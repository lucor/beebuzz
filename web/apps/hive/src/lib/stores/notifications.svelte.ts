import { browser } from '$app/environment';
import {
	notificationsRepository,
	StorageQuotaError,
	type StoredNotificationRecord
} from '$lib/services/notifications-repository';
import { SvelteSet } from 'svelte/reactivity';
import type { Notification, NotificationPriority } from '@beebuzz/shared/types';

export type TopicSummary = {
	name: string;
	count: number;
	unreadCount: number;
	lastActivityAt: number;
};

/** Aggregates notifications by topic, sorted by most recent activity. */
function computeTopicSummaries(
	notifications: Notification[],
	unreadIds: Set<string>
): TopicSummary[] {
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- pure function, not reactive state
	const topicMap = new Map<string, TopicSummary>();

	for (const notification of notifications) {
		if (!notification.topic) continue;

		const sentAt = notification.sentAt.getTime();
		const existing = topicMap.get(notification.topic);

		if (!existing) {
			topicMap.set(notification.topic, {
				name: notification.topic,
				count: 1,
				unreadCount: unreadIds.has(notification.id) ? 1 : 0,
				lastActivityAt: sentAt
			});
			continue;
		}

		existing.count += 1;
		if (unreadIds.has(notification.id)) existing.unreadCount += 1;
		if (sentAt > existing.lastActivityAt) existing.lastActivityAt = sentAt;
	}

	return [...topicMap.values()].sort((a, b) => {
		if (b.lastActivityAt !== a.lastActivityAt) return b.lastActivityAt - a.lastActivityAt;
		if (b.count !== a.count) return b.count - a.count;
		return a.name.localeCompare(b.name);
	});
}

function createNotificationsStore() {
	let notifications = $state<Notification[]>([]);
	let activeDeviceId = $state<string | null>(null);
	let syncCursor = $state<string | null>(null);
	let storageError = $state<string | null>(null);
	const unreadIds = new SvelteSet<string>();

	function parseStoredNotification(record: unknown): Notification | null {
		if (!record || typeof record !== 'object') return null;
		const r = record as Record<string, unknown>;

		if (
			typeof r.id !== 'string' ||
			typeof r.title !== 'string' ||
			typeof r.body !== 'string' ||
			typeof r.sentAt !== 'string'
		) {
			return null;
		}

		const sentAt = new Date(r.sentAt);
		if (Number.isNaN(sentAt.getTime())) {
			return null;
		}

		return {
			id: r.id,
			title: r.title,
			body: r.body,
			topicId: (r.topicId as string | null) ?? null,
			// Writers persist an empty string for "no topic"; normalize it back to
			// null so the in-memory representation matches Notification.topic.
			topic: typeof r.topic === 'string' && r.topic !== '' ? r.topic : null,
			sentAt,
			priority: (r.priority as NotificationPriority) ?? 'normal',
			attachment: r.attachment as Notification['attachment']
		};
	}

	function normalizeStorageError(error: unknown): string {
		if (
			error instanceof StorageQuotaError ||
			(error &&
				typeof error === 'object' &&
				(error as { name?: unknown }).name === 'StorageQuotaError')
		) {
			return 'Browser storage is full. New notification history may not persist.';
		}
		return error instanceof Error ? error.message : 'Notification storage failed';
	}

	function recordStorageError(error: unknown) {
		storageError = normalizeStorageError(error);
		console.error('[NotificationsStore] Storage operation failed', error);
	}

	function clearStorageError() {
		storageError = null;
	}

	function toStoredRecord(notification: Notification, deviceId: string): StoredNotificationRecord {
		return {
			id: notification.id,
			deviceId,
			title: notification.title,
			body: notification.body,
			topicId: notification.topicId ?? undefined,
			topic: notification.topic,
			sentAt: notification.sentAt.toISOString(),
			priority: notification.priority,
			attachment: notification.attachment
		};
	}

	function saveReadIds() {
		if (!browser || !activeDeviceId) return;
		const deviceId = activeDeviceId;
		const readArray = notifications.map((n) => n.id).filter((id) => !unreadIds.has(id));
		void notificationsRepository.saveReadIds(deviceId, readArray).catch(recordStorageError);
	}

	function persistNotification(notification: Notification) {
		if (!browser || !activeDeviceId) return;
		const deviceId = activeDeviceId;
		void notificationsRepository
			.save(toStoredRecord(notification, deviceId))
			.then(clearStorageError)
			.catch(recordStorageError);
		saveReadIds();
	}

	function saveSyncCursor() {
		if (!browser || !activeDeviceId) return;
		const deviceId = activeDeviceId;
		void notificationsRepository
			.saveSyncCursor(deviceId, syncCursor)
			.then(clearStorageError)
			.catch(recordStorageError);
	}

	function activateDevice(deviceId: string) {
		if (activeDeviceId === deviceId) return;
		activeDeviceId = deviceId;
		notifications = [];
		unreadIds.clear();
		syncCursor = null;
		storageError = null;
	}

	function deactivateDevice() {
		activeDeviceId = null;
		notifications = [];
		unreadIds.clear();
		syncCursor = null;
		storageError = null;
	}

	async function loadFromIndexedDB(): Promise<void> {
		if (!browser || !activeDeviceId) return;
		const deviceId = activeDeviceId;
		try {
			const [records, state] = await Promise.all([
				notificationsRepository.listByDevice(deviceId),
				notificationsRepository.getState(deviceId)
			]);
			if (activeDeviceId !== deviceId) return;

			const idbNotifications: Notification[] = [];
			for (const record of records) {
				const parsed = parseStoredNotification(record);
				if (!parsed) {
					console.error('[NotificationsStore] Skipped malformed IndexedDB notification record', {
						id: record.id
					});
					continue;
				}
				idbNotifications.push(parsed);
			}

			// eslint-disable-next-line svelte/prefer-svelte-reactivity -- local dedupe map, not reactive state
			const byId = new Map<string, Notification>();
			for (const notification of [...notifications, ...idbNotifications]) {
				byId.set(notification.id, notification);
			}
			notifications = [...byId.values()].sort((a, b) => b.sentAt.getTime() - a.sentAt.getTime());

			const readSet = new Set(state.readIds);
			unreadIds.clear();
			for (const notification of notifications) {
				if (!readSet.has(notification.id)) {
					unreadIds.add(notification.id);
				}
			}
			syncCursor = state.syncCursor;
			clearStorageError();
		} catch (error) {
			recordStorageError(error);
		}
	}

	function add(
		title: string,
		body: string,
		topic: string | null = null,
		topicId: string | null = null,
		sentAt: string,
		attachment?: unknown,
		priority?: string,
		id?: string
	): boolean {
		if (!activeDeviceId) return false;
		if (!id) return false;
		if (notifications.some((n) => n.id === id)) return false;

		const DEFAULT_PRIORITY: NotificationPriority = 'normal';
		const parsedSentAt = new Date(sentAt);
		if (Number.isNaN(parsedSentAt.getTime())) {
			console.error('[NotificationsStore] Rejected notification with invalid sentAt');
			return false;
		}

		const notification: Notification = {
			id,
			title,
			body,
			topicId,
			topic,
			sentAt: parsedSentAt,
			priority: (priority as NotificationPriority) ?? DEFAULT_PRIORITY,
			attachment: attachment as Notification['attachment']
		};
		notifications = [notification, ...notifications];
		unreadIds.add(notification.id);
		persistNotification(notification);
		return true;
	}

	function latestNotificationId(): string | undefined {
		return notifications[0]?.id;
	}

	function remove(id: string) {
		if (!activeDeviceId) return;
		const deviceId = activeDeviceId;
		notifications = notifications.filter((n) => n.id !== id);
		unreadIds.delete(id);
		void notificationsRepository
			.deleteMany([id])
			.then(() => notificationsRepository.saveReadIds(deviceId, readIdsForCurrentNotifications()))
			.then(clearStorageError)
			.catch(recordStorageError);
	}

	/** Removes multiple notifications in one pass. */
	function removeMany(ids: string[]) {
		if (!activeDeviceId) return;
		if (ids.length === 0) return;
		const deviceId = activeDeviceId;

		const idSet = new Set(ids);
		notifications = notifications.filter((notification) => !idSet.has(notification.id));
		for (const id of ids) {
			unreadIds.delete(id);
		}
		void notificationsRepository
			.deleteMany(ids)
			.then(() => notificationsRepository.saveReadIds(deviceId, readIdsForCurrentNotifications()))
			.then(clearStorageError)
			.catch(recordStorageError);
	}

	function clearAll() {
		if (!activeDeviceId) return;
		const deviceId = activeDeviceId;
		notifications = [];
		unreadIds.clear();
		syncCursor = null;
		void Promise.all([
			notificationsRepository.clearByDevice(deviceId),
			notificationsRepository.clearState(deviceId)
		])
			.then(clearStorageError)
			.catch(recordStorageError);
	}

	function readIdsForCurrentNotifications(): string[] {
		return notifications.map((n) => n.id).filter((id) => !unreadIds.has(id));
	}

	function markAsRead(id: string) {
		if (!activeDeviceId) return;
		unreadIds.delete(id);
		saveReadIds();
	}

	function markAsUnread(id: string) {
		if (!activeDeviceId) return;
		unreadIds.add(id);
		saveReadIds();
	}

	/** Marks the provided notifications as read in one pass. */
	function markManyAsRead(ids: string[]) {
		if (!activeDeviceId) return;
		for (const id of ids) {
			unreadIds.delete(id);
		}
		saveReadIds();
	}

	/** Marks the provided notifications as unread in one pass. */
	function markManyAsUnread(ids: string[]) {
		if (!activeDeviceId) return;
		for (const id of ids) {
			unreadIds.add(id);
		}
		saveReadIds();
	}

	return {
		get activeDeviceId() {
			return activeDeviceId;
		},
		get list() {
			return notifications;
		},
		get unreadIds() {
			return unreadIds;
		},
		get unreadCount() {
			return unreadIds.size;
		},
		get count() {
			return notifications.length;
		},
		get isEmpty() {
			return notifications.length === 0;
		},
		get latestNotificationId() {
			return latestNotificationId();
		},
		get syncCursor() {
			return syncCursor;
		},
		set syncCursor(value: string | null) {
			syncCursor = value;
			saveSyncCursor();
		},
		get storageError() {
			return storageError;
		},
		get topicSummaries() {
			return computeTopicSummaries(notifications, unreadIds);
		},
		add,
		remove,
		removeMany,
		clearAll,
		markAsRead,
		markAsUnread,
		markManyAsRead,
		markManyAsUnread,
		activateDevice,
		deactivateDevice,
		loadFromIndexedDB
	};
}

export const notificationsStore = createNotificationsStore();

// Utility to group notifications by day (pure function, not reactive)
export function groupByDay(notificationsList: Notification[]): {
	groups: Map<string, Notification[]>;
	orderedLabels: string[];
} {
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- pure function, not reactive state
	const groups = new Map<string, Notification[]>();

	for (const n of notificationsList) {
		const label = getDayLabel(n.sentAt);
		const existing = groups.get(label);
		if (existing) {
			existing.push(n);
		} else {
			groups.set(label, [n]);
		}
	}

	// Order: Today, Yesterday, then other dates
	const priority = ['Today', 'Yesterday'];
	const orderedLabels = [...groups.keys()].sort((a, b) => {
		const aIdx = priority.indexOf(a);
		const bIdx = priority.indexOf(b);
		if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx;
		if (aIdx !== -1) return -1;
		if (bIdx !== -1) return 1;
		return 0;
	});

	return { groups, orderedLabels };
}

function getDayLabel(date: Date): string {
	const now = new Date();
	const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
	const target = new Date(date.getFullYear(), date.getMonth(), date.getDate());
	const diffDays = Math.floor((today.getTime() - target.getTime()) / 86400000);

	if (diffDays === 0) return 'Today';
	if (diffDays === 1) return 'Yesterday';

	return target.toLocaleDateString('en-US', {
		month: 'short',
		day: 'numeric',
		year: target.getFullYear() !== today.getFullYear() ? 'numeric' : undefined
	});
}

export function formatTime(date: Date): string {
	return date.toLocaleTimeString('en-US', {
		hour: '2-digit',
		minute: '2-digit',
		hour12: false
	});
}

export function formatRelativeTime(date: Date): string {
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- pure utility, not reactive state
	const now = new Date();
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- pure utility, not reactive state
	const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- pure utility, not reactive state
	const target = new Date(date.getFullYear(), date.getMonth(), date.getDate());
	const diffDays = Math.floor((today.getTime() - target.getTime()) / 86400000);

	// Not today: show absolute time for clarity across midnight
	if (diffDays !== 0) {
		return formatTime(date);
	}

	const diffMs = Date.now() - date.getTime();
	const diffSeconds = Math.floor(diffMs / 1000);

	if (diffSeconds < 10) {
		return 'now';
	}

	if (diffSeconds < 60) {
		return `${diffSeconds}s`;
	}

	const diffMinutes = Math.floor(diffMs / 60000);
	if (diffMinutes < 60) {
		return `${diffMinutes}m`;
	}

	return formatTime(date);
}
