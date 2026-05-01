<script lang="ts">
	import {
		Image,
		FileDown,
		Video,
		ChevronUp,
		EllipsisVertical,
		Check,
		ListChecks,
		Mail,
		Trash2
	} from '@lucide/svelte';
	import type { Notification, NotificationPriority } from '@beebuzz/shared/types';
	import { notificationsStore, formatRelativeTime } from '$lib/stores/notifications.svelte';
	import { fetchAndCacheAttachment, isImageMime, isVideoMime } from '$lib/utils/attachmentCache';
	import type { CachedAttachment } from '$lib/utils/attachmentCache';
	import { parseHttpsLinkSegments } from '$lib/utils/linkify';
	import { tick } from 'svelte';

	const PRIORITY_HIGH: NotificationPriority = 'high';

	interface Props {
		notification: Notification;
		onSelectTopic?: (topic: string) => void;
		selectionMode?: boolean;
		selected?: boolean;
		hideTopic?: boolean;
		onToggleSelection?: (id: string) => void;
		onEnterSelection?: (id: string) => void;
	}

	let {
		notification,
		onSelectTopic,
		selectionMode = false,
		selected = false,
		hideTopic = false,
		onToggleSelection,
		onEnterSelection
	}: Props = $props();

	/** Whether this notification is unread. */
	const isUnread = $derived(notificationsStore.unreadIds.has(notification.id));

	/** CSS classes for the left border accent based on priority. */
	const priorityBorderClass = $derived.by(() => {
		const p = notification.priority ?? 'normal';
		if (p === PRIORITY_HIGH) return 'border-l-4 border-l-error';
		return '';
	});

	let cachedAttachment = $state<CachedAttachment | null>(null);
	let showImageModal = $state(false);
	let showVideoModal = $state(false);
	let attachmentLoading = $state(false);
	let imageDialog = $state<HTMLDialogElement | undefined>(undefined);
	let videoDialog = $state<HTMLDialogElement | undefined>(undefined);
	// Why: Safari can't reliably play <video> from data: URLs (no range requests).
	// Create a blob URL on demand for the modal and revoke it on close.
	let videoObjectUrl = $state<string | null>(null);

	// Why: DaisyUI's focus-based dropdown breaks in Safari — focus leaves the
	// trigger before onclick fires on menu items, swallowing the event.
	// Manual state control avoids this.
	let menuOpen = $state(false);
	let menuRef = $state<HTMLDivElement | undefined>(undefined);

	/** Toggle the actions menu open/closed. */
	function toggleMenu() {
		menuOpen = !menuOpen;
	}

	$effect(() => {
		if (!menuOpen) return;

		/** Close the menu when clicking outside the dropdown container. */
		const handleClickOutside = (e: MouseEvent) => {
			if (menuRef && !menuRef.contains(e.target as Node)) {
				menuOpen = false;
			}
		};

		document.addEventListener('click', handleClickOutside, true);
		return () => document.removeEventListener('click', handleClickOutside, true);
	});

	$effect(() => {
		if (!imageDialog) return;
		if (showImageModal && cachedAttachment) {
			imageDialog.showModal();
		} else {
			imageDialog.close();
		}
	});

	$effect(() => {
		if (!videoDialog) return;
		if (showVideoModal) {
			videoDialog.showModal();
		} else {
			videoDialog.close();
		}
	});

	/** Create the blob URL and open the video modal. */
	function openVideoModal() {
		if (!cachedAttachment) return;
		if (!videoObjectUrl) {
			videoObjectUrl = URL.createObjectURL(cachedAttachment.blob);
		}
		showVideoModal = true;
	}

	/**
	 * Close the video modal and revoke the blob URL to free memory.
	 * Awaits tick() so the <video> element unmounts before the URL is revoked,
	 * avoiding a microtask window where the still-mounted element holds a
	 * revoked src.
	 */
	async function closeVideoModal() {
		showVideoModal = false;
		await tick();
		if (videoObjectUrl) {
			URL.revokeObjectURL(videoObjectUrl);
			videoObjectUrl = null;
		}
	}

	function handleMarkRead() {
		notificationsStore.markAsRead(notification.id);
		menuOpen = false;
	}

	function handleMarkUnread() {
		notificationsStore.markAsUnread(notification.id);
		menuOpen = false;
	}

	function handleEnterSelection() {
		onEnterSelection?.(notification.id);
		menuOpen = false;
	}

	function handleDelete() {
		notificationsStore.remove(notification.id);
		menuOpen = false;
	}

	/** Trigger browser download for non-image attachments. */
	function triggerDownload(dataUrl: string, filename: string) {
		const a = document.createElement('a');
		a.href = dataUrl;
		a.download = filename;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
	}

	/** Show image modal or trigger download depending on MIME type. */
	function handleLoaded() {
		if (!cachedAttachment) return;
		if (isImageMime(cachedAttachment.mimeType)) {
			showImageModal = true;
		} else if (isVideoMime(cachedAttachment.mimeType)) {
			openVideoModal();
		} else {
			const filename = notification.attachment?.filename || 'attachment.bin';
			triggerDownload(cachedAttachment.dataUrl, filename);
		}
	}

	/** Returns a cached attachment assembled from inline base64 notification data. */
	function buildInlineAttachment(): CachedAttachment | null {
		const inlineData = notification.attachment?.data;
		if (!inlineData) {
			return null;
		}

		const mimeType = notification.attachment?.mime || 'application/octet-stream';
		const binary = atob(inlineData);
		const bytes = new Uint8Array(binary.length);
		for (let i = 0; i < binary.length; i++) {
			bytes[i] = binary.charCodeAt(i);
		}
		const blob = new Blob([bytes], { type: mimeType });
		return {
			dataUrl: `data:${mimeType};base64,${inlineData}`,
			blob,
			mimeType,
			timestamp: Date.now()
		};
	}

	/** Compute the label shown in the attachment chip. */
	const attachmentLabel = $derived.by(() => {
		const filename = notification.attachment?.filename;
		if (filename) return filename;
		if (notification.attachment?.mime && isImageMime(notification.attachment.mime)) {
			return 'Image attachment';
		}
		if (notification.attachment?.mime && isVideoMime(notification.attachment.mime)) {
			return 'Video attachment';
		}
		return 'File attachment';
	});

	const relativeTime = $derived(formatRelativeTime(notification.sentAt));
	const absoluteTime = $derived(
		notification.sentAt.toLocaleString('en-US', {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
			hour12: false
		})
	);

	const bodySegments = $derived.by(() => {
		if (!notification.body) return [];
		return parseHttpsLinkSegments(notification.body);
	});

	function handleTopicClick() {
		if (selectionMode) return;
		if (!notification.topic || !onSelectTopic) return;
		onSelectTopic(notification.topic);
	}

	function handleSelectionToggle() {
		if (!selectionMode) return;
		onToggleSelection?.(notification.id);
	}

	/** Fetch and decrypt attachment on click (lazy). */
	async function loadAttachment() {
		if (selectionMode) return;
		if (cachedAttachment) {
			handleLoaded();
			return;
		}

		const inlineAttachment = buildInlineAttachment();
		if (inlineAttachment) {
			cachedAttachment = inlineAttachment;
			handleLoaded();
			return;
		}

		if (!notification.attachment?.token) return;

		const url = `/attachments/${notification.attachment.token}`;
		attachmentLoading = true;
		try {
			cachedAttachment = await fetchAndCacheAttachment(url, notification.attachment.mime, true);
			handleLoaded();
		} catch (err) {
			console.error('[NotificationCard] Failed to load attachment:', err);
		} finally {
			attachmentLoading = false;
		}
	}
</script>

<div
	class="relative rounded-box border border-base-300 bg-base-100 p-4 transition-colors {priorityBorderClass} {isUnread
		? ''
		: 'opacity-60'} {selected ? 'border-primary bg-base-200' : ''}"
>
	{#if selectionMode}
		<button
			type="button"
			class="absolute inset-0 z-10 rounded-box"
			aria-pressed={selected}
			aria-label="{selected ? 'Deselect' : 'Select'} message: {notification.title}"
			onclick={handleSelectionToggle}
		>
			<span class="sr-only">{selected ? 'Deselect message' : 'Select message'}</span>
		</button>
	{/if}

	<div class="flex items-start justify-between gap-3">
		<div class="flex items-start gap-2 flex-1 min-w-0">
			{#if isUnread}
				<span class="mt-2 h-2.5 w-2.5 rounded-full bg-primary shrink-0" aria-label="Unread"></span>
			{/if}

			<div class="flex-1 min-w-0">
				<div class="flex items-center gap-2">
					{#if notification.priority === PRIORITY_HIGH}
						<ChevronUp size={16} class="text-error shrink-0" aria-label="High priority" />
					{/if}
					<h3
						class="font-bold text-base {notification.priority === PRIORITY_HIGH
							? 'text-error'
							: 'text-base-content'}"
					>
						{notification.title}
					</h3>
				</div>
				{#if notification.body}
					<p class="mt-2 text-sm text-base-content/85 [overflow-wrap:anywhere]">
						{#each bodySegments as segment, index (index)}
							{#if segment.kind === 'link'}
								<a
									href={segment.href}
									target="_blank"
									rel="noopener noreferrer"
									class="link link-neutral font-medium hover:underline [overflow-wrap:anywhere]"
								>
									{segment.text}
								</a>
							{:else}
								{segment.text}
							{/if}
						{/each}
					</p>
				{/if}
			</div>
		</div>

		<div class="flex items-start gap-1 shrink-0">
			{#if selectionMode}
				<span
					class="flex h-6 w-6 items-center justify-center rounded-full border border-base-300 {selected
						? 'bg-primary text-primary-content border-primary'
						: 'bg-base-100 text-base-content/50'}"
					aria-hidden="true"
				>
					<Check size={14} />
				</span>
			{:else}
				<!-- Actions menu — uses manual state instead of DaisyUI focus-based
				     dropdown because Safari fires focus-loss before onclick, swallowing the event -->
				<div class="relative" bind:this={menuRef}>
					<button
						type="button"
						class="btn btn-ghost btn-circle btn-sm text-base-content/60"
						aria-label="Notification actions"
						aria-expanded={menuOpen}
						onclick={toggleMenu}
					>
						<EllipsisVertical size={16} />
					</button>
					{#if menuOpen}
						<ul
							role="menu"
							class="menu absolute right-0 z-20 mt-1 w-44 rounded-box border border-base-200 bg-base-100 p-2 shadow"
						>
							<li>
								<button type="button" onclick={handleEnterSelection}>
									<ListChecks size={16} />
									Select
								</button>
							</li>
							{#if isUnread}
								<li>
									<button type="button" onclick={handleMarkRead}>
										<Check size={16} />
										Mark as read
									</button>
								</li>
							{:else}
								<li>
									<button type="button" onclick={handleMarkUnread}>
										<Mail size={16} />
										Mark as unread
									</button>
								</li>
							{/if}
							<li>
								<button type="button" class="text-error" onclick={handleDelete}>
									<Trash2 size={16} />
									Delete
								</button>
							</li>
						</ul>
					{/if}
				</div>
			{/if}
		</div>
	</div>

	<div class="mt-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-base-content/70">
		{#if notification.topic && !hideTopic}
			<button
				type="button"
				class="badge badge-sm badge-ghost border-base-300 gap-0.5 font-medium hover:bg-base-200"
				disabled={selectionMode}
				onclick={handleTopicClick}
			>
				#{notification.topic}
			</button>
			<span aria-hidden="true">·</span>
		{/if}
		<time
			datetime={notification.sentAt.toISOString()}
			title={absoluteTime}
			class="whitespace-nowrap"
		>
			{relativeTime}
		</time>
	</div>

	{#if notification.attachment?.token || notification.attachment?.data}
		<div class="mt-3">
			<button
				onclick={loadAttachment}
				disabled={attachmentLoading || selectionMode}
				type="button"
				class="btn btn-ghost btn-sm justify-start gap-2 px-2 text-left hover:bg-base-200"
				aria-label="Open attachment: {attachmentLabel}"
			>
				{#if notification.attachment?.mime && isImageMime(notification.attachment.mime)}
					<Image size={16} />
				{:else if notification.attachment?.mime && isVideoMime(notification.attachment.mime)}
					<Video size={16} />
				{:else}
					<FileDown size={16} />
				{/if}
				<span class="truncate text-xs">{attachmentLabel}</span>
				{#if attachmentLoading}
					<span class="loading loading-spinner loading-xs" aria-hidden="true"></span>
				{/if}
			</button>
		</div>
	{/if}
</div>

<!-- Image modal — rendered outside the card to avoid inheriting opacity-60 on read messages -->
<dialog bind:this={imageDialog} class="modal" onclose={() => (showImageModal = false)}>
	<div class="modal-box max-w-4xl w-11/12 flex flex-col items-center">
		<form method="dialog">
			<button
				type="submit"
				class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
				aria-label="Close image viewer"
			>
				✕
			</button>
		</form>
		<img
			src={cachedAttachment?.dataUrl}
			alt="{notification.title} attachment"
			class="rounded-lg max-w-full max-h-96"
		/>
	</div>
	<form method="dialog" class="modal-backdrop"><button type="submit">close</button></form>
</dialog>

<!-- Video modal — iOS PWAs handle programmatic downloads poorly, so playable media opens inline -->
<dialog bind:this={videoDialog} class="modal" onclose={closeVideoModal}>
	<div class="modal-box max-w-4xl w-11/12 flex flex-col items-center">
		<form method="dialog">
			<button
				type="submit"
				class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
				aria-label="Close video viewer"
			>
				✕
			</button>
		</form>
		{#if showVideoModal && videoObjectUrl}
			<video
				src={videoObjectUrl}
				controls
				muted
				playsinline
				preload="metadata"
				class="rounded-lg max-w-full max-h-[80vh]"
			>
				<a
					href={cachedAttachment?.dataUrl || ''}
					download={notification.attachment?.filename || 'attachment'}
				>
					Download video
				</a>
			</video>
		{/if}
	</div>
	<form method="dialog" class="modal-backdrop"><button type="submit">close</button></form>
</dialog>
