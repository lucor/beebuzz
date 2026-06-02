<script lang="ts">
	import type { ApiSpec, ApiOperation } from '$lib/api-docs/types';
	import MethodBadge from './MethodBadge.svelte';
	import OperationCard from './OperationCard.svelte';
	import AuthenticationSection from './AuthenticationSection.svelte';
	import { Search, Lock } from '@lucide/svelte';

	let { spec }: { spec: ApiSpec } = $props();

	let query = $state('');

	const normalizedQuery = $derived(query.trim().toLowerCase());
	const hasBearerAuth = $derived(
		spec.securitySchemes.bearerAuth?.type === 'http' &&
			spec.securitySchemes.bearerAuth?.scheme === 'bearer'
	);

	function matches(op: ApiOperation): boolean {
		if (!normalizedQuery) return true;
		const haystack = [op.path, op.summary, op.operationId, op.method]
			.filter((value): value is string => typeof value === 'string')
			.join(' ')
			.toLowerCase();
		return haystack.includes(normalizedQuery);
	}

	const filteredTags = $derived(
		spec.tags
			.map((tag) => ({
				...tag,
				operations: tag.operations.filter(matches)
			}))
			.filter((tag) => tag.operations.length > 0)
	);

	function operationAnchor(op: ApiOperation): string {
		if (op.operationId) return `op-${op.operationId}`;
		return `op-${op.method}-${op.path.replace(/[^a-z0-9]+/gi, '-').replace(/^-+|-+$/g, '')}`;
	}
</script>

<div class="not-prose flex gap-6 min-h-full">
	<aside
		class="hidden md:block w-64 shrink-0 sticky top-20 self-start max-h-[calc(100dvh-6rem)] overflow-y-auto pr-2"
	>
		<div class="mb-3">
			<label class="input input-sm w-full">
				<Search size={14} aria-hidden="true" />
				<input
					type="search"
					placeholder="Search endpoints"
					bind:value={query}
					aria-label="Search endpoints"
				/>
			</label>
		</div>
		<nav class="space-y-4 text-sm">
			{#if hasBearerAuth}
				<div>
					<a
						href="#authentication"
						class="flex items-center gap-2 rounded px-2 py-1 hover:bg-base-200"
					>
						<Lock size={14} aria-hidden="true" />
						<span class="font-medium">Authentication</span>
					</a>
				</div>
			{/if}
			{#each filteredTags as tag (tag.name)}
				<div>
					<a
						href="#tag-{tag.name}"
						class="block mb-1 text-xs font-semibold uppercase tracking-wide text-base-content/70 hover:text-base-content"
					>
						{tag.name}
					</a>
					<ul class="space-y-0.5">
						{#each tag.operations as op (operationAnchor(op))}
							<li>
								<a
									href="#{operationAnchor(op)}"
									class="flex items-center gap-2 rounded px-2 py-1 !no-underline hover:bg-base-200"
								>
									<MethodBadge method={op.method} size="xs" />
									<span class="text-xs truncate" title={op.path}>
										{op.summary ?? op.path}
									</span>
								</a>
							</li>
						{/each}
					</ul>
				</div>
			{:else}
				<p class="text-sm italic text-base-content/60">No results.</p>
			{/each}
		</nav>
	</aside>

	<div class="flex-1 min-w-0 space-y-8">
		<header class="not-prose">
			<h1 class="text-3xl font-bold mb-1">{spec.title}</h1>
			{#if spec.version}
				<p class="text-sm text-base-content/70">Version {spec.version}</p>
			{/if}
			{#if spec.description}
				<p class="mt-3 text-base text-base-content/80 whitespace-pre-wrap">
					{spec.description}
				</p>
			{/if}
			<div class="md:hidden mt-4">
				<label class="input input-sm w-full">
					<Search size={14} aria-hidden="true" />
					<input
						type="search"
						placeholder="Search endpoints"
						bind:value={query}
						aria-label="Search endpoints"
					/>
				</label>
			</div>
		</header>

		<AuthenticationSection schemes={spec.securitySchemes} />

		{#each filteredTags as tag (tag.name)}
			<section id="tag-{tag.name}" class="space-y-4 scroll-mt-20">
				<div class="not-prose">
					<h2 class="text-2xl font-bold mb-1">{tag.name}</h2>
					{#if tag.description}
						<p class="text-sm text-base-content/70">{tag.description}</p>
					{/if}
				</div>
				<div class="space-y-4">
					{#each tag.operations as op (operationAnchor(op))}
						<OperationCard operation={op} securitySchemes={spec.securitySchemes} />
					{/each}
				</div>
			</section>
		{:else}
			<p class="not-prose text-base-content/60 italic">No endpoints match your search.</p>
		{/each}
	</div>
</div>
