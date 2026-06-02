<script lang="ts">
	import { page } from '$app/state';
	import SectionLayout from '$lib/components/section-layout.svelte';
	import { Zap, type Icon } from '@lucide/svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	type NavSection = {
		title: string;
		icon: typeof Icon;
		items: { label: string; href: string }[];
	};

	const navSections: NavSection[] = [
		{
			title: 'Docs',
			icon: Zap,
			items: [
				{ label: 'Quickstart', href: '/docs/quickstart' },
				{ label: 'Webhooks', href: '/docs/webhooks' },
				{ label: 'Local Dev', href: '/docs/local-dev' },
				{ label: 'Browser Support', href: '/docs/browser-support' }
			]
		}
	];

	let currentPath = $derived(page.url.pathname);
</script>

<SectionLayout {navSections} activeHref={currentPath} useMarkdown={true}>
	{@render children()}
</SectionLayout>
