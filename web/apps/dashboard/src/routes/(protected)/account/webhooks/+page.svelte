<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from '@beebuzz/shared/stores';
	import type { Webhook, Topic, InspectSession, InspectSessionStatus } from '@beebuzz/shared/api';
	import { accountApi, topicsApi } from '@beebuzz/shared/api';
	import { ApiError } from '@beebuzz/shared/errors';
	import { SettingsModal, JsonTreeViewer, TopicChip } from '@beebuzz/shared/components';
	import {
		Settings,
		FileText,
		Copy,
		Check,
		Plus,
		Search,
		Eye,
		EllipsisVertical,
		Pencil,
		RefreshCw,
		Trash2,
		ChevronsUp
	} from '@lucide/svelte';
	import {
		MAX_DISPLAY_NAME_LEN,
		MAX_DISPLAY_NAME_SOFT_LEN,
		MAX_DESCRIPTION_LEN
	} from '@beebuzz/shared';

	let webhooks = $state<Webhook[]>([]);
	let topics = $state<Topic[]>([]);
	let isLoading = $state(false);
	let showCreateModal = $state(false);

	// Create form state
	let newWebhookName = $state('');
	let newWebhookDescription = $state('');
	let newWebhookPayloadType = $state<'beebuzz' | 'custom'>('beebuzz');
	let newWebhookTitlePath = $state('');
	let newWebhookBodyPath = $state('');
	let newWebhookPriority = $state<'normal' | 'high'>('normal');
	let selectedTopics = $state<string[]>([]);

	// Inspect mode state
	let enableInspectMode = $state(false);
	let inspectSession = $state<InspectSession | null>(null);
	let inspectStatus = $state<InspectSessionStatus | null>(null);
	let inspectPollInterval: ReturnType<typeof setInterval> | null = null;

	// Created/regenerated webhook token modal — shows full URL once
	let revealedToken = $state<string | null>(null);
	let revealedWebhookName = $state<string>('');
	let isRegenerated = $state(false);
	let isCopied = $state(false);
	let copiedInspectUrl = $state(false);

	// Edit state
	let editingWebhookId = $state<string | null>(null);
	let editWebhookName = $state('');
	let editWebhookDescription = $state('');
	let editWebhookPayloadType = $state<'beebuzz' | 'custom'>('beebuzz');
	let editWebhookTitlePath = $state('');
	let editWebhookBodyPath = $state('');
	let editWebhookPriority = $state<'normal' | 'high'>('normal');
	let editSelectedTopics = $state<string[]>([]);
	let webhookPendingDelete = $state<Webhook | null>(null);

	// Detail view state
	let expandedWebhookId = $state<string | null>(null);
	let copiedCurlWebhookId = $state<string | null>(null);

	// Regenerate token state
	let webhookPendingRegenerate = $state<Webhook | null>(null);
	let actionsMenuOpenWebhookId = $state<string | null>(null);
	let actionsMenuRef = $state<HTMLElement | undefined>(undefined);

	onMount(async () => {
		await loadWebhooks();
		await loadTopics();
	});

	$effect(() => {
		if (!actionsMenuOpenWebhookId) return;

		// Close the actions menu when the click lands outside the current menu container.
		const handleClickOutside = (e: MouseEvent) => {
			if (actionsMenuRef && !actionsMenuRef.contains(e.target as Node)) {
				actionsMenuOpenWebhookId = null;
			}
		};

		document.addEventListener('click', handleClickOutside, true);
		return () => document.removeEventListener('click', handleClickOutside, true);
	});

	async function loadWebhooks() {
		isLoading = true;
		try {
			webhooks = await accountApi.listWebhooks();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load webhooks');
		} finally {
			isLoading = false;
		}
	}

	async function loadTopics() {
		try {
			topics = await topicsApi.listTopics();
		} catch (err) {
			console.error('Failed to load topics:', err);
		}
	}

	// --- INSPECT MODE ---
	async function startInspectMode() {
		if (!newWebhookName.trim()) {
			toast.error('Webhook name is required');
			return;
		}

		if (selectedTopics.length === 0) {
			toast.error('At least one topic is required');
			return;
		}

		inspectSession = {
			token: '',
			url: 'Connecting...',
			status: 'waiting',
			expires_at: new Date(Date.now() + 10 * 60 * 1000).toISOString()
		};
		inspectStatus = {
			status: 'waiting',
			expires_at: new Date(Date.now() + 10 * 60 * 1000).toISOString()
		};
		startPolling();

		try {
			const session = await accountApi.createInspectSession(
				newWebhookName,
				newWebhookDescription,
				newWebhookPriority,
				selectedTopics
			);
			inspectSession = session;
		} catch (err) {
			stopPolling();
			inspectSession = null;
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to start inspect mode');
		}
	}

	function startPolling() {
		inspectPollInterval = setInterval(() => {
			void (async () => {
				try {
					inspectStatus = await accountApi.getInspectSession();
					if (inspectStatus?.status === 'captured') {
						stopPolling();
					}
				} catch (err) {
					if (err instanceof ApiError && err.status === 404) {
						resetInspectState();
						toast.error('Inspect session expired');
					}
				}
			})();
		}, 1000);
	}

	function stopPolling() {
		if (inspectPollInterval) {
			clearInterval(inspectPollInterval);
			inspectPollInterval = null;
		}
	}

	function handlePathClick(path: string, _value: string) {
		void _value;
		if (!newWebhookTitlePath) {
			newWebhookTitlePath = path;
		} else if (!newWebhookBodyPath) {
			newWebhookBodyPath = path;
		}
	}

	async function finalizeInspect() {
		if (!inspectSession) return;

		if (!newWebhookTitlePath.trim()) {
			toast.error('Click on a field in the JSON to set the Title path');
			return;
		}

		if (!newWebhookBodyPath.trim()) {
			toast.error('Click on a field in the JSON to set the Body path');
			return;
		}

		isLoading = true;
		try {
			const webhook = await accountApi.finalizeInspect(newWebhookTitlePath, newWebhookBodyPath);
			revealedToken = webhook.token;
			revealedWebhookName = webhook.name;
			isRegenerated = false;
			resetInspectState();
			showCreateModal = false;
			toast.success('Webhook created');
			await loadWebhooks();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to create webhook');
		} finally {
			isLoading = false;
		}
	}

	function resetInspectState() {
		stopPolling();
		inspectSession = null;
		inspectStatus = null;
		enableInspectMode = false;
		newWebhookTitlePath = '';
		newWebhookBodyPath = '';
	}

	// --- CREATE HANDLER ---
	async function handleCreate() {
		if (!newWebhookName.trim()) {
			toast.error('Webhook name is required');
			return;
		}

		if (selectedTopics.length === 0) {
			toast.error('At least one topic is required');
			return;
		}

		// Only validate paths for custom payload type
		if (newWebhookPayloadType === 'custom') {
			if (!newWebhookTitlePath.trim()) {
				toast.error('Title path is required for custom payloads (e.g., data.title)');
				return;
			}

			if (newWebhookTitlePath.trim().startsWith('.')) {
				toast.error('Title path must not start with a dot');
				return;
			}

			if (!newWebhookBodyPath.trim()) {
				toast.error('Body path is required for custom payloads (e.g., data.body)');
				return;
			}

			if (newWebhookBodyPath.trim().startsWith('.')) {
				toast.error('Body path must not start with a dot');
				return;
			}
		}

		isLoading = true;
		try {
			const webhook = await accountApi.createWebhook(
				newWebhookName,
				newWebhookDescription,
				newWebhookPayloadType,
				newWebhookTitlePath,
				newWebhookBodyPath,
				newWebhookPriority,
				selectedTopics
			);
			revealedToken = webhook.token;
			revealedWebhookName = webhook.name;
			isRegenerated = false;
			newWebhookName = '';
			newWebhookDescription = '';
			newWebhookPayloadType = 'beebuzz';
			newWebhookTitlePath = '';
			newWebhookBodyPath = '';
			newWebhookPriority = 'normal';
			selectedTopics = [];
			showCreateModal = false;
			toast.success('Webhook created');
			await loadWebhooks();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to create webhook');
		} finally {
			isLoading = false;
		}
	}

	async function handleDelete() {
		if (!webhookPendingDelete) return;

		try {
			await accountApi.deleteWebhook(webhookPendingDelete.id);
			toast.success('Webhook deleted');
			webhookPendingDelete = null;
			await loadWebhooks();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to delete webhook');
		}
	}

	// --- MODAL & CLIPBOARD ---
	function copyRevealedUrl() {
		if (!revealedToken) return;
		void navigator.clipboard.writeText(accountApi.getWebhookReceiveUrl(revealedToken));
		isCopied = true;
		setTimeout(() => {
			isCopied = false;
		}, 2000);
	}

	function copyInspectUrl() {
		if (!inspectSession) return;
		void navigator.clipboard.writeText(inspectSession.url);
		copiedInspectUrl = true;
		setTimeout(() => {
			copiedInspectUrl = false;
		}, 2000);
	}

	function closeRevealedModal() {
		revealedToken = null;
		revealedWebhookName = '';
		isRegenerated = false;
	}

	// Auto-select the general topic when it's the only one (and thus read-only)
	$effect(() => {
		const singleGeneral = topics.length === 1 && topics[0].name === 'general';
		if (singleGeneral && selectedTopics.length === 0) {
			selectedTopics = [topics[0].id];
		}
	});

	// --- TOPIC SELECTION ---
	function toggleTopic(topicId: string) {
		if (selectedTopics.includes(topicId)) {
			selectedTopics = selectedTopics.filter((t) => t !== topicId);
		} else {
			selectedTopics = [...selectedTopics, topicId];
		}
	}

	function toggleEditTopic(topicId: string) {
		if (editSelectedTopics.includes(topicId)) {
			editSelectedTopics = editSelectedTopics.filter((t) => t !== topicId);
		} else {
			editSelectedTopics = [...editSelectedTopics, topicId];
		}
	}

	// --- EDIT HANDLERS ---
	function startEdit(webhook: Webhook) {
		actionsMenuOpenWebhookId = null;
		editingWebhookId = webhook.id;
		editWebhookName = webhook.name;
		editWebhookDescription = webhook.description || '';
		editWebhookPayloadType = webhook.payload_type;
		editWebhookTitlePath = webhook.title_path ?? '';
		editWebhookBodyPath = webhook.body_path ?? '';
		editWebhookPriority = webhook.priority;
		editSelectedTopics = webhook.topic_ids || [];
		expandedWebhookId = null;
	}

	function cancelEdit() {
		editingWebhookId = null;
		editWebhookName = '';
		editWebhookDescription = '';
		editWebhookPayloadType = 'beebuzz';
		editWebhookTitlePath = '';
		editWebhookBodyPath = '';
		editWebhookPriority = 'normal';
		editSelectedTopics = [];
	}

	function toggleWebhookActions(webhookId: string) {
		actionsMenuOpenWebhookId = actionsMenuOpenWebhookId === webhookId ? null : webhookId;
	}

	function openRegenerateDialog(webhook: Webhook) {
		actionsMenuOpenWebhookId = null;
		webhookPendingRegenerate = webhook;
	}

	function openDeleteDialog(webhook: Webhook) {
		actionsMenuOpenWebhookId = null;
		webhookPendingDelete = webhook;
	}

	async function handleEdit(webhookId: string) {
		if (!editWebhookName.trim()) {
			toast.error('Webhook name is required');
			return;
		}

		if (editSelectedTopics.length === 0) {
			toast.error('At least one topic must be selected');
			return;
		}

		// Only validate paths for custom payload type
		if (editWebhookPayloadType === 'custom') {
			if (!editWebhookTitlePath.trim()) {
				toast.error('Title path is required for custom payloads');
				return;
			}

			if (editWebhookTitlePath.trim().startsWith('.')) {
				toast.error('Title path must not start with a dot');
				return;
			}

			if (!editWebhookBodyPath.trim()) {
				toast.error('Body path is required for custom payloads');
				return;
			}

			if (editWebhookBodyPath.trim().startsWith('.')) {
				toast.error('Body path must not start with a dot');
				return;
			}
		}

		isLoading = true;
		try {
			await accountApi.updateWebhook(
				webhookId,
				editWebhookName,
				editWebhookDescription,
				editWebhookPayloadType,
				editWebhookTitlePath,
				editWebhookBodyPath,
				editWebhookPriority,
				editSelectedTopics
			);
			toast.success('Webhook updated');
			editingWebhookId = null;
			expandedWebhookId = null;
			await loadWebhooks();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to update webhook');
		} finally {
			isLoading = false;
		}
	}

	// --- REGENERATE TOKEN ---
	async function handleRegenerate() {
		if (!webhookPendingRegenerate) return;

		try {
			const { token } = await accountApi.regenerateWebhookToken(webhookPendingRegenerate.id);
			revealedToken = token;
			revealedWebhookName = webhookPendingRegenerate.name;
			isRegenerated = true;
			webhookPendingRegenerate = null;
			toast.success('Token regenerated');
			await loadWebhooks();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to regenerate token');
		}
	}

	// --- FORM VALIDATION ---
	function isCreateFormValid(): boolean {
		if (!newWebhookName.trim()) return false;
		if (newWebhookName.length > MAX_DISPLAY_NAME_LEN) return false;
		if (newWebhookDescription.length > MAX_DESCRIPTION_LEN) return false;
		if (selectedTopics.length === 0) return false;
		// Only validate paths for custom payload type
		if (newWebhookPayloadType === 'custom') {
			if (!newWebhookTitlePath.trim()) return false;
			if (newWebhookTitlePath.trim().startsWith('.')) return false;
			if (!newWebhookBodyPath.trim()) return false;
			if (newWebhookBodyPath.trim().startsWith('.')) return false;
		}
		return true;
	}

	function isEditFormValid(): boolean {
		if (!editWebhookName.trim()) return false;
		if (editWebhookName.length > MAX_DISPLAY_NAME_LEN) return false;
		if (editWebhookDescription.length > MAX_DESCRIPTION_LEN) return false;
		if (editSelectedTopics.length === 0) return false;
		// Only validate paths for custom payload type
		if (editWebhookPayloadType === 'custom') {
			if (!editWebhookTitlePath.trim()) return false;
			if (editWebhookTitlePath.trim().startsWith('.')) return false;
			if (!editWebhookBodyPath.trim()) return false;
			if (editWebhookBodyPath.trim().startsWith('.')) return false;
		}
		return true;
	}

	/** Build a nested object from dot-separated path entries. */
	function buildNestedJson(entries: [string, string][]): Record<string, unknown> {
		const root: Record<string, unknown> = {};
		for (const [path, value] of entries) {
			const segments = path
				.replace(/^\.+|\.+$/g, '')
				.split('.')
				.filter(Boolean);
			if (segments.length === 0) continue;
			let current: Record<string, unknown> = root;
			for (let i = 0; i < segments.length - 1; i++) {
				const key = segments[i];
				if (typeof current[key] !== 'object' || current[key] === null) {
					current[key] = {};
				}
				current = current[key] as Record<string, unknown>;
			}
			current[segments[segments.length - 1]] = value;
		}
		return root;
	}

	/** Build a ready-to-use curl example for the given webhook. */
	function buildExampleCurl(webhook: Webhook): string {
		const url = accountApi.getWebhookReceiveUrl('<your-token>');
		let body: string;

		if (webhook.payload_type === 'beebuzz') {
			body = JSON.stringify({ title: 'Hello', body: 'World' });
		} else {
			const nested = buildNestedJson([
				[webhook.title_path ?? '', 'Hello'],
				[webhook.body_path ?? '', 'World']
			]);
			body = JSON.stringify(nested);
		}

		return `curl -X POST "${url}" \\\n  -H "Content-Type: application/json" \\\n  -d '${body}'`;
	}

	/** Copy the example curl to clipboard. */
	function copyCurlExample(webhook: Webhook) {
		void navigator.clipboard.writeText(buildExampleCurl(webhook));
		copiedCurlWebhookId = webhook.id;
		setTimeout(() => {
			copiedCurlWebhookId = null;
		}, 2000);
	}
</script>

<div class="p-6">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-base-content">Webhooks</h1>
		<button
			type="button"
			onclick={() => (showCreateModal = true)}
			disabled={isLoading}
			class="btn btn-primary gap-2"
		>
			<Plus size={20} />
			Create Webhook
		</button>
	</div>

	<!-- Token Reveal Modal (create + regenerate) -->
	<SettingsModal
		open={Boolean(revealedToken)}
		title={isRegenerated ? 'Token Regenerated' : 'Webhook Created'}
		description={revealedWebhookName}
		onClose={closeRevealedModal}
	>
		{#if revealedToken}
			<div class="alert alert-warning mb-4 text-sm">
				This URL will not be shown again. Copy it now.
			</div>
			<div class="bg-base-200 border border-base-300 rounded p-4">
				<p class="text-xs text-base-content/70 mb-2">Webhook URL</p>
				<p class="font-mono text-xs text-base-content break-all">
					{accountApi.getWebhookReceiveUrl(revealedToken)}
				</p>
			</div>
		{/if}
		{#snippet actions()}
			<button type="button" onclick={copyRevealedUrl} class="btn btn-sm btn-outline gap-2">
				{#if isCopied}
					<Check size={16} />
					Copied!
				{:else}
					<Copy size={16} />
					Copy URL
				{/if}
			</button>
			<button type="button" onclick={closeRevealedModal} class="btn btn-primary btn-sm">Done</button
			>
		{/snippet}
	</SettingsModal>

	<!-- Create Modal -->
	<SettingsModal
		open={showCreateModal}
		title="Create Webhook"
		onClose={() => (showCreateModal = false)}
		size="xl"
	>
		<form
			id="create-webhook-form"
			onsubmit={(e) => {
				e.preventDefault();
				void handleCreate();
			}}
			class="space-y-4"
		>
			<!-- Name -->
			<div>
				<div class="mb-2 flex items-center justify-between gap-3">
					<label for="create-name" class="text-sm font-semibold text-base-content">
						Webhook Name
					</label>
					<p
						class="text-xs tabular-nums whitespace-nowrap"
						class:text-base-content-70={newWebhookName.length <= MAX_DISPLAY_NAME_SOFT_LEN}
						class:text-warning={newWebhookName.length > MAX_DISPLAY_NAME_SOFT_LEN &&
							newWebhookName.length <= MAX_DISPLAY_NAME_LEN}
						class:text-error={newWebhookName.length > MAX_DISPLAY_NAME_LEN}
					>
						({newWebhookName.length}/{MAX_DISPLAY_NAME_LEN})
					</p>
				</div>
				<input
					type="text"
					id="create-name"
					placeholder="e.g., Home Assistant Alerts"
					class="input input-bordered w-full bg-base-100 text-base-content"
					bind:value={newWebhookName}
					disabled={isLoading || inspectSession !== null}
					maxlength={MAX_DISPLAY_NAME_LEN}
					required
				/>
			</div>

			<!-- Description -->
			<div>
				<div class="mb-2 flex items-center justify-between gap-3">
					<label for="create-desc" class="text-sm font-semibold text-base-content">
						Description
					</label>
					<p class="text-xs text-base-content/70 tabular-nums whitespace-nowrap">
						({newWebhookDescription.length}/{MAX_DESCRIPTION_LEN})
					</p>
				</div>
				<textarea
					id="create-desc"
					placeholder="What is this webhook for?"
					class="textarea textarea-bordered w-full bg-base-100 text-base-content"
					bind:value={newWebhookDescription}
					disabled={isLoading || inspectSession !== null}
					maxlength={MAX_DESCRIPTION_LEN}
					rows="2"
				></textarea>
			</div>

			<div>
				<label for="create-priority" class="block text-sm font-semibold text-base-content mb-2">
					Delivery Priority
				</label>
				<select
					id="create-priority"
					class="select select-bordered w-full bg-base-100 text-base-content"
					bind:value={newWebhookPriority}
					disabled={isLoading || inspectSession !== null}
				>
					<option value="normal">Normal</option>
					<option value="high">High</option>
				</select>
				<p class="text-xs text-base-content/70 mt-1">
					Controls the delivery urgency used for notifications sent by this webhook.
				</p>
			</div>

			<!-- Topics -->
			{#if topics.length > 0}
				<fieldset>
					<legend class="block text-sm font-semibold text-base-content mb-2">
						Topics
						<span class="text-xs font-normal text-base-content/70">(At least one required)</span>
					</legend>
					<div class="space-y-2 max-h-40 overflow-y-auto">
						{#each topics as topic (topic.id)}
							{@const isGeneral = topic.name === 'general'}
							{@const isOnlyTopic = topics.length === 1}
							{@const isReadOnly = isGeneral && isOnlyTopic}
							<label
								class={`flex items-center gap-2 ${isReadOnly ? 'opacity-70' : 'cursor-pointer'}`}
							>
								<input
									type="checkbox"
									checked={selectedTopics.includes(topic.id)}
									onchange={() => !isReadOnly && toggleTopic(topic.id)}
									disabled={isLoading || isReadOnly || inspectSession !== null}
									class="checkbox checkbox-sm"
								/>
								<TopicChip name={topic.name} muted={isReadOnly} />
								{#if isGeneral}
									<span class="text-xs text-base-content/70">(default)</span>
								{/if}
							</label>
						{/each}
					</div>
				</fieldset>
			{:else}
				<div class="alert alert-warning bg-warning/10 border border-warning/30">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="stroke-current shrink-0 h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 9v2m0 4v2m0 0v2m-6-4v2m0 0v2m12-6v2m0 0v2"
						/></svg
					>
					<span class="text-sm text-base-content"
						>Default 'general' topic is available for all webhooks.</span
					>
				</div>
			{/if}

			<!-- Payload Type -->
			<fieldset>
				<legend class="block text-sm font-semibold text-base-content mb-3"> Payload Type </legend>
				<div class="space-y-2">
					<label
						class="flex items-center gap-3 cursor-pointer p-3 border border-base-300 rounded hover:bg-base-200"
					>
						<input
							type="radio"
							name="payload-type"
							value="beebuzz"
							bind:group={newWebhookPayloadType}
							disabled={isLoading}
							class="radio radio-sm"
						/>
						<div>
							<div class="font-medium text-base-content flex items-center gap-2">
								<FileText size={16} />
								Beebuzz Standard
							</div>
							<div class="text-xs text-base-content/70">Simple payload with title and message</div>
						</div>
					</label>

					<label
						class="flex items-center gap-3 cursor-pointer p-3 border border-base-300 rounded hover:bg-base-200"
					>
						<input
							type="radio"
							name="payload-type"
							value="custom"
							bind:group={newWebhookPayloadType}
							disabled={isLoading}
							class="radio radio-sm"
						/>
						<div>
							<div class="font-medium text-base-content flex items-center gap-2">
								<Settings size={16} />
								Custom Payload Mapping
							</div>
							<div class="text-xs text-base-content/70">Map JSON fields to title and body</div>
						</div>
					</label>

					{#if newWebhookPayloadType === 'custom' && !inspectSession}
						<label
							class="flex items-center gap-3 cursor-pointer p-3 border border-primary/50 bg-primary/5 rounded hover:bg-primary/10"
						>
							<input
								type="checkbox"
								bind:checked={enableInspectMode}
								disabled={isLoading}
								class="checkbox checkbox-sm checkbox-primary"
							/>
							<div>
								<div class="font-medium text-base-content flex items-center gap-2">
									<Search size={16} />
									Test & Inspect
								</div>
								<div class="text-xs text-base-content/70">
									Capture a sample payload to discover the correct paths
								</div>
							</div>
						</label>
					{/if}
				</div>
			</fieldset>

			<!-- Inspect Mode UI -->
			{#if enableInspectMode && newWebhookPayloadType === 'custom'}
				<div class="border-t border-base-300 pt-4 space-y-4">
					{#if !inspectSession}
						<!-- Show Start Inspect button -->
						<div class="alert alert-info bg-info/10 border border-info/30">
							<Eye size={16} />
							<div class="text-sm">
								<strong>Test & Inspect</strong> - Send a sample payload to discover the correct JSON paths.
								The next request to your webhook URL will be captured and displayed below.
							</div>
						</div>
					{:else if inspectStatus?.status === 'waiting'}
						<!-- Waiting for payload -->
						<div class="border border-warning/30 rounded bg-warning/5">
							<div class="p-3 border-b border-warning/20">
								<div class="flex items-center gap-2">
									<span class="loading loading-spinner loading-sm"></span>
									<div class="text-sm">
										{#if inspectSession.url === 'Connecting...'}
											<strong>Creating temporary webhook...</strong>
										{:else}
											<strong>Waiting for payload...</strong>
										{/if}
									</div>
								</div>
							</div>
							<div class="p-3 bg-base-200/50">
								<p class="text-xs text-base-content/70 mb-2">Send a POST request to this URL:</p>
								<div class="flex items-center justify-between gap-2">
									<code class="text-xs text-base-content break-all flex-1"
										>{inspectSession.url}</code
									>
									<button
										type="button"
										onclick={() => copyInspectUrl()}
										class="btn btn-xs btn-ghost gap-1 flex-shrink-0"
										disabled={inspectSession.url === 'Connecting...'}
									>
										{#if copiedInspectUrl}
											<Check size={12} />
											Copied
										{:else}
											<Copy size={12} />
											Copy URL
										{/if}
									</button>
								</div>
								{#if inspectSession.url !== 'Connecting...'}
									<p class="text-xs text-base-content/50 mt-2">
										This URL will expire in 10 minutes
									</p>
								{/if}
							</div>
						</div>
					{:else if inspectStatus?.status === 'captured' && inspectStatus.payload}
						<!-- Payload captured - show JSON tree -->
						<div class="alert alert-success bg-success/10 border border-success/30 mb-4">
							<Check size={16} />
							<div class="text-sm">
								<strong>Payload received!</strong> Click on any text field in the JSON below to map it
								to title or body.
							</div>
						</div>

						<div class="bg-info/5 border border-info/20 rounded p-3 mb-4">
							<div class="flex items-start gap-2">
								<Search size={16} class="mt-0.5 text-info flex-shrink-0" />
								<div class="text-sm text-base-content">
									<p class="font-medium mb-1">How to configure your webhook:</p>
									<ol class="list-decimal list-inside space-y-1 text-xs text-base-content/80">
										<li>
											<strong>Select Title:</strong> Click on the JSON field containing the
											notification title (e.g., <code>text</code>, <code>title</code>, or
											<code>subject</code>)
										</li>
										<li>
											<strong>Select Body:</strong> Click on the field containing the message
											content (e.g., <code>message</code>, <code>body</code>, or
											<code>description</code>)
										</li>
										<li>The paths will appear in the fields below automatically</li>
									</ol>
								</div>
							</div>
						</div>

						<div class="mb-4">
							<JsonTreeViewer data={inspectStatus.payload} onPathClick={handlePathClick} />
						</div>

						<div class="space-y-3">
							<div>
								<label
									for="inspect-titlePath"
									class="block text-sm font-semibold text-base-content mb-1"
								>
									Title Path
									<span class="text-xs font-normal text-base-content/70">— Click a field above</span
									>
								</label>
								<input
									type="text"
									id="inspect-titlePath"
									placeholder="e.g., text, title, or subject field"
									class="input input-bordered w-full bg-base-100 text-base-content"
									bind:value={newWebhookTitlePath}
									disabled={isLoading}
								/>
							</div>

							<div>
								<label
									for="inspect-bodyPath"
									class="block text-sm font-semibold text-base-content mb-1"
								>
									Body Path
									<span class="text-xs font-normal text-base-content/70">— Click a field above</span
									>
								</label>
								<input
									type="text"
									id="inspect-bodyPath"
									placeholder="e.g., message, body, or description field"
									class="input input-bordered w-full bg-base-100 text-base-content"
									bind:value={newWebhookBodyPath}
									disabled={isLoading}
								/>
							</div>
						</div>
					{/if}
				</div>
			{:else if newWebhookPayloadType === 'custom' && !enableInspectMode}
				<!-- Manual path entry (when not using inspect mode) -->
				<div class="border-t border-base-300 pt-4 space-y-4">
					<div>
						<label
							for="create-titlePath"
							class="block text-sm font-semibold text-base-content mb-2"
						>
							Title Path
							<span class="text-xs font-normal text-base-content/70">*</span>
						</label>
						<input
							type="text"
							id="create-titlePath"
							placeholder="e.g., data.title or message.subject"
							class="input input-bordered w-full bg-base-100 text-base-content"
							bind:value={newWebhookTitlePath}
							disabled={isLoading}
							required
						/>
						<p class="text-xs text-base-content/70 mt-1">
							Enter JSON path without leading dot (uses gjson syntax)
						</p>
					</div>

					<div>
						<label for="create-bodyPath" class="block text-sm font-semibold text-base-content mb-2">
							Body Path
							<span class="text-xs font-normal text-base-content/70">*</span>
						</label>
						<input
							type="text"
							id="create-bodyPath"
							placeholder="e.g., data.body or message.content"
							class="input input-bordered w-full bg-base-100 text-base-content"
							bind:value={newWebhookBodyPath}
							disabled={isLoading}
							required
						/>
						<p class="text-xs text-base-content/70 mt-1">
							Enter JSON path without leading dot (uses gjson syntax)
						</p>
					</div>
				</div>
			{/if}
		</form>
		{#snippet actions()}
			<button
				type="button"
				class="btn btn-outline"
				onclick={() => {
					resetInspectState();
					showCreateModal = false;
				}}
				disabled={isLoading}
			>
				Cancel
			</button>
			{#if enableInspectMode && !inspectSession}
				<button
					type="button"
					class="btn btn-primary gap-2"
					disabled={isLoading || !newWebhookName.trim() || selectedTopics.length === 0}
					onclick={() => void startInspectMode()}
				>
					{#if isLoading}
						<span class="loading loading-spinner loading-sm"></span>
						Starting...
					{:else}
						<Eye size={16} />
						Start Inspect
					{/if}
				</button>
			{:else if enableInspectMode && inspectStatus?.status === 'captured'}
				<button
					type="button"
					class="btn btn-primary"
					disabled={isLoading || !newWebhookTitlePath.trim() || !newWebhookBodyPath.trim()}
					onclick={() => void finalizeInspect()}
				>
					{#if isLoading}
						<span class="loading loading-spinner loading-sm"></span>
						Creating...
					{:else}
						Create Webhook
					{/if}
				</button>
			{:else}
				<button
					type="submit"
					class="btn btn-primary"
					disabled={isLoading || !isCreateFormValid()}
					form="create-webhook-form"
				>
					{#if isLoading}
						<span class="loading loading-spinner loading-sm"></span>
						Creating...
					{:else}
						Create Webhook
					{/if}
				</button>
			{/if}
		{/snippet}
	</SettingsModal>

	<!-- Webhooks List -->
	<div class="space-y-4">
		{#if isLoading && webhooks.length === 0}
			<div class="flex justify-center py-8">
				<span class="loading loading-spinner loading-lg text-primary"></span>
			</div>
		{:else if webhooks.length === 0}
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
					<h3 class="font-bold text-base-content">No webhooks yet</h3>
					<div class="text-sm text-base-content/70">
						Create your first webhook to receive JSON notifications
					</div>
				</div>
			</div>
		{:else}
			{#each webhooks as webhook (webhook.id)}
				{#if editingWebhookId === webhook.id}
					<!-- EDIT MODE -->
					<div class="card bg-base-100 shadow border border-base-300 p-4">
						<form
							onsubmit={(e) => {
								e.preventDefault();
								void handleEdit(webhook.id);
							}}
							class="space-y-4"
						>
							<div>
								<div class="mb-2 flex items-center justify-between gap-3">
									<label
										for="edit-name-{webhook.id}"
										class="text-sm font-semibold text-base-content"
									>
										Webhook Name
									</label>
									<p
										class="text-xs tabular-nums whitespace-nowrap"
										class:text-base-content-70={editWebhookName.length <= MAX_DISPLAY_NAME_SOFT_LEN}
										class:text-warning={editWebhookName.length > MAX_DISPLAY_NAME_SOFT_LEN &&
											editWebhookName.length <= MAX_DISPLAY_NAME_LEN}
										class:text-error={editWebhookName.length > MAX_DISPLAY_NAME_LEN}
									>
										({editWebhookName.length}/{MAX_DISPLAY_NAME_LEN})
									</p>
								</div>
								<input
									type="text"
									id="edit-name-{webhook.id}"
									placeholder="e.g., Home Assistant Alerts"
									class="input input-bordered w-full bg-base-100 text-base-content"
									bind:value={editWebhookName}
									disabled={isLoading}
									maxlength={MAX_DISPLAY_NAME_LEN}
									required
								/>
							</div>

							<div>
								<div class="mb-2 flex items-center justify-between gap-3">
									<label
										for="edit-desc-{webhook.id}"
										class="text-sm font-semibold text-base-content"
									>
										Description
									</label>
									<p class="text-xs text-base-content/70 tabular-nums whitespace-nowrap">
										({editWebhookDescription.length}/{MAX_DESCRIPTION_LEN})
									</p>
								</div>
								<textarea
									id="edit-desc-{webhook.id}"
									placeholder="What is this webhook for?"
									class="textarea textarea-bordered w-full bg-base-100 text-base-content"
									bind:value={editWebhookDescription}
									disabled={isLoading}
									maxlength={MAX_DESCRIPTION_LEN}
									rows="2"
								></textarea>
							</div>

							<div>
								<label
									for="edit-priority-{webhook.id}"
									class="block text-sm font-semibold text-base-content mb-2"
								>
									Delivery Priority
								</label>
								<select
									id="edit-priority-{webhook.id}"
									class="select select-bordered w-full bg-base-100 text-base-content"
									bind:value={editWebhookPriority}
									disabled={isLoading}
								>
									<option value="normal">Normal</option>
									<option value="high">High</option>
								</select>
								<p class="text-xs text-base-content/70 mt-1">
									Controls the delivery urgency used for notifications sent by this webhook.
								</p>
							</div>

							{#if topics.length > 0}
								<fieldset>
									<legend class="block text-sm font-semibold text-base-content mb-2">
										Topics
										<span class="text-xs font-normal text-base-content/70"
											>(At least one required)</span
										>
									</legend>
									<div class="space-y-2 max-h-40 overflow-y-auto">
										{#each topics as topic (topic.id)}
											<label class="flex items-center gap-2 cursor-pointer">
												<input
													type="checkbox"
													checked={editSelectedTopics.includes(topic.id)}
													onchange={() => toggleEditTopic(topic.id)}
													disabled={isLoading}
													class="checkbox checkbox-sm"
												/>
												<TopicChip name={topic.name} />
											</label>
										{/each}
									</div>
								</fieldset>
							{/if}

							<fieldset>
								<legend class="block text-sm font-semibold text-base-content mb-3">
									Payload Type
								</legend>
								<div class="space-y-2">
									<label
										class="flex items-center gap-3 cursor-pointer p-3 border border-base-300 rounded hover:bg-base-200"
									>
										<input
											type="radio"
											name="edit-payload-type"
											value="beebuzz"
											bind:group={editWebhookPayloadType}
											disabled={isLoading}
											class="radio radio-sm"
										/>
										<div>
											<div class="font-medium text-base-content flex items-center gap-2">
												<FileText size={16} />
												Beebuzz Standard
											</div>
										</div>
									</label>

									<label
										class="flex items-center gap-3 cursor-pointer p-3 border border-base-300 rounded hover:bg-base-200"
									>
										<input
											type="radio"
											name="edit-payload-type"
											value="custom"
											bind:group={editWebhookPayloadType}
											disabled={isLoading}
											class="radio radio-sm"
										/>
										<div>
											<div class="font-medium text-base-content flex items-center gap-2">
												<Settings size={16} />
												Custom Payload Mapping
											</div>
										</div>
									</label>
								</div>
							</fieldset>

							{#if editWebhookPayloadType === 'custom'}
								<div class="border-t border-base-300 pt-4 space-y-4">
									<div>
										<label
											for="edit-titlePath-{webhook.id}"
											class="block text-sm font-semibold text-base-content mb-2"
										>
											Title Path
											<span class="text-xs font-normal text-base-content/70">*</span>
										</label>
										<input
											type="text"
											id="edit-titlePath-{webhook.id}"
											placeholder="e.g., data.title or message.subject"
											class="input input-bordered w-full bg-base-100 text-base-content"
											bind:value={editWebhookTitlePath}
											disabled={isLoading}
											required
										/>
										<p class="text-xs text-base-content/70 mt-1">
											Enter the JSON path without leading dot (uses gjson syntax)
										</p>
									</div>

									<div>
										<label
											for="edit-bodyPath-{webhook.id}"
											class="block text-sm font-semibold text-base-content mb-2"
										>
											Body Path
											<span class="text-xs font-normal text-base-content/70">*</span>
										</label>
										<input
											type="text"
											id="edit-bodyPath-{webhook.id}"
											placeholder="e.g., data.body or message.content"
											class="input input-bordered w-full bg-base-100 text-base-content"
											bind:value={editWebhookBodyPath}
											disabled={isLoading}
											required
										/>
										<p class="text-xs text-base-content/70 mt-1">
											Enter the JSON path without leading dot (uses gjson syntax)
										</p>
									</div>
								</div>
							{/if}

							<div class="flex flex-col gap-2 pt-4 sm:flex-row sm:justify-end">
								<button
									type="button"
									class="btn btn-outline btn-sm"
									onclick={cancelEdit}
									disabled={isLoading}
								>
									Cancel
								</button>
								<button
									type="submit"
									class="btn btn-primary btn-sm"
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
					<!-- VIEW MODE -->
					<div class="card bg-base-100 shadow border border-base-300">
						<!-- Summary -->
						<div class="p-4 space-y-4">
							<!-- Name with Action Buttons -->
							<div class="flex justify-between items-start">
								<div class="flex items-center gap-2 pr-3">
									<p class="font-semibold text-base-content">{webhook.name}</p>
									{#if webhook.priority === 'high'}
										<div class="tooltip tooltip-bottom" data-tip="High priority">
											<span
												class="inline-flex h-5 w-5 items-center justify-center rounded-full border border-primary/30 bg-primary/15 text-secondary"
												aria-label="High priority"
											>
												<ChevronsUp size={12} />
											</span>
										</div>
									{/if}
								</div>
								<div class="relative">
									<button
										type="button"
										onclick={() => toggleWebhookActions(webhook.id)}
										disabled={isLoading}
										class="btn btn-ghost btn-circle btn-xs text-base-content/60"
										aria-label={`Open actions for ${webhook.name}`}
										aria-expanded={actionsMenuOpenWebhookId === webhook.id}
									>
										<EllipsisVertical size={14} />
									</button>
									{#if actionsMenuOpenWebhookId === webhook.id}
										<ul
											role="menu"
											bind:this={actionsMenuRef}
											class="menu absolute right-0 z-20 mt-1 w-44 rounded-box border border-base-300 bg-base-100 p-2 shadow"
										>
											<li>
												<button type="button" onclick={() => startEdit(webhook)}>
													<Pencil size={16} />
													Edit
												</button>
											</li>
											<li>
												<button type="button" onclick={() => openRegenerateDialog(webhook)}>
													<RefreshCw size={16} />
													Regenerate
												</button>
											</li>
											<li>
												<button
													type="button"
													class="text-error"
													onclick={() => openDeleteDialog(webhook)}
												>
													<Trash2 size={16} />
													Delete
												</button>
											</li>
										</ul>
									{/if}
								</div>
							</div>

							<!-- Description -->
							{#if webhook.description}
								<p class="text-sm text-base-content/70">{webhook.description}</p>
							{/if}

							<!-- Topics -->
							{#if webhook.topic_ids && webhook.topic_ids.length > 0}
								<div>
									<div class="flex flex-wrap gap-2">
										{#each webhook.topic_ids as topicId, idx (idx)}
											<TopicChip name={topics.find((t) => t.id === topicId)?.name ?? topicId} />
										{/each}
									</div>
								</div>
							{/if}

							<!-- Last used -->
							<p class="text-xs text-base-content/70">
								{#if webhook.last_used_at}
									Last used {new Date(webhook.last_used_at).toLocaleDateString()}
								{:else}
									Never used
								{/if}
							</p>
						</div>

						<!-- Expandable Detail View -->
						{#if expandedWebhookId === webhook.id}
							<div class="border-t border-base-300 p-4 space-y-4 bg-base-200">
								<!-- Mapping Rules (only for custom) -->
								{#if webhook.payload_type === 'custom'}
									<div>
										<div class="block text-sm font-semibold text-base-content mb-2">
											Mapping Rules
										</div>
										<div class="bg-base-100 border border-base-300 rounded p-3 space-y-2 text-sm">
											<div>
												<span class="font-medium text-base-content">Title Path:</span>
												<code class="text-xs bg-base-200 px-2 py-1 rounded ml-2">
													{webhook.title_path}
												</code>
											</div>
											<div>
												<span class="font-medium text-base-content">Body Path:</span>
												<code class="text-xs bg-base-200 px-2 py-1 rounded ml-2">
													{webhook.body_path}
												</code>
											</div>
										</div>
									</div>
								{/if}

								<!-- Example curl -->
								<div>
									<div class="flex items-center justify-between mb-2">
										<div class="block text-sm font-semibold text-base-content">Example</div>
										<button
											type="button"
											onclick={() => copyCurlExample(webhook)}
											class="btn btn-xs btn-ghost gap-1"
										>
											{#if copiedCurlWebhookId === webhook.id}
												<Check size={12} />
												Copied
											{:else}
												<Copy size={12} />
												Copy cURL
											{/if}
										</button>
									</div>
									<pre class="bg-base-100 border border-base-300 rounded p-3 overflow-x-auto"><code
											class="font-mono text-xs text-base-content">{buildExampleCurl(webhook)}</code
										></pre>
								</div>
							</div>
						{/if}

						<!-- Expand/Collapse Button -->
						<div class="border-t border-base-300 p-2 text-center">
							<button
								type="button"
								onclick={() => {
									if (expandedWebhookId === webhook.id) {
										expandedWebhookId = null;
									} else {
										expandedWebhookId = webhook.id;
									}
								}}
								class="text-xs text-base-content/70 hover:text-base-content"
							>
								{expandedWebhookId === webhook.id ? '▲ Hide Details' : '▼ Show Details'}
							</button>
						</div>
					</div>
				{/if}
			{/each}
		{/if}
	</div>

	<SettingsModal
		open={Boolean(webhookPendingRegenerate)}
		title="Regenerate Token"
		description={webhookPendingRegenerate ? webhookPendingRegenerate.name : undefined}
		onClose={() => {
			webhookPendingRegenerate = null;
		}}
		size="sm"
	>
		<p class="text-sm text-base-content/70">
			This will invalidate the current webhook URL. Any external services using it will stop working
			until updated.
		</p>
		{#snippet actions()}
			<button
				type="button"
				class="btn btn-outline"
				onclick={() => (webhookPendingRegenerate = null)}
			>
				Cancel
			</button>
			<button type="button" class="btn btn-warning" onclick={handleRegenerate}>Regenerate</button>
		{/snippet}
	</SettingsModal>

	<SettingsModal
		open={Boolean(webhookPendingDelete)}
		title="Delete webhook"
		description={webhookPendingDelete
			? `Delete "${webhookPendingDelete.name}"? This cannot be undone.`
			: undefined}
		onClose={() => {
			webhookPendingDelete = null;
		}}
		size="sm"
	>
		<p class="text-sm text-base-content/70">
			Any external services using this webhook will stop working.
		</p>
		{#snippet actions()}
			<button type="button" class="btn btn-outline" onclick={() => (webhookPendingDelete = null)}>
				Cancel
			</button>
			<button type="button" class="btn btn-error" onclick={handleDelete}>Delete Webhook</button>
		{/snippet}
	</SettingsModal>
</div>
