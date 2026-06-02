<script lang="ts">
	import '../app.css';
	import { dev } from '$app/environment';
	import { Toast } from '@beebuzz/shared/components';

	interface Props {
		children: import('svelte').Snippet;
	}

	const { children }: Props = $props();

	const APP_NAME = dev ? 'BeeBuzz DEV' : 'BeeBuzz';
	const THEME_COLOR = dev ? '#7FB5CC' : '#f59e0b';
	const FAVICON_PATH = dev ? '/assets/dev/favicon-196.png' : '/assets/favicon-196.png';
	const APPLE_ICON_PATH = dev ? '/assets/dev/apple-icon-180.png' : '/apple-touch-icon.png';
</script>

<svelte:head>
	<title>{APP_NAME}</title>
	<meta name="theme-color" content={THEME_COLOR} />
	<meta name="apple-mobile-web-app-title" content={APP_NAME} />
	<link rel="manifest" href="/manifest.json" />
	<link rel="icon" type="image/png" sizes="196x196" href={FAVICON_PATH} />
	<link rel="apple-touch-icon" href={APPLE_ICON_PATH} />
</svelte:head>

{@render children()}
<Toast />
