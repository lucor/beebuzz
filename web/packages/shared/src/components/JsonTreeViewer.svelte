<script lang="ts">
	interface Props {
		data: Record<string, unknown>;
		selectedPath?: string;
		onPathClick?: (path: string, value: string) => void;
	}

	let { data, selectedPath, onPathClick }: Props = $props();

	function getValueType(
		value: unknown
	): 'string' | 'number' | 'boolean' | 'object' | 'array' | 'null' {
		if (value === null) return 'null';
		if (Array.isArray(value)) return 'array';
		return typeof value as 'string' | 'number' | 'boolean' | 'object';
	}

	function buildPath(parentPath: string, key: string): string {
		if (!parentPath) return key;
		return `${parentPath}.${key}`;
	}

	function isScalar(value: unknown): boolean {
		const type = getValueType(value);
		return type === 'string' || type === 'number' || type === 'boolean' || type === 'null';
	}

	function formatValue(value: unknown): string {
		if (value === null) return 'null';
		if (typeof value === 'string') return `"${value}"`;
		if (typeof value === 'object') return JSON.stringify(value);
		return '' + (value as string | number | boolean);
	}

	function getValueClass(type: string): string {
		if (type === 'string') return 'text-success';
		if (type === 'null') return 'text-warning';
		return 'text-accent';
	}

	function getSelectedClass(isSelected: boolean): string {
		return isSelected
			? 'bg-primary-focus/20 rounded px-1'
			: 'hover:bg-base-300 rounded px-1 cursor-pointer';
	}
</script>

{#snippet renderNode(entries: [string, unknown][], parentPath: string)}
	{#each entries as [key, value] (key)}
		{@const path = buildPath(parentPath, key)}
		{@const type = getValueType(value)}
		{@const isSelected = selectedPath === path}
		<div class="py-0.5">
			{#if isSelected}
				<span class="bg-primary-focus/20 text-primary rounded px-1 mr-1">▶</span>
			{/if}
			{#if isScalar(value)}
				<button
					type="button"
					class={getSelectedClass(isSelected)}
					onclick={() => onPathClick?.(path, formatValue(value))}
				>
					<span class="text-primary">{key}</span>
					<span class="text-base-content/50">: </span>
					<span class={getValueClass(type)}>{formatValue(value)}</span>
				</button>
			{:else if type === 'object'}
				<span class="text-primary">{key}</span>
				<span class="text-base-content/50">: &#123;</span>
				<div class="pl-4 border-l border-base-content/10 ml-2">
					{@render renderNode(Object.entries(value as Record<string, unknown>), path)}
				</div>
				<span class="text-base-content/50">&#125;</span>
			{:else if type === 'array'}
				<span class="text-primary">{key}</span>
				<span class="text-base-content/50">: [</span>
				<div class="pl-4 border-l border-base-content/10 ml-2">
					{#each value as unknown[] as item, idx (idx)}
						<div class="py-0.5">
							{#if isScalar(item)}
								{@const itemType = getValueType(item)}
								<button
									type="button"
									class={getSelectedClass(selectedPath === `${path}.${idx}`)}
									onclick={() => onPathClick?.(`${path}.${idx}`, formatValue(item))}
								>
									<span class={getValueClass(itemType)}>{formatValue(item)}</span>
								</button>
							{:else if typeof item === 'object' && item !== null}
								<span class="text-base-content/50">&#123;</span>
								<div class="pl-4 border-l border-base-content/10 ml-2">
									{@render renderNode(
										Object.entries(item as Record<string, unknown>),
										`${path}.${idx}`
									)}
								</div>
								<span class="text-base-content/50">&#125;</span>
							{:else}
								<span class="text-warning">null</span>
							{/if}
						</div>
					{/each}
				</div>
				<span class="text-base-content/50">]</span>
			{/if}
		</div>
	{/each}
{/snippet}

<div class="font-mono text-xs bg-base-200 rounded p-3 overflow-x-auto">
	{@render renderNode(Object.entries(data), '')}
</div>
