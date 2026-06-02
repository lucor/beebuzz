<script lang="ts">
	import { page } from '$app/state';
	import SectionLayout from '$lib/components/section-layout.svelte';
	import { ScrollText, type Icon } from '@lucide/svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	type NavSection = {
		title: string;
		icon: typeof Icon;
		items: { label: string; href: string }[];
	};

	const navSections: NavSection[] = [
		{
			title: 'Policies',
			icon: ScrollText,
			items: [
				{ label: 'Privacy Policy', href: '#privacy' },
				{ label: 'Terms of Service', href: '#terms' },
				{ label: 'Cookie Policy', href: '#cookies' }
			]
		}
	];

	let currentHash = $derived(page.url.hash);
</script>

<SectionLayout {navSections} activeHref={currentHash}>
	{@render children()}
</SectionLayout>
