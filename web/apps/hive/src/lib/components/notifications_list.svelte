<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { inboxUiStore } from '$lib/stores/inbox_ui.svelte';
	import { notificationsStore, groupByDay } from '$lib/stores/notifications.svelte';
	import NotificationCard from '$lib/components/notification_card.svelte';
	import InboxHeader from '$lib/components/inbox_header.svelte';
	import TopicsPanel from '$lib/components/topics_panel.svelte';
	import TopicChips from '$lib/components/topic_chips.svelte';
	import { Bell } from '@lucide/svelte';

	let isTopicsModalOpen = $state(false);
	let topicsDialog = $state<HTMLDialogElement | undefined>(undefined);
	let bulkActionDialog = $state<HTMLDialogElement | undefined>(undefined);
	let selectedTopic = $derived(page.url.searchParams.get('topic'));
	let pendingBulkAction = $state<{
		type: 'delete_visible' | 'delete_selected';
		ids: string[];
		title: string;
		description: string;
	} | null>(null);

	const topics = $derived(notificationsStore.topicSummaries);

	const filteredList = $derived.by(() => {
		if (!selectedTopic) return notificationsStore.list;
		return notificationsStore.list.filter((notification) => notification.topic === selectedTopic);
	});

	const grouped = $derived(groupByDay(filteredList));
	const selectedTopicCount = $derived(filteredList.length);
	const selectedCount = $derived(inboxUiStore.selectedCount);
	const unreadFilteredCount = $derived.by(() =>
		filteredList.reduce((count, notification) => {
			return notificationsStore.unreadIds.has(notification.id) ? count + 1 : count;
		}, 0)
	);
	const selectionSnapshot = $derived.by(() =>
		filteredList.filter((notification) => inboxUiStore.selectedIds.has(notification.id))
	);
	const unreadSelectionCount = $derived.by(() =>
		selectionSnapshot.reduce((count, notification) => {
			return notificationsStore.unreadIds.has(notification.id) ? count + 1 : count;
		}, 0)
	);
	const readSelectionCount = $derived(selectionSnapshot.length - unreadSelectionCount);

	const visibleIds = $derived(filteredList.map((n) => n.id));
	const allVisibleSelected = $derived(
		visibleIds.length > 0 && visibleIds.every((id) => inboxUiStore.selectedIds.has(id))
	);
	const someVisibleSelected = $derived(
		visibleIds.some((id) => inboxUiStore.selectedIds.has(id)) && !allVisibleSelected
	);

	type HeaderMode = 'inbox' | 'topic' | 'selection';
	const headerMode: HeaderMode = $derived.by(() => {
		if (inboxUiStore.selectionMode) return 'selection';
		if (selectedTopic) return 'topic';
		return 'inbox';
	});

	$effect(() => {
		inboxUiStore.pruneSelection(notificationsStore.list.map((notification) => notification.id));
	});

	$effect(() => {
		if (!inboxUiStore.selectionMode) return;

		const handleKeydown = (event: KeyboardEvent) => {
			if (event.key !== 'Escape') return;
			inboxUiStore.exitSelection();
		};

		document.addEventListener('keydown', handleKeydown);
		return () => document.removeEventListener('keydown', handleKeydown);
	});

	$effect(() => {
		if (!topicsDialog) return;
		if (isTopicsModalOpen) {
			topicsDialog.showModal();
		} else {
			topicsDialog.close();
		}
	});

	$effect(() => {
		if (!bulkActionDialog) return;
		if (pendingBulkAction) {
			bulkActionDialog.showModal();
		} else {
			bulkActionDialog.close();
		}
	});

	function handleTopicsDialogClose() {
		isTopicsModalOpen = false;
	}

	function handleBulkActionDialogClose() {
		pendingBulkAction = null;
	}

	async function selectTopic(topic: string | null) {
		if (inboxUiStore.selectionMode) return;

		const url = new URL(page.url);

		if (topic) {
			url.searchParams.set('topic', topic);
		} else {
			url.searchParams.delete('topic');
		}

		isTopicsModalOpen = false;
		await goto(`${url.pathname}${url.search}${url.hash}`, {
			keepFocus: true,
			noScroll: true
		});
	}

	function clearTopicFilter() {
		void selectTopic(null);
	}

	function markFilteredAsRead() {
		const unreadIds = filteredList
			.filter((notification) => notificationsStore.unreadIds.has(notification.id))
			.map((notification) => notification.id);

		if (unreadIds.length === 0) return;
		notificationsStore.markManyAsRead(unreadIds);
	}

	function startSelectionMode(id?: string) {
		inboxUiStore.enterSelection(id);
	}

	function toggleSelection(id: string) {
		inboxUiStore.toggleSelection(id);
	}

	function confirmDeleteVisible() {
		const ids = filteredList.map((notification) => notification.id);
		if (ids.length === 0 || !selectedTopic) return;

		pendingBulkAction = {
			type: 'delete_visible',
			ids: [...ids],
			title: `Delete ${ids.length} ${ids.length === 1 ? 'message' : 'messages'}?`,
			description: `This will delete ${ids.length} ${
				ids.length === 1 ? 'message' : 'messages'
			} in #${selectedTopic} from this device.`
		};
	}

	function confirmDeleteSelected() {
		const ids = [...selectionSnapshot.map((notification) => notification.id)];
		if (ids.length === 0) return;

		pendingBulkAction = {
			type: 'delete_selected',
			ids,
			title: `Delete ${ids.length} ${ids.length === 1 ? 'message' : 'messages'}?`,
			description: `This will delete the selected ${ids.length} ${
				ids.length === 1 ? 'message' : 'messages'
			} from this device.`
		};
	}

	function confirmBulkAction() {
		if (!pendingBulkAction) return;

		notificationsStore.removeMany(pendingBulkAction.ids);
		if (pendingBulkAction.type === 'delete_selected') {
			inboxUiStore.exitSelection();
		}
		pendingBulkAction = null;
	}

	function markSelectedAsRead() {
		const ids = [...selectionSnapshot.map((notification) => notification.id)].filter((id) =>
			notificationsStore.unreadIds.has(id)
		);
		if (ids.length === 0) return;
		notificationsStore.markManyAsRead(ids);
	}

	function markSelectedAsUnread() {
		const ids = [...selectionSnapshot.map((notification) => notification.id)].filter(
			(id) => !notificationsStore.unreadIds.has(id)
		);
		if (ids.length === 0) return;
		notificationsStore.markManyAsUnread(ids);
	}
</script>

<div class="mx-auto max-w-6xl app-content px-4 py-6 lg:px-6">
	{#if notificationsStore.isEmpty}
		<div class="card bg-base-100 shadow-sm p-10 text-center">
			<div class="flex justify-center mb-4">
				<Bell size={64} class="text-base-content/50" />
			</div>
			<h3 class="text-lg font-bold text-base-content">No notifications yet</h3>
			<p class="text-base-content/60">You'll see new messages here when they arrive</p>
		</div>
	{:else}
		<div class="grid gap-6 lg:grid-cols-[240px_minmax(0,1fr)]">
			<!-- Desktop: topic rail -->
			{#if topics.length > 0}
				<aside class="hidden lg:block">
					<div class="sticky top-4">
						<TopicsPanel
							{topics}
							{selectedTopic}
							onSelectTopic={selectTopic}
							disabled={inboxUiStore.selectionMode}
						/>
					</div>
				</aside>
			{/if}

			<!-- Main feed column -->
			<section class="min-w-0 {topics.length === 0 ? 'lg:col-span-full' : ''}">
				<!-- Contextual header: inbox / topic / selection -->
				<InboxHeader
					mode={headerMode}
					{selectedTopic}
					topicMessageCount={selectedTopicCount}
					{selectedCount}
					{unreadFilteredCount}
					{allVisibleSelected}
					{someVisibleSelected}
					hasSelection={selectionSnapshot.length > 0}
					hasUnreadSelection={unreadSelectionCount > 0}
					hasReadSelection={readSelectionCount > 0}
					onStartSelection={() => startSelectionMode()}
					onExitSelection={inboxUiStore.exitSelection}
					onToggleAllVisible={() => inboxUiStore.toggleAll(visibleIds)}
					onMarkSelectedRead={markSelectedAsRead}
					onMarkSelectedUnread={markSelectedAsUnread}
					onDeleteSelected={confirmDeleteSelected}
					onClearTopic={clearTopicFilter}
					onMarkAllRead={markFilteredAsRead}
					onDeleteVisible={confirmDeleteVisible}
				/>

				<!-- Mobile: topic chip row -->
				{#if topics.length > 0 && !inboxUiStore.selectionMode}
					<div class="mb-4 lg:hidden">
						<TopicChips
							{topics}
							{selectedTopic}
							onSelectTopic={selectTopic}
							onShowAll={() => {
								isTopicsModalOpen = true;
							}}
							disabled={inboxUiStore.selectionMode}
						/>
					</div>
				{/if}

				<div class="space-y-4">
					<div class="sr-only" aria-live="polite">
						{selectedCount} selected
					</div>

					{#each grouped.orderedLabels as dayLabel (dayLabel)}
						<div class="space-y-3">
							<h2 class="text-sm font-semibold uppercase tracking-wide text-base-content/60">
								{dayLabel}
							</h2>
							{#each grouped.groups.get(dayLabel) ?? [] as notification (notification.id)}
								<NotificationCard
									{notification}
									onSelectTopic={selectTopic}
									selectionMode={inboxUiStore.selectionMode}
									selected={inboxUiStore.selectedIds.has(notification.id)}
									hideTopic={Boolean(selectedTopic)}
									onToggleSelection={toggleSelection}
									onEnterSelection={startSelectionMode}
								/>
							{/each}
						</div>
					{/each}

					{#if filteredList.length === 0}
						<div class="card bg-base-100 shadow-sm p-10 text-center">
							<p class="text-base-content/70">No messages for #{selectedTopic}</p>
							<div class="mt-4">
								<button type="button" class="btn btn-ghost btn-sm" onclick={clearTopicFilter}>
									Clear filter
								</button>
							</div>
						</div>
					{/if}
				</div>
			</section>
		</div>
	{/if}
</div>

<!-- Mobile: topics bottom sheet -->
<dialog
	bind:this={topicsDialog}
	class="modal modal-bottom lg:hidden"
	onclose={handleTopicsDialogClose}
>
	<div class="modal-box max-h-[80dvh] rounded-t-2xl p-0">
		<div class="flex justify-center py-2">
			<span class="h-1 w-10 rounded-full bg-base-300"></span>
		</div>
		<TopicsPanel
			{topics}
			{selectedTopic}
			onSelectTopic={selectTopic}
			disabled={inboxUiStore.selectionMode}
		/>
		<div class="border-t border-base-300 px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))]">
			<form method="dialog">
				<button type="submit" class="btn btn-ghost btn-sm w-full">Close</button>
			</form>
		</div>
	</div>
	<form method="dialog" class="modal-backdrop"><button type="submit">close</button></form>
</dialog>

<dialog bind:this={bulkActionDialog} class="modal" onclose={handleBulkActionDialogClose}>
	<div class="modal-box">
		<h3 class="text-lg font-bold">{pendingBulkAction?.title}</h3>
		<p class="py-4 text-sm text-base-content/70">{pendingBulkAction?.description}</p>
		<div class="modal-action flex flex-col gap-2 sm:flex-row sm:justify-end">
			<form method="dialog" class="w-full sm:w-auto">
				<button type="submit" class="btn btn-outline w-full">Cancel</button>
			</form>
			<button type="button" class="btn btn-error" onclick={confirmBulkAction}>Delete</button>
		</div>
	</div>
	<form method="dialog" class="modal-backdrop"><button type="submit">close</button></form>
</dialog>
