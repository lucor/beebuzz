<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { isLoggedIn } from '@beebuzz/shared/utils/cookie';
	import { me } from '@beebuzz/shared/services/account';

	let { children }: { children: import('svelte').Snippet } = $props();
	let ready = $state(false);

	onMount(() => {
		const checkAuth = async () => {
			if (!isLoggedIn()) {
				ready = true;
				return;
			}

			try {
				await me();
				await goto(resolve('/account'));
			} catch {
				ready = true;
			}
		};

		void checkAuth();
	});
</script>

{#if !ready}
	<div class="flex items-center justify-center min-h-dvh">
		<p class="text-base-content/70">Loading...</p>
	</div>
{:else}
	<div class="min-h-dvh flex flex-col">
		<main class="flex-1 p-8 px-6">
			{@render children()}
		</main>
	</div>
{/if}
