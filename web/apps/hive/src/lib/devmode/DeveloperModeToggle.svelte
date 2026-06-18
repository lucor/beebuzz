<script lang="ts">
	import { onMount } from 'svelte';
	import {
		loadDeveloperSettings,
		setDeveloperModeEnabled,
		deleteDeveloperDatabase,
		clearAllDeveloperDiagnostics
	} from '$lib/devmode/storage';
	import { developerSettings } from '$lib/devmode/settings';

	let {
		enabled = $bindable(false),
		onEnable = () => {},
		onDisable = () => {},
		showClear = false,
		onClear = () => {}
	}: {
		enabled: boolean;
		onEnable?: () => void | Promise<void>;
		onDisable?: () => void | Promise<void>;
		showClear?: boolean;
		onClear?: () => void | Promise<void>;
	} = $props();

	let showEnableConfirm = $state(false);
	let showDisableConfirm = $state(false);
	let disabling = $state(false);
	let disableError = $state<string | null>(null);
	let clearing = $state(false);

	onMount(async () => {
		const settings = await loadDeveloperSettings();
		enabled = settings.enabled;
	});

	const handleToggle = () => {
		if (!enabled) {
			showEnableConfirm = true;
			return;
		}
		disableError = null;
		showDisableConfirm = true;
	};

	const doEnable = async () => {
		showEnableConfirm = false;
		enabled = true;
		await setDeveloperModeEnabled(true);
		developerSettings.set({ enabled: true });
		await onEnable();
	};

	const doDisable = async () => {
		disabling = true;
		disableError = null;
		developerSettings.set({ enabled: false });
		try {
			await deleteDeveloperDatabase();
		} catch (error) {
			developerSettings.set({ enabled: true });
			await setDeveloperModeEnabled(true);
			disableError = error instanceof Error ? error.message : 'Failed to disable Developer Mode';
			disabling = false;
			return;
		}
		developerSettings.set({ enabled: false });
		enabled = false;
		disabling = false;
		showDisableConfirm = false;
		await onDisable();
	};

	const handleClear = async () => {
		clearing = true;
		try {
			await clearAllDeveloperDiagnostics();
			await onClear();
		} finally {
			clearing = false;
		}
	};
</script>

<div class="flex items-center justify-between gap-3">
	<div>
		<p class="font-medium">Developer Mode</p>
		<p class="text-sm text-base-content/70">Capture local diagnostics on this device.</p>
	</div>
	<button
		type="button"
		role="switch"
		aria-checked={enabled}
		aria-label={enabled ? 'Disable Developer Mode' : 'Enable Developer Mode'}
		class="relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 {enabled
			? 'border-primary bg-primary'
			: 'border-base-content/10 bg-base-300'}"
		onclick={handleToggle}
		disabled={disabling}
	>
		<span
			class="inline-block h-5 w-5 rounded-full bg-base-100 shadow-sm transition-transform {enabled
				? 'translate-x-5'
				: 'translate-x-0.5'}"
			aria-hidden="true"
		></span>
	</button>
</div>

{#if showClear}
	<div class="flex items-center justify-between gap-3">
		<div>
			<p class="font-medium">Clear diagnostics</p>
			<p class="text-sm text-base-content/70">
				Remove logs, warnings, and error reports from this device.
			</p>
		</div>
		<button
			class="btn btn-ghost btn-sm text-error"
			onclick={() => void handleClear()}
			disabled={clearing}
		>
			{clearing ? 'Clearing...' : 'Clear'}
		</button>
	</div>
{/if}

<div class="rounded-xl border border-base-300 bg-base-200 p-4 text-sm text-base-content/60">
	<p>Diagnostics are stored locally for up to 24 hours. Reports are never sent automatically.</p>
	<p class="mt-1">Console warnings and errors are captured only while Developer Mode is enabled.</p>
</div>

{#if showEnableConfirm}
	<div class="modal modal-open" role="dialog" aria-modal="true" aria-labelledby="enable-title">
		<div class="modal-box">
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
		<button
			class="modal-backdrop"
			aria-label="Close modal"
			onclick={() => (showEnableConfirm = false)}
		></button>
	</div>
{/if}

{#if showDisableConfirm}
	<div class="modal modal-open" role="dialog" aria-modal="true" aria-labelledby="disable-title">
		<div class="modal-box">
			<h2 id="disable-title" class="text-lg font-semibold text-base-content">
				Disable Developer Mode?
			</h2>
			<p class="mt-2 text-sm text-base-content/70">
				This deletes all local developer diagnostics on this device.
			</p>
			<div class="mt-6 flex justify-end gap-3">
				<button
					class="btn btn-ghost"
					onclick={() => {
						disableError = null;
						showDisableConfirm = false;
					}}
					disabled={disabling}>Cancel</button
				>
				<button class="btn btn-error" onclick={() => void doDisable()} disabled={disabling}>
					{disabling ? 'Disabling...' : 'Disable'}
				</button>
			</div>
			{#if disableError}
				<div class="alert alert-error mt-4 text-sm" role="alert">
					<span>{disableError}</span>
				</div>
			{/if}
		</div>
		<button
			class="modal-backdrop"
			aria-label="Close modal"
			onclick={() => {
				disableError = null;
				showDisableConfirm = false;
			}}
		></button>
	</div>
{/if}
