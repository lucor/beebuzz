<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { me } from '@beebuzz/shared/services/account';
	import { auth } from '@beebuzz/shared/stores';

	let { children } = $props<{ children: import('svelte').Snippet }>();

	onMount(() => {
		const checkAuth = async () => {
			try {
				await me();
			} catch {
				await goto(resolve('/login'));
			}
		};
		void checkAuth();
	});
</script>

{#if !auth.user}
	<div class="flex items-center justify-center min-h-dvh">
		<p class="text-base-content/70">Loading...</p>
	</div>
{:else}
	{@render children()}
{/if}
