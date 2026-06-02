<script lang="ts">
	import type { ApiParameter } from '$lib/api-docs/types';
	import { typeLabel } from '$lib/api-docs/examples';

	let { parameters, title }: { parameters: ApiParameter[]; title: string } = $props();
</script>

{#if parameters.length > 0}
	<div class="not-prose mt-4">
		<h4 class="mb-2 text-sm font-semibold uppercase tracking-wide text-base-content/70">
			{title}
		</h4>
		<div class="overflow-x-auto rounded-box border border-base-300">
			<table class="table table-zebra table-sm w-full">
				<thead>
					<tr>
						<th class="w-1/4">Name</th>
						<th class="w-1/4">Type</th>
						<th>Description</th>
					</tr>
				</thead>
				<tbody>
					{#each parameters as param (param.in + ':' + param.name)}
						<tr>
							<td class="align-top">
								<div class="font-mono text-sm font-semibold">{param.name}</div>
								<div class="mt-1 flex flex-wrap gap-1">
									{#if param.required}
										<span class="badge badge-error badge-xs">required</span>
									{/if}
									{#if param.deprecated}
										<span class="badge badge-warning badge-xs">deprecated</span>
									{/if}
								</div>
							</td>
							<td class="align-top font-mono text-xs text-base-content/80">
								{typeLabel(param.schema)}
							</td>
							<td class="align-top text-sm">
								{param.description ?? ''}
								{#if param.schema?.enum?.length}
									<div class="mt-1 flex flex-wrap gap-1">
										{#each param.schema.enum as value (String(value))}
											<code class="rounded bg-base-200 px-1.5 py-0.5 text-xs">{String(value)}</code>
										{/each}
									</div>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>
{/if}
