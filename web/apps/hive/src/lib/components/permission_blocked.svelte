<script lang="ts">
	import { onboarding } from '../onboarding.svelte';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { BellOff, Settings, RefreshCw } from '@lucide/svelte';

	let retrying = $state(false);

	/** Handles the retry button click with loading state. */
	const handleRetry = async () => {
		retrying = true;
		await onboarding.retryPermission();
		retrying = false;
	};
</script>

<main class="flex items-center justify-center min-h-dvh bg-[#FDF7ED] px-4">
	<div class="w-full max-w-md">
		<!-- Header -->
		<div class="text-center mb-8">
			<div class="flex flex-col items-center justify-center mb-4">
				<BeeBuzzLogo variant="img" class="w-16 h-16 mb-2" />
			</div>
		</div>

		<!-- Card -->
		<div class="bg-white rounded-lg shadow-md border border-[#E6E1D8] p-8">
			<div class="flex items-center justify-center mb-4">
				<div class="p-3 bg-red-100 rounded-full">
					<BellOff size={32} class="text-red-500" />
				</div>
			</div>

			<h2 class="text-lg font-bold text-[#2D3748] text-center mb-2">Notifications Blocked</h2>

			<p class="text-[#6B7280] text-sm text-center mb-6">
				BeeBuzz needs notification permission to deliver messages. You have blocked notifications
				for this site.
			</p>

			<div class="bg-[#FDF7ED] rounded-lg p-4 mb-6">
				<p class="text-sm font-semibold text-[#2D3748] mb-2 flex items-center gap-2">
					<Settings size={16} />
					How to re-enable
				</p>
				<ol class="text-xs text-[#6B7280] space-y-1 list-decimal list-inside">
					<li>Open your browser settings</li>
					<li>Find "Site Settings" or "Permissions"</li>
					<li>Locate this site and allow notifications</li>
					<li>Come back and tap the button below</li>
				</ol>
			</div>

			<button
				type="button"
				class="btn btn-block btn-primary font-semibold"
				onclick={handleRetry}
				disabled={retrying}
			>
				{#if retrying}
					<span class="loading loading-spinner loading-sm"></span>
					Checking...
				{:else}
					<RefreshCw size={18} />
					I've enabled notifications
				{/if}
			</button>
		</div>
	</div>
</main>
