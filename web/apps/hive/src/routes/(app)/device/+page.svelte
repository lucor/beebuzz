<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { toast } from '@beebuzz/shared/stores';
	import { health } from '@beebuzz/shared/stores/health.svelte';
	import { ApiError } from '@beebuzz/shared/errors';
	import { logger } from '@beebuzz/shared/logger';
	import { notificationsStore } from '$lib/stores/notifications.svelte';
	import { paired } from '$lib/stores/paired.svelte';
	import {
		RECONNECT_REQUIRED_REASON,
		type ReconnectRequiredReason
	} from '$lib/services/pairing-check';
	import { PUSH_STATE_STATUS, reconcilePushState } from '$lib/onboarding.svelte';
	import {
		deleteAllKeys,
		getStoredDeviceKey,
		type StoredDeviceKey
	} from '$lib/services/encryption';
	import { unsubscribeFromPush } from '$lib/services/push';
	import { cleanupStalePairingState } from '$lib/services/startup-recovery';
	import {
		Bell,
		Link,
		Globe,
		Shield,
		RefreshCw,
		Server,
		Copy,
		Check,
		ChevronDown,
		Loader,
		Unplug,
		ExternalLink,
		CircleCheck,
		CircleX,
		CircleAlert,
		TriangleAlert
	} from '@lucide/svelte';

	const STATUS_PAGE_URL = 'https://status.beebuzz.app';

	type ServiceWorkerState = 'active' | 'installing' | 'waiting' | 'missing';

	interface DeviceSnapshot {
		encryptionKey: StoredDeviceKey | null;
		notificationPermission: NotificationPermission | 'unsupported';
		hasPushSubscription: boolean;
		serviceWorkerState: ServiceWorkerState;
		updateAvailable: boolean;
	}

	let loading = $state(true);
	let reconnectRequiredReason = $state<ReconnectRequiredReason | null>(null);
	let backendUnavailable = $state(false);
	let reconnecting = $state(false);
	let copyingPublicKey = $state(false);
	let disconnecting = $state(false);
	let techDetailsOpen = $state(false);
	let disconnectDialog = $state<HTMLDialogElement | undefined>(undefined);

	let snapshot = $state<DeviceSnapshot>({
		encryptionKey: null,
		notificationPermission: 'default',
		hasPushSubscription: false,
		serviceWorkerState: 'missing',
		updateAvailable: false
	});

	/** Loads the current device state shown in the overview cards. */
	const loadDeviceSnapshot = async () => {
		const registration = await navigator.serviceWorker.getRegistration();
		const subscription = registration ? await registration.pushManager.getSubscription() : null;
		const encryptionKey = await getStoredDeviceKey();

		let serviceWorkerState: ServiceWorkerState = 'missing';
		if (registration?.waiting) {
			serviceWorkerState = 'waiting';
		} else if (registration?.installing) {
			serviceWorkerState = 'installing';
		} else if (registration?.active) {
			serviceWorkerState = 'active';
		}

		snapshot = {
			encryptionKey,
			notificationPermission:
				typeof Notification === 'undefined' ? 'unsupported' : Notification.permission,
			hasPushSubscription: subscription !== null,
			serviceWorkerState,
			updateAvailable: Boolean(registration?.waiting)
		};

		const pushState = await reconcilePushState();
		reconnectRequiredReason =
			pushState.status === PUSH_STATE_STATUS.RECONNECT_REQUIRED ? pushState.reason : null;
		backendUnavailable = pushState.status === PUSH_STATE_STATUS.TRANSIENT_BACKEND_ERROR;
	};

	onMount(() => {
		const init = async () => {
			try {
				if (health.status === 'unknown' && !health.loading) {
					await health.check();
				}
				await loadDeviceSnapshot();
			} finally {
				loading = false;
			}
		};

		void init();
	});

	$effect(() => {
		if (!disconnectDialog) return;
		if (disconnecting) {
			disconnectDialog.showModal();
		} else {
			disconnectDialog.close();
		}
	});

	const notificationsReady = $derived(
		!reconnectRequiredReason &&
			snapshot.notificationPermission === 'granted' &&
			snapshot.hasPushSubscription &&
			health.status === 'ok' &&
			(snapshot.serviceWorkerState === 'active' || snapshot.serviceWorkerState === 'waiting')
	);

	const summaryTitle = $derived.by(() => {
		if (notificationsReady) return 'Ready to receive notifications';
		if (reconnectRequiredReason) return reconnectReasonTitle;
		if (backendUnavailable) return 'Server sync pending';
		return 'This device needs attention';
	});

	const heroBorderClass = $derived(notificationsReady ? 'border-base-300' : 'border-warning/30');
	const heroBgClass = $derived(notificationsReady ? 'bg-base-200/60' : 'bg-warning/5');
	const heroDotClass = $derived(notificationsReady ? 'bg-success' : 'bg-warning');

	const summaryBody = $derived.by(() => {
		if (notificationsReady && snapshot.serviceWorkerState === 'waiting') {
			return 'This device is ready to receive notifications. A newer version is available, and updating is recommended for the latest fixes and improvements.';
		}

		if (notificationsReady) {
			return 'This device is ready to receive notifications.';
		}

		if (reconnectRequiredReason) {
			return reconnectReasonBody(reconnectRequiredReason);
		}

		if (backendUnavailable) {
			return 'BeeBuzz is temporarily unreachable, so remote device health could not be verified.';
		}

		if (snapshot.notificationPermission !== 'granted') {
			return 'Notifications are not allowed in this browser, so new pushes cannot be delivered.';
		}

		if (!snapshot.hasPushSubscription) {
			return 'This device is no longer connected for push notifications. Pair this device again to restore notifications.';
		}

		if (health.status === 'error') {
			return 'The device is paired, but the server is currently unavailable.';
		}

		return 'The app is not fully ready yet, so notifications may be unreliable.';
	});

	/* --- Essential checks derived state --- */

	const notificationStatus = $derived.by(() => {
		switch (snapshot.notificationPermission) {
			case 'granted':
				return { label: 'Allowed', tone: 'badge-success' as const, icon: CircleCheck };
			case 'default':
				return { label: 'Not decided', tone: 'badge-warning' as const, icon: CircleAlert };
			case 'unsupported':
				return { label: 'Not supported', tone: 'badge-error' as const, icon: CircleX };
			default:
				return { label: 'Blocked', tone: 'badge-error' as const, icon: CircleX };
		}
	});

	const connectionStatus = $derived.by(() => {
		if (reconnectRequiredReason) {
			return { label: 'Unknown', tone: 'badge-neutral' as const, icon: CircleAlert };
		}
		if (snapshot.hasPushSubscription) {
			return { label: 'Active', tone: 'badge-success' as const, icon: CircleCheck };
		}
		return { label: 'Not connected', tone: 'badge-error' as const, icon: CircleX };
	});

	const serverStatus = $derived.by(() => {
		if (health.loading || health.status === 'unknown') {
			return {
				label: 'Checking',
				tone: 'badge-warning' as const,
				loading: true,
				icon: CircleAlert
			};
		}
		if (health.status === 'ok') {
			return {
				label: 'Connected',
				tone: 'badge-success' as const,
				loading: false,
				icon: CircleCheck
			};
		}
		return { label: 'Unavailable', tone: 'badge-error' as const, loading: false, icon: CircleX };
	});

	const pairingVerificationStatus = $derived.by(() => {
		if (reconnectRequiredReason) {
			return {
				label: 'Reconnect',
				tone: 'badge-warning' as const,
				icon: TriangleAlert
			};
		}
		if (backendUnavailable) {
			return {
				label: 'Offline',
				tone: 'badge-neutral' as const,
				icon: CircleAlert
			};
		}
		return {
			label: 'Paired',
			tone: 'badge-success' as const,
			icon: CircleCheck
		};
	});

	/* --- Technical details derived state --- */

	const secureDeliveryStatus = $derived(
		snapshot.encryptionKey
			? { label: 'Configured', tone: 'badge-success', icon: CircleCheck }
			: { label: 'Not configured', tone: 'badge-error', icon: CircleX }
	);

	const serviceWorkerStatus = $derived.by(() => {
		switch (snapshot.serviceWorkerState) {
			case 'active':
				return { label: 'Active', tone: 'badge-success', icon: CircleCheck };
			case 'installing':
				return { label: 'Installing', tone: 'badge-warning', icon: CircleAlert };
			case 'waiting':
				return { label: 'Update ready', tone: 'badge-warning', icon: CircleAlert };
			default:
				return { label: 'Missing', tone: 'badge-error', icon: CircleX };
		}
	});

	const publicKeyCreatedAt = $derived.by(() => {
		if (!snapshot.encryptionKey?.createdAt) {
			return null;
		}

		return new Intl.DateTimeFormat(undefined, {
			dateStyle: 'medium',
			timeStyle: 'short'
		}).format(new Date(snapshot.encryptionKey.createdAt));
	});

	/** Copies the public age recipient to the clipboard. */
	const copyPublicKey = async () => {
		if (!snapshot.encryptionKey?.recipient) return;

		try {
			copyingPublicKey = true;
			await navigator.clipboard.writeText(snapshot.encryptionKey.recipient);
			toast.success('Public key copied');
		} catch (error: unknown) {
			logger.error('Public key copy failed', { error: String(error) });
			toast.error('Failed to copy public key');
		} finally {
			window.setTimeout(() => {
				copyingPublicKey = false;
			}, 1200);
		}
	};

	/** Disconnects this device and sends the user back to the pairing flow. */
	const handleDisconnect = async () => {
		disconnecting = false;
		disconnectDialog?.close();

		try {
			await unsubscribeFromPush();
			await deleteAllKeys();
			notificationsStore.clearAll();
			paired.clear();
			toast.success('Device disconnected');
			await goto('/pair');
		} catch (error: unknown) {
			logger.error('Device disconnect failed', { error: String(error) });
			toast.error(error instanceof ApiError ? error.userMessage : 'Disconnect failed');
			paired.clear();
			await goto('/pair');
		}
	};

	const handleReconnect = async () => {
		reconnecting = true;
		try {
			await cleanupStalePairingState();
			notificationsStore.clearAll();
			paired.clear();
			toast.info('Please enter a new pairing code to reconnect.');
			await goto('/pair');
		} catch (error: unknown) {
			logger.error('Reconnect failed', { error: String(error) });
			toast.error('Reconnect failed. Please try again.');
		} finally {
			reconnecting = false;
		}
	};

	function handleDisconnectDialogClose() {
		disconnecting = false;
	}

	const reconnectReasonTitle = $derived.by(() => {
		if (!reconnectRequiredReason) {
			return 'Reconnect required';
		}
		switch (reconnectRequiredReason) {
			case RECONNECT_REQUIRED_REASON.SUBSCRIPTION_GONE:
				return 'Push subscription lost';
			case RECONNECT_REQUIRED_REASON.INVALID_DEVICE_TOKEN:
				return 'Device credentials expired';
			case RECONNECT_REQUIRED_REASON.MISSING_DEVICE_TOKEN:
				return 'Device credentials missing';
			case RECONNECT_REQUIRED_REASON.UNPAIRED:
				return 'Device no longer paired';
			default:
				return 'Pairing incomplete';
		}
	});

	function reconnectReasonBody(reason: ReconnectRequiredReason): string {
		switch (reason) {
			case RECONNECT_REQUIRED_REASON.SUBSCRIPTION_GONE:
				return "The push service invalidated this device's subscription. Generate a new pairing code from your account to reconnect.";
			case RECONNECT_REQUIRED_REASON.INVALID_DEVICE_TOKEN:
				return 'This device no longer has valid BeeBuzz pairing credentials. Reconnect it using a new pairing code from your account.';
			case RECONNECT_REQUIRED_REASON.MISSING_DEVICE_TOKEN:
				return 'This device is missing the local credentials needed to verify its pairing status. Reconnect it once to restore them.';
			case RECONNECT_REQUIRED_REASON.UNPAIRED:
				return 'BeeBuzz no longer considers this device paired. Generate a new pairing code from your account and reconnect.';
			case RECONNECT_REQUIRED_REASON.PENDING:
				return 'BeeBuzz still considers this device pairing incomplete. Generate a fresh pairing code from your account and reconnect.';
		}
	}
</script>

{#if loading}
	<div class="flex min-h-[40vh] items-center justify-center">
		<span class="loading loading-spinner loading-lg text-primary"></span>
	</div>
{:else}
	<div class="space-y-6">
		<!-- Hero summary -->
		<section class={`rounded-3xl border ${heroBorderClass} ${heroBgClass} p-6 shadow-sm`}>
			<div class="space-y-4">
				<div class="space-y-2">
					<div class="flex items-center gap-2">
						<span class={`h-2.5 w-2.5 rounded-full ${heroDotClass}`}></span>
						<p class="text-sm font-semibold uppercase tracking-[0.18em] text-base-content/60">
							My Device
						</p>
					</div>
					<h1 class="text-3xl font-bold text-base-content">{summaryTitle}</h1>
					<p class="text-base-content/75">{summaryBody}</p>
				</div>
				{#if reconnectRequiredReason}
					<div class="flex justify-start">
						<button
							type="button"
							class="btn btn-warning btn-sm"
							onclick={() => void handleReconnect()}
							disabled={reconnecting}
						>
							{reconnecting ? 'Reconnecting...' : 'Reconnect'}
						</button>
					</div>
				{/if}
			</div>
		</section>

		<!-- Essential checks -->
		<section class="card border border-base-300 bg-base-100 shadow-sm">
			<div class="card-body gap-5">
				<h2 class="card-title text-xl">Essential checks</h2>
				<div class="space-y-3">
					<!-- Notifications -->
					<div
						class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
					>
						<div class="flex items-center gap-3">
							<Bell size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
							<div>
								<p class="font-medium text-base-content">Notifications</p>
								<p class="text-sm text-base-content/65">Browser permission for this device.</p>
							</div>
						</div>
						<span class={`badge ${notificationStatus.tone} gap-1`}>
							<notificationStatus.icon size={12} aria-hidden="true" />
							{notificationStatus.label}
						</span>
					</div>

					<!-- Device connection -->
					<div
						class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
					>
						<div class="flex items-center gap-3">
							<Link size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
							<div>
								<p class="font-medium text-base-content">Device connection</p>
								<p class="text-sm text-base-content/65">Push subscription for this device.</p>
							</div>
						</div>
						<span class={`badge ${connectionStatus.tone} gap-1`}>
							<connectionStatus.icon size={12} aria-hidden="true" />
							{connectionStatus.label}
						</span>
					</div>

					<!-- BeeBuzz connection -->
					<div
						class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
					>
						<div class="flex items-center gap-3">
							<Globe size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
							<div>
								<p class="font-medium text-base-content">Server connection</p>
								<p class="text-sm text-base-content/65">Connection to the BeeBuzz server.</p>
							</div>
						</div>
						{#if serverStatus.loading}
							<span class="inline-flex items-center gap-2 text-sm text-base-content/70">
								<Loader size={14} class="animate-spin" aria-hidden="true" />
								Checking
							</span>
						{:else}
							<span class={`badge ${serverStatus.tone} gap-1`}>
								<serverStatus.icon size={12} aria-hidden="true" />
								{serverStatus.label}
							</span>
						{/if}
					</div>

					<!-- Pairing verification -->
					<div
						class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
					>
						<div class="flex items-center gap-3">
							<Shield size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
							<div>
								<p class="font-medium text-base-content">Pairing verification</p>
								<p class="text-sm text-base-content/65">Remote pairing status for this device.</p>
							</div>
						</div>
						<span class={`badge ${pairingVerificationStatus.tone} gap-1`}>
							<pairingVerificationStatus.icon size={12} aria-hidden="true" />
							{pairingVerificationStatus.label}
						</span>
					</div>
				</div>

				<!-- System status link -->
				<a
					href={STATUS_PAGE_URL}
					class="link link-hover inline-flex items-center gap-1.5 text-sm text-base-content/60"
					target="_blank"
					rel="noopener noreferrer"
				>
					<ExternalLink size={14} aria-hidden="true" />
					View system status
				</a>
			</div>
		</section>

		<!-- Technical details (collapsed) -->
		<section class="card border border-base-300 bg-base-100 shadow-sm">
			<div class="card-body gap-0 p-0">
				<button
					type="button"
					class="flex w-full items-center justify-between px-6 py-5"
					onclick={() => (techDetailsOpen = !techDetailsOpen)}
					aria-expanded={techDetailsOpen}
				>
					<h2 class="text-xl font-semibold text-base-content">Technical details</h2>
					<ChevronDown
						size={20}
						class="shrink-0 text-base-content/50 transition-transform duration-200 {techDetailsOpen
							? 'rotate-180'
							: ''}"
						aria-hidden="true"
					/>
				</button>

				{#if techDetailsOpen}
					<div class="space-y-4 px-6 pb-6">
						<!-- Secure delivery -->
						<div
							class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
						>
							<div class="flex items-center gap-3">
								<Shield size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
								<p class="font-medium text-base-content">Secure delivery</p>
							</div>
							<span class={`badge ${secureDeliveryStatus.tone} gap-1`}>
								<secureDeliveryStatus.icon size={12} aria-hidden="true" />
								{secureDeliveryStatus.label}
							</span>
						</div>

						<!-- App runtime -->
						<div
							class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
						>
							<div class="flex items-center gap-3">
								<RefreshCw size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
								<p class="font-medium text-base-content">Service worker</p>
							</div>
							<span class={`badge ${serviceWorkerStatus.tone} gap-1`}>
								<serviceWorkerStatus.icon size={12} aria-hidden="true" />
								{serviceWorkerStatus.label}
							</span>
						</div>

						<!-- Backend version -->
						<div
							class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
						>
							<div class="flex items-center gap-3">
								<Server size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
								<p class="font-medium text-base-content">Backend version</p>
							</div>
							{#if health.version}
								<span class="font-mono text-sm text-base-content/70">{health.version}</span>
							{:else}
								<span class="badge badge-error gap-1">
									<CircleX size={12} aria-hidden="true" />
									Unavailable
								</span>
							{/if}
						</div>

						<!-- Public key -->
						{#if snapshot.encryptionKey}
							<div class="space-y-3 rounded-2xl border border-base-300 px-4 py-4">
								<div class="flex items-start justify-between gap-3">
									<div>
										<p class="font-medium text-base-content">Public key</p>
										{#if publicKeyCreatedAt}
											<p class="text-sm text-base-content/65">
												Generated {publicKeyCreatedAt}
											</p>
										{/if}
									</div>
									<button
										type="button"
										class="btn btn-ghost btn-sm"
										onclick={() => void copyPublicKey()}
										aria-label="Copy public key"
									>
										{#if copyingPublicKey}
											<Check size={16} aria-hidden="true" />
										{:else}
											<Copy size={16} aria-hidden="true" />
										{/if}
									</button>
								</div>
								<code
									class="block overflow-x-auto rounded-xl bg-base-200 px-3 py-3 text-sm text-base-content"
								>
									{snapshot.encryptionKey.recipient}
								</code>
								<div
									class="flex items-center justify-between gap-3 rounded-xl bg-base-200 px-3 py-3"
								>
									<div>
										<p class="text-xs font-medium uppercase tracking-[0.16em] text-base-content/55">
											Fingerprint
										</p>
										<p class="font-mono text-sm text-base-content">
											{snapshot.encryptionKey.fingerprint}
										</p>
									</div>
								</div>
							</div>
						{/if}
					</div>
				{/if}
			</div>
		</section>

		<!-- Disconnect device -->
		<section class="card border border-error/20 bg-base-100 shadow-sm">
			<div class="card-body gap-5">
				<div class="flex items-start gap-4">
					<div class="rounded-2xl bg-error/10 p-3 text-error">
						<Unplug size={22} aria-hidden="true" />
					</div>
					<div class="space-y-1">
						<h2 class="card-title text-xl">Disconnect device</h2>
						<p class="text-sm text-base-content/70">
							Stop notifications, remove encryption keys, and clear notification history.
						</p>
					</div>
				</div>
				<div
					class="flex flex-col gap-4 rounded-2xl border border-error/20 bg-error/5 px-4 py-4 md:flex-row md:items-center md:justify-between"
				>
					<div>
						<p class="font-medium text-base-content">Disconnect this device</p>
						<p class="text-sm text-base-content/70">
							You'll need to pair again to receive notifications.
						</p>
					</div>
					<button type="button" class="btn btn-error" onclick={() => (disconnecting = true)}>
						Disconnect this device
					</button>
				</div>
			</div>
		</section>
	</div>

	<dialog bind:this={disconnectDialog} class="modal" onclose={handleDisconnectDialogClose}>
		<div class="modal-box">
			<h3 class="text-lg font-bold">Disconnect this device?</h3>
			<p class="py-4 text-base-content/80">
				This will stop notifications, remove encryption keys, and clear notification history. You'll
				need to pair again to receive notifications.
			</p>
			<div class="modal-action flex flex-col gap-2 sm:flex-row sm:justify-end">
				<form method="dialog" class="w-full sm:w-auto">
					<button type="submit" class="btn btn-outline w-full">Cancel</button>
				</form>
				<button type="button" class="btn btn-error" onclick={handleDisconnect}>Disconnect</button>
			</div>
		</div>
		<form method="dialog" class="modal-backdrop"><button type="submit">close</button></form>
	</dialog>
{/if}
