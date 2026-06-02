<script lang="ts">
	import { markdownCodeCopy } from './markdown-code-copy';

	interface Props {
		children?: import('svelte').Snippet;
	}

	let { children }: Props = $props();
</script>

<div class="markdown-content" use:markdownCodeCopy>
	{@render children?.()}
</div>
