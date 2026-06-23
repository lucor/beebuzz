<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		loadDeveloperSettings,
		listLogs,
		clearLogs,
		listSnapshots,
		clearSnapshots,
		listConsoleDiagnostics,
		clearConsoleDiagnostics
	} from '$lib/devmode/storage';
	import { subscribeToLogs } from '$lib/devmode/safe-logger';
	import { developerSettings } from '$lib/devmode/settings';
	import { filterHiveLogs, listHiveLogFilterOptions } from '$lib/devmode/log-filters';
	import { subscribeToConsoleDiagnostics } from '$lib/devmode/console-diagnostics';
	import { captureHiveError } from '$lib/devmode/error-capture';
	import { runEncryptionProbe } from '$lib/services/encryption-diagnostics';
	import {
		loadPushDebugSnapshot,
		updateServiceWorkerRegistration,
		activateWaitingServiceWorker,
		unregisterServiceWorker
	} from '$lib/services/debug-diagnostics';
	import {
		ArrowDownToLine,
		ArrowUpFromLine,
		Check,
		CircleAlert,
		CircleCheck,
		CircleX,
		Copy,
		RefreshCcw,
		RefreshCw,
		Server,
		Shield,
		Smartphone
	} from '@lucide/svelte';
	import type {
		HiveLogEntry,
		HiveLogData,
		HiveErrorSnapshot,
		HiveConsoleDiagnosticEntry
	} from '$lib/devmode/types';
	import { HIVE_BOUNDARY, HIVE_TRANSPORT } from '$lib/devmode/types';
	import type { EncryptionProbeResult, PushDebugSnapshot } from '$lib/types/encryption';
	import { getStoredDeviceKey, type StoredDeviceKey } from '$lib/services/encryption';
	import {
		getNotificationRuntimeMetadata,
		type NotificationRuntimeMetadata
	} from '$lib/services/runtime-metadata-repository';
	import { health } from '@beebuzz/shared/stores/health.svelte';
	import { formatVersionDisplay } from '@beebuzz/shared/utils/version';
	import { toast } from '@beebuzz/shared/stores';
	import { logger } from '@beebuzz/shared/logger';

	type Tab = 'overview' | 'logs' | 'issues' | 'tools';

	let activeTab = $state<Tab>('overview');
	let devModeEnabled = $state(false);
	let initDone = $state(false);
	let logs = $state<HiveLogEntry[]>([]);
	let liveLogs = $state<HiveLogEntry[]>([]);
	let snapshots = $state<HiveErrorSnapshot[]>([]);
	let consoleDiagnostics = $state<HiveConsoleDiagnosticEntry[]>([]);
	const ALL_FILTER_VALUE = '__all__';
	let scopeSelect = $state<string>(ALL_FILTER_VALUE);
	let eventSelect = $state<string>(ALL_FILTER_VALUE);
	let traceSelect = $state<string>(ALL_FILTER_VALUE);
	let notificationSelect = $state<string>(ALL_FILTER_VALUE);
	let logSearch = $state<string>('');
	let logPaused = $state(false);
	let logFollowTail = $state(true);
	let logListElement = $state<HTMLDivElement | null>(null);
	let selectedLog = $state<HiveLogEntry | null>(null);
	let selectedSnapshot = $state<HiveErrorSnapshot | null>(null);
	let selectedConsoleDiag = $state<HiveConsoleDiagnosticEntry | null>(null);
	let deviceKey = $state<StoredDeviceKey | null>(null);
	const hiveVersionDisplay = $derived(
		formatVersionDisplay({
			version: String(import.meta.env.VITE_BEEBUZZ_VERSION || 'dev'),
			commit: String(import.meta.env.VITE_BEEBUZZ_COMMIT || 'dev'),
			dirty: import.meta.env.VITE_BEEBUZZ_DIRTY === true
		})
	);
	const backendVersionDisplay = $derived(
		formatVersionDisplay({
			version: health.version ?? 'dev',
			commit: health.commit
		})
	);
	let notificationRuntime = $state<NotificationRuntimeMetadata | null>(null);
	let copyingPublicKey = $state(false);
	let showPublicKey = $state(false);
	let pushSnapshot = $state<PushDebugSnapshot | null>(null);
	let refreshingPush = $state(false);
	let pushError = $state<string | null>(null);
	let updatingSw = $state(false);
	let activatingSw = $state(false);
	let unregisteringSw = $state(false);
	let showAdvancedRecovery = $state(false);
	let showUnregisterConfirm = $state(false);
	let initError = $state<string | null>(null);

	let probeStatus = $state<'idle' | 'running' | 'passed' | 'failed'>('idle');
	let probeResult = $state<EncryptionProbeResult | null>(null);
	let probeError = $state<string | null>(null);

	let unsubLogs: (() => void) | null = null;
	let unsubConsoleDiagnostics: (() => void) | null = null;

	const issueCount = $derived(snapshots.length + consoleDiagnostics.length);

	const fromSelectValue = (value: string): string => (value === ALL_FILTER_VALUE ? '' : value);
	const toSelectValue = (value: string | undefined): string => value || ALL_FILTER_VALUE;
	const selectedFilterValue = (select: HTMLSelectElement): string =>
		select.selectedIndex === 0 ? ALL_FILTER_VALUE : select.value || ALL_FILTER_VALUE;

	const handleScopeFilterChange = (select: HTMLSelectElement) => {
		scopeSelect = selectedFilterValue(select);
	};

	const handleEventFilterChange = (select: HTMLSelectElement) => {
		eventSelect = selectedFilterValue(select);
	};

	const handleNotificationFilterChange = (select: HTMLSelectElement) => {
		notificationSelect = selectedFilterValue(select);
	};

	const handleTraceFilterChange = (select: HTMLSelectElement) => {
		traceSelect = selectedFilterValue(select);
	};

	const filteredLogs = $derived(
		filterHiveLogs(logs, liveLogs, !logPaused, {
			scope: fromSelectValue(scopeSelect),
			event: fromSelectValue(eventSelect),
			trace: fromSelectValue(traceSelect),
			notification: fromSelectValue(notificationSelect),
			search: logSearch
		})
	);

	const filterOptions = $derived(listHiveLogFilterOptions(logs, liveLogs));

	const loadInitialData = async () => {
		const settings = await loadDeveloperSettings();
		if (!settings.enabled) {
			await goto('/device');
			return;
		}
		devModeEnabled = true;
		developerSettings.set(settings);

		try {
			logs = await listLogs();
		} catch (error) {
			logger.warn('Developer logs load failed', { error: String(error) });
			initError = 'Some local diagnostics could not be loaded.';
		}
		try {
			snapshots = await listSnapshots();
		} catch (error) {
			logger.warn('Developer snapshots load failed', { error: String(error) });
			initError = 'Some local diagnostics could not be loaded.';
		}
		try {
			consoleDiagnostics = await listConsoleDiagnostics(200);
		} catch (error) {
			logger.warn('Developer console diagnostics load failed', { error: String(error) });
			initError = 'Some local diagnostics could not be loaded.';
		}
		try {
			deviceKey = await getStoredDeviceKey();
		} catch (error) {
			logger.warn('Developer device key load failed', { error: String(error) });
			initError = 'Some local diagnostics could not be loaded.';
		}
		try {
			notificationRuntime = await getNotificationRuntimeMetadata();
		} catch (error) {
			logger.warn('Developer runtime metadata load failed', { error: String(error) });
			initError = 'Some local diagnostics could not be loaded.';
		}
		try {
			pushSnapshot = await loadPushDebugSnapshot();
		} catch (error) {
			logger.warn('Developer push snapshot load failed', { error: String(error) });
			pushError = error instanceof Error ? error.message : String(error);
		}
		initDone = true;
	};

	const refreshRuntimeMetadata = async () => {
		try {
			notificationRuntime = await getNotificationRuntimeMetadata();
		} catch (error) {
			logger.warn('Developer runtime metadata refresh failed', { error: String(error) });
		}
	};

	const startLiveLogSubscription = () => {
		if (unsubLogs) return;
		unsubLogs = subscribeToLogs((entry) => {
			liveLogs = [entry, ...liveLogs].slice(0, 500);
			if (
				entry.event === 'push.received' ||
				entry.event === 'notification.displayed' ||
				entry.event === 'outbox.notification_imported'
			) {
				void refreshRuntimeMetadata();
			}
		});
	};

	const stopLiveLogSubscription = () => {
		if (!unsubLogs) return;
		unsubLogs();
		unsubLogs = null;
	};

	const handleVisibilityChange = () => {
		if (document.visibilityState === 'visible') {
			void refreshRuntimeMetadata();
		}
	};

	const startConsoleDiagnosticsSubscription = () => {
		if (unsubConsoleDiagnostics) return;
		unsubConsoleDiagnostics = subscribeToConsoleDiagnostics((entry) => {
			consoleDiagnostics = [entry, ...consoleDiagnostics].slice(0, 200);
		});
	};

	const stopConsoleDiagnosticsSubscription = () => {
		if (!unsubConsoleDiagnostics) return;
		unsubConsoleDiagnostics();
		unsubConsoleDiagnostics = null;
	};

	$effect(() => {
		if (!initDone) return;
		const unsub = developerSettings.subscribe((s) => {
			if (!s.enabled) {
				void goto('/device');
			}
		});
		return () => unsub();
	});

	const handleClearLogs = async () => {
		await clearLogs();
		logs = [];
		liveLogs = [];
	};

	const handleClearReports = async () => {
		await clearSnapshots();
		snapshots = [];
		selectedSnapshot = null;
	};

	const handleClearRuntimeWarnings = async () => {
		await clearConsoleDiagnostics();
		consoleDiagnostics = [];
		selectedConsoleDiag = null;
	};

	const handleCopyLogs = async () => {
		try {
			const text = JSON.stringify(filteredLogs, null, 2);
			await navigator.clipboard.writeText(text);
		} catch {
			// ignore
		}
	};

	const handleDownloadLogs = () => {
		const blob = new Blob([JSON.stringify(filteredLogs, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `hive-logs-${new Date().toISOString()}.json`;
		a.click();
		URL.revokeObjectURL(url);
	};

	const traceColor = (id: string): string => {
		const palette = [
			'#0ea5e9',
			'#10b981',
			'#f59e0b',
			'#f43f5e',
			'#8b5cf6',
			'#06b6d4',
			'#84cc16',
			'#d946ef'
		];

		let hash = 0;
		for (let i = 0; i < id.length; i += 1) {
			hash = (hash * 31 + id.charCodeAt(i)) >>> 0;
		}
		return palette[hash % palette.length];
	};

	const formatLogTime = (ts: string): string => {
		const date = new Date(ts);
		if (Number.isNaN(date.getTime())) return ts;

		const pad = (value: number, length = 2) => String(value).padStart(length, '0');
		return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${pad(
			date.getMilliseconds(),
			3
		)}`;
	};

	const formatRuntimeTime = (ts: string | null | undefined): string => {
		if (!ts) return 'Never';
		const date = new Date(ts);
		if (Number.isNaN(date.getTime())) return ts;
		return new Intl.DateTimeFormat(undefined, {
			dateStyle: 'medium',
			timeStyle: 'short'
		}).format(date);
	};

	const shortToken = (value: string): string => {
		if (value.length <= 12) return value;
		return `${value.slice(0, 8)}...${value.slice(-4)}`;
	};

	const boundaryLabel = (data?: HiveLogData): string => {
		switch (data?.boundary) {
			case HIVE_BOUNDARY.INBOUND:
				return 'IN';
			case HIVE_BOUNDARY.OUTBOUND:
				return 'OUT';
			case HIVE_BOUNDARY.INTERNAL:
				return 'INT';
			default:
				return '';
		}
	};

	const transportLabel = (data?: HiveLogData): string => {
		switch (data?.transport) {
			case HIVE_TRANSPORT.WEB_PUSH:
				return 'PUSH';
			case HIVE_TRANSPORT.HTTPS:
				return 'HTTPS';
			case HIVE_TRANSPORT.POST_MESSAGE:
				return 'MSG';
			case HIVE_TRANSPORT.INDEXEDDB:
				return 'IDB';
			case HIVE_TRANSPORT.LOCAL_STORAGE:
				return 'LS';
			case HIVE_TRANSPORT.NOTIFICATION_CENTER:
				return 'OS';
			case HIVE_TRANSPORT.SERVICE_WORKER:
				return 'SW';
			case HIVE_TRANSPORT.CRYPTO:
				return 'CRYPTO';
			default:
				return '';
		}
	};

	const eventLabel = (event: HiveLogEntry['event'] | undefined): string => {
		switch (event) {
			case 'app.started':
				return 'Hive started';
			case 'service_worker.registered':
				return 'Service worker ready';
			case 'push.received':
				return 'Push arrived';
			case 'payload.resolve':
				return 'Payload read';
			case 'payload.detected_plain':
				return 'Plain payload detected';
			case 'payload.detected_encrypted':
				return 'Encrypted payload detected';
			case 'payload.decrypt_failed':
				return 'Decrypt failed';
			case 'payload.invalid':
				return 'Invalid payload';
			case 'push.resolved':
				return 'Notification decoded';
			case 'notification.persist_started':
				return 'Saving notification';
			case 'notification.persisted':
				return 'Saved locally';
			case 'notification.displayed':
				return 'Shown by OS';
			case 'clients.notified':
				return 'App notified';
			case 'notification.imported':
				return 'Imported in Hive';
			case 'notification.clicked':
				return 'User clicked';
			case 'outbox.sync_started':
				return 'Outbox sync started';
			case 'outbox.request_started':
				return 'Outbox request sent';
			case 'outbox.response_received':
				return 'Outbox response received';
			case 'outbox.notification_resolve_started':
				return 'Synced notification decoding';
			case 'outbox.notification_imported':
				return 'Synced notification imported';
			case 'outbox.cursor_updated':
				return 'Sync cursor updated';
			case 'outbox.gap_detected':
				return 'Sync gap detected';
			case 'outbox.sync_completed':
				return 'Outbox sync completed';
			case 'outbox.sync_failed':
				return 'Outbox sync failed';
			default:
				return event ?? 'Unknown event';
		}
	};

	const eventDetail = (log: HiveLogEntry): string => {
		const parts: string[] = [];
		if (log.data?.duration_ms !== undefined) parts.push(`${log.data.duration_ms}ms`);
		if (log.data?.item_count !== undefined) parts.push(`${log.data.item_count} item`);
		if (log.data?.imported_count !== undefined) parts.push(`${log.data.imported_count} imported`);
		if (log.data?.page_count !== undefined) parts.push(`page ${log.data.page_count}`);
		if (log.data?.delivery_mode) parts.push(log.data.delivery_mode);
		return parts.join(' · ');
	};

	const compactDetail = (log: HiveLogEntry): string => {
		if (log.data?.endpoint) {
			const method = log.data.method ?? '';
			return method ? `${method} ${log.data.endpoint}` : log.data.endpoint;
		}
		const detail = eventDetail(log);
		if (detail) return detail;
		const label = eventLabel(log.event);
		if (log.message && !label.includes(log.message)) {
			return log.message;
		}
		return '';
	};

	const clearLogFilters = () => {
		scopeSelect = ALL_FILTER_VALUE;
		eventSelect = ALL_FILTER_VALUE;
		traceSelect = ALL_FILTER_VALUE;
		notificationSelect = ALL_FILTER_VALUE;
		logSearch = '';
	};

	const copyPublicKey = async () => {
		if (!deviceKey?.recipient) return;

		try {
			copyingPublicKey = true;
			await navigator.clipboard.writeText(deviceKey.recipient);
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

	const runProbe = async () => {
		probeStatus = 'running';
		probeError = null;
		probeResult = null;
		try {
			const r = await runEncryptionProbe();
			probeResult = r;
			probeStatus = r.keyPersistence.ok && r.wrappingKey.ok ? 'passed' : 'failed';
		} catch (error) {
			probeError = error instanceof Error ? error.message : String(error);
			probeStatus = 'failed';
		}
	};

	const refreshPushDiag = async () => {
		refreshingPush = true;
		pushError = null;
		try {
			pushSnapshot = await loadPushDebugSnapshot();
		} catch (error) {
			pushError = error instanceof Error ? error.message : String(error);
		} finally {
			refreshingPush = false;
		}
	};

	const handleUpdateServiceWorker = async () => {
		updatingSw = true;
		pushError = null;
		try {
			await updateServiceWorkerRegistration();
			await refreshPushDiag();
		} catch (error) {
			pushError = error instanceof Error ? error.message : String(error);
		} finally {
			updatingSw = false;
		}
	};

	const handleActivateServiceWorker = async () => {
		activatingSw = true;
		pushError = null;
		try {
			await activateWaitingServiceWorker();
			await refreshPushDiag();
		} catch (error) {
			pushError = error instanceof Error ? error.message : String(error);
		} finally {
			activatingSw = false;
		}
	};

	const handleUnregister = async () => {
		showUnregisterConfirm = false;
		unregisteringSw = true;
		pushError = null;
		try {
			await unregisterServiceWorker();
			await refreshPushDiag();
		} catch (error) {
			pushError = error instanceof Error ? error.message : String(error);
		} finally {
			unregisteringSw = false;
		}
	};

	onMount(async () => {
		try {
			await loadInitialData();
		} catch (error) {
			logger.warn('Developer Mode init failed (non-blocking)', { error: String(error) });
			initError = error instanceof Error ? error.message : 'Developer Mode could not load fully.';
			initDone = true;
		}
		document.addEventListener('visibilitychange', handleVisibilityChange);

		if (devModeEnabled) {
			startLiveLogSubscription();
			startConsoleDiagnosticsSubscription();
		}
	});

	onDestroy(() => {
		document.removeEventListener('visibilitychange', handleVisibilityChange);
		stopLiveLogSubscription();
		stopConsoleDiagnosticsSubscription();
	});

	$effect(() => {
		if (activeTab === 'overview') {
			void refreshRuntimeMetadata();
		}
	});

	$effect(() => {
		if (filteredLogs.length === 0 || !logFollowTail || activeTab !== 'logs' || !logListElement) {
			return;
		}

		void tick().then(() => {
			if (!logListElement) return;
			logListElement.scrollTop = logListElement.scrollHeight;
		});
	});
</script>

<div class="flex flex-col gap-6">
	<div>
		<h1 class="text-3xl font-bold text-base-content">Developer Mode</h1>
		<p class="text-sm text-base-content/70">Privacy-safe local diagnostics for Hive</p>
	</div>

	{#if initError}
		<div class="alert alert-warning text-sm" role="status">
			<span>{initError}</span>
		</div>
	{/if}

	{#if !devModeEnabled}
		<div class="card bg-base-100 shadow-md">
			<div class="card-body items-center gap-4 py-12 text-center">
				<p class="text-lg text-base-content/60">Developer Mode is off.</p>
				<p class="text-sm text-base-content/50 max-w-md">
					Enable it to keep local privacy-safe diagnostic events on this device. Nothing is sent to
					BeeBuzz automatically.
				</p>
				<button class="btn btn-primary" onclick={() => void goto('/device')}>
					Enable Developer Mode
				</button>
			</div>
		</div>
	{:else}
		<div role="tablist" class="tabs tabs-lift">
			<button
				type="button"
				role="tab"
				aria-selected={activeTab === 'overview'}
				class="tab {activeTab === 'overview' ? 'tab-active' : ''}"
				onclick={() => (activeTab = 'overview')}
			>
				Overview
			</button>
			<button
				type="button"
				role="tab"
				aria-selected={activeTab === 'logs'}
				class="tab {activeTab === 'logs' ? 'tab-active' : ''}"
				onclick={() => (activeTab = 'logs')}
			>
				Logs
			</button>
			<button
				type="button"
				role="tab"
				aria-selected={activeTab === 'issues'}
				class="tab {activeTab === 'issues' ? 'tab-active' : ''}"
				onclick={() => (activeTab = 'issues')}
			>
				Issues
				{#if issueCount > 0}
					<span class="badge badge-warning badge-sm">{issueCount}</span>
				{/if}
			</button>
			<button
				type="button"
				role="tab"
				aria-selected={activeTab === 'tools'}
				class="tab {activeTab === 'tools' ? 'tab-active' : ''}"
				onclick={() => (activeTab = 'tools')}
			>
				Tools
			</button>
		</div>

		{#if activeTab === 'overview'}
			<div class="space-y-4">
				<div class="card bg-base-100 shadow-md">
					<div class="card-body gap-6">
						<h2 class="card-title text-lg">Device Internals</h2>

						<div class="grid gap-4 sm:grid-cols-2">
							<div
								class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
							>
								<div class="flex items-center gap-3">
									<Shield size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
									<p class="font-medium text-base-content">Secure delivery</p>
								</div>
								{#if deviceKey}
									<span class="badge badge-success gap-1">
										<CircleCheck size={12} aria-hidden="true" />
										Configured
									</span>
								{:else}
									<span class="badge badge-error gap-1">
										<CircleX size={12} aria-hidden="true" />
										Not configured
									</span>
								{/if}
							</div>

							<div
								class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
							>
								<div class="flex items-center gap-3">
									<RefreshCw size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
									<p class="font-medium text-base-content">Service worker</p>
								</div>
								{#if pushSnapshot}
									<span class="badge badge-success gap-1">
										<CircleCheck size={12} aria-hidden="true" />
										{pushSnapshot.controllerState ?? 'active'}
									</span>
								{:else}
									<span class="badge badge-ghost gap-1">
										<CircleAlert size={12} aria-hidden="true" />
										Unknown
									</span>
								{/if}
							</div>
						</div>

						<div
							class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
						>
							<div class="flex items-center gap-3">
								<Smartphone size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
								<p class="font-medium text-base-content">Hive version</p>
							</div>
							<div class="text-right">
								<span class="font-mono text-sm text-base-content/70"
									>{hiveVersionDisplay.primary}</span
								>
								{#if hiveVersionDisplay.badge}
									<span class="badge badge-warning badge-sm ml-2">{hiveVersionDisplay.badge}</span>
								{/if}
								{#if hiveVersionDisplay.secondary}
									<p class="text-xs font-mono text-base-content/40">
										{hiveVersionDisplay.secondary}
									</p>
								{/if}
							</div>
						</div>

						<div
							class="flex items-center justify-between gap-3 rounded-2xl border border-base-300 px-4 py-3"
						>
							<div class="flex items-center gap-3">
								<Server size={18} class="shrink-0 text-base-content/50" aria-hidden="true" />
								<p class="font-medium text-base-content">Backend version</p>
							</div>
							{#if health.version}
								<div class="text-right">
									<span class="font-mono text-sm text-base-content/70"
										>{backendVersionDisplay.primary}</span
									>
									{#if backendVersionDisplay.badge}
										<span class="badge badge-warning badge-sm ml-2"
											>{backendVersionDisplay.badge}</span
										>
									{/if}
									{#if backendVersionDisplay.secondary}
										<p class="text-xs font-mono text-base-content/40">
											{backendVersionDisplay.secondary}
										</p>
									{/if}
								</div>
							{:else}
								<span class="badge badge-error gap-1">
									<CircleX size={12} aria-hidden="true" />
									Unavailable
								</span>
							{/if}
						</div>

						<div class="rounded-2xl border border-base-300 px-4 py-3">
							<p class="text-sm font-medium text-base-content">Last notification received</p>
							<p class="mt-1 font-mono text-sm text-base-content/70">
								{formatRuntimeTime(notificationRuntime?.lastNotificationReceivedAt)}
								{#if notificationRuntime?.lastNotificationReceivedVia}
									<span
										class="badge {notificationRuntime.lastNotificationReceivedVia === 'push'
											? 'badge-info'
											: 'badge-ghost'} badge-xs ml-2 align-middle"
									>
										{notificationRuntime.lastNotificationReceivedVia}
									</span>
								{/if}
							</p>
						</div>

						{#if deviceKey}
							<div class="rounded-2xl border border-base-300 px-4 py-4">
								<div class="flex items-start justify-between gap-3">
									<div>
										<p class="font-medium text-base-content">Public key</p>
										{#if deviceKey.createdAt}
											<p class="text-sm text-base-content/65">
												Generated
												{new Intl.DateTimeFormat(undefined, {
													dateStyle: 'medium',
													timeStyle: 'short'
												}).format(new Date(deviceKey.createdAt))}
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

								<div
									class="mt-3 flex items-center justify-between gap-3 rounded-xl bg-base-200 px-3 py-3"
								>
									<div>
										<p class="text-xs font-medium uppercase tracking-[0.16em] text-base-content/55">
											Fingerprint
										</p>
										<p class="font-mono text-sm text-base-content">
											{deviceKey.fingerprint}
										</p>
									</div>
								</div>

								<button
									class="mt-3 link link-hover text-sm"
									onclick={() => (showPublicKey = !showPublicKey)}
								>
									{showPublicKey ? 'Hide key' : 'Show key'}
								</button>

								{#if showPublicKey}
									<code
										class="mt-2 block overflow-x-auto rounded-xl bg-base-200 px-3 py-3 text-sm text-base-content"
									>
										{deviceKey.recipient}
									</code>
								{/if}
							</div>
						{/if}
					</div>
				</div>
			</div>
		{:else if activeTab === 'logs'}
			<div class="card bg-base-100 shadow-md">
				<div class="card-body gap-4">
					<div class="rounded-xl border border-base-300 bg-base-100 p-3">
						<p class="mb-3 text-sm font-medium text-base-content">Filters</p>

						<div class="grid gap-2 md:grid-cols-[9rem_minmax(0,1fr)_9rem_10rem]">
							<select
								class="select select-bordered select-sm w-full"
								value={scopeSelect}
								onchange={(event) => handleScopeFilterChange(event.currentTarget)}
							>
								<option value={ALL_FILTER_VALUE}>All scopes</option>
								{#each filterOptions.scopes as s (s)}
									<option value={s}>{s}</option>
								{/each}
							</select>

							<select
								class="select select-bordered select-sm w-full"
								value={eventSelect}
								onchange={(event) => handleEventFilterChange(event.currentTarget)}
							>
								<option value={ALL_FILTER_VALUE}>All events</option>
								{#each filterOptions.events as ev (ev)}
									<option value={ev}>{eventLabel(ev)}</option>
								{/each}
							</select>

							<select
								class="select select-bordered select-sm w-full"
								value={notificationSelect}
								onchange={(event) => handleNotificationFilterChange(event.currentTarget)}
							>
								<option value={ALL_FILTER_VALUE}>All notifications</option>
								{#each filterOptions.notifications as n (n)}
									<option value={n}>{shortToken(n)}</option>
								{/each}
							</select>

							<select
								class="select select-bordered select-sm w-full"
								value={traceSelect}
								onchange={(event) => handleTraceFilterChange(event.currentTarget)}
							>
								<option value={ALL_FILTER_VALUE}>All traces</option>
								{#each filterOptions.traces as t (t)}
									<option value={t}>{shortToken(t)}</option>
								{/each}
							</select>
						</div>

						<div class="mt-2 flex items-center gap-2">
							<label class="input input-bordered input-sm flex min-w-0 flex-1 items-center gap-2">
								<input
									type="search"
									class="grow"
									placeholder="Search logs..."
									bind:value={logSearch}
								/>
							</label>
							<button class="btn btn-sm btn-ghost shrink-0" onclick={clearLogFilters}>Reset</button>
						</div>

						<hr class="my-3 border-base-300" />

						<div class="flex flex-wrap items-center justify-between gap-2">
							<div class="flex flex-wrap items-center gap-3 text-sm">
								<label class="flex items-center gap-2">
									<span>Live logs:</span>
									<input
										type="checkbox"
										class="toggle toggle-primary toggle-sm"
										checked={!logPaused}
										onchange={(e) => (logPaused = !e.currentTarget.checked)}
									/>
									<span class="text-xs text-base-content/60">{logPaused ? 'Off' : 'On'}</span>
								</label>

								<label class="flex items-center gap-2">
									<span>Auto-scroll:</span>
									<input
										type="checkbox"
										class="toggle toggle-primary toggle-sm"
										checked={logFollowTail}
										onchange={(e) => (logFollowTail = e.currentTarget.checked)}
									/>
									<span class="text-xs text-base-content/60">{logFollowTail ? 'On' : 'Off'}</span>
								</label>
							</div>

							<div class="flex flex-wrap items-center gap-2">
								<button class="btn btn-sm btn-outline" onclick={() => void handleCopyLogs()}>
									Copy
								</button>
								<button class="btn btn-sm btn-outline" onclick={() => void handleDownloadLogs()}>
									Download
								</button>
								<button
									class="btn btn-sm btn-ghost text-error"
									onclick={() => void handleClearLogs()}
								>
									Clear logs
								</button>
							</div>
						</div>
					</div>

					<div class="grid min-h-96 gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
						<div
							class="max-h-[34rem] overflow-y-auto rounded-lg border border-base-300 bg-base-200/50"
							bind:this={logListElement}
						>
							{#if filteredLogs.length === 0}
								<p class="p-4 text-center text-sm text-base-content/50">No log entries.</p>
							{:else}
								{#each filteredLogs as log (log.id)}
									<button
										class="flex flex-col gap-1 w-full p-2 px-3 border-b border-base-300/50 border-l-4 text-left transition-colors duration-100 cursor-pointer hover:bg-base-300/30 {selectedLog?.id ===
										log.id
											? 'bg-base-300/50'
											: ''}"
										style={log.data?.push_trace_id
											? `--trace-border-color: ${traceColor(log.data.push_trace_id)}; border-left-color: ${traceColor(log.data.push_trace_id)}`
											: 'border-left-color: transparent'}
										onclick={() => (selectedLog = selectedLog?.id === log.id ? null : log)}
									>
										<div class="flex flex-wrap items-center gap-1.5 min-w-0">
											<time class="font-mono text-xs text-base-content/40"
												>{formatLogTime(log.ts)}</time
											>
											<span
												class="inline-flex items-center rounded-full px-1.5 py-px text-[0.625rem] font-semibold tracking-wide uppercase whitespace-nowrap"
												data-scope={log.scope}>{log.scope}</span
											>
											{#if log.data?.boundary}
												<span
													class="inline-flex items-center gap-0.5 text-[0.625rem] font-semibold whitespace-nowrap {boundaryLabel(
														log.data
													) === 'IN'
														? 'text-[hsl(160_70%_35%)]'
														: boundaryLabel(log.data) === 'OUT'
															? 'text-[hsl(200_70%_40%)]'
															: 'text-[hsl(38_85%_40%)]'}"
												>
													{#if log.data.boundary === HIVE_BOUNDARY.INBOUND}
														<ArrowDownToLine class="size-3" />
													{:else if log.data.boundary === HIVE_BOUNDARY.OUTBOUND}
														<ArrowUpFromLine class="size-3" />
													{:else if log.data.boundary === HIVE_BOUNDARY.INTERNAL}
														<RefreshCcw class="size-3" />
													{/if}
													{boundaryLabel(log.data)}
												</span>
											{/if}
											<span class="text-sm font-medium truncate min-w-0"
												>{eventLabel(log.event)}</span
											>
										</div>
										{#if compactDetail(log)}
											<div class="font-mono text-xs text-base-content/60 [overflow-wrap:anywhere]">
												{compactDetail(log)}
											</div>
										{/if}
									</button>
								{/each}
							{/if}
						</div>

						<aside class="rounded-lg border border-base-300 bg-base-200/50 p-4">
							{#if selectedLog}
								<div class="flex items-start justify-between gap-3">
									<p class="font-mono text-xs text-base-content/50">
										{formatLogTime(selectedLog.ts)} · {selectedLog.ts}
									</p>
									<button class="btn btn-ghost btn-xs" onclick={() => (selectedLog = null)}>
										Close
									</button>
								</div>

								<div class="mt-4 grid grid-cols-2 gap-2 text-xs">
									<div class="col-span-2">
										<p class="text-base-content/50">Event</p>
										<p class="font-mono break-all">{selectedLog.event}</p>
									</div>
									<div class="col-span-2">
										<p class="text-base-content/50">Title</p>
										<p class="font-medium">{eventLabel(selectedLog.event)}</p>
									</div>
									<div>
										<p class="text-base-content/50">Scope</p>
										<p class="font-mono">{selectedLog.scope}</p>
									</div>
									<div>
										<p class="text-base-content/50">Boundary</p>
										<p class="font-mono">{boundaryLabel(selectedLog.data) || '—'}</p>
									</div>
									<div>
										<p class="text-base-content/50">Transport</p>
										<p class="font-mono">{transportLabel(selectedLog.data) || '—'}</p>
									</div>
									{#if selectedLog.data?.duration_ms !== undefined}
										<div>
											<p class="text-base-content/50">Duration</p>
											<p class="font-mono">{selectedLog.data.duration_ms}ms</p>
										</div>
									{/if}
									{#if selectedLog.data?.endpoint}
										<div class="col-span-2">
											<p class="text-base-content/50">Endpoint</p>
											<p class="truncate font-mono">
												{selectedLog.data.method ?? ''}
												{selectedLog.data.endpoint}
											</p>
										</div>
									{/if}
									{#if selectedLog.data?.notification_id}
										<div class="col-span-2">
											<p class="text-base-content/50">Notification</p>
											<button
												class="btn btn-xs mt-0.5 max-w-full justify-start truncate font-mono"
												onclick={() =>
													(notificationSelect = toSelectValue(selectedLog?.data?.notification_id))}
											>
												{selectedLog.data.notification_id}
											</button>
										</div>
									{/if}
									{#if selectedLog.data?.push_trace_id}
										<div class="col-span-2">
											<p class="text-base-content/50">Trace</p>
											<button
												class="btn btn-xs mt-0.5 max-w-full justify-start truncate font-mono"
												onclick={() =>
													(traceSelect = toSelectValue(selectedLog?.data?.push_trace_id))}
											>
												{selectedLog.data.push_trace_id}
											</button>
										</div>
									{/if}
								</div>

								{#if selectedLog.data}
									<div class="mt-4">
										<p class="mb-1 text-xs text-base-content/50">Raw JSON</p>
										<pre
											class="max-h-52 overflow-auto rounded-lg bg-base-300 p-3 font-mono text-xs">
{JSON.stringify(selectedLog.data, null, 2)}</pre>
									</div>
								{/if}
							{:else}
								<p class="text-sm text-base-content/50">Select an event to inspect metadata.</p>
							{/if}
						</aside>
					</div>
				</div>
			</div>
		{:else if activeTab === 'issues'}
			<div class="card bg-base-100 shadow-md">
				<div class="card-body gap-4">
					<h2 class="text-lg font-semibold text-base-content">Error reports</h2>
					<p class="text-sm text-base-content/70">
						Captured errors that can be reviewed and submitted.
					</p>

					<div class="flex items-center justify-between gap-3">
						<div class="text-sm text-base-content/60">Nothing is sent automatically.</div>
						{#if snapshots.length > 0}
							<button
								class="btn btn-ghost btn-sm text-error"
								onclick={() => void handleClearReports()}
							>
								Clear reports
							</button>
						{/if}
					</div>

					{#if snapshots.length === 0}
						<p class="text-center text-sm text-base-content/50 py-4">No diagnostic reports yet.</p>
					{:else}
						{#each snapshots as snapshot (snapshot.id)}
							<div class="rounded-xl border border-base-300 p-4">
								<div class="flex items-start justify-between gap-3">
									<div>
										<p class="font-mono text-xs text-base-content/60">{snapshot.id}</p>
										<p class="text-sm text-base-content/70">{snapshot.ts}</p>
										<p class="mt-1 font-medium text-error">
											{snapshot.message}
										</p>
										<p class="text-xs text-base-content/60">
											{snapshot.severity} · {snapshot.scope} · {snapshot.event}
											{#if snapshot.error}
												· {snapshot.error.name}
											{/if}
										</p>
									</div>
									<button
										class="btn btn-outline btn-sm"
										onclick={() =>
											void navigator.clipboard.writeText(JSON.stringify(snapshot, null, 2))}
									>
										Copy snapshot
									</button>
								</div>

								<button
									class="mt-2 text-xs link link-hover"
									onclick={() =>
										(selectedSnapshot = selectedSnapshot?.id === snapshot.id ? null : snapshot)}
								>
									{selectedSnapshot?.id === snapshot.id ? 'Hide details' : 'Show details'}
								</button>

								{#if selectedSnapshot?.id === snapshot.id}
									<pre
										class="mt-3 rounded-lg bg-base-300 p-3 font-mono text-xs overflow-x-auto max-h-96">
{JSON.stringify(snapshot, null, 2)}</pre>
								{/if}
							</div>
						{/each}
					{/if}

					<hr class="border-base-300" />

					<h2 class="text-lg font-semibold text-base-content">Runtime warnings</h2>
					<div
						class="rounded-xl border border-base-300 bg-base-200/50 px-3 py-2 text-xs text-base-content/60"
					>
						Runtime warnings and errors captured while Developer Mode is enabled.
					</div>

					{#if consoleDiagnostics.length > 0}
						<div class="flex justify-end">
							<button
								class="btn btn-sm btn-ghost text-error"
								onclick={() => void handleClearRuntimeWarnings()}
							>
								Clear warnings
							</button>
						</div>
					{/if}

					{#if consoleDiagnostics.length === 0}
						<p class="text-center text-sm text-base-content/50 py-4">
							No warnings or errors captured.
						</p>
					{:else}
						<div class="space-y-2">
							{#each consoleDiagnostics as entry (entry.id)}
								<button
									class="w-full rounded-lg border border-base-300 bg-base-100 px-3 py-2 text-left text-xs hover:bg-base-200"
									onclick={() =>
										(selectedConsoleDiag = selectedConsoleDiag?.id === entry.id ? null : entry)}
								>
									<div class="flex items-center gap-2">
										<span
											class="badge {entry.level === 'error'
												? 'badge-error'
												: 'badge-warning'} badge-xs"
										>
											{entry.level}
										</span>
										<span class="font-mono text-base-content/50">{entry.source}</span>
										<time class="ml-auto text-base-content/40"
											>{new Date(entry.ts).toLocaleTimeString()}</time
										>
									</div>
									<p class="mt-1 truncate font-medium text-base-content">{entry.message}</p>
								</button>
								{#if selectedConsoleDiag?.id === entry.id}
									<div class="mt-2 space-y-2">
										{#if entry.stack && entry.stack.length > 0}
											<div>
												<p class="text-xs font-semibold text-base-content/60 mb-1">Stack trace</p>
												<pre
													class="rounded-lg bg-base-300 p-3 font-mono text-xs overflow-x-auto max-h-48">{entry.stack.join(
														'\n'
													)}</pre>
											</div>
										{/if}
										<pre
											class="rounded-lg bg-base-300 p-3 font-mono text-xs overflow-x-auto max-h-48">{JSON.stringify(
												entry,
												null,
												2
											)}</pre>
									</div>
								{/if}
							{/each}
						</div>
					{/if}
				</div>
			</div>
		{:else if activeTab === 'tools'}
			<div class="space-y-4">
				<div class="card bg-base-100 shadow-md">
					<div class="card-body gap-4">
						<h2 class="card-title text-lg">Push &amp; service worker</h2>
						<p class="text-sm text-base-content/70">Inspect and refresh the local push runtime.</p>

						<div class="flex flex-wrap gap-2">
							<button
								class="btn btn-outline btn-sm"
								onclick={() => void refreshPushDiag()}
								disabled={refreshingPush}
							>
								Refresh
							</button>
							<button
								class="btn btn-outline btn-sm"
								onclick={() => void handleUpdateServiceWorker()}
								disabled={updatingSw}
							>
								Check update
							</button>
							<button
								class="btn btn-outline btn-sm"
								onclick={() => void handleActivateServiceWorker()}
								disabled={activatingSw}
							>
								Activate update
							</button>
						</div>

						{#if pushError}
							<div class="alert alert-error text-sm"><span>{pushError}</span></div>
						{/if}

						{#if pushSnapshot}
							<div class="grid gap-4 sm:grid-cols-2">
								<div class="rounded-xl border border-base-300 bg-base-200/50 p-4">
									<p
										class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60"
									>
										Service Worker
									</p>
									<div class="space-y-2 font-mono text-xs text-base-content/80">
										<p>controller: {pushSnapshot.controllerScriptURL ?? 'none'}</p>
										<p>state: {pushSnapshot.controllerState ?? 'none'}</p>
										<p>scope: {pushSnapshot.registrationScope ?? 'none'}</p>
										<p>installing: {pushSnapshot.registrationInstallingState ?? 'none'}</p>
										<p>waiting: {pushSnapshot.registrationWaitingState ?? 'none'}</p>
										<p>active: {pushSnapshot.registrationActiveState ?? 'none'}</p>
									</div>
								</div>
								<div class="rounded-xl border border-base-300 bg-base-200/50 p-4">
									<p
										class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60"
									>
										Push Subscription
									</p>
									<div class="space-y-2 font-mono text-xs text-base-content/80">
										<p>subscription: {pushSnapshot.subscriptionPresent ? 'present' : 'none'}</p>
										<p>key material: {pushSnapshot.subscriptionKeysPresent ? 'present' : 'none'}</p>
									</div>
								</div>
							</div>
						{/if}

						<button
							class="link link-hover text-xs text-base-content/50"
							onclick={() => (showAdvancedRecovery = !showAdvancedRecovery)}
						>
							{showAdvancedRecovery ? 'Hide' : 'Show'} advanced recovery
						</button>

						{#if showAdvancedRecovery}
							<div class="rounded-xl border border-error/20 bg-error/5 px-4 py-3">
								<div class="flex items-center justify-between gap-3">
									<div>
										<p class="text-sm font-medium text-base-content">Unregister service worker</p>
										<p class="text-xs text-base-content/60">
											Clears the active service worker. Notifications will stop until next page
											load.
										</p>
									</div>
									<button
										class="btn btn-ghost btn-sm text-error"
										onclick={() => (showUnregisterConfirm = true)}
										disabled={unregisteringSw}
									>
										{unregisteringSw ? 'Unregistering...' : 'Unregister'}
									</button>
								</div>
							</div>
						{/if}
					</div>
				</div>

				<div class="card bg-base-100 shadow-md">
					<div class="card-body gap-4">
						<div class="flex flex-wrap items-center justify-between gap-3">
							<div>
								<h2 class="card-title text-lg">Encryption</h2>
								<p class="text-sm text-base-content/70">
									Verify local key persistence and wrapping support.
								</p>
							</div>
							<span
								class="badge
								{probeStatus === 'passed'
									? 'badge-success'
									: probeStatus === 'failed'
										? 'badge-error'
										: probeStatus === 'running'
											? 'badge-warning'
											: 'badge-ghost'}"
							>
								{probeStatus}
							</span>
						</div>

						<div class="flex gap-2">
							<button
								class="btn btn-primary btn-sm"
								onclick={() => void runProbe()}
								disabled={probeStatus === 'running'}
							>
								{probeStatus === 'running' ? 'Running...' : 'Run probe'}
							</button>
						</div>

						{#if probeError}
							<div class="alert alert-error text-sm"><span>{probeError}</span></div>
						{/if}

						{#if probeResult}
							<div class="rounded-xl border border-base-300 bg-base-200/50 p-4">
								<p class="text-xs text-base-content/80">
									Key persistence: {probeResult.keyPersistence.ok ? 'pass' : 'fail'}
									| Wrapping key: {probeResult.wrappingKey.ok ? 'pass' : 'fail'}
								</p>
							</div>
						{/if}
					</div>
				</div>

				<div class="card bg-base-100 shadow-md">
					<div class="card-body gap-4">
						<h2 class="card-title text-lg">Test diagnostics</h2>
						<p class="text-sm text-base-content/70">Generate local test warnings and reports.</p>

						<div class="flex flex-wrap gap-2">
							<button
								class="btn btn-outline btn-sm"
								onclick={() => {
									try {
										throw new Error('Test warn from Developer Mode');
									} catch (e) {
										console.warn(e);
									}
									try {
										throw new Error('Test error from Developer Mode');
									} catch (e) {
										console.error(e);
									}
								}}
							>
								Test capture
							</button>
							<button
								class="btn btn-outline btn-sm"
								onclick={async () => {
									const err = new Error('Test snapshot from Developer Mode');
									await captureHiveError({
										scope: 'app',
										event: 'app.bootstrap_failed',
										message: 'User-triggered test snapshot',
										error: err,
										severity: 'error'
									});
									snapshots = await listSnapshots();
								}}
							>
								Test snapshot
							</button>
						</div>
					</div>
				</div>
			</div>
		{/if}
	{/if}
</div>

{#if showUnregisterConfirm}
	<div class="modal modal-open" role="dialog" aria-modal="true" aria-labelledby="unregister-title">
		<div class="modal-box">
			<h2 id="unregister-title" class="text-lg font-semibold text-base-content">
				Unregister service worker?
			</h2>
			<p class="mt-2 text-sm text-base-content/70">
				This will stop all notifications until the next page load. The service worker will be
				re-registered automatically on the next visit.
			</p>
			<div class="mt-6 flex justify-end gap-3">
				<button class="btn btn-ghost" onclick={() => (showUnregisterConfirm = false)}>
					Cancel
				</button>
				<button class="btn btn-error" onclick={() => void handleUnregister()}> Unregister </button>
			</div>
		</div>
		<button
			class="modal-backdrop"
			aria-label="Close modal"
			onclick={() => (showUnregisterConfirm = false)}
		></button>
	</div>
{/if}

<style>
	[data-scope='app'] {
		background: hsl(220 14% 70% / 0.2);
		color: hsl(220 14% 45%);
	}
	[data-scope='push'] {
		background: hsl(170 70% 40% / 0.15);
		color: hsl(170 70% 30%);
	}
	[data-scope='payload'] {
		background: hsl(240 60% 55% / 0.15);
		color: hsl(240 60% 40%);
	}
	[data-scope='notification'] {
		background: hsl(38 85% 50% / 0.15);
		color: hsl(38 85% 35%);
	}
	[data-scope='outbox'] {
		background: hsl(210 85% 50% / 0.15);
		color: hsl(210 85% 35%);
	}
	[data-scope='service_worker'] {
		background: hsl(260 40% 60% / 0.15);
		color: hsl(260 40% 40%);
	}
	[data-scope='storage'] {
		background: hsl(200 30% 55% / 0.15);
		color: hsl(200 30% 40%);
	}
	[data-scope='pairing'] {
		background: hsl(330 60% 50% / 0.15);
		color: hsl(330 60% 35%);
	}
	[data-scope='network'] {
		background: hsl(160 50% 45% / 0.15);
		color: hsl(160 50% 30%);
	}
	[data-scope='encryption'] {
		background: hsl(270 50% 55% / 0.15);
		color: hsl(270 50% 40%);
	}
	[data-scope='default'] {
		background: hsl(0 0% 60% / 0.15);
		color: hsl(0 0% 45%);
	}
</style>
