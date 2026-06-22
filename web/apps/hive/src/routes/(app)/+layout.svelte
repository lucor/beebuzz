<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto, preloadCode } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { logger } from '@beebuzz/shared/logger';
	import { safeLogger } from '$lib/devmode/safe-logger';
	import { captureHiveError } from '$lib/devmode/error-capture';
	import {
		startConsoleDiagnosticsCapture,
		stopConsoleDiagnosticsCapture
	} from '$lib/devmode/console-diagnostics';
	import {
		isHiveDiagnosticEvent,
		isHiveLogScope,
		type HiveLogData,
		type HiveDiagnosticDescriptor
	} from '$lib/devmode/types';
	import { developerSettings } from '$lib/devmode/settings';
	import { HIVE_BOUNDARY, HIVE_TRANSPORT, HIVE_DIAGNOSTIC } from '$lib/devmode/types';
	import { loadDeveloperSettings } from '$lib/devmode/storage';
	import { health } from '@beebuzz/shared/stores/health.svelte';
	import { toast } from '@beebuzz/shared/stores';
	import { paired } from '$lib/stores/paired.svelte';
	import { connectivity } from '$lib/stores/connectivity.svelte';
	import { notificationsStore } from '$lib/stores/notifications.svelte';
	import { getVapidKey, registerServiceWorker } from '$lib/services/push';
	import { deviceKeysRepository } from '$lib/services/device-keys-repository';
	import { notificationsRepository } from '$lib/services/notifications-repository';
	import { bootstrapAppShell } from '$lib/services/app-bootstrap';
	import { syncRecentNotifications } from '$lib/services/notification-sync';
	import { cleanupStalePairingState } from '$lib/services/startup-recovery';
	import { formatStartupError } from '$lib/services/startup-error';
	import {
		LOCAL_REPAIR_REQUIRED_REASON,
		PUSH_STATE_STATUS,
		reconcilePushState,
		type PushStateResult
	} from '$lib/onboarding.svelte';
	import { withTimeout } from '$lib/utils/async';
	import {
		Menu,
		X,
		Activity,
		Hexagon,
		Bug,
		TriangleAlert,
		ExternalLink,
		Loader,
		type Icon
	} from '@lucide/svelte';
	import type { PushMessage } from '@beebuzz/shared/types';

	const GITHUB_RELEASES_URL = 'https://codeberg.org/beebuzz/cli/releases';
	const STARTUP_TIMEOUT_MS = 10000;

	/** Runs a background task with a logged warning instead of throwing. */
	const runOptional = async (label: string, task: () => Promise<unknown>): Promise<void> => {
		try {
			await task();
		} catch (error) {
			logger.warn(`${label} failed (non-blocking)`, { error: String(error) });
		}
	};

	let { children }: { children: import('svelte').Snippet } = $props();

	type NavItem = {
		label: string;
		href: string;
		icon: typeof Icon;
	};

	let sidebarOpen = $state(false);
	let ready = $state(false);
	let startupError = $state<string | null>(null);
	let updateAvailable = $state(false);
	let activatingUpdate = $state(false);
	let hasSeenController = $state(false);
	let reloadingForUpdate = $state(false);
	let pushStateResult = $state<PushStateResult | null>(null);

	let currentPath = $derived(page.url.pathname);

	let devModeEnabled = $state(false);

	$effect(() => {
		const unsub = developerSettings.subscribe((s) => {
			devModeEnabled = s.enabled;
		});
		return () => unsub();
	});

	const navItems = $derived.by(() => {
		const items: NavItem[] = [
			{ label: 'Hive', href: resolve('/'), icon: Hexagon },
			{ label: 'My Device', href: resolve('/device'), icon: Activity }
		];
		if (devModeEnabled) {
			items.push({ label: 'Developer Mode', href: resolve('/developer'), icon: Bug });
		}
		return items;
	});

	/** Handles messages from the service worker. */
	const handleServiceWorkerMessage = (event: MessageEvent<PushMessage>) => {
		const eventData = event.data as Record<string, unknown>;
		if (eventData?.type === 'DIAGNOSTIC_EVENT') {
			const msg = eventData;
			if (
				typeof msg.scope === 'string' &&
				typeof msg.event === 'string' &&
				typeof msg.message === 'string' &&
				isHiveLogScope(msg.scope) &&
				isHiveDiagnosticEvent(msg.event)
			) {
				const data =
					msg.data && typeof msg.data === 'object' && !Array.isArray(msg.data)
						? (msg.data as HiveLogData)
						: undefined;
				const descriptor = {
					scope: msg.scope,
					event: msg.event
				} satisfies HiveDiagnosticDescriptor;
				safeLogger.log(descriptor, msg.message, data);
			}
			return;
		}

		if (event.data?.type === 'PUSH_RECEIVED') {
			if (event.data.deviceId !== notificationsStore.activeDeviceId) return;
			notificationsStore.add(
				event.data.title,
				event.data.body,
				event.data.topic,
				event.data.topicId ?? null,
				event.data.sentAt,
				event.data.attachment,
				event.data.priority,
				event.data.id
			);
			safeLogger.log(HIVE_DIAGNOSTIC.NOTIFICATION_IMPORTED, 'Push imported into app state', {
				notification_id: event.data.id,
				push_trace_id: event.data.pushTraceId,
				boundary: HIVE_BOUNDARY.INTERNAL,
				transport: HIVE_TRANSPORT.POST_MESSAGE
			});
			void syncNotificationsFromBackend();
		} else if (event.data?.type === 'NOTIFICATION_CLICKED') {
			const clickedNotification = event.data.notification;
			// User tapped a system notification — always reload from IndexedDB
			// in case the postMessage for the original push didn't reach us
			// (common on iOS, and required when the click happens before the
			// app shell has activated a device id). Run this BEFORE any
			// device-id gating so the durable record is still recovered when
			// `deviceId` on the click message is missing or stale.
			void notificationsStore.loadFromIndexedDB();
			if (clickedNotification?.deviceId !== notificationsStore.activeDeviceId) return;
			if (
				clickedNotification?.id &&
				typeof clickedNotification.title === 'string' &&
				typeof clickedNotification.body === 'string' &&
				typeof clickedNotification.sentAt === 'string'
			) {
				notificationsStore.add(
					clickedNotification.title,
					clickedNotification.body,
					clickedNotification.topic ?? null,
					clickedNotification.topicId ?? null,
					clickedNotification.sentAt,
					clickedNotification.attachment,
					clickedNotification.priority,
					clickedNotification.id
				);
				safeLogger.log(
					HIVE_DIAGNOSTIC.NOTIFICATION_IMPORTED,
					'Clicked notification imported into app state',
					{
						notification_id: clickedNotification.id,
						push_trace_id: clickedNotification.pushTraceId,
						boundary: HIVE_BOUNDARY.INTERNAL,
						transport: HIVE_TRANSPORT.POST_MESSAGE
					}
				);
			}
		} else if (event.data?.type === 'SUBSCRIPTION_CHANGED') {
			paired.clear();
			toast.info('Push subscription expired. Please reconnect.');
		}
	};

	let swMessageListener: ((event: MessageEvent<PushMessage>) => void) | null = null;

	// iOS Safari PWA workaround: the SW's clients.matchAll() returns zero clients
	// even when the app is in the foreground, so postMessage never arrives.
	// This is a confirmed WebKit bug — fix merged in WebKit PR #11848:
	// https://github.com/WebKit/WebKit/pull/11848
	// Tracked via: https://github.com/firebase/firebase-js-sdk/issues/7309
	// Until the fix ships on all target iOS versions, poll IndexedDB while
	// visible to pick up notifications written by the SW.
	// Cost is negligible: one getAll() that no-ops when empty.
	const POLL_INTERVAL_MS = 3000;
	let pollTimer: ReturnType<typeof setInterval> | null = null;

	const startPolling = () => {
		if (pollTimer) return;
		pollTimer = setInterval(() => {
			void notificationsStore.loadFromIndexedDB();
		}, POLL_INTERVAL_MS);
	};

	const stopPolling = () => {
		if (!pollTimer) return;
		clearInterval(pollTimer);
		pollTimer = null;
	};

	const handleVisibilityChange = () => {
		if (document.visibilityState === 'visible' && ready) {
			void notificationsStore.loadFromIndexedDB();
			startPolling();
			void refreshRemoteState();
		} else {
			stopPolling();
		}
	};

	const syncWaitingWorker = (registration: ServiceWorkerRegistration | undefined | null) => {
		updateAvailable = Boolean(registration?.waiting);
		if (!updateAvailable) {
			activatingUpdate = false;
		}
	};

	const syncNotificationsFromBackend = async () => {
		if (!connectivity.online) return;
		try {
			const credentials = await deviceKeysRepository.getDeviceCredentials();
			if (!credentials) return;
			await syncRecentNotifications(credentials.deviceId, credentials.deviceToken);
		} catch (error) {
			logger.warn('Notification HTTPS sync failed', { error: String(error) });
		}
	};

	/** Background reconcile — avoids blocking the shell when the backend is unreachable. */
	const backgroundReconcilePushState = async () => {
		if (!connectivity.online) return;
		try {
			const pushState = await withTimeout(
				reconcilePushState(),
				STARTUP_TIMEOUT_MS,
				'Push state validation'
			);
			pushStateResult = pushState;

			if (pushState.status === PUSH_STATE_STATUS.OK) return;
			if (pushState.status === PUSH_STATE_STATUS.RECONNECT_REQUIRED) return;
			if (pushState.status === PUSH_STATE_STATUS.TRANSIENT_BACKEND_ERROR) return;

			// Local-only repair — key lost or subscription expired.
			await cleanupStalePairingState();
			paired.clear();
			toast.info(
				pushState.reason === LOCAL_REPAIR_REQUIRED_REASON.KEYS_LOST
					? 'Device key missing. Please reconnect this device.'
					: 'Push subscription expired. Please reconnect.'
			);
			await goto('/pair');
		} catch (error) {
			logger.warn('Push state reconcile failed (non-blocking)', { error: String(error) });
		}
	};

	/** Called when the browser transitions from offline → online. */
	const handleOnlineEvent = () => {
		if (!ready) return;
		safeLogger.log(HIVE_DIAGNOSTIC.APP_STARTED, 'Connection restored, refreshing state');
		void refreshRemoteState();
	};

	/** Soft-failing wrapper around checkForServiceWorkerUpdate. */
	const refreshServiceWorkerUpdate = async () => {
		try {
			const registration = await navigator.serviceWorker.getRegistration();
			if (!registration) {
				updateAvailable = false;
				activatingUpdate = false;
				return;
			}
			await registration.update();
			syncWaitingWorker(registration);
		} catch (error) {
			logger.warn('SW update check failed (non-blocking)', { error: String(error) });
		}
	};

	/** Preloads route chunks that should remain available when the device goes offline. */
	const preloadOfflineRoutes = async () => {
		if (!connectivity.online) return;
		await runOptional('Offline route preload', async () => {
			await Promise.all([preloadCode(resolve('/device')), preloadCode(resolve('/developer'))]);
		});
	};

	/**
	 * Post-pairing startup checks — local SW bookkeeping first, then
	 * optional VAPID/health/update when online.
	 */
	const runRemoteStartupChecks = async (registration: ServiceWorkerRegistration) => {
		watchServiceWorkerRegistration(registration);
		safeLogger.log(
			HIVE_DIAGNOSTIC.SERVICE_WORKER_REGISTERED,
			'Service worker registered and active'
		);
		syncWaitingWorker(registration);

		if (!connectivity.online) return;

		await runOptional('VAPID key fetch', () =>
			withTimeout(getVapidKey(), STARTUP_TIMEOUT_MS, 'VAPID key fetch')
		);

		if (health.status === 'unknown' && !health.loading) {
			await runOptional('Health check', () =>
				withTimeout(health.check(), STARTUP_TIMEOUT_MS, 'Health check')
			);
		}

		await refreshServiceWorkerUpdate();
		await preloadOfflineRoutes();
	};

	/** Refreshes remote state after the shell is ready: health, push reconcile, sync, SW update. */
	const refreshRemoteState = () => {
		if (!connectivity.online) return;
		void health.check();
		void backgroundReconcilePushState();
		void syncNotificationsFromBackend();
		void refreshServiceWorkerUpdate();
		void preloadOfflineRoutes();
	};

	const handleServiceWorkerControllerChange = () => {
		if (!hasSeenController) {
			hasSeenController = true;
			return;
		}

		if (reloadingForUpdate) {
			return;
		}

		reloadingForUpdate = true;
		window.location.reload();
	};

	const watchServiceWorkerRegistration = (registration: ServiceWorkerRegistration) => {
		syncWaitingWorker(registration);

		registration.addEventListener('updatefound', () => {
			const installingWorker = registration.installing;
			if (!installingWorker) {
				syncWaitingWorker(registration);
				return;
			}

			installingWorker.addEventListener('statechange', () => {
				if (installingWorker.state === 'installed') {
					syncWaitingWorker(registration);
				}
			});
		});
	};

	const activateServiceWorkerUpdate = async () => {
		activatingUpdate = true;

		const registration = await navigator.serviceWorker.getRegistration();
		const waitingWorker = registration?.waiting;
		if (!waitingWorker) {
			syncWaitingWorker(registration);
			return;
		}

		waitingWorker.postMessage({ type: 'SKIP_WAITING' });
	};

	/** Boots the paired app shell and surfaces startup failures instead of hanging forever. */
	const bootstrapApp = async () => {
		startupError = null;
		ready = false;
		stopPolling();
		notificationsStore.deactivateDevice();
		if (swMessageListener) {
			navigator.serviceWorker.removeEventListener('message', swMessageListener);
			swMessageListener = null;
		}
		navigator.serviceWorker.removeEventListener(
			'controllerchange',
			handleServiceWorkerControllerChange
		);

		const devSettings = await loadDeveloperSettings();
		developerSettings.set(devSettings);
		startConsoleDiagnosticsCapture();

		safeLogger.log(HIVE_DIAGNOSTIC.APP_STARTED, 'Hive app bootstrapping');

		try {
			// The bootstrap order matters: paired state and credentials determine the
			// notification history scope, so no local notification cache is loaded until
			// the current backend device ID is known.
			const { isPaired } = await bootstrapAppShell({
				registerServiceWorker: () =>
					withTimeout(registerServiceWorker(), STARTUP_TIMEOUT_MS, 'Service worker registration'),
				checkPaired: () => withTimeout(paired.check(), STARTUP_TIMEOUT_MS, 'Paired device check'),
				getDeviceId: async () => {
					const credentials = await withTimeout(
						deviceKeysRepository.getDeviceCredentials(),
						STARTUP_TIMEOUT_MS,
						'Device credentials load'
					);
					return credentials?.deviceId ?? null;
				},
				activateNotifications: (deviceId) => {
					notificationsStore.activateDevice(deviceId);
				},
				attachServiceWorkerListeners: () => {
					navigator.serviceWorker.addEventListener('message', handleServiceWorkerMessage);
					swMessageListener = handleServiceWorkerMessage;
					hasSeenController = navigator.serviceWorker.controller !== null;
					navigator.serviceWorker.addEventListener(
						'controllerchange',
						handleServiceWorkerControllerChange
					);
				},
				migrateLegacyNotifications: (deviceId) =>
					withTimeout(
						notificationsRepository.migrateLegacyNotifications(deviceId),
						STARTUP_TIMEOUT_MS,
						'Legacy notification migration'
					),
				loadPersistedNotifications: (phase) =>
					withTimeout(
						notificationsStore.loadFromIndexedDB(),
						STARTUP_TIMEOUT_MS,
						phase === 'initial'
							? 'Initial notification cache load'
							: 'Final notification cache load'
					),
				runPostPairingChecks: async (registration) => {
					await runOptional('Post-pairing startup checks', () =>
						runRemoteStartupChecks(registration)
					);
				}
			});

			if (!isPaired) {
				await cleanupStalePairingState();
				safeLogger.log(
					HIVE_DIAGNOSTIC.PAIRING_RECONNECT_REQUIRED,
					'Device not paired, redirecting to pairing'
				);
				await goto('/pair');
				return;
			}

			// Local cache loaded — shell is ready immediately.
			ready = true;
			safeLogger.log(HIVE_DIAGNOSTIC.APP_STARTED, 'Hive app ready');
			startPolling();

			// Remote state refresh runs asynchronously; defers to handleOnlineEvent when offline.
			void refreshRemoteState();
		} catch (error: unknown) {
			startupError = formatStartupError(error);
			logger.error('Hive app bootstrap failed', { error: String(error) });
			safeLogger.log(HIVE_DIAGNOSTIC.APP_BOOTSTRAP_FAILED, startupError);
			void captureHiveError({
				scope: 'app',
				event: 'app.bootstrap_failed',
				message: 'Hive app bootstrap failed',
				error,
				severity: 'error'
			});
		}
	};

	onMount(() => {
		connectivity.init();
		document.addEventListener('visibilitychange', handleVisibilityChange);
		window.addEventListener('online', handleOnlineEvent);
		void bootstrapApp();
	});

	onDestroy(() => {
		if (swMessageListener) {
			navigator.serviceWorker.removeEventListener('message', swMessageListener);
			swMessageListener = null;
		}
		navigator.serviceWorker.removeEventListener(
			'controllerchange',
			handleServiceWorkerControllerChange
		);
		document.removeEventListener('visibilitychange', handleVisibilityChange);
		window.removeEventListener('online', handleOnlineEvent);
		stopPolling();
		connectivity.destroy();
		stopConsoleDiagnosticsCapture();
	});

	// Reactive guard: if paired state is lost (e.g. from SW message), redirect
	$effect(() => {
		if (ready && !paired.isPaired) {
			void goto('/pair');
		}
	});

	type DeviceStatusTone = 'healthy' | 'check' | 'offline' | 'checking';

	const navbarStatusTone = $derived.by<DeviceStatusTone>(() => {
		if (!connectivity.online) {
			return 'offline';
		}

		if (paired.loading || health.loading || health.status === 'unknown') {
			return 'checking';
		}

		if (!paired.isPaired) {
			return 'check';
		}

		if (pushStateResult?.status === PUSH_STATE_STATUS.RECONNECT_REQUIRED) {
			return 'check';
		}

		if (pushStateResult?.status === PUSH_STATE_STATUS.TRANSIENT_BACKEND_ERROR) {
			return health.status === 'error' ? 'offline' : 'check';
		}

		if (health.status === 'error') {
			return 'offline';
		}

		return 'healthy';
	});

	const navbarStatusText = $derived.by(() => {
		switch (navbarStatusTone) {
			case 'checking':
				return 'Checking';
			case 'check':
				return 'Attention';
			case 'offline':
				return 'Offline';
			default:
				return 'Healthy';
		}
	});

	const navbarStatusLabel = $derived.by(() => {
		switch (navbarStatusTone) {
			case 'checking':
				return 'Device status: checking';
			case 'check':
				return 'Device status: attention needed';
			case 'offline':
				return 'Device status: offline';
			default:
				return 'Device status: healthy';
		}
	});

	const navbarStatusDotClass = $derived.by(() => {
		switch (navbarStatusTone) {
			case 'checking':
				return 'bg-warning';
			case 'check':
				return 'bg-warning';
			case 'offline':
				return 'bg-error';
			default:
				return 'bg-success';
		}
	});

	const navbarStatusChipClass = $derived.by(() => {
		switch (navbarStatusTone) {
			case 'checking':
				return 'border-warning/40 bg-warning/10 text-warning-content';
			case 'check':
				return 'border-warning/40 bg-warning/10 text-warning-content';
			case 'offline':
				return 'border-error/30 bg-error/10 text-error-content';
			default:
				return 'border-base-300 bg-base-100 text-base-content/80';
		}
	});
</script>

{#if !ready}
	<main class="flex min-h-dvh items-center justify-center bg-base-100 px-4">
		{#if startupError}
			<div class="w-full max-w-md rounded-2xl border border-warning/30 bg-base-100 p-6 shadow-sm">
				<div class="flex items-start gap-3">
					<TriangleAlert size={20} class="mt-0.5 shrink-0 text-warning" aria-hidden="true" />
					<div class="space-y-4">
						<div class="space-y-1">
							<h1 class="text-lg font-semibold text-base-content">Startup failed</h1>
							<p class="text-sm text-base-content/70">{startupError}</p>
						</div>
						<div class="flex flex-wrap gap-2">
							<button
								type="button"
								class="btn btn-primary btn-sm"
								onclick={() => void bootstrapApp()}
							>
								Retry
							</button>
							<a href={resolve('/pair')} class="btn btn-ghost btn-sm"> Open Pairing </a>
						</div>
					</div>
				</div>
			</div>
		{:else}
			<span class="loading loading-spinner loading-lg text-primary"></span>
		{/if}
	</main>
{:else}
	<div class="flex flex-col h-screen">
		<!-- Fixed Navbar -->
		<nav class="navbar bg-base-100 shadow-sm fixed top-0 left-0 right-0 z-50 px-4 md:px-8">
			<!-- Left: Hamburger + Logo -->
			<div class="navbar-start flex items-center gap-4">
				<button
					aria-label="Toggle sidebar"
					class="btn btn-square btn-ghost lg:hidden"
					onclick={() => (sidebarOpen = !sidebarOpen)}
				>
					{#if sidebarOpen}
						<X size={24} />
					{:else}
						<Menu size={24} />
					{/if}
				</button>

				<a href={resolve('/')} class="hidden sm:flex">
					<BeeBuzzLogo variant="full" class="h-10 w-auto" />
				</a>
			</div>

			<!-- Center: Logo (mobile only) -->
			<div class="navbar-center sm:hidden">
				<a href={resolve('/')} class="flex items-center">
					<BeeBuzzLogo variant="img" class="h-9 w-9" />
				</a>
			</div>

			<div class="navbar-end">
				<div class="tooltip tooltip-bottom" data-tip={navbarStatusLabel}>
					<a
						href={resolve('/device')}
						class={`inline-flex items-center gap-2 rounded-full border px-2.5 py-1.5 text-xs font-medium transition-colors hover:border-base-content/20 hover:bg-base-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 ${navbarStatusChipClass}`}
						aria-label={navbarStatusLabel}
					>
						{#if navbarStatusTone === 'checking'}
							<Loader size={12} class="animate-spin" aria-hidden="true" />
						{:else}
							<span class={`h-2.5 w-2.5 rounded-full ${navbarStatusDotClass}`}></span>
						{/if}
						<span class="hidden sm:inline">Device status</span>
						<span class="hidden md:inline text-current/80">{navbarStatusText}</span>
						<span class="sm:hidden">{navbarStatusText}</span>
					</a>
				</div>
			</div>
		</nav>

		<!-- Main Layout with Sidebar -->
		<div class="flex flex-1 pt-16 overflow-hidden">
			<!-- Sidebar Overlay (mobile) -->
			{#if sidebarOpen}
				<button
					class="fixed inset-0 bg-black/50 z-30 lg:hidden"
					onclick={() => (sidebarOpen = false)}
					aria-label="Close sidebar"
					type="button"
				></button>
			{/if}

			<!-- Sidebar -->
			<aside
				class="fixed left-0 top-16 bottom-0 w-64 bg-base-200 border-r border-base-300 shadow-lg transition-transform duration-300 z-40 lg:relative lg:top-0 overflow-y-auto
					{sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}"
			>
				<div class="flex min-h-full flex-col p-4 md:p-8">
					<ul class="menu w-full gap-2 text-base-content">
						{#each navItems as item (item.href)}
							<li>
								<a
									href={item.href}
									class={`rounded-lg transition-colors ${
										currentPath === item.href
											? 'bg-primary text-primary-content font-semibold'
											: 'hover:bg-base-300'
									}`}
									onclick={() => (sidebarOpen = false)}
								>
									<item.icon size={20} aria-hidden="true" />
									<span>{item.label}</span>
								</a>
							</li>
						{/each}
					</ul>
				</div>
			</aside>

			<!-- Main Content -->
			<main class="flex-1 overflow-y-auto bg-base-100">
				<div class="p-4 md:p-8 max-w-6xl mx-auto">
					{#if updateAvailable}
						<div class="sticky top-0 z-20 mb-4">
							<div
								class="alert items-start gap-3 px-4 py-3 shadow-sm border border-primary/20 bg-primary/10"
								role="status"
							>
								<TriangleAlert size={20} class="mt-0.5 shrink-0 text-primary" aria-hidden="true" />
								<div class="min-w-0 flex-1 w-full">
									<div class="flex w-full items-center gap-3 md:hidden">
										<p class="min-w-0 flex-1 font-semibold text-base-content">
											A new version is available.
										</p>
										<button
											type="button"
											class="btn btn-primary btn-sm ml-auto shrink-0"
											onclick={() => void activateServiceWorkerUpdate()}
											disabled={activatingUpdate}
										>
											{activatingUpdate ? 'Updating...' : 'Update'}
										</button>
									</div>
									<a
										href={GITHUB_RELEASES_URL}
										class="link link-hover mt-2 inline-flex items-center gap-1 text-sm font-medium md:hidden"
										target="_blank"
										rel="noopener noreferrer"
									>
										<ExternalLink size={14} aria-hidden="true" />
										Release notes
									</a>
									<div class="hidden w-full items-center gap-3 md:flex">
										<p class="min-w-0 flex-1 font-semibold text-base-content">
											A new version is available.
										</p>
										<div class="ml-auto flex shrink-0 items-center gap-3">
											<button
												type="button"
												class="btn btn-primary btn-sm shrink-0"
												onclick={() => void activateServiceWorkerUpdate()}
												disabled={activatingUpdate}
											>
												{activatingUpdate ? 'Updating...' : 'Update'}
											</button>
											<a
												href={GITHUB_RELEASES_URL}
												class="link link-hover inline-flex items-center gap-1 text-sm font-medium"
												target="_blank"
												rel="noopener noreferrer"
											>
												<ExternalLink size={14} aria-hidden="true" />
												Release notes
											</a>
										</div>
									</div>
								</div>
							</div>
						</div>
					{/if}
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{/if}
