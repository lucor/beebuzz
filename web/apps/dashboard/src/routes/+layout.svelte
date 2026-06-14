<script lang="ts">
	import '../app.css';
	import { Toast } from '@beebuzz/shared/components';

	let { children }: { children: import('svelte').Snippet } = $props();

	const APP_NAME = 'BeeBuzz';
	const THEME_COLOR = '#f59e0b';
	const FAVICON_PATH = '/assets/favicon-196.png';
</script>

<svelte:head>
	<title>{APP_NAME}</title>
	<meta name="theme-color" content={THEME_COLOR} />
	<link rel="icon" type="image/png" sizes="196x196" href={FAVICON_PATH} />
</svelte:head>

{@render children()}

<Toast />
