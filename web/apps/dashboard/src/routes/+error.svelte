<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { House, ArrowLeft } from '@lucide/svelte';
</script>

<div class="min-h-dvh bg-base-100 flex flex-col items-center justify-center px-4">
	<div class="text-center max-w-md">
		<BeeBuzzLogo variant="img" class="w-24 h-24 mb-6 mx-auto" />

		<h1 class="text-6xl font-bold text-primary mb-2">{page.status}</h1>
		<p class="text-xl text-base-content mb-2">
			{#if page.status === 404}
				Page not found
			{:else}
				Something went wrong
			{/if}
		</p>
		<p class="text-base-content/60 mb-8">
			{#if page.error?.message}
				{page.error.message}
			{:else if page.status === 404}
				The page you're looking for doesn't exist or has been moved.
			{:else}
				An unexpected error occurred. Please try again.
			{/if}
		</p>

		<div class="flex flex-col sm:flex-row gap-3 justify-center">
			<button onclick={() => history.back()} class="btn btn-outline">
				<ArrowLeft size={18} />
				Go Back
			</button>
			<a href={resolve('/')} class="btn btn-primary">
				<House size={18} />
				Home
			</a>
		</div>
	</div>
</div>
