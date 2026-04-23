<script lang="ts">
	import { health } from '../stores/health.svelte';
	import { CircleCheck, CircleAlert, Loader } from '@lucide/svelte';
	import { onMount } from 'svelte';

	onMount(() => {
		if (health.status !== 'unknown' || health.loading) {
			return;
		}

		void health.check();
	});
</script>

<div class="flex items-center gap-2">
	<div class="flex items-center gap-2 text-xs text-base-content/70">
		{#if health.loading}
			<Loader size={14} class="animate-spin" />
		{:else if health.status === 'ok'}
			<CircleCheck size={14} class="text-success" />
		{:else if health.status === 'error'}
			<CircleAlert size={14} class="text-warning" />
		{:else}
			<div class="w-3.5 h-3.5 rounded-full bg-base-300"></div>
		{/if}

		{#if health.version}
			<span>{health.version}</span>
		{/if}
	</div>
</div>
