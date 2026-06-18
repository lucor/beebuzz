<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { isLoggedIn } from '@beebuzz/shared/utils/cookie';
	import AuthEntry from '$lib/components/AuthEntry.svelte';

	onMount(() => {
		if (isLoggedIn()) {
			void goto(resolve('/account'));
		}
	});
</script>

<div class="min-h-dvh flex flex-col">
	<main class="flex-1 p-8 px-6">
		<AuthEntry redirectAfterSubmit="/auth/verify" />
	</main>
</div>
