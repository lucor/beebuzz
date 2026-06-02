<script lang="ts">
	import '../app.css';
	import { dev } from '$app/environment';
	import { Toast } from '@beebuzz/shared/components';

	let { children }: { children: import('svelte').Snippet } = $props();

	const APP_NAME = dev ? 'BeeBuzz DEV' : 'BeeBuzz';
	const THEME_COLOR = dev ? '#7FB5CC' : '#f59e0b';
	const FAVICON_PATH = dev ? '/assets/dev/favicon-196.png' : '/assets/favicon-196.png';
</script>

<svelte:head>
	<title>{APP_NAME}</title>
	<meta name="theme-color" content={THEME_COLOR} />
	<link rel="icon" type="image/png" sizes="196x196" href={FAVICON_PATH} />
</svelte:head>

{@render children()}

<Toast />
