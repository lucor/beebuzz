<script lang="ts">
	import { Copy, Check } from '@lucide/svelte';
	import { onDestroy } from 'svelte';

	let { value, label = 'Example' }: { value: unknown; label?: string } = $props();

	const text = $derived(JSON.stringify(value, null, 2));

	let copied = $state(false);
	let resetTimeout: ReturnType<typeof setTimeout> | null = null;

	/** Copy JSON to clipboard and show a brief check-mark. */
	async function handleCopy() {
		try {
			await navigator.clipboard.writeText(text);
			copied = true;
			if (resetTimeout) clearTimeout(resetTimeout);
			resetTimeout = setTimeout(() => {
				copied = false;
				resetTimeout = null;
			}, 2000);
		} catch {
			/* clipboard access denied — silently ignored */
		}
	}

	onDestroy(() => {
		if (resetTimeout) clearTimeout(resetTimeout);
	});
</script>

<div class="docs-terminal not-prose">
	<div class="docs-terminal-bar">
		<span class="docs-terminal-title">{label}</span>
		<button
			type="button"
			class="docs-terminal-copy btn btn-ghost btn-xs btn-square"
			onclick={handleCopy}
			aria-label="Copy example"
			title="Copy example"
		>
			{#if copied}
				<Check size={14} />
			{:else}
				<Copy size={14} />
			{/if}
		</button>
	</div>
	<div class="docs-terminal-body">
		<pre><code>{text}</code></pre>
	</div>
</div>
