<script lang="ts">
	import { resolve } from '$app/paths';
	import { Toast, BeeBuzzLogo } from '@beebuzz/shared/components';
	import MarkdownRenderer from '@beebuzz/shared/components/markdown/MarkdownRenderer.svelte';
	import { Menu, X, type Icon } from '@lucide/svelte';
	import PublicFooter from '$lib/components/PublicFooter.svelte';

	let {
		children,
		navSections,
		activeHref,
		useMarkdown = false
	}: {
		children: import('svelte').Snippet;
		navSections: {
			title: string;
			icon: typeof Icon;
			items: { label: string; href: string }[];
		}[];
		activeHref: string;
		useMarkdown?: boolean;
	} = $props();

	let sidebarOpen = $state(false);

	/** Closes the sidebar. */
	function closeSidebar() {
		sidebarOpen = false;
	}
</script>

<div class="flex h-screen flex-col">
	<nav class="navbar fixed left-0 right-0 top-0 z-50 bg-base-100 px-4 shadow-sm md:px-8">
		<div class="flex flex-1 items-center gap-4">
			<button
				aria-label="Toggle sidebar"
				class="btn btn-square btn-ghost lg:hidden"
				onclick={() => (sidebarOpen = !sidebarOpen)}
			>
				{#if sidebarOpen}
					<X size={24} />
				{:else}
					<Menu size={24} />
				{/if}
			</button>

			<a href={resolve('/')} class="hidden items-center gap-2 sm:flex">
				<BeeBuzzLogo variant="img" class="h-10 w-10" />
				<BeeBuzzLogo variant="text" class="hidden h-8 w-24 md:block" />
			</a>
		</div>

		<div class="navbar-center sm:hidden">
			<a href={resolve('/')} class="flex items-center gap-2">
				<BeeBuzzLogo variant="img" class="h-10 w-10" />
			</a>
		</div>

		<div class="navbar-end"></div>
	</nav>

	<div class="flex flex-1 overflow-hidden pt-16">
		{#if sidebarOpen}
			<button
				class="fixed inset-0 z-30 bg-black/50 lg:hidden"
				onclick={() => (sidebarOpen = false)}
				aria-label="Close sidebar"
				type="button"
			></button>
		{/if}

		<aside
			class="fixed bottom-0 left-0 top-16 z-40 w-64 overflow-y-auto border-r border-base-300 bg-base-200 shadow-lg transition-transform duration-300 lg:relative lg:top-0
			{sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}"
		>
			<nav class="w-full p-4 text-base-content md:p-8">
				{#each navSections as section (section.title)}
					<div class="mb-6">
						<div
							class="mb-2 flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-base-content/70"
						>
							<section.icon size={16} aria-hidden="true" />
							<span>{section.title}</span>
						</div>
						<ul class="space-y-1">
							{#each section.items as item (item.href)}
								<li>
									<a
										href={item.href}
										class={`block rounded-lg px-3 py-2 text-sm transition-colors ${
											activeHref === item.href
												? 'bg-primary font-medium text-primary-content'
												: 'hover:bg-base-300'
										}`}
										onclick={closeSidebar}
									>
										{item.label}
									</a>
								</li>
							{/each}
						</ul>
					</div>
				{/each}
			</nav>
		</aside>

		<main class="flex-1 overflow-y-auto bg-base-100">
			<article
				class="prose prose-sm mx-auto flex min-h-full max-w-6xl flex-col p-4 md:prose-base md:p-8"
			>
				<div class="flex-1">
					{#if useMarkdown}
						<MarkdownRenderer>{@render children()}</MarkdownRenderer>
					{:else}
						{@render children()}
					{/if}
				</div>

				<div class="not-prose mt-12">
					<PublicFooter />
				</div>
			</article>
		</main>
	</div>
</div>

<Toast />
