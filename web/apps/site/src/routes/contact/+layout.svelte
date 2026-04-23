<script lang="ts">
	import { page } from '$app/state';
	import SectionLayout from '$lib/components/section-layout.svelte';
	import { MessageSquare, type Icon } from '@lucide/svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	type NavSection = {
		title: string;
		icon: typeof Icon;
		items: { label: string; href: string }[];
	};

	const navSections: NavSection[] = [
		{
			title: 'Contact',
			icon: MessageSquare,
			items: [
				{ label: 'Security', href: '#security' },
				{ label: 'Support', href: '#support' },
				{ label: 'Privacy', href: '#privacy' }
			]
		}
	];

	let currentHash = $derived(page.url.hash);
</script>

<SectionLayout {navSections} activeHref={currentHash}>
	{@render children()}
</SectionLayout>
