<script lang="ts">
	import {
		CheckCheck,
		Trash2,
		X,
		ListChecks,
		Mail,
		ArrowLeft,
		SquareCheck,
		Square
	} from '@lucide/svelte';

	type HeaderMode = 'inbox' | 'topic' | 'selection';

	interface Props {
		mode: HeaderMode;
		selectedTopic?: string | null;
		topicMessageCount?: number;
		selectedCount?: number;
		unreadFilteredCount?: number;
		allVisibleSelected?: boolean;
		someVisibleSelected?: boolean;
		hasSelection?: boolean;
		hasUnreadSelection?: boolean;
		hasReadSelection?: boolean;
		onStartSelection?: () => void;
		onExitSelection?: () => void;
		onToggleAllVisible?: () => void;
		onMarkSelectedRead?: () => void;
		onMarkSelectedUnread?: () => void;
		onDeleteSelected?: () => void;
		onClearTopic?: () => void;
		onMarkAllRead?: () => void;
		onDeleteVisible?: () => void;
	}

	let {
		mode,
		selectedTopic = null,
		topicMessageCount = 0,
		selectedCount = 0,
		unreadFilteredCount = 0,
		allVisibleSelected = false,
		someVisibleSelected = false,
		hasSelection = false,
		hasUnreadSelection = false,
		hasReadSelection = false,
		onStartSelection,
		onExitSelection,
		onToggleAllVisible,
		onMarkSelectedRead,
		onMarkSelectedUnread,
		onDeleteSelected,
		onClearTopic,
		onMarkAllRead,
		onDeleteVisible
	}: Props = $props();
</script>

{#if mode === 'selection'}
	<div
		class="sticky top-0 z-10 mb-4 flex flex-wrap items-center gap-2 rounded-box border border-base-300 bg-base-100 px-4 py-3 shadow-sm"
	>
		<button
			type="button"
			class="btn btn-ghost btn-sm btn-square"
			aria-label={allVisibleSelected ? 'Deselect all in view' : 'Select all in view'}
			onclick={onToggleAllVisible}
		>
			{#if allVisibleSelected}
				<SquareCheck size={18} class="text-primary" />
			{:else if someVisibleSelected}
				<SquareCheck size={18} class="text-base-content/50" />
			{:else}
				<Square size={18} />
			{/if}
		</button>
		<div class="mr-auto min-w-0">
			<p class="truncate text-sm font-semibold text-base-content">
				{selectedCount} selected
			</p>
		</div>
		<button
			type="button"
			class="btn btn-ghost btn-sm"
			disabled={!hasUnreadSelection}
			onclick={onMarkSelectedRead}
		>
			<CheckCheck size={16} />
			<span class="hidden sm:inline">Mark as read</span>
		</button>
		<button
			type="button"
			class="btn btn-ghost btn-sm"
			disabled={!hasReadSelection}
			onclick={onMarkSelectedUnread}
		>
			<Mail size={16} />
			<span class="hidden sm:inline">Mark unread</span>
		</button>
		<button
			type="button"
			class="btn btn-ghost btn-sm text-error"
			disabled={!hasSelection}
			onclick={onDeleteSelected}
		>
			<Trash2 size={16} />
			<span class="hidden sm:inline">Delete</span>
		</button>
		<button
			type="button"
			class="btn btn-ghost btn-sm"
			aria-label="Exit selection mode"
			onclick={onExitSelection}
		>
			<X size={16} />
			<span class="hidden sm:inline">Cancel</span>
		</button>
	</div>
{:else if mode === 'topic' && selectedTopic}
	<div
		class="sticky top-0 z-10 mb-4 flex items-center gap-2 rounded-box border border-base-300 bg-base-200/60 px-4 py-3 shadow-sm"
	>
		<button
			type="button"
			class="btn btn-ghost btn-sm btn-square"
			aria-label="Back to all messages"
			onclick={onClearTopic}
		>
			<ArrowLeft size={16} />
		</button>
		<div class="mr-auto min-w-0">
			<p class="truncate text-sm font-semibold text-base-content">#{selectedTopic}</p>
			<p class="text-xs text-base-content/55">
				{topicMessageCount}
				{topicMessageCount === 1 ? 'message' : 'messages'}
			</p>
		</div>
		<button
			type="button"
			class="btn btn-ghost btn-sm"
			disabled={unreadFilteredCount === 0}
			onclick={onMarkAllRead}
		>
			<CheckCheck size={16} />
			<span class="hidden sm:inline">Mark all read</span>
		</button>
		<button type="button" class="btn btn-ghost btn-sm text-error" onclick={onDeleteVisible}>
			<Trash2 size={16} />
			<span class="hidden sm:inline">Delete</span>
		</button>
	</div>
{:else}
	<div class="mb-4 flex items-center justify-between gap-3">
		<div class="min-w-0">
			<h1 class="text-lg font-semibold text-base-content sm:text-2xl">Inbox</h1>
			<p class="hidden text-sm text-base-content/60 sm:block">
				Your notifications, grouped by day.
			</p>
		</div>
		<div class="flex items-center gap-2">
			<button
				type="button"
				class="btn btn-ghost btn-sm"
				aria-label="Select notifications"
				onclick={onStartSelection}
			>
				<ListChecks size={16} />
				Select
			</button>
		</div>
	</div>
{/if}
