<script lang="ts">
	import { onboarding } from '../onboarding.svelte';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { TriangleAlert, RefreshCw } from '@lucide/svelte';
</script>

<main class="flex items-center justify-center min-h-dvh bg-base-100 px-4">
	<div class="w-full max-w-md">
		<!-- Header -->
		<div class="text-center mb-8">
			<div class="flex flex-col items-center justify-center mb-4">
				<BeeBuzzLogo variant="img" class="w-16 h-16 mb-2" />
			</div>
		</div>

		<!-- Card -->
		<div class="rounded-lg border border-base-300 bg-base-100 p-8 shadow-md">
			<div class="flex items-center justify-center mb-4">
				<div class="p-3 bg-amber-100 rounded-full">
					<TriangleAlert size={32} class="text-amber-500" />
				</div>
			</div>

			<h2 class="text-lg font-bold text-base-content text-center mb-2">Something went wrong</h2>

			<p class="text-base-content/70 text-sm text-center mb-6">
				{onboarding.errorMessage ?? 'An unexpected error occurred during pairing.'}
			</p>

			<button
				type="button"
				class="btn btn-block btn-primary font-semibold"
				onclick={() => onboarding.retry()}
			>
				<RefreshCw size={18} />
				Try Again
			</button>
		</div>
	</div>
</main>
