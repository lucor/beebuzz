<script lang="ts">
	import { Copy, Check } from '@lucide/svelte';
	import { onDestroy } from 'svelte';

	interface CodeLine {
		/** Text content of the line. */
		text: string;
		/** Prefix shown before the line (e.g. "$", ">", "1"). */
		prefix?: string;
		/** Optional CSS class for the <pre> element (e.g. "text-warning", "text-success"). */
		class?: string;
	}

	interface Props {
		/** Lines to display inside the mockup-code block. */
		lines: CodeLine[];
		/** Accessible label for the copy button. */
		copyLabel?: string;
	}

	let { lines, copyLabel = 'Copy code' }: Props = $props();

	let copied = $state(false);
	let resetTimeout: ReturnType<typeof setTimeout> | null = null;

	/** Build plain text from lines for clipboard copy. */
	function buildPlainText(): string {
		return lines.map((l) => l.text).join('\n');
	}

	/** Copy content to clipboard and show a brief check-mark. */
	async function handleCopy() {
		try {
			await navigator.clipboard.writeText(buildPlainText());
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

<div class="mockup-code relative">
	<button
		type="button"
		class="btn btn-ghost btn-xs btn-square absolute top-2 right-2 text-neutral-content hover:bg-neutral-content/10 z-10"
		onclick={handleCopy}
		aria-label={copyLabel}
		title={copyLabel}
	>
		{#if copied}
			<Check size={14} />
		{:else}
			<Copy size={14} />
		{/if}
	</button>
	{#each lines as line, idx (idx)}
		<pre data-prefix={line.prefix} class={line.class}><code>{line.text}</code></pre>
	{/each}
</div>
