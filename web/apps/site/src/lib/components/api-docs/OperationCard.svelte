<script lang="ts" module>
	export function statusBadgeClass(status: string): string {
		const n = parseInt(status, 10);
		if (!Number.isFinite(n)) return 'badge-ghost';
		if (n >= 200 && n < 300) return 'badge-success';
		if (n >= 300 && n < 400) return 'badge-info';
		if (n >= 400 && n < 500) return 'badge-warning';
		if (n >= 500) return 'badge-error';
		return 'badge-ghost';
	}
</script>

<script lang="ts">
	import type { ApiOperation, ApiSecurityScheme } from '$lib/api-docs/types';
	import { buildExample } from '$lib/api-docs/examples';
	import MethodBadge from './MethodBadge.svelte';
	import ParamTable from './ParamTable.svelte';
	import SchemaView from './SchemaView.svelte';
	import JsonExample from './JsonExample.svelte';
	import { Lock } from '@lucide/svelte';

	let {
		operation,
		securitySchemes
	}: { operation: ApiOperation; securitySchemes: Record<string, ApiSecurityScheme> } = $props();

	const requiredSchemes = $derived(collectRequiredSchemes(operation, securitySchemes));

	function collectRequiredSchemes(
		op: ApiOperation,
		schemes: Record<string, ApiSecurityScheme>
	): ApiSecurityScheme[] {
		const seen: Record<string, true> = {};
		const result: ApiSecurityScheme[] = [];
		for (const requirement of op.security) {
			for (const name of Object.keys(requirement)) {
				if (seen[name]) continue;
				seen[name] = true;
				if (schemes[name]) result.push(schemes[name]);
			}
		}
		return result;
	}

	function schemeLabel(scheme: ApiSecurityScheme): string {
		if (scheme.type === 'http' && scheme.scheme === 'bearer') return 'Bearer';
		if (scheme.type === 'apiKey') return `API key (${scheme.in ?? 'header'})`;
		if (scheme.type === 'oauth2') return 'OAuth 2.0';
		return scheme.name;
	}

	type TabKey = 'request' | 'responses' | 'schema';

	let activeTab = $state<TabKey>('request');

	const pathParams = $derived(operation.parameters.filter((p) => p.in === 'path'));
	const queryParams = $derived(operation.parameters.filter((p) => p.in === 'query'));
	const headerParams = $derived(operation.parameters.filter((p) => p.in === 'header'));

	const jsonBody = $derived(
		operation.requestBody?.content.find((c) => c.mediaType.includes('json'))
	);
	const requestExample = $derived(
		jsonBody ? (jsonBody.example ?? buildExample(jsonBody.schema)) : null
	);

	const anchorId = $derived(slug(operation));

	function slug(op: ApiOperation): string {
		if (op.operationId) return `op-${op.operationId}`;
		return `op-${op.method}-${op.path.replace(/[^a-z0-9]+/gi, '-').replace(/^-+|-+$/g, '')}`;
	}
</script>

<article
	id={anchorId}
	class="card bg-base-100 border border-base-300 shadow-sm scroll-mt-20"
	class:opacity-70={operation.deprecated}
>
	<div class="card-body p-4 md:p-6 gap-4">
		<header class="not-prose flex flex-col gap-2">
			<div class="flex flex-wrap items-center gap-2">
				<MethodBadge method={operation.method} />
				<span class="text-sm md:text-base break-all">{operation.path}</span>
				{#if operation.deprecated}
					<span class="badge badge-warning badge-sm">deprecated</span>
				{/if}
			</div>
			{#if operation.summary}
				<h3 class="text-lg font-semibold m-0">{operation.summary}</h3>
			{/if}
			<div class="flex flex-wrap gap-1">
				{#if operation.stability}
					<span class="badge badge-outline badge-sm">{operation.stability}</span>
				{/if}
				{#if operation.audience}
					{#each operation.audience as audience (audience)}
						<span class="badge badge-outline badge-sm">{audience}</span>
					{/each}
				{/if}
				{#each requiredSchemes as scheme (scheme.name)}
					<span class="badge badge-warning badge-sm gap-1">
						<Lock size={10} aria-hidden="true" />
						{schemeLabel(scheme)}
					</span>
				{/each}
			</div>
		</header>

		{#if operation.description}
			<!-- eslint-disable svelte/no-at-html-tags -- safe HTML from our OpenAPI spec -->
			<p class="op-desc not-prose text-sm text-base-content/80 whitespace-pre-wrap break-words m-0">
				{@html operation.description}
			</p>
			<!-- eslint-enable svelte/no-at-html-tags -->
		{/if}

		<div role="tablist" class="not-prose tabs tabs-border w-full">
			<button
				role="tab"
				type="button"
				class="tab"
				class:tab-active={activeTab === 'request'}
				onclick={() => (activeTab = 'request')}
			>
				Request
			</button>
			<button
				role="tab"
				type="button"
				class="tab"
				class:tab-active={activeTab === 'responses'}
				onclick={() => (activeTab = 'responses')}
			>
				Responses
				<span class="badge badge-ghost badge-xs ml-2">{operation.responses.length}</span>
			</button>
			{#if jsonBody?.schema}
				<button
					role="tab"
					type="button"
					class="tab"
					class:tab-active={activeTab === 'schema'}
					onclick={() => (activeTab = 'schema')}
				>
					Body schema
				</button>
			{/if}
		</div>

		{#if activeTab === 'request'}
			<div class="not-prose space-y-3">
				<ParamTable parameters={pathParams} title="Path parameters" />
				<ParamTable parameters={queryParams} title="Query parameters" />
				<ParamTable parameters={headerParams} title="Header parameters" />

				{#if operation.requestBody}
					<div class="mt-2">
						<h4 class="mb-2 text-sm font-semibold uppercase tracking-wide text-base-content/70">
							Request body
							{#if operation.requestBody.required}
								<span class="badge badge-error badge-xs ml-2">required</span>
							{/if}
						</h4>
						{#if operation.requestBody.description}
							<p class="text-sm text-base-content/80 mb-2">{operation.requestBody.description}</p>
						{/if}
						<div class="flex flex-wrap gap-1 mb-2">
							{#each operation.requestBody.content as media (media.mediaType)}
								<span class="badge badge-outline badge-sm font-mono">{media.mediaType}</span>
							{/each}
						</div>
						{#if requestExample !== null}
							<JsonExample value={requestExample} label="Example request" />
						{/if}
					</div>
				{:else if pathParams.length === 0 && queryParams.length === 0 && headerParams.length === 0}
					<p class="text-sm italic text-base-content/60">No parameters or body.</p>
				{/if}
			</div>
		{/if}

		{#if activeTab === 'responses'}
			<div class="not-prose space-y-3">
				{#each operation.responses as resp (resp.status)}
					{@const statusBadge = statusBadgeClass(resp.status)}
					{@const jsonResp = resp.content.find((c) => c.mediaType.includes('json'))}
					<div class="rounded-box border border-base-300 bg-base-100 p-3">
						<div class="flex flex-wrap items-baseline gap-2 mb-2">
							<span class="badge {statusBadge} badge-sm font-mono">{resp.status}</span>
							<span class="text-sm">{resp.description ?? ''}</span>
						</div>
						{#if resp.headers && Object.keys(resp.headers).length > 0}
							<div class="mb-2">
								<h5 class="text-xs font-semibold uppercase text-base-content/70 mb-1">Headers</h5>
								<ul class="text-xs space-y-0.5">
									{#each Object.entries(resp.headers) as [name, h] (name)}
										<li>
											<code class="font-mono">{name}</code>
											{#if h.description}<span class="text-base-content/70">— {h.description}</span
												>{/if}
										</li>
									{/each}
								</ul>
							</div>
						{/if}
						{#if resp.content.length > 0}
							<div class="flex flex-wrap gap-1 mb-2">
								{#each resp.content as media (media.mediaType)}
									<span class="badge badge-outline badge-xs font-mono">{media.mediaType}</span>
								{/each}
							</div>
						{/if}
						{#if jsonResp?.schema}
							<JsonExample
								value={jsonResp.example ?? buildExample(jsonResp.schema)}
								label="Example response"
							/>
						{/if}
					</div>
				{:else}
					<p class="text-sm italic text-base-content/60">No documented responses.</p>
				{/each}
			</div>
		{/if}

		{#if activeTab === 'schema' && jsonBody?.schema}
			<div class="not-prose">
				<SchemaView schema={jsonBody.schema} />
			</div>
		{/if}
	</div>
</article>

<style>
	.op-desc :global(code) {
		background-color: color-mix(in oklab, var(--color-base-300) 50%, transparent);
		padding-inline: 0.375rem;
		padding-block: 0.125rem;
		border-radius: 0.25rem;
		font-family: monospace;
		font-size: 0.75rem;
	}
</style>
