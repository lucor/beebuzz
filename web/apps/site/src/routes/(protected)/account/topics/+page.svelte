<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from '@beebuzz/shared/stores';
	import type { Topic } from '@beebuzz/shared/api';
	import { topicsApi } from '@beebuzz/shared/api';
	import { ApiError } from '@beebuzz/shared/errors';
	import { SettingsModal, TopicChip } from '@beebuzz/shared/components';
	import { Plus, EllipsisVertical, Pencil, Trash2 } from '@lucide/svelte';
	import { MAX_DESCRIPTION_LEN } from '@beebuzz/shared';

	let topics = $state<Topic[]>([]);
	let isLoading = $state(false);
	let showCreateModal = $state(false);
	let newTopicName = $state('');
	let newTopicDescription = $state('');
	let editingTopicId = $state<string | null>(null);
	let topicPendingDelete = $state<Topic | null>(null);
	let editDescription = $state('');
	let actionsMenuOpenTopicId = $state<string | null>(null);
	let actionsMenuRef = $state<HTMLElement | undefined>(undefined);

	onMount(async () => {
		await loadTopics();
	});

	$effect(() => {
		if (!actionsMenuOpenTopicId) return;

		// Close the actions menu when the click lands outside the current menu container.
		const handleClickOutside = (e: MouseEvent) => {
			if (actionsMenuRef && !actionsMenuRef.contains(e.target as Node)) {
				actionsMenuOpenTopicId = null;
			}
		};

		document.addEventListener('click', handleClickOutside, true);
		return () => document.removeEventListener('click', handleClickOutside, true);
	});

	async function loadTopics() {
		isLoading = true;
		try {
			topics = await topicsApi.listTopics();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load topics');
		} finally {
			isLoading = false;
		}
	}

	async function handleCreate() {
		if (!newTopicName.trim()) {
			toast.error('Topic name is required');
			return;
		}

		isLoading = true;
		try {
			await topicsApi.createTopic(newTopicName, newTopicDescription);
			newTopicName = '';
			newTopicDescription = '';
			showCreateModal = false;
			toast.success('Topic created');
			await loadTopics();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to create topic');
		} finally {
			isLoading = false;
		}
	}

	function isValidTopicName(name: string): boolean {
		const regex = /^[a-z][a-z0-9_]{0,31}$/;
		return regex.test(name);
	}

	function startEdit(topic: Topic) {
		actionsMenuOpenTopicId = null;
		editingTopicId = topic.id;
		editDescription = topic.description || '';
	}

	function cancelEdit() {
		editingTopicId = null;
		editDescription = '';
	}

	function toggleTopicActions(topicId: string) {
		actionsMenuOpenTopicId = actionsMenuOpenTopicId === topicId ? null : topicId;
	}

	function openDeleteDialog(topic: Topic) {
		actionsMenuOpenTopicId = null;
		topicPendingDelete = topic;
	}

	async function handleUpdate() {
		if (!editingTopicId) return;

		isLoading = true;
		try {
			await topicsApi.updateTopic(editingTopicId, editDescription);
			toast.success('Topic updated');
			await loadTopics();
			cancelEdit();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to update topic');
		} finally {
			isLoading = false;
		}
	}

	async function confirmDeleteTopic() {
		if (!topicPendingDelete) return;
		isLoading = true;
		try {
			await topicsApi.deleteTopic(topicPendingDelete.id);
			toast.success('Topic deleted');
			topicPendingDelete = null;
			await loadTopics();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to delete topic');
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="p-6">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-base-content">Topics</h1>
		<button
			onclick={() => {
				showCreateModal = true;
			}}
			disabled={isLoading}
			class="btn btn-primary gap-2"
		>
			<Plus size={20} />
			Create Topic
		</button>
	</div>

	<div class="divider my-4"></div>

	<p class="text-sm text-base-content/70 mb-6">
		Organize notifications by topic. Create custom topics to categorize your notifications.
	</p>

	<SettingsModal
		open={showCreateModal}
		title="Create Topic"
		onClose={() => {
			showCreateModal = false;
		}}
	>
		<form
			id="create-topic-form"
			onsubmit={(e) => {
				e.preventDefault();
				void handleCreate();
			}}
			class="space-y-4"
		>
			<div>
				<label for="topic-name" class="block text-sm font-semibold text-base-content mb-2">
					Topic Name
				</label>
				<input
					type="text"
					id="topic-name"
					placeholder="e.g., alerts, weather, updates"
					class={`input input-bordered w-full bg-base-100 text-base-content ${!isValidTopicName(newTopicName.trim()) && newTopicName.trim() !== '' ? 'input-error' : ''}`}
					bind:value={newTopicName}
					disabled={isLoading}
					required
				/>
				<p class="text-xs text-base-content/70 mt-2">
					Lowercase letters, numbers, underscores only. Starts with letter (max 32 characters)
				</p>
			</div>

			<div>
				<div class="mb-2 flex items-center justify-between gap-3">
					<label for="topic-description" class="text-sm font-semibold text-base-content">
						Description
					</label>
					<p class="text-xs text-base-content/70 tabular-nums whitespace-nowrap">
						({newTopicDescription.length}/{MAX_DESCRIPTION_LEN})
					</p>
				</div>
				<textarea
					id="topic-description"
					placeholder="e.g., Critical system alerts and warnings"
					class="textarea textarea-bordered w-full bg-base-100 text-base-content"
					bind:value={newTopicDescription}
					disabled={isLoading}
					maxlength={MAX_DESCRIPTION_LEN}
					rows="2"
				></textarea>
			</div>
		</form>
		{#snippet actions()}
			<button
				type="button"
				class="btn btn-outline"
				onclick={() => {
					showCreateModal = false;
				}}
				disabled={isLoading}
			>
				Cancel
			</button>
			<button
				type="submit"
				class="btn btn-primary"
				disabled={isLoading || !isValidTopicName(newTopicName.trim())}
				form="create-topic-form"
			>
				{#if isLoading}
					<span class="loading loading-spinner loading-sm"></span>
					Creating...
				{:else}
					Create Topic
				{/if}
			</button>
		{/snippet}
	</SettingsModal>

	<div class="space-y-4">
		{#if isLoading && topics.length === 0}
			<div class="flex justify-center py-8">
				<span class="loading loading-spinner loading-lg text-primary"></span>
			</div>
		{:else if topics.length === 0}
			<div class="alert alert-info bg-base-200 border border-base-300">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					class="stroke-current shrink-0 w-6 h-6"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					></path>
				</svg>
				<div>
					<h3 class="font-bold text-base-content">No custom topics yet</h3>
					<div class="text-sm text-base-content/70">
						Create your first topic to organize your notifications
					</div>
				</div>
			</div>
		{:else}
			{#each topics as topic (topic.id)}
				{#if editingTopicId === topic.id}
					<!-- Edit mode -->
					<div class="card bg-base-100 shadow border border-primary p-4">
						<form
							onsubmit={(e) => {
								e.preventDefault();
								void handleUpdate();
							}}
							class="space-y-4"
						>
							<div>
								<div class="mb-2 flex items-center justify-between gap-3">
									<label for="edit-description" class="text-sm font-semibold text-base-content">
										Description
									</label>
									<p class="text-xs text-base-content/70 tabular-nums whitespace-nowrap">
										({editDescription.length}/{MAX_DESCRIPTION_LEN})
									</p>
								</div>
								<textarea
									id="edit-description"
									class="textarea textarea-bordered w-full bg-base-100 text-base-content"
									bind:value={editDescription}
									disabled={isLoading}
									maxlength={MAX_DESCRIPTION_LEN}
									rows="2"
								></textarea>
							</div>

							<div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
								<button
									type="button"
									class="btn btn-sm btn-outline"
									onclick={cancelEdit}
									disabled={isLoading}
								>
									Cancel
								</button>
								<button type="submit" class="btn btn-sm btn-primary" disabled={isLoading}>
									{#if isLoading}
										<span class="loading loading-spinner loading-sm"></span>
										Saving...
									{:else}
										Save Changes
									{/if}
								</button>
							</div>
						</form>
					</div>
				{:else}
					<!-- View mode -->
					<div class="card bg-base-100 shadow border border-base-300 p-4">
						<div class="flex justify-between items-start">
							<div class="flex-1">
								<div class="flex items-center gap-2">
									<TopicChip name={topic.name} />
									{#if topic.name === 'general'}
										<span
											class="badge badge-sm bg-primary text-primary-content font-semibold border-0"
										>
											System
										</span>
									{/if}
								</div>
								{#if topic.description}
									<p class="text-sm text-base-content mt-2">{topic.description}</p>
								{/if}
								{#if topic.name !== 'general'}
									<p class="text-xs text-base-content/70 mt-2">
										Created {new Date(topic.created_at).toLocaleDateString()}
									</p>
								{/if}
							</div>
							<div class="relative">
								<button
									type="button"
									onclick={() => toggleTopicActions(topic.id)}
									disabled={isLoading}
									class="btn btn-ghost btn-circle btn-xs text-base-content/60"
									aria-label={`Open actions for ${topic.name}`}
									aria-expanded={actionsMenuOpenTopicId === topic.id}
								>
									<EllipsisVertical size={14} />
								</button>
								{#if actionsMenuOpenTopicId === topic.id}
									<ul
										role="menu"
										bind:this={actionsMenuRef}
										class="menu absolute right-0 z-20 mt-1 w-44 rounded-box border border-base-300 bg-base-100 p-2 shadow"
									>
										<li>
											<button type="button" onclick={() => startEdit(topic)}>
												<Pencil size={16} />
												Edit
											</button>
										</li>
										{#if topic.name !== 'general'}
											<li>
												<button
													type="button"
													class="text-error"
													onclick={() => openDeleteDialog(topic)}
												>
													<Trash2 size={16} />
													Delete
												</button>
											</li>
										{/if}
									</ul>
								{/if}
							</div>
						</div>
					</div>
				{/if}
				{#if topic.name === 'general' && topics.length > 1}
					<div class="divider text-xs text-base-content/50 my-0">Custom topics</div>
				{/if}
			{/each}
		{/if}
	</div>

	<SettingsModal
		open={Boolean(topicPendingDelete)}
		title="Delete topic"
		description={topicPendingDelete
			? `Delete "${topicPendingDelete.name}"? This cannot be undone.`
			: undefined}
		onClose={() => {
			topicPendingDelete = null;
		}}
		size="sm"
	>
		<div class="text-sm text-base-content/70">
			All tokens or devices linked to this topic will lose access to it.
		</div>
		{#snippet actions()}
			<button
				type="button"
				class="btn btn-outline"
				onclick={() => {
					topicPendingDelete = null;
				}}
				disabled={isLoading}
			>
				Cancel
			</button>
			<button type="button" class="btn btn-error" onclick={confirmDeleteTopic} disabled={isLoading}>
				{#if isLoading}
					<span class="loading loading-spinner loading-sm"></span>
					Deleting...
				{:else}
					Delete Topic
				{/if}
			</button>
		{/snippet}
	</SettingsModal>
</div>
