import { browser } from '$app/environment';
import { notificationsRepository } from '$lib/services/notifications-repository';
import { SvelteSet } from 'svelte/reactivity';
import type { Notification, NotificationPriority } from '@beebuzz/shared/types';

const STORAGE_KEY_PREFIX = 'notifications:';
const READ_IDS_KEY_PREFIX = 'notifications_read_ids:';
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
			topic: (r.topic as string | null) ?? null,
			sentAt,
			priority: (r.priority as NotificationPriority) ?? 'normal',
			attachment: r.attachment as Notification['attachment']
		};
	}

	/** Persists read IDs to localStorage. */
	function saveReadIds() {
		if (!browser || !activeDeviceId) return;
		const readArray = notifications.map((n) => n.id).filter((id) => !unreadIds.has(id));
		localStorage.setItem(`${READ_IDS_KEY_PREFIX}${activeDeviceId}`, JSON.stringify(readArray));
	}

	function save() {
		if (!browser || !activeDeviceId) return;
		const toSave = notifications.map((n) => ({
			id: n.id,
			title: n.title,
			body: n.body,
			topicId: n.topicId,
			topic: n.topic,
			sentAt: n.sentAt.toISOString(),
			priority: n.priority,
			attachment: n.attachment
		}));
		localStorage.setItem(`${STORAGE_KEY_PREFIX}${activeDeviceId}`, JSON.stringify(toSave));
		saveReadIds();
	}

	function loadForActiveDevice() {
		if (!browser || !activeDeviceId) return;
		const saved = localStorage.getItem(`${STORAGE_KEY_PREFIX}${activeDeviceId}`);
		notifications = [];
		unreadIds.clear();
		if (saved) {
			try {
				const parsed: unknown = JSON.parse(saved);
				if (!Array.isArray(parsed)) {
					notifications = [];
					return;
				}
				notifications = parsed
					.map((n) => parseStoredNotification(n))
					.filter((n): n is Notification => n !== null);

				// Restore read state: only mark as unread those not in the persisted read list
				unreadIds.clear();
				// eslint-disable-next-line svelte/prefer-svelte-reactivity -- local temp set inside load, not reactive state
				const readSet = new Set<string>();
				const savedReadIds = localStorage.getItem(`${READ_IDS_KEY_PREFIX}${activeDeviceId}`);
				if (savedReadIds) {
					try {
						const readArray: unknown = JSON.parse(savedReadIds);
						if (Array.isArray(readArray)) {
							readArray.forEach((id) => readSet.add(id as string));
						}
					} catch {
						// ignore malformed data
					}
				}
				for (const n of notifications) {
					if (!readSet.has(n.id)) {
						unreadIds.add(n.id);
					}
				}
			} catch {
				notifications = [];
				unreadIds.clear();
			}
		}
	}

	/** Removes localStorage entries belonging to devices other than the active one. */
	function removeStaleLocalStorage() {
		if (!browser || !activeDeviceId) return;
		const activeStorageKey = `${STORAGE_KEY_PREFIX}${activeDeviceId}`;
		const activeReadKey = `${READ_IDS_KEY_PREFIX}${activeDeviceId}`;
		const keysToRemove: string[] = [];
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i);
			if (!key) continue;
			if (
				(key.startsWith(STORAGE_KEY_PREFIX) || key.startsWith(READ_IDS_KEY_PREFIX)) &&
				key !== activeStorageKey &&
				key !== activeReadKey
			) {
				keysToRemove.push(key);
			}
		}
		for (const key of keysToRemove) {
			localStorage.removeItem(key);
		}
		// One-shot cleanup of legacy unscoped keys (pre-device-scoping).
		localStorage.removeItem('notifications');
		localStorage.removeItem('notifications_read_ids');
	}

	function activateDevice(deviceId: string) {
		if (activeDeviceId === deviceId) return;
		activeDeviceId = deviceId;
		removeStaleLocalStorage();
		loadForActiveDevice();
	}

	function deactivateDevice() {
		activeDeviceId = null;
		notifications = [];
		unreadIds.clear();
	}

	async function loadFromIndexedDB(): Promise<void> {
		if (!browser || !activeDeviceId) return;
		const deviceId = activeDeviceId;

		return new Promise((resolve) => {
			try {
				void notificationsRepository
					.listByDevice(deviceId)
					.then((records) => {
						if (activeDeviceId !== deviceId) {
							resolve();
							return;
						}

						const importedIds: string[] = [];
						const idbNotifications: Notification[] = [];

						for (const record of records) {
							const parsed = parseStoredNotification(record);
							if (!parsed) {
								console.error(
									'[NotificationsStore] Skipped malformed IndexedDB notification record',
									{ id: record.id }
								);
								continue;
							}

							idbNotifications.push(parsed);
							importedIds.push(parsed.id);
						}

						// Deduplicate: the SW always persists to IndexedDB, so if the app
						// was visible it may have already received the same notification
						// via postMessage. Match on id to skip dupes.
						const existingIds = new Set(notifications.map((n) => n.id));
						const newNotifications = idbNotifications.filter((n) => !existingIds.has(n.id));

						if (newNotifications.length) {
							notifications = [...newNotifications, ...notifications].sort(
								(a, b) => b.sentAt.getTime() - a.sentAt.getTime()
							);
							newNotifications.forEach((n) => unreadIds.add(n.id));
							save();
						}

						// Delete only records we successfully parsed so malformed
						// entries stay available for inspection instead of being lost.
						void notificationsRepository.deleteMany(importedIds).finally(() => resolve());
					})
					.catch(() => resolve());
			} catch {
				resolve();
			}
		});
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
	) {
		if (!activeDeviceId) return;
		if (!id) return;
		if (notifications.some((n) => n.id === id)) return;

		const DEFAULT_PRIORITY: NotificationPriority = 'normal';
		const parsedSentAt = new Date(sentAt);
		if (Number.isNaN(parsedSentAt.getTime())) {
			console.error('[NotificationsStore] Rejected notification with invalid sentAt');
			return;
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
		save();
	}

	function remove(id: string) {
		if (!activeDeviceId) return;
		notifications = notifications.filter((n) => n.id !== id);
		unreadIds.delete(id);
		save();
	}

	/** Removes multiple notifications in one pass. */
	function removeMany(ids: string[]) {
		if (!activeDeviceId) return;
		if (ids.length === 0) return;

		const idSet = new Set(ids);
		notifications = notifications.filter((notification) => !idSet.has(notification.id));
		for (const id of ids) {
			unreadIds.delete(id);
		}
		save();
	}

	function clearAll() {
		if (!activeDeviceId) return;
		notifications = [];
		unreadIds.clear();
		save();
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
