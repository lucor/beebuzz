<script lang="ts">
	import { onboarding } from '../onboarding.svelte';
	import type { InstallPlatform } from '../onboarding.svelte';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { Share, Plus, Download } from '@lucide/svelte';

	interface Props {
		platform: InstallPlatform;
	}

	let { platform }: Props = $props();

	let installed = $derived(onboarding.state !== 'not-installed');
</script>

<main class="flex items-center justify-center min-h-dvh bg-base-100 px-4">
	<div class="w-full max-w-md">
		<!-- Header -->
		<div class="text-center mb-8">
			<div class="flex flex-col items-center justify-center mb-4">
				<BeeBuzzLogo variant="img" class="w-16 h-16 mb-2" />
				<BeeBuzzLogo variant="text" class="w-48 h-12" />
			</div>
			<p class="text-lg text-base-content/80 font-medium">Install BeeBuzz to get started</p>
		</div>

		<!-- Card -->
		<div class="rounded-lg border border-base-300 bg-base-100 p-8 mb-6 shadow-md">
			{#if platform === 'ios'}
				<h2 class="text-lg font-bold text-base-content mb-4">Add to Home Screen</h2>
				<p class="text-base-content/70 text-sm mb-6">
					BeeBuzz requires installation on iOS to receive push notifications.
				</p>

				<ol class="space-y-4 text-sm text-base-content">
					<li class="flex items-start gap-3">
						<span
							class="flex-shrink-0 w-7 h-7 rounded-full bg-warning text-warning-content flex items-center justify-center font-bold text-xs"
							>1</span
						>
						<div>
							<p class="font-semibold">Tap the Share icon</p>
							<p class="text-base-content/70 mt-0.5">
								<Share size={14} class="inline-block mb-0.5" /> at the bottom of Safari
							</p>
						</div>
					</li>
					<li class="flex items-start gap-3">
						<span
							class="flex-shrink-0 w-7 h-7 rounded-full bg-warning text-warning-content flex items-center justify-center font-bold text-xs"
							>2</span
						>
						<div>
							<p class="font-semibold">Tap "Add to Home Screen"</p>
							<p class="text-base-content/70 mt-0.5">
								<Plus size={14} class="inline-block mb-0.5" /> Scroll down in the share sheet
							</p>
						</div>
					</li>
					<li class="flex items-start gap-3">
						<span
							class="flex-shrink-0 w-7 h-7 rounded-full bg-warning text-warning-content flex items-center justify-center font-bold text-xs"
							>3</span
						>
						<div>
							<p class="font-semibold">Open BeeBuzz from your Home Screen</p>
							<p class="text-base-content/70 mt-0.5">The app will continue from this step</p>
						</div>
					</li>
				</ol>
			{:else if platform === 'safari-macos'}
				<h2 class="text-lg font-bold text-base-content mb-4">Add to Dock</h2>
				<p class="text-base-content/70 text-sm mb-6">
					Install BeeBuzz as a standalone app for reliable notifications.
				</p>

				<ol class="space-y-4 text-sm text-base-content">
					<li class="flex items-start gap-3">
						<span
							class="flex-shrink-0 w-7 h-7 rounded-full bg-warning text-warning-content flex items-center justify-center font-bold text-xs"
							>1</span
						>
						<div>
							<p class="font-semibold">Select "Add to Dock"</p>
							<p class="text-base-content/70 mt-0.5">
								From the <Share size={14} class="inline-block mb-0.5" /> Share menu or File menu
							</p>
						</div>
					</li>
					<li class="flex items-start gap-3">
						<span
							class="flex-shrink-0 w-7 h-7 rounded-full bg-warning text-warning-content flex items-center justify-center font-bold text-xs"
							>2</span
						>
						<div>
							<p class="font-semibold">Open BeeBuzz from the Dock</p>
							<p class="text-base-content/70 mt-0.5">The app will continue from this step</p>
						</div>
					</li>
				</ol>

				<div class="mt-4 rounded-lg bg-base-200 p-3">
					<p class="text-xs text-base-content/70">
						Tip: For auto-start, go to System Settings, General, Login Items and add BeeBuzz.
					</p>
				</div>
			{:else if platform === 'chromium'}
				<h2 class="text-lg font-bold text-base-content mb-4">Install BeeBuzz</h2>
				<p class="text-base-content/70 text-sm mb-6">
					Install the app for a dedicated window and reliable notifications.
				</p>

				<button
					type="button"
					class="btn btn-block btn-lg btn-primary font-semibold mb-4"
					onclick={() => onboarding.handleNativeInstall()}
					disabled={installed}
				>
					<Download size={20} />
					Install BeeBuzz
				</button>

				{#if installed}
					<p class="text-sm text-center text-base-content/70">
						Now open BeeBuzz from your apps to continue.
					</p>
				{/if}
			{:else if platform === 'already-installed'}
				<h2 class="text-lg font-bold text-base-content mb-4">BeeBuzz is already installed</h2>
				<p class="text-base-content/70 text-sm mb-6">
					Open the app to continue setup. If you installed it recently, check your home screen or
					app drawer.
				</p>
				<button
					type="button"
					class="text-sm text-base-content/70 hover:text-base-content underline"
					onclick={() => onboarding.skipInstall()}
				>
					Can't find it? Continue in browser
				</button>
			{:else}
				<!-- browser-fallback: Chromium without beforeinstallprompt, or unknown browser -->
				<h2 class="text-lg font-bold text-base-content mb-4">Install BeeBuzz</h2>
				<p class="text-base-content/70 text-sm mb-6">
					Install BeeBuzz for reliable notifications. Look for the install icon in your browser's
					address bar, or use your browser's menu to install this app.
				</p>
			{/if}
		</div>

		<!-- Can't install? fallback -->
		<div class="text-center">
			<button
				type="button"
				class="text-sm text-base-content/70 hover:text-base-content underline"
				onclick={() => onboarding.skipInstall()}
			>
				Can't install? Continue in browser
			</button>
			<p class="text-xs text-base-content/70 mt-1 max-w-xs mx-auto">
				Notification reliability may be reduced without installation.
			</p>
		</div>
	</div>
</main>
