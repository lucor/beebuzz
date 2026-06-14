<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';

	onMount(async () => {
		// Redirect to overview page
		await goto(resolve('/account/overview'));
	});
</script>

<!-- This page redirects to overview -->
<div class="text-center py-8">
	<span class="loading loading-spinner loading-lg text-primary"></span>
</div>
