<script lang="ts">
	import type { TopicSummary } from '$lib/stores/notifications.svelte';

	const MAX_VISIBLE_CHIPS = 5;

	interface Props {
		topics: TopicSummary[];
		selectedTopic: string | null;
		onSelectTopic: (topic: string | null) => void;
		onShowAll: () => void;
		disabled?: boolean;
	}

	let { topics, selectedTopic, onSelectTopic, onShowAll, disabled = false }: Props = $props();

	/** Visible chips: top N topics, ensuring the selected topic is always shown. */
	const visibleTopics = $derived.by(() => {
		const top = topics.slice(0, MAX_VISIBLE_CHIPS);

		if (selectedTopic && !top.some((t) => t.name === selectedTopic)) {
			const selected = topics.find((t) => t.name === selectedTopic);
			if (selected) {
				top.pop();
				top.unshift(selected);
			}
		}

		return top;
	});

	const hasMore = $derived(topics.length > MAX_VISIBLE_CHIPS);
</script>

{#if topics.length > 0}
	<div class="flex items-center gap-2 overflow-x-auto pb-1 scrollbar-none">
		<button
			type="button"
			class="badge whitespace-nowrap {selectedTopic === null
				? 'badge-primary'
				: 'badge-ghost border-base-300'}"
			{disabled}
			onclick={() => onSelectTopic(null)}
		>
			All
		</button>

		{#each visibleTopics as topic (topic.name)}
			<button
				type="button"
				class="badge whitespace-nowrap gap-1 {selectedTopic === topic.name
					? 'badge-primary'
					: 'badge-ghost border-base-300'}"
				{disabled}
				onclick={() => onSelectTopic(topic.name)}
			>
				<span class="truncate max-w-28">#{topic.name}</span>
				{#if topic.unreadCount > 0}
					<span
						class="inline-flex h-4 min-w-4 items-center justify-center rounded-full text-[10px] font-bold {selectedTopic ===
						topic.name
							? 'bg-primary-content text-primary'
							: 'bg-primary text-primary-content'}"
					>
						{topic.unreadCount}
					</span>
				{/if}
			</button>
		{/each}

		{#if hasMore}
			<button
				type="button"
				class="badge badge-ghost border-base-300 whitespace-nowrap"
				{disabled}
				onclick={onShowAll}
			>
				More…
			</button>
		{/if}
	</div>
{/if}
