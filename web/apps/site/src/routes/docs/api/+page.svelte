<script lang="ts">
	import ApiReference from '$lib/components/api-docs/ApiReference.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>API Reference — BeeBuzz</title>
	<meta
		name="description"
		content="HTTP API reference for the BeeBuzz server, generated from the public OpenAPI contract."
	/>
</svelte:head>

<ApiReference spec={data.spec} />
