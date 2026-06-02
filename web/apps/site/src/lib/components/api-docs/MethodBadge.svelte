<script lang="ts">
	import type { HttpMethod } from '$lib/api-docs/types';

	let { method, size = 'sm' }: { method: HttpMethod; size?: 'xs' | 'sm' } = $props();

	const colorMap: Record<HttpMethod, string> = {
		get: 'text-info',
		post: 'text-success',
		put: 'text-warning',
		patch: 'text-accent',
		delete: 'text-error',
		head: 'text-base-content/50',
		options: 'text-base-content/50'
	};

	const sizeClass = $derived(size === 'xs' ? 'text-xs' : 'text-sm');
</script>

<span class="{colorMap[method]} {sizeClass} font-mono font-semibold uppercase tracking-wide">
	{method}
</span>
