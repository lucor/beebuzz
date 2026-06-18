<script lang="ts">
	import type { TopicSummary } from '$lib/stores/notifications.svelte';

	interface Props {
		topics: TopicSummary[];
		selectedTopic: string | null;
		onSelectTopic: (topic: string | null) => void;
		showAllMessages?: boolean;
		disabled?: boolean;
	}

	let {
		topics,
		selectedTopic,
		onSelectTopic,
		showAllMessages = true,
		disabled = false
	}: Props = $props();

	function handleSelect(topic: string | null) {
		if (disabled) return;
		onSelectTopic(topic);
	}
</script>

<div class="flex h-full flex-col bg-base-100">
	<div class="border-b border-base-300 px-4 py-4">
		<h2 class="text-sm font-semibold uppercase tracking-wide text-base-content/60">Topics</h2>
	</div>

	<ul class="menu w-full gap-1 p-3">
		{#if showAllMessages}
			<li>
				<button
					type="button"
					class="flex items-center justify-between rounded-box px-3 py-2 text-sm {selectedTopic ===
					null
						? 'bg-base-200/80 font-semibold'
						: ''}"
					{disabled}
					onclick={() => handleSelect(null)}
				>
					<span>All messages</span>
				</button>
			</li>
		{/if}

		{#each topics as topic (topic.name)}
			<li>
				<button
					type="button"
					class="flex items-center justify-between rounded-box px-3 py-2 text-sm {selectedTopic ===
					topic.name
						? 'bg-base-200/80 font-semibold'
						: ''}"
					{disabled}
					onclick={() => handleSelect(topic.name)}
				>
					<span class="min-w-0 truncate">#{topic.name}</span>
					<span class="flex items-center gap-1.5">
						{#if topic.unreadCount > 0}
							<span class="badge badge-primary badge-xs">{topic.unreadCount}</span>
						{/if}
						<span class="text-xs text-base-content/50">{topic.count}</span>
					</span>
				</button>
			</li>
		{/each}
	</ul>
</div>
