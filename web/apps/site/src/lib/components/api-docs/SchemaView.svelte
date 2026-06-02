<script lang="ts">
	import Self from './SchemaView.svelte';
	import type { ApiSchema } from '$lib/api-docs/types';
	import { typeLabel } from '$lib/api-docs/examples';

	let { schema, depth = 0 }: { schema: ApiSchema | undefined; depth?: number } = $props();

	const required = $derived(new Set(schema?.required ?? []));
	const properties = $derived(schema?.properties ? Object.entries(schema.properties) : []);
	const isObject = $derived(properties.length > 0);
	const isArray = $derived(
		(Array.isArray(schema?.type) ? schema?.type.includes('array') : schema?.type === 'array') ||
			!!schema?.items
	);
</script>

{#if !schema}
	<span class="text-base-content/60 italic text-sm">no schema</span>
{:else if isObject}
	<ul class="not-prose space-y-2">
		{#each properties as [name, prop] (name)}
			<li class="rounded-box border border-base-300 bg-base-100 p-3">
				<div class="flex flex-wrap items-baseline gap-2">
					<code class="font-mono text-sm font-semibold">{name}</code>
					<span class="font-mono text-xs text-base-content/70">{typeLabel(prop)}</span>
					{#if required.has(name)}
						<span class="badge badge-error badge-xs">required</span>
					{/if}
					{#if prop.deprecated}
						<span class="badge badge-warning badge-xs">deprecated</span>
					{/if}
					{#if prop.readOnly}
						<span class="badge badge-ghost badge-xs">read-only</span>
					{/if}
				</div>
				{#if prop.description}
					<p class="mt-1 text-sm text-base-content/80">{prop.description}</p>
				{/if}
				{#if prop.enum?.length}
					<div class="mt-2 flex flex-wrap gap-1">
						{#each prop.enum as value (String(value))}
							<code class="rounded bg-base-200 px-1.5 py-0.5 text-xs">{String(value)}</code>
						{/each}
					</div>
				{/if}
				{#if depth < 3 && (prop.properties || prop.items?.properties)}
					<details class="mt-2">
						<summary class="cursor-pointer text-xs text-base-content/70 hover:text-base-content">
							Show nested
						</summary>
						<div class="mt-2 pl-3 border-l-2 border-base-300">
							{#if prop.properties}
								<Self schema={prop} depth={depth + 1} />
							{:else if prop.items}
								<Self schema={prop.items} depth={depth + 1} />
							{/if}
						</div>
					</details>
				{/if}
			</li>
		{/each}
	</ul>
{:else if isArray && schema.items}
	<div class="not-prose">
		<p class="text-sm text-base-content/70 mb-2">
			Array of <code class="font-mono">{typeLabel(schema.items)}</code>
		</p>
		<Self schema={schema.items} {depth} />
	</div>
{:else}
	<div class="not-prose flex flex-wrap items-baseline gap-2 text-sm">
		<span class="font-mono text-xs text-base-content/70">{typeLabel(schema)}</span>
		{#if schema.description}
			<span class="text-base-content/80">{schema.description}</span>
		{/if}
		{#if schema.enum?.length}
			<div class="flex flex-wrap gap-1">
				{#each schema.enum as value (String(value))}
					<code class="rounded bg-base-200 px-1.5 py-0.5 text-xs">{String(value)}</code>
				{/each}
			</div>
		{/if}
	</div>
{/if}
