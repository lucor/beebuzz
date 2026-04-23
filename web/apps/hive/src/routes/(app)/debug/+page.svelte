<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { runEncryptionProbe } from '$lib/services/encryption-diagnostics';
	import {
		activateWaitingServiceWorker,
		loadPushDebugSnapshot,
		unregisterServiceWorker,
		updateServiceWorkerRegistration
	} from '$lib/services/debug-diagnostics';
	import type {
		EncryptionProbeResult,
		EncryptionProbeScenarioResult,
		EncryptionProbeStatus,
		PushDebugSnapshot
	} from '$lib/types/encryption';

	const DEBUG = import.meta.env.VITE_BEEBUZZ_DEBUG === true;

	const IDLE_STATUS = 'idle';
	const RUNNING_STATUS = 'running';
	const PASSED_STATUS = 'passed';
	const FAILED_STATUS = 'failed';

	let status = $state<EncryptionProbeStatus>(IDLE_STATUS);
	let result = $state<EncryptionProbeResult | null>(null);
	let errorMessage = $state<string | null>(null);
	let pushSnapshot = $state<PushDebugSnapshot | null>(null);
	let pushErrorMessage = $state<string | null>(null);
	let refreshingPushLogs = $state(false);
	let updatingServiceWorker = $state(false);
	let activatingServiceWorker = $state(false);
	let unregisteringServiceWorker = $state(false);
	let copyingProbe = $state(false);
	let probeCopied = $state(false);

	/** Returns the number of passed steps for a scenario. */
	const getPassedStepsCount = (scenario: EncryptionProbeScenarioResult): number => {
		return scenario.steps.filter((step) => step.ok).length;
	};

	/** Maps a boolean result to the corresponding badge class. */
	const getBadgeClass = (ok: boolean): string => {
		return ok ? 'badge badge-success badge-outline' : 'badge badge-error badge-outline';
	};

	/** Formats the full encryption probe result as plain text for copy/paste. */
	const formatProbeResult = (probeResult: EncryptionProbeResult): string => {
		const lines: string[] = [
			'Encryption Probe',
			`Run At: ${probeResult.runAt}`,
			`User Agent: ${probeResult.userAgent}`,
			'',
			'Key Persistence',
			`Status: ${probeResult.keyPersistence.ok ? 'pass' : 'fail'}`,
			`Structured clone: ${probeResult.keyPersistence.structuredClone.ok ? 'pass' : 'fail'}`,
			`Structured clone detail: ${probeResult.keyPersistence.structuredClone.detail}`
		];

		for (const scenario of probeResult.keyPersistence.scenarios) {
			lines.push('');
			lines.push(`Scenario: ${scenario.label}`);
			lines.push(`Status: ${scenario.ok ? 'pass' : 'fail'}`);
			if (scenario.recipient) {
				lines.push(`Recipient: ${scenario.recipient}`);
			}
			for (const step of scenario.steps) {
				lines.push(`- ${step.ok ? 'pass' : 'fail'} | ${step.label} | ${step.detail}`);
			}
		}

		lines.push('');
		lines.push('Wrapping Key');
		lines.push(`Status: ${probeResult.wrappingKey.ok ? 'pass' : 'fail'}`);
		for (const step of probeResult.wrappingKey.steps) {
			lines.push(`- ${step.ok ? 'pass' : 'fail'} | ${step.label} | ${step.detail}`);
		}

		return lines.join('\n');
	};

	/** Copies the current encryption probe result in a plain-text format. */
	const copyProbeResult = async (): Promise<void> => {
		if (!result) {
			return;
		}

		copyingProbe = true;
		probeCopied = false;

		try {
			await navigator.clipboard.writeText(formatProbeResult(result));
			probeCopied = true;
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : String(error);
		} finally {
			copyingProbe = false;
		}
	};

	/** Executes the encryption probe and stores the result for display. */
	const executeProbe = async (): Promise<void> => {
		if (!DEBUG) {
			return;
		}

		status = RUNNING_STATUS;
		errorMessage = null;
		result = null;
		probeCopied = false;

		try {
			const probeResult = await runEncryptionProbe();
			result = probeResult;
			status =
				probeResult.keyPersistence.ok && probeResult.wrappingKey.ok ? PASSED_STATUS : FAILED_STATUS;
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : String(error);
			status = FAILED_STATUS;
		}
	};

	/** Loads the current push/service worker diagnostics snapshot. */
	const refreshPushDiagnostics = async (): Promise<void> => {
		refreshingPushLogs = true;
		pushErrorMessage = null;

		try {
			pushSnapshot = await loadPushDebugSnapshot();
		} catch (error) {
			pushErrorMessage = error instanceof Error ? error.message : String(error);
		} finally {
			refreshingPushLogs = false;
		}
	};

	/** Forces a service worker update check and refreshes local diagnostics. */
	const handleServiceWorkerUpdate = async (): Promise<void> => {
		updatingServiceWorker = true;
		pushErrorMessage = null;

		try {
			await updateServiceWorkerRegistration();
			await refreshPushDiagnostics();
		} catch (error) {
			pushErrorMessage = error instanceof Error ? error.message : String(error);
		} finally {
			updatingServiceWorker = false;
		}
	};

	/** Activates a waiting service worker and reloads once control changes. */
	const handleServiceWorkerActivate = async (): Promise<void> => {
		activatingServiceWorker = true;
		pushErrorMessage = null;

		try {
			const activated = await activateWaitingServiceWorker();
			if (!activated) {
				pushErrorMessage = 'no waiting service worker to activate';
				return;
			}

			await refreshPushDiagnostics();
		} catch (error) {
			pushErrorMessage = error instanceof Error ? error.message : String(error);
		} finally {
			activatingServiceWorker = false;
		}
	};

	/** Unregisters the current service worker so the next load can start clean. */
	const handleServiceWorkerUnregister = async (): Promise<void> => {
		unregisteringServiceWorker = true;
		pushErrorMessage = null;

		try {
			const unregistered = await unregisterServiceWorker();
			if (!unregistered) {
				pushErrorMessage = 'service worker registration not found';
				return;
			}

			await refreshPushDiagnostics();
		} catch (error) {
			pushErrorMessage = error instanceof Error ? error.message : String(error);
		} finally {
			unregisteringServiceWorker = false;
		}
	};

	onMount(() => {
		if (!DEBUG) {
			void goto('/');
			return;
		}

		const handleControllerChange = (): void => {
			window.location.reload();
		};

		navigator.serviceWorker.addEventListener('controllerchange', handleControllerChange);

		void executeProbe();
		void refreshPushDiagnostics();

		return () => {
			navigator.serviceWorker.removeEventListener('controllerchange', handleControllerChange);
		};
	});
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center gap-3">
		<BeeBuzzLogo variant="img" class="h-12 w-12" />
		<div>
			<h1 class="text-3xl font-bold text-base-content">Debug</h1>
			<p class="text-sm text-base-content/70">
				Client, encryption, and push diagnostics for development builds.
			</p>
		</div>
	</div>

	<div class="card bg-base-100 shadow-md">
		<div class="card-body gap-4">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div class="flex items-center gap-3">
					<h2 class="text-lg font-semibold text-base-content">Push Diagnostics</h2>
					{#if refreshingPushLogs}
						<span class="badge badge-warning">refreshing</span>
					{/if}
				</div>
				<div class="flex flex-wrap gap-2">
					<button
						class="btn btn-outline btn-sm"
						onclick={() => void refreshPushDiagnostics()}
						disabled={refreshingPushLogs}
					>
						Refresh
					</button>
					<button
						class="btn btn-outline btn-sm"
						onclick={() => void handleServiceWorkerUpdate()}
						disabled={updatingServiceWorker}
					>
						{updatingServiceWorker ? 'Checking...' : 'Check update'}
					</button>
					<button
						class="btn btn-outline btn-sm"
						onclick={() => void handleServiceWorkerActivate()}
						disabled={activatingServiceWorker}
					>
						{activatingServiceWorker ? 'Activating...' : 'Activate update'}
					</button>
					<button
						class="btn btn-outline btn-sm"
						onclick={() => void handleServiceWorkerUnregister()}
						disabled={unregisteringServiceWorker}
					>
						{unregisteringServiceWorker ? 'Removing...' : 'Unregister'}
					</button>
				</div>
			</div>

			{#if pushErrorMessage}
				<div class="alert alert-error">
					<span>{pushErrorMessage}</span>
				</div>
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
							<p>endpoint host: {pushSnapshot.subscriptionEndpointHost ?? 'none'}</p>
							<p>p256dh length: {pushSnapshot.subscriptionP256dhLength}</p>
							<p>auth length: {pushSnapshot.subscriptionAuthLength}</p>
						</div>
					</div>
				</div>
			{/if}
		</div>
	</div>

	<div class="card bg-base-100 shadow-md">
		<div class="card-body gap-4">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div class="flex items-center gap-3">
					<h2 class="text-lg font-semibold text-base-content">Encryption Probe</h2>
					<span
						class={`badge ${
							status === PASSED_STATUS
								? 'badge-success'
								: status === FAILED_STATUS
									? 'badge-error'
									: status === RUNNING_STATUS
										? 'badge-warning'
										: 'badge-ghost'
						}`}
					>
						{status}
					</span>
				</div>

				<button
					class="btn btn-primary btn-sm"
					onclick={() => void executeProbe()}
					disabled={status === RUNNING_STATUS}
				>
					{status === RUNNING_STATUS ? 'Running...' : 'Run again'}
				</button>
				<button
					class="btn btn-outline btn-sm"
					onclick={() => void copyProbeResult()}
					disabled={!result || copyingProbe}
				>
					{copyingProbe ? 'Copying...' : probeCopied ? 'Copied' : 'Copy results'}
				</button>
			</div>

			{#if errorMessage}
				<div class="alert alert-error">
					<span>{errorMessage}</span>
				</div>
			{/if}

			{#if result}
				<div class="rounded-xl border border-base-300 bg-base-200/50 p-4">
					<p class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60">
						User Agent
					</p>
					<p class="break-all font-mono text-xs text-base-content/80">{result.userAgent}</p>
				</div>

				<div class="rounded-xl border border-base-300 p-4">
					<div class="mb-3 flex items-center justify-between gap-3">
						<h2 class="text-lg font-semibold text-base-content">Key Persistence</h2>
						<span class={getBadgeClass(result.keyPersistence.ok)}>
							{result.keyPersistence.ok ? 'pass' : 'fail'}
						</span>
					</div>
					<div class="rounded-lg border border-base-300 bg-base-200/50 p-3">
						<p class="mb-2 text-xs text-base-content/60">
							This verifies the original failure mode: direct X25519 CryptoKey persistence.
						</p>
						<div class="mb-1 flex items-center justify-between gap-3">
							<p class="text-sm font-medium text-base-content">Structured clone CryptoKey</p>
							<span class={getBadgeClass(result.keyPersistence.structuredClone.ok)}>
								{result.keyPersistence.structuredClone.ok ? 'pass' : 'fail'}
							</span>
						</div>
						<p class="break-words font-mono text-xs text-base-content/70">
							{result.keyPersistence.structuredClone.detail}
						</p>
					</div>
				</div>

				<div class="grid gap-4 lg:grid-cols-2">
					{#each result.keyPersistence.scenarios as scenario (scenario.id)}
						<section class="rounded-2xl border border-base-300 bg-base-100 p-5">
							<div class="mb-4 flex items-start justify-between gap-3">
								<div>
									<h2 class="text-lg font-semibold text-base-content">{scenario.label}</h2>
									<p class="text-xs text-base-content/60">
										{getPassedStepsCount(scenario)}/{scenario.steps.length} steps passed
									</p>
								</div>
								<span class={getBadgeClass(scenario.ok)}>{scenario.ok ? 'pass' : 'fail'}</span>
							</div>

							{#if scenario.recipient}
								<p
									class="mb-4 break-all rounded-lg bg-base-200 px-3 py-2 font-mono text-xs text-base-content/80"
								>
									{scenario.recipient}
								</p>
							{/if}

							<div class="space-y-3">
								{#each scenario.steps as step (step.id)}
									<div class="rounded-xl border border-base-300 p-3">
										<div class="mb-1 flex items-center justify-between gap-3">
											<p class="text-sm font-medium text-base-content">{step.label}</p>
											<span class={getBadgeClass(step.ok)}>{step.ok ? 'pass' : 'fail'}</span>
										</div>
										<p class="break-words font-mono text-xs text-base-content/70">{step.detail}</p>
									</div>
								{/each}
							</div>
						</section>
					{/each}
				</div>

				<section class="rounded-2xl border border-base-300 bg-base-100 p-5">
					<div class="mb-4 flex items-start justify-between gap-3">
						<div>
							<h2 class="text-lg font-semibold text-base-content">Wrapping Key Probe</h2>
							<p class="mt-1 text-xs text-base-content/60">
								This verifies the production path: persist the AES-GCM wrapping key, then
								wrap/unwrap X25519 as non-extractable.
							</p>
							<p class="text-xs text-base-content/60">
								{result.wrappingKey.steps.filter((step) => step.ok).length}/{result.wrappingKey
									.steps.length}
								steps passed
							</p>
						</div>
						<span class={getBadgeClass(result.wrappingKey.ok)}>
							{result.wrappingKey.ok ? 'pass' : 'fail'}
						</span>
					</div>

					<div class="space-y-3">
						{#each result.wrappingKey.steps as step (step.id)}
							<div class="rounded-xl border border-base-300 p-3">
								<div class="mb-1 flex items-center justify-between gap-3">
									<p class="text-sm font-medium text-base-content">{step.label}</p>
									<span class={getBadgeClass(step.ok)}>{step.ok ? 'pass' : 'fail'}</span>
								</div>
								<p class="break-words font-mono text-xs text-base-content/70">{step.detail}</p>
							</div>
						{/each}
					</div>
				</section>
			{:else if status === RUNNING_STATUS}
				<div class="flex items-center gap-3 rounded-xl border border-base-300 bg-base-200/50 p-4">
					<span class="loading loading-spinner loading-md"></span>
					<p class="text-sm text-base-content/70">
						Running key persistence and wrapping-key probes.
					</p>
				</div>
			{/if}
		</div>
	</div>
</div>
