<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		loadDeveloperSettings,
		setDeveloperModeEnabled,
		listLogs,
		clearLogs,
		listSnapshots,
		clearSnapshots,
		clearAllDeveloperDiagnostics
	} from '$lib/devmode/storage';
	import { subscribeToLogs } from '$lib/devmode/safe-logger';
	import { developerSettings } from '$lib/devmode/settings';
	import { buildHiveDebugReport } from '$lib/devmode/reports';
	import { submitDebugReport } from '$lib/devmode/api';
	import { runEncryptionProbe } from '$lib/services/encryption-diagnostics';
	import {
		loadPushDebugSnapshot,
		updateServiceWorkerRegistration,
		activateWaitingServiceWorker,
		unregisterServiceWorker
	} from '$lib/services/debug-diagnostics';
	import type { HiveLogEntry, HiveErrorSnapshot } from '$lib/devmode/types';
	import type { EncryptionProbeResult, PushDebugSnapshot } from '$lib/types/encryption';

	type Tab = 'console' | 'reports' | 'settings';

	let activeTab = $state<Tab>('console');
	let devModeEnabled = $state(false);
	let showEnableConfirm = $state(false);
	let logs = $state<HiveLogEntry[]>([]);
	let liveLogs = $state<HiveLogEntry[]>([]);
	let snapshots = $state<HiveErrorSnapshot[]>([]);
	let logFilterKind = $state<'all' | 'main' | 'developer'>('all');
	let logFilterScope = $state<string>('');
	let logSearch = $state<string>('');
	let logPaused = $state(false);
	let logFollowTail = $state(true);
	let selectedLog = $state<HiveLogEntry | null>(null);
	let selectedSnapshot = $state<HiveErrorSnapshot | null>(null);
	let submitting = $state<string | null>(null);
	let submitResult = $state<string | null>(null);
	let submitError = $state<string | null>(null);

	let probeStatus = $state<'idle' | 'running' | 'passed' | 'failed'>('idle');
	let probeResult = $state<EncryptionProbeResult | null>(null);
	let probeError = $state<string | null>(null);
	let pushSnapshot = $state<PushDebugSnapshot | null>(null);
	let pushError = $state<string | null>(null);
	let refreshingPush = $state(false);
	let updatingSw = $state(false);
	let activatingSw = $state(false);
	let unregisteringSw = $state(false);

	let unsubLogs: (() => void) | null = null;

	const filteredLogs = $derived.by(() => {
		let items = logPaused ? logs : [...liveLogs, ...logs];
		items.sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime());

		if (logFilterKind !== 'all') {
			items = items.filter((l) => l.kind === logFilterKind);
		}
		if (logFilterScope) {
			items = items.filter((l) => l.scope === logFilterScope);
		}
		if (logSearch) {
			const q = logSearch.toLowerCase();
			items = items.filter(
				(l) =>
					l.event.toLowerCase().includes(q) ||
					l.message.toLowerCase().includes(q) ||
					l.scope.toLowerCase().includes(q)
			);
		}
		return items;
	});

	const scopes = $derived.by<string[]>(() => {
		const all = [...liveLogs, ...logs];
		return [...new Set(all.map((l) => l.scope))].sort();
	});

	const loadInitialData = async () => {
		const settings = await loadDeveloperSettings();
		devModeEnabled = settings.enabled;
		developerSettings.set(settings);

		if (settings.enabled) {
			logs = await listLogs();
			snapshots = await listSnapshots();
		}
	};

	const handleToggle = async () => {
		if (!devModeEnabled) {
			showEnableConfirm = true;
			return;
		}
		await doDisable();
	};

	const startLiveLogSubscription = () => {
		if (unsubLogs) return;
		unsubLogs = subscribeToLogs((entry) => {
			liveLogs = [entry, ...liveLogs].slice(0, 500);
		});
	};

	const stopLiveLogSubscription = () => {
		if (!unsubLogs) return;
		unsubLogs();
		unsubLogs = null;
	};

	const doEnable = async () => {
		showEnableConfirm = false;
		devModeEnabled = true;
		await setDeveloperModeEnabled(true);
		developerSettings.set({ enabled: true });
		logs = await listLogs();
		snapshots = await listSnapshots();
		startLiveLogSubscription();
	};

	const doDisable = async () => {
		devModeEnabled = false;
		selectedLog = null;
		selectedSnapshot = null;
		stopLiveLogSubscription();
		await setDeveloperModeEnabled(false);
		developerSettings.set({ enabled: false });
		logs = [];
		liveLogs = [];
		snapshots = [];
	};

	const handleClearAll = async () => {
		await clearAllDeveloperDiagnostics();
		logs = [];
		liveLogs = [];
		snapshots = [];
		selectedLog = null;
		selectedSnapshot = null;
	};

	const handleClearLogs = async () => {
		await clearLogs();
		logs = [];
		liveLogs = [];
	};

	const handleClearSnapshots = async () => {
		await clearSnapshots();
		snapshots = [];
	};

	const handleSubmitReport = async (snapshot: HiveErrorSnapshot) => {
		submitting = snapshot.id;
		submitResult = null;
		submitError = null;

		try {
			const report = buildHiveDebugReport(snapshot);
			const response = await submitDebugReport(report);
			if (response) {
				submitResult = `Report submitted: ${response.report_id}`;
			} else {
				submitError = 'Failed to submit report';
			}
		} catch (error) {
			submitError = error instanceof Error ? error.message : 'Unknown error';
		} finally {
			submitting = null;
		}
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

	onMount(async () => {
		await loadInitialData();

		if (devModeEnabled) {
			startLiveLogSubscription();
		}
	});

	onDestroy(() => {
		stopLiveLogSubscription();
	});
</script>

<div class="flex flex-col gap-6">
	<div>
		<h1 class="text-3xl font-bold text-base-content">Developer Mode</h1>
		<p class="text-sm text-base-content/70">Privacy-safe local diagnostics for Hive</p>
	</div>

	{#if devModeEnabled}
		<div role="tablist" class="tabs tabs-bordered">
			<button
				role="tab"
				class="tab {activeTab === 'console' ? 'tab-active' : ''}"
				onclick={() => (activeTab = 'console')}
			>
				Console
			</button>
			<button
				role="tab"
				class="tab {activeTab === 'reports' ? 'tab-active' : ''}"
				onclick={() => (activeTab = 'reports')}
			>
				Reports
			</button>
			<button
				role="tab"
				class="tab {activeTab === 'settings' ? 'tab-active' : ''}"
				onclick={() => (activeTab = 'settings')}
			>
				Settings
			</button>
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
				<button class="btn btn-primary" onclick={() => (showEnableConfirm = true)}>
					Enable Developer Mode
				</button>
			</div>
		</div>
	{:else if activeTab === 'console'}
		<div class="card bg-base-100 shadow-md">
			<div class="card-body gap-4">
				<div class="flex flex-wrap items-center justify-between gap-3">
					<div class="flex flex-wrap items-center gap-2">
						<div class="join">
							<button
								class="join-item btn btn-sm {logFilterKind === 'all' ? 'btn-active' : ''}"
								onclick={() => (logFilterKind = 'all')}
							>
								All
							</button>
							<button
								class="join-item btn btn-sm {logFilterKind === 'main' ? 'btn-active' : ''}"
								onclick={() => (logFilterKind = 'main')}
							>
								Main
							</button>
							<button
								class="join-item btn btn-sm {logFilterKind === 'developer' ? 'btn-active' : ''}"
								onclick={() => (logFilterKind = 'developer')}
							>
								Developer
							</button>
						</div>

						<select
							class="select select-bordered select-sm"
							value={logFilterScope}
							onchange={(e) => (logFilterScope = e.currentTarget.value)}
						>
							<option value="">All scopes</option>
							{#each scopes as scope (scope)}
								<option value={scope}>{scope}</option>
							{/each}
						</select>

						<label class="input input-bordered input-sm flex items-center gap-2">
							<input type="search" class="grow" placeholder="Search..." bind:value={logSearch} />
						</label>
					</div>

					<div class="flex flex-wrap items-center gap-2">
						<button
							class="btn btn-sm {logPaused ? 'btn-warning' : 'btn-outline'}"
							onclick={() => (logPaused = !logPaused)}
						>
							{logPaused ? 'Paused' : 'Pause'}
						</button>
						<button
							class="btn btn-sm {logFollowTail ? 'btn-primary' : 'btn-outline'}"
							onclick={() => (logFollowTail = !logFollowTail)}
						>
							Follow
						</button>
						<button class="btn btn-sm btn-outline" onclick={() => void handleCopyLogs()}>
							Copy
						</button>
						<button class="btn btn-sm btn-outline" onclick={() => void handleDownloadLogs()}>
							Download
						</button>
						<button class="btn btn-sm btn-ghost text-error" onclick={() => void handleClearLogs()}>
							Clear
						</button>
					</div>
				</div>

				{#if selectedLog}
					<div class="rounded-xl border border-base-300 bg-base-200/50 p-4">
						<div class="flex items-start justify-between gap-3">
							<div class="space-y-1">
								<p class="font-mono text-xs text-base-content/60">{selectedLog.id}</p>
								<p class="font-mono text-xs text-base-content/60">{selectedLog.ts}</p>
								<p class="mt-2">
									<span class="badge badge-sm">{selectedLog.kind}</span>
									<span class="badge badge-sm badge-outline ml-1">{selectedLog.scope}</span>
								</p>
								<p class="mt-2 font-medium">{selectedLog.event}</p>
								<p class="text-sm text-base-content/70">{selectedLog.message}</p>
								{#if selectedLog.data}
									<pre class="mt-2 rounded-lg bg-base-300 p-3 font-mono text-xs overflow-x-auto">
{JSON.stringify(selectedLog.data, null, 2)}</pre>
								{/if}
							</div>
							<button class="btn btn-ghost btn-xs" onclick={() => (selectedLog = null)}>
								Close
							</button>
						</div>
					</div>
				{/if}

				<div class="max-h-96 overflow-y-auto rounded-xl border border-base-300 bg-base-200/50">
					{#if filteredLogs.length === 0}
						<p class="p-4 text-center text-sm text-base-content/50">No log entries.</p>
					{:else}
						{#each filteredLogs as log (log.id)}
							<button
								class="flex w-full items-start gap-3 border-b border-base-300/50 px-4 py-2 text-left hover:bg-base-300/50 transition-colors"
								onclick={() => (selectedLog = selectedLog?.id === log.id ? null : log)}
							>
								<span
									class="badge badge-sm mt-0.5 shrink-0
									{log.kind === 'main' ? 'badge-primary' : 'badge-ghost'}"
								>
									{log.kind}
								</span>
								<span class="badge badge-sm badge-outline mt-0.5 shrink-0">{log.scope}</span>
								<span class="min-w-0 flex-1 font-mono text-xs">
									<span class="font-semibold">{log.event}</span>
									<span class="text-base-content/50 ml-1">{log.message}</span>
								</span>
								<span class="shrink-0 font-mono text-xs text-base-content/40">
									{new Date(log.ts).toLocaleTimeString()}
								</span>
							</button>
						{/each}
					{/if}
				</div>
			</div>
		</div>
	{:else if activeTab === 'reports'}
		<div class="card bg-base-100 shadow-md">
			<div class="card-body gap-4">
				<div class="flex items-center justify-between gap-3">
					<p class="text-sm text-base-content/70">
						Error snapshots are captured automatically when errors occur.
					</p>
					{#if submitResult}
						<div class="alert alert-success text-sm">{submitResult}</div>
					{/if}
					{#if submitError}
						<div class="alert alert-error text-sm">{submitError}</div>
					{/if}
					<button
						class="btn btn-ghost btn-sm text-error"
						onclick={() => void handleClearSnapshots()}
					>
						Clear all
					</button>
				</div>

				<div class="rounded-xl border border-base-300 bg-base-200 p-4 text-sm text-base-content/60">
					<p>Nothing is sent automatically.</p>
					<p>
						Only the report shown here will be sent to BeeBuzz. Notification content, attachments,
						API keys, device private keys, full tokens, and push subscription details are never
						collected.
					</p>
				</div>

				{#if snapshots.length === 0}
					<p class="text-center text-sm text-base-content/50 py-8">No error snapshots yet.</p>
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
									class="btn btn-primary btn-sm"
									onclick={() => void handleSubmitReport(snapshot)}
									disabled={submitting === snapshot.id}
								>
									{submitting === snapshot.id ? 'Submitting...' : 'Submit report'}
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
			</div>
		</div>
	{:else if activeTab === 'settings'}
		<div class="card bg-base-100 shadow-md">
			<div class="card-body gap-6">
				<div class="flex items-center justify-between gap-3">
					<div>
						<p class="font-medium">Developer Mode</p>
						<p class="text-sm text-base-content/70">Enable privacy-safe local diagnostics</p>
					</div>
					<input
						type="checkbox"
						class="toggle toggle-primary"
						checked={devModeEnabled}
						onchange={() => void handleToggle()}
					/>
				</div>

				<div class="flex items-center justify-between gap-3">
					<div>
						<p class="font-medium">Clear local diagnostics</p>
						<p class="text-sm text-base-content/70">
							Remove all logs and error snapshots from this device
						</p>
					</div>
					<button class="btn btn-ghost btn-sm text-error" onclick={() => void handleClearAll()}>
						Clear
					</button>
				</div>

				<div
					class="rounded-xl border border-base-300 bg-base-200 p-4 text-sm text-base-content/60 space-y-1"
				>
					<p>Retention: 24 hours</p>
					<p>Max local events: 1,000</p>
					<p>Network metadata: always safe / normalized</p>
					<p>Safe stack traces: captured only when Developer Mode is On</p>
					<p>Auto-clear on logout: always On</p>
					<p>Browser console output: Off in production</p>
					<p>Auto-submit reports: never</p>
				</div>
			</div>
		</div>

		<div class="card bg-base-100 shadow-md">
			<div class="card-body gap-4">
				<div class="flex flex-wrap items-center justify-between gap-3">
					<div>
						<h2 class="text-lg font-semibold text-base-content">Diagnostic Tools</h2>
						<p class="text-sm text-base-content/70">
							Local tools for inspecting service worker and encryption behavior.
						</p>
					</div>
				</div>

				<div class="flex flex-wrap items-center justify-between gap-3">
					<h3 class="text-base font-semibold text-base-content">Push Diagnostics</h3>
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
							onclick={() => void updateServiceWorkerRegistration().then(() => refreshPushDiag())}
							disabled={updatingSw}
						>
							Check update
						</button>
						<button
							class="btn btn-outline btn-sm"
							onclick={() => void activateWaitingServiceWorker().then(() => refreshPushDiag())}
							disabled={activatingSw}
						>
							Activate update
						</button>
						<button
							class="btn btn-outline btn-sm"
							onclick={() => void unregisterServiceWorker().then(() => refreshPushDiag())}
							disabled={unregisteringSw}
						>
							Unregister
						</button>
					</div>
				</div>

				{#if pushError}
					<div class="alert alert-error"><span>{pushError}</span></div>
				{/if}

				{#if pushSnapshot}
					<div class="grid gap-4 lg:grid-cols-2">
						<div class="rounded-xl border border-base-300 bg-base-200/50 p-4">
							<p class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60">
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
							<p class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60">
								Push Subscription
							</p>
							<div class="space-y-2 font-mono text-xs text-base-content/80">
								<p>subscription: {pushSnapshot.subscriptionPresent ? 'present' : 'none'}</p>
								<p>key material: {pushSnapshot.subscriptionKeysPresent ? 'present' : 'none'}</p>
							</div>
						</div>
					</div>
				{/if}
			</div>
		</div>

		<div class="card bg-base-100 shadow-md">
			<div class="card-body gap-4">
				<div class="flex flex-wrap items-center justify-between gap-3">
					<h3 class="text-base font-semibold text-base-content">Encryption Probe</h3>
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
					<div class="alert alert-error"><span>{probeError}</span></div>
				{/if}

				{#if probeResult}
					<div class="rounded-xl border border-base-300 bg-base-200/50 p-4">
						<p class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60">
							Result
						</p>
						<p class="font-mono text-xs text-base-content/80">
							Key persistence: {probeResult.keyPersistence.ok ? 'pass' : 'fail'}
							| Wrapping key: {probeResult.wrappingKey.ok ? 'pass' : 'fail'}
						</p>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

{#if showEnableConfirm}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		role="dialog"
		aria-modal="true"
		aria-labelledby="enable-title"
	>
		<div class="mx-4 max-w-md rounded-2xl bg-base-100 p-6 shadow-xl">
			<h2 id="enable-title" class="text-lg font-semibold text-base-content">
				Enable Developer Mode?
			</h2>
			<p class="mt-2 text-sm text-base-content/70">
				Developer Mode stores privacy-safe diagnostic events locally on this device for up to 24
				hours, including safe stack traces for captured errors. Hive does not collect notification
				content, attachments, API keys, device private keys, full tokens, or push subscription
				secrets. Reports are sent only after you review and submit them.
			</p>
			<div class="mt-6 flex justify-end gap-3">
				<button class="btn btn-ghost" onclick={() => (showEnableConfirm = false)}>Cancel</button>
				<button class="btn btn-primary" onclick={() => void doEnable()}>Enable</button>
			</div>
		</div>
	</div>
{/if}
