<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from '@beebuzz/shared/stores';
	import type { ApiToken, CreatedApiToken, Topic, AccountUsage } from '@beebuzz/shared/api';
	import { accountApi, topicsApi } from '@beebuzz/shared/api';
	import { ApiError } from '@beebuzz/shared/errors';
	import { CodeBlock, SettingsModal, TopicChip } from '@beebuzz/shared/components';
	import { Dot, Plus, ExternalLink, EllipsisVertical, Pencil, Trash2, Send } from '@lucide/svelte';
	import {
		MAX_DISPLAY_NAME_LEN,
		MAX_DISPLAY_NAME_SOFT_LEN,
		MAX_DESCRIPTION_LEN
	} from '@beebuzz/shared';
	import { API_URL, PUSH_URL, SITE_URL } from '@beebuzz/shared/config';
	import { isSaasMode } from '$lib/config/deployment';

	const TEST_NOTIFICATION_TITLE = 'BeeBuzz test notification';
	const TEST_NOTIFICATION_BODY = 'This is a test message sent with your API token.';

	let tokens = $state<ApiToken[]>([]);
	let topics = $state<Topic[]>([]);
	let isLoading = $state(false);
	let showCreateModal = $state(false);
	let newTokenName = $state('');
	let newTokenDescription = $state('');
	let selectedTopics = $state<string[]>([]);
	let createdToken = $state<(CreatedApiToken & { id?: string }) | null>(null);
	let tokenModalTab = $state<'curl' | 'cli'>('curl');
	let reminderTab = $state<'curl' | 'cli'>('curl');
	let editingTokenId = $state<string | null>(null);
	let editTokenName = $state('');
	let editTokenDescription = $state('');
	let editSelectedTopics = $state<string[]>([]);
	let tokenPendingDelete = $state<ApiToken | null>(null);
	let actionsMenuOpenTokenId = $state<string | null>(null);
	let actionsMenuRef = $state<HTMLElement | undefined>(undefined);
	let usage = $state<AccountUsage | null>(null);
	let usageLoaded = $state(false);
	let reminderDismissed = $state(false);
	let isTokenCreationOnboarding = $state(false);
	let isSendingTestNotification = $state(false);
	let testNotificationError = $state<string | null>(null);
	let testNotificationSuccess = $state<string | null>(null);

	onMount(async () => {
		await Promise.all([loadTokens(), loadTopics(), loadUsage()]);
		const searchParams = new URLSearchParams(window.location.search);
		isTokenCreationOnboarding = searchParams.get('onboarding') === 'send-first-message';
		if (isTokenCreationOnboarding) {
			reminderDismissed = false;
		}
	});

	$effect(() => {
		if (!actionsMenuOpenTokenId) return;

		// Close the actions menu when the click lands outside the current menu container.
		const handleClickOutside = (e: MouseEvent) => {
			if (actionsMenuRef && !actionsMenuRef.contains(e.target as Node)) {
				actionsMenuOpenTokenId = null;
			}
		};

		document.addEventListener('click', handleClickOutside, true);
		return () => document.removeEventListener('click', handleClickOutside, true);
	});

	async function loadTokens() {
		isLoading = true;
		try {
			tokens = await accountApi.listApiTokens();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load tokens');
		} finally {
			isLoading = false;
		}
	}

	async function loadTopics() {
		try {
			topics = await topicsApi.listTopics();
		} catch {
			// Silently fail - topics are optional
		}
	}

	async function loadUsage() {
		try {
			usage = await accountApi.getUsage(0);
			usageLoaded = true;
		} catch {
			// Silently fail - reminder is helpful but not critical
		}
	}

	async function handleCreate() {
		if (!newTokenName.trim()) {
			toast.error('Token name is required');
			return;
		}

		if (newTokenName.length > MAX_DISPLAY_NAME_LEN) {
			toast.error(`Token name must be ${MAX_DISPLAY_NAME_LEN} characters or less`);
			return;
		}

		if (newTokenDescription.length > MAX_DESCRIPTION_LEN) {
			toast.error(`Description must be ${MAX_DESCRIPTION_LEN} characters or less`);
			return;
		}

		if (topics.length > 0 && selectedTopics.length === 0) {
			toast.error('At least one topic must be selected');
			return;
		}

		isLoading = true;
		try {
			const token = await accountApi.createApiToken(
				newTokenName,
				selectedTopics,
				newTokenDescription
			);
			createdToken = token;
			newTokenName = '';
			newTokenDescription = '';
			selectedTopics = [];
			showCreateModal = false;
			tokenModalTab = 'curl';
			reminderDismissed = false;
			toast.success('API token created');
			await Promise.all([loadTokens(), loadUsage()]);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to create token');
		} finally {
			isLoading = false;
		}
	}

	async function handleDelete() {
		if (!tokenPendingDelete) return;

		try {
			await accountApi.deleteApiToken(tokenPendingDelete.id);
			toast.success('API token revoked');
			tokenPendingDelete = null;
			await loadTokens();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to delete token');
		}
	}

	interface CodeLine {
		text: string;
		prefix?: string;
		class?: string;
	}

	function buildCurlExample(token: string): CodeLine[] {
		return [
			{ text: `curl ${PUSH_URL} \\`, prefix: '$' },
			{ text: `  -H "Authorization: Bearer ${token}" \\`, prefix: '>' },
			{ text: `  -F title="${TEST_NOTIFICATION_TITLE}" \\`, prefix: '>' },
			{ text: `  -F body="${TEST_NOTIFICATION_BODY}"`, prefix: '>' }
		];
	}

	function buildCliExample(token: string): CodeLine[] {
		return [
			{ text: 'go install lucor.dev/beebuzz/cmd/beebuzz@latest', prefix: '$' },
			{ text: `beebuzz connect --api-token "${token}"`, prefix: '$' },
			{ text: `beebuzz send "${TEST_NOTIFICATION_TITLE}"`, prefix: '$' }
		];
	}

	function closeCreatedModal() {
		createdToken = null;
		isSendingTestNotification = false;
		testNotificationError = null;
		testNotificationSuccess = null;
	}

	interface BrowserSendResponse {
		status: 'delivered' | 'partial' | 'failed';
		sent_count: number;
		total_count: number;
		failed_count: number;
	}

	async function sendTestNotification() {
		if (!createdToken?.token || isSendingTestNotification) {
			return;
		}

		isSendingTestNotification = true;
		testNotificationError = null;
		testNotificationSuccess = null;

		try {
			const response = await fetch(`${API_URL}/v1/push`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${createdToken.token}`,
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					title: TEST_NOTIFICATION_TITLE,
					body: TEST_NOTIFICATION_BODY
				})
			});

			const responseBody = (await response.json().catch(() => null)) as
				| BrowserSendResponse
				| { message?: string; code?: string }
				| null;

			if (!response.ok) {
				testNotificationError =
					responseBody && 'message' in responseBody && responseBody.message
						? responseBody.message
						: 'Failed to send test notification';
				return;
			}

			testNotificationSuccess =
				responseBody && 'sent_count' in responseBody
					? `Test sent to ${responseBody.sent_count} of ${responseBody.total_count} device(s).`
					: 'Test notification sent.';
			await loadUsage();
		} catch {
			testNotificationError = 'Network error while sending test notification';
		} finally {
			isSendingTestNotification = false;
		}
	}

	function toggleTopic(topicId: string) {
		if (selectedTopics.includes(topicId)) {
			selectedTopics = selectedTopics.filter((t) => t !== topicId);
		} else {
			selectedTopics = [...selectedTopics, topicId];
		}
	}

	function startEdit(token: ApiToken) {
		actionsMenuOpenTokenId = null;
		editingTokenId = token.id;
		editTokenName = token.name || '';
		editTokenDescription = token.description || '';
		editSelectedTopics = token.topic_ids || [];
	}

	function cancelEdit() {
		editingTokenId = null;
		editTokenName = '';
		editTokenDescription = '';
		editSelectedTopics = [];
	}

	function toggleTokenActions(tokenId: string) {
		actionsMenuOpenTokenId = actionsMenuOpenTokenId === tokenId ? null : tokenId;
	}

	function openDeleteDialog(token: ApiToken) {
		actionsMenuOpenTokenId = null;
		tokenPendingDelete = token;
	}

	async function handleEdit(tokenId: string) {
		if (!editTokenName.trim()) {
			toast.error('Token name is required');
			return;
		}

		if (editTokenName.length > MAX_DISPLAY_NAME_LEN) {
			toast.error(`Token name must be ${MAX_DISPLAY_NAME_LEN} characters or less`);
			return;
		}

		if (editTokenDescription.length > MAX_DESCRIPTION_LEN) {
			toast.error(`Description must be ${MAX_DESCRIPTION_LEN} characters or less`);
			return;
		}

		if (topics.length > 0 && editSelectedTopics.length === 0) {
			toast.error('At least one topic must be selected');
			return;
		}

		isLoading = true;
		try {
			await accountApi.updateApiToken(
				tokenId,
				editTokenName,
				editTokenDescription,
				editSelectedTopics
			);
			toast.success('API token updated');
			cancelEdit();
			await loadTokens();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to update token');
		} finally {
			isLoading = false;
		}
	}

	function toggleEditTopic(topicId: string) {
		if (editSelectedTopics.includes(topicId)) {
			editSelectedTopics = editSelectedTopics.filter((t) => t !== topicId);
		} else {
			editSelectedTopics = [...editSelectedTopics, topicId];
		}
	}

	// Auto-select the general topic when it's the only one (and thus read-only)
	$effect(() => {
		const singleGeneral = topics.length === 1 && topics[0].name === 'general';
		if (singleGeneral && selectedTopics.length === 0) {
			selectedTopics = [topics[0].id];
		}
	});

	$effect(() => {
		const singleGeneral = topics.length === 1 && topics[0].name === 'general';
		if (singleGeneral && editSelectedTopics.length === 0) {
			editSelectedTopics = [topics[0].id];
		}
	});

	function isCreateFormValid(): boolean {
		if (!newTokenName.trim()) return false;
		if (newTokenName.length > MAX_DISPLAY_NAME_LEN) return false;
		if (newTokenDescription.length > MAX_DESCRIPTION_LEN) return false;
		if (topics.length > 0 && selectedTopics.length === 0) return false;
		return true;
	}

	function isEditFormValid(): boolean {
		if (!editTokenName.trim()) return false;
		if (editTokenName.length > MAX_DISPLAY_NAME_LEN) return false;
		if (editTokenDescription.length > MAX_DESCRIPTION_LEN) return false;
		if (topics.length > 0 && editSelectedTopics.length === 0) return false;
		return true;
	}

	let hasSentFirstMessage = $derived.by(() => {
		if (!usage) {
			return false;
		}

		return usage.data.some((day) => day.notifications_created > 0);
	});

	let shouldShowFirstMessageReminder = $derived(
		usageLoaded && tokens.length > 0 && !hasSentFirstMessage && !reminderDismissed
	);

	let activeExampleLines = $derived(
		createdToken
			? tokenModalTab === 'curl'
				? buildCurlExample(createdToken.token)
				: buildCliExample(createdToken.token)
			: []
	);

	let reminderExampleLines = $derived(
		reminderTab === 'curl'
			? buildCurlExample('<your-api-token>')
			: buildCliExample('<your-api-token>')
	);
</script>

<div class="p-6">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-base-content">API Tokens</h1>
		<button
			onclick={() => {
				showCreateModal = true;
			}}
			disabled={isLoading}
			class="btn btn-primary"
		>
			<Plus size={20} />
			Create Token
		</button>
	</div>

	<div class="divider my-4"></div>

	<p class="text-sm text-base-content/70 mb-6">
		Create and manage API tokens for programmatic access
	</p>

	{#if shouldShowFirstMessageReminder}
		<div class="card bg-primary/5 border border-primary/20 mb-6">
			<div class="card-body gap-4">
				<div class="flex items-start justify-between gap-3">
					<div>
						<h2 class="card-title text-base-content">Send your first test message</h2>
						<p class="text-sm text-base-content/70">
							You already have a token. Use it with <code>curl</code> or the CLI and once the first message
							is sent this reminder disappears.
						</p>
					</div>
					<button class="btn btn-ghost btn-sm" onclick={() => (reminderDismissed = true)}
						>Dismiss</button
					>
				</div>

				<div class="space-y-4">
					<div role="tablist" class="flex gap-1">
						<button
							type="button"
							role="tab"
							class={`btn btn-sm ${reminderTab === 'curl' ? 'btn-active' : 'btn-ghost'}`}
							aria-selected={reminderTab === 'curl'}
							onclick={() => (reminderTab = 'curl')}
						>
							curl
						</button>
						<button
							type="button"
							role="tab"
							class={`btn btn-sm ${reminderTab === 'cli' ? 'btn-active' : 'btn-ghost'}`}
							aria-selected={reminderTab === 'cli'}
							onclick={() => (reminderTab = 'cli')}
						>
							CLI
						</button>
					</div>

					{#key reminderTab}
						<CodeBlock
							lines={reminderExampleLines}
							copyLabel={`Copy ${reminderTab === 'curl' ? 'curl' : 'CLI'} example`}
						/>
					{/key}
				</div>

				{#if isSaasMode}
					<div class="flex items-center justify-between gap-3 text-sm text-base-content/70">
						<p>Need the full setup guide?</p>
						<a
							href={`${SITE_URL}/docs/quickstart`}
							class="inline-flex items-center gap-1 transition-colors hover:text-base-content"
							target="_blank"
							rel="noreferrer"
						>
							Open quickstart
							<ExternalLink size={14} />
						</a>
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<SettingsModal
		open={Boolean(createdToken)}
		title="API Token Created"
		description={isTokenCreationOnboarding
			? 'Save the token now, then send a first test message.'
			: undefined}
		size="xl"
		onClose={closeCreatedModal}
	>
		{#if createdToken}
			<div class={isTokenCreationOnboarding ? 'space-y-6' : 'space-y-5'}>
				<div class="rounded-xl border border-warning/30 bg-warning/10 p-4">
					<div class="space-y-1">
						{#if isTokenCreationOnboarding}
							<p class="text-xs font-semibold uppercase tracking-[0.16em] text-warning-content/80">
								Step 1
							</p>
						{/if}
						<p class="text-sm font-semibold text-base-content">Save this token now.</p>
						<p class="text-sm text-base-content/70">
							{isTokenCreationOnboarding
								? 'Store it in your password manager or local secret store before closing this dialog.'
								: "You won't be able to see it again after closing this dialog."}
						</p>
					</div>
				</div>
				<CodeBlock lines={[{ text: createdToken.token }]} copyLabel="Copy token" />

				<div class="space-y-4">
					<div class="space-y-1">
						{#if isTokenCreationOnboarding}
							<p class="text-xs font-semibold uppercase tracking-[0.16em] text-base-content/50">
								Step 2
							</p>
						{/if}
						<p class="text-sm font-semibold text-base-content">Send a first test message</p>
						<p class="text-sm text-base-content/70">
							{isTokenCreationOnboarding
								? 'Choose the fastest path for your device. `curl` is usually the quickest.'
								: 'Try the token with `curl` or the CLI.'}
						</p>
					</div>

					<div role="tablist" class="flex gap-1">
						<button
							type="button"
							role="tab"
							class={`btn btn-sm ${tokenModalTab === 'curl' ? 'btn-active' : 'btn-ghost'}`}
							aria-selected={tokenModalTab === 'curl'}
							onclick={() => (tokenModalTab = 'curl')}
						>
							curl
						</button>
						<button
							type="button"
							role="tab"
							class={`btn btn-sm ${tokenModalTab === 'cli' ? 'btn-active' : 'btn-ghost'}`}
							aria-selected={tokenModalTab === 'cli'}
							onclick={() => (tokenModalTab = 'cli')}
						>
							CLI
						</button>
					</div>

					{#key tokenModalTab}
						<CodeBlock
							lines={activeExampleLines}
							copyLabel={`Copy ${tokenModalTab === 'curl' ? 'curl' : 'CLI'} example`}
						/>
					{/key}
				</div>

				<div class="rounded-xl border border-base-300 bg-base-100 p-4">
					<div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
						<div>
							<p class="text-sm font-semibold text-base-content">
								{isTokenCreationOnboarding ? 'No terminal available?' : 'Browser fallback'}
							</p>
							<p class="text-sm text-base-content/70">
								{isTokenCreationOnboarding
									? 'Send a browser-based test directly from this page, useful on mobile too.'
									: 'Send a test directly from this page when `curl` or the CLI are not available.'}
							</p>
						</div>
						<button
							type="button"
							class="btn btn-primary"
							onclick={() => void sendTestNotification()}
							disabled={isSendingTestNotification}
						>
							{#if isSendingTestNotification}
								<span class="loading loading-spinner loading-sm"></span>
								Sending...
							{:else}
								<Send size={16} />
								Send test notification now
							{/if}
						</button>
					</div>
					{#if testNotificationSuccess}
						<p class="mt-3 text-sm text-success">{testNotificationSuccess}</p>
					{:else if testNotificationError}
						<p class="mt-3 text-sm text-error">{testNotificationError}</p>
					{/if}
				</div>
				{#if isSaasMode}
					<div class="flex items-center justify-between gap-3 text-sm text-base-content/70">
						<p>
							{isTokenCreationOnboarding
								? 'Need the full setup guide?'
								: 'More details in the docs'}
						</p>
						<a
							href={`${SITE_URL}/docs/quickstart`}
							class="inline-flex items-center gap-1 transition-colors hover:text-base-content"
							target="_blank"
							rel="noreferrer"
						>
							Open quickstart
							<ExternalLink size={14} />
						</a>
					</div>
				{/if}
			</div>
		{/if}
		{#snippet actions()}
			<button type="button" onclick={closeCreatedModal} class="btn btn-sm btn-primary">Done</button>
		{/snippet}
	</SettingsModal>

	<SettingsModal
		open={showCreateModal}
		title="Create API Token"
		onClose={() => {
			showCreateModal = false;
		}}
	>
		<form
			id="create-token-form"
			onsubmit={(e) => {
				e.preventDefault();
				void handleCreate();
			}}
			class="space-y-4"
		>
			<div>
				<div class="mb-2 flex items-center justify-between gap-3">
					<label for="name" class="text-sm font-semibold text-base-content">Token Name</label>
					<p
						class="text-xs tabular-nums whitespace-nowrap"
						class:text-base-content-70={newTokenName.length <= MAX_DISPLAY_NAME_SOFT_LEN}
						class:text-warning={newTokenName.length > MAX_DISPLAY_NAME_SOFT_LEN &&
							newTokenName.length <= MAX_DISPLAY_NAME_LEN}
						class:text-error={newTokenName.length > MAX_DISPLAY_NAME_LEN}
					>
						({newTokenName.length}/{MAX_DISPLAY_NAME_LEN})
					</p>
				</div>
				<input
					type="text"
					id="name"
					placeholder="e.g., Home Assistant"
					class="input input-bordered w-full bg-base-100 text-base-content"
					bind:value={newTokenName}
					disabled={isLoading}
					maxlength={MAX_DISPLAY_NAME_LEN}
					required
				/>
			</div>

			<div>
				<div class="mb-2 flex items-center justify-between gap-3">
					<label for="description" class="text-sm font-semibold text-base-content">
						Description
					</label>
					<p class="text-xs text-base-content/70 tabular-nums whitespace-nowrap">
						({newTokenDescription.length}/{MAX_DESCRIPTION_LEN})
					</p>
				</div>
				<textarea
					id="description"
					placeholder="Optional description..."
					class="textarea textarea-bordered w-full bg-base-100 text-base-content"
					bind:value={newTokenDescription}
					disabled={isLoading}
					maxlength={MAX_DESCRIPTION_LEN}
					rows="2"
				></textarea>
			</div>

			{#if topics.length > 0}
				<fieldset>
					<legend class="block text-sm font-semibold text-base-content mb-2">
						Topics
						<span class="text-xs font-normal text-base-content/70"
							>(Required - select at least one)</span
						>
					</legend>
					<div class="space-y-2">
						{#each topics as topic (topic.id)}
							{@const isReadOnly = topic.name === 'general' && topics.length === 1}
							<label
								class={`flex items-center gap-2 ${isReadOnly ? 'opacity-70' : 'cursor-pointer'}`}
							>
								<input
									type="checkbox"
									checked={selectedTopics.includes(topic.id)}
									onchange={() => !isReadOnly && toggleTopic(topic.id)}
									disabled={isLoading || isReadOnly}
									class="checkbox checkbox-sm"
								/>
								<TopicChip name={topic.name} muted={isReadOnly} />
								{#if topic.name === 'general'}
									<span class="text-xs text-base-content/70">(default)</span>
								{/if}
							</label>
						{/each}
					</div>
				</fieldset>
			{/if}
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
				disabled={isLoading || !isCreateFormValid()}
				form="create-token-form"
			>
				{#if isLoading}
					<span class="loading loading-spinner loading-sm"></span>
					Creating...
				{:else}
					Create Token
				{/if}
			</button>
		{/snippet}
	</SettingsModal>

	<div class="space-y-4">
		{#if isLoading && tokens.length === 0}
			<div class="flex justify-center py-8">
				<span class="loading loading-spinner loading-lg text-primary"></span>
			</div>
		{:else if tokens.length === 0}
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
					<h3 class="font-bold text-base-content">No API tokens yet</h3>
					<div class="text-sm text-base-content/70 mb-3">
						Create your first API token to get started
					</div>
					<div class="flex gap-2">
						<button
							type="button"
							class="btn btn-primary btn-sm"
							onclick={() => (showCreateModal = true)}
						>
							<Plus size={16} />
							Create Your First Token
						</button>
						{#if isSaasMode}
							<a
								href={`${SITE_URL}/docs/quickstart`}
								class="btn btn-ghost btn-sm"
								target="_blank"
								rel="noopener noreferrer"
							>
								Learn how
								<ExternalLink size={14} />
							</a>
						{/if}
					</div>
				</div>
			</div>
		{:else}
			{#each tokens as token (token.id)}
				{#if editingTokenId === token.id}
					<div class="card bg-base-100 shadow border border-base-300 p-4">
						<form
							onsubmit={(e) => {
								e.preventDefault();
								void handleEdit(token.id);
							}}
							class="space-y-4"
						>
							<div>
								<div class="mb-2 flex items-center justify-between gap-3">
									<label for="name-{token.id}" class="text-sm font-semibold text-base-content">
										Token Name
									</label>
									<p
										class="text-xs tabular-nums whitespace-nowrap"
										class:text-base-content-70={editTokenName.length <= MAX_DISPLAY_NAME_SOFT_LEN}
										class:text-warning={editTokenName.length > MAX_DISPLAY_NAME_SOFT_LEN &&
											editTokenName.length <= MAX_DISPLAY_NAME_LEN}
										class:text-error={editTokenName.length > MAX_DISPLAY_NAME_LEN}
									>
										({editTokenName.length}/{MAX_DISPLAY_NAME_LEN})
									</p>
								</div>
								<input
									type="text"
									id="name-{token.id}"
									placeholder="e.g., Home Assistant"
									class="input input-bordered w-full bg-base-100 text-base-content"
									bind:value={editTokenName}
									disabled={isLoading}
									maxlength={MAX_DISPLAY_NAME_LEN}
									required
								/>
							</div>

							<div>
								<div class="mb-2 flex items-center justify-between gap-3">
									<label for="desc-{token.id}" class="text-sm font-semibold text-base-content">
										Description
									</label>
									<p class="text-xs text-base-content/70 tabular-nums whitespace-nowrap">
										({editTokenDescription.length}/{MAX_DESCRIPTION_LEN})
									</p>
								</div>
								<textarea
									id="desc-{token.id}"
									placeholder="Optional description..."
									class="textarea textarea-bordered w-full bg-base-100 text-base-content"
									bind:value={editTokenDescription}
									disabled={isLoading}
									maxlength={MAX_DESCRIPTION_LEN}
									rows="2"
								></textarea>
							</div>

							{#if topics.length > 0}
								<fieldset>
									<legend class="block text-sm font-semibold text-base-content mb-2">
										Topics
										<span class="text-xs font-normal text-base-content/70"
											>(Required - select at least one)</span
										>
									</legend>
									<div class="space-y-2">
										{#each topics as topic (topic.id)}
											{@const isReadOnly = topic.name === 'general' && topics.length === 1}
											<label
												class={`flex items-center gap-2 ${isReadOnly ? 'opacity-70' : 'cursor-pointer'}`}
											>
												<input
													type="checkbox"
													checked={editSelectedTopics.includes(topic.id)}
													onchange={() => !isReadOnly && toggleEditTopic(topic.id)}
													disabled={isLoading || isReadOnly}
													class="checkbox checkbox-sm"
												/>
												<TopicChip name={topic.name} muted={isReadOnly} />
												{#if topic.name === 'general'}
													<span class="text-xs text-base-content/70">(default)</span>
												{/if}
											</label>
										{/each}
									</div>
								</fieldset>
							{/if}

							<div class="flex flex-col gap-2 pt-4 sm:flex-row sm:justify-end">
								<button
									type="button"
									class="btn btn-outline"
									onclick={cancelEdit}
									disabled={isLoading}
								>
									Cancel
								</button>
								<button
									type="submit"
									class="btn btn-primary"
									disabled={isLoading || !isEditFormValid()}
								>
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
					<div class="card bg-base-100 shadow border border-base-300 p-4">
						<div class="flex justify-between items-start">
							<div class="flex-1">
								<p class="font-semibold text-base-content">{token.name}</p>
								{#if token.description}
									<p class="text-sm text-base-content/70 mt-1">{token.description}</p>
								{/if}
								<p class="text-xs text-base-content/70 mt-1 flex items-center gap-1">
									<span>
										{#if token.last_used_at}
											Last used {new Date(token.last_used_at).toLocaleDateString()}
										{:else}
											Never used
										{/if}
									</span>
									<Dot size={12} class="flex-shrink-0" />
									<span>
										{#if token.expires_at}
											{new Date(token.expires_at) > new Date()
												? `Expires on ${new Date(token.expires_at).toLocaleDateString()}`
												: `Expired on ${new Date(token.expires_at).toLocaleDateString()}`}
										{:else}
											Never expires
										{/if}
									</span>
								</p>
								{#if token.topic_ids && token.topic_ids.length > 0}
									<div class="flex flex-wrap gap-2 mt-3">
										{#each token.topic_ids as topicId, idx (idx)}
											<TopicChip name={topics.find((t) => t.id === topicId)?.name ?? topicId} />
										{/each}
									</div>
								{:else if token.topic_ids !== undefined}
									<p class="text-xs text-base-content/70 mt-2">No topics</p>
								{/if}
							</div>
							<div class="relative">
								<button
									type="button"
									onclick={() => toggleTokenActions(token.id)}
									disabled={isLoading}
									class="btn btn-ghost btn-circle btn-xs text-base-content/60"
									aria-label={`Open actions for ${token.name}`}
									aria-expanded={actionsMenuOpenTokenId === token.id}
								>
									<EllipsisVertical size={14} />
								</button>
								{#if actionsMenuOpenTokenId === token.id}
									<ul
										role="menu"
										bind:this={actionsMenuRef}
										class="menu absolute right-0 z-20 mt-1 w-44 rounded-box border border-base-300 bg-base-100 p-2 shadow"
									>
										<li>
											<button type="button" onclick={() => startEdit(token)}>
												<Pencil size={16} />
												Edit
											</button>
										</li>
										<li>
											<button
												type="button"
												class="text-error"
												onclick={() => openDeleteDialog(token)}
											>
												<Trash2 size={16} />
												Revoke
											</button>
										</li>
									</ul>
								{/if}
							</div>
						</div>
					</div>
				{/if}
			{/each}
		{/if}
	</div>

	<SettingsModal
		open={Boolean(tokenPendingDelete)}
		title="Revoke API token"
		description={tokenPendingDelete
			? `Revoke "${tokenPendingDelete.name}"? It cannot be undone.`
			: undefined}
		onClose={() => {
			tokenPendingDelete = null;
		}}
		size="sm"
	>
		<p class="text-sm text-base-content/70">
			Services using this token will lose access immediately.
		</p>
		{#snippet actions()}
			<button type="button" class="btn btn-outline" onclick={() => (tokenPendingDelete = null)}>
				Cancel
			</button>
			<button type="button" class="btn btn-error" onclick={handleDelete}>Revoke Token</button>
		{/snippet}
	</SettingsModal>
</div>
