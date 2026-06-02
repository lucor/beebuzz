<script lang="ts">
	import logoImg from '../assets/beebuzz-logo-img.svg?raw';
	import logoText from '../assets/beebuzz-logo-text.svg?raw';

	interface Props {
		variant?: 'img' | 'text';
		class?: string;
	}

	let { variant = 'img', class: className = '' }: Props = $props();

	let svg = $derived(variant === 'img' ? logoImg : logoText);
	let svgWithClass = $derived.by(() => svg.replace('<svg', `<svg class="${className}"`));
</script>

<!-- SVG content is trusted (loaded from static assets via import) -->
<!-- eslint-disable-next-line svelte/no-at-html-tags -->
{@html svgWithClass}

<style>
	:global(svg) {
		display: inline-block;
	}
</style>
