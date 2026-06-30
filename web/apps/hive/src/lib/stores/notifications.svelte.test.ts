import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NOTIFICATION_STATE_STORE, NOTIFICATIONS_STORE, openHiveDB } from '$lib/services/hive-db';
import { notificationsRepository, StorageQuotaError } from '$lib/services/notifications-repository';

async function loadStore() {
	vi.resetModules();
	return import('./notifications.svelte');
}

async function clearNotificationStores(): Promise<void> {
	const db = await openHiveDB();
	return new Promise((resolve, reject) => {
		const tx = db.transaction([NOTIFICATIONS_STORE, NOTIFICATION_STATE_STORE], 'readwrite');
		tx.objectStore(NOTIFICATIONS_STORE).clear();
		tx.objectStore(NOTIFICATION_STATE_STORE).clear();
		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(new Error(tx.error?.message ?? 'Clear failed'));
	});
}

async function waitForAssertion(assertion: () => void | Promise<void>): Promise<void> {
	let lastError: unknown;
	for (let i = 0; i < 10; i++) {
		try {
			await assertion();
			return;
		} catch (error) {
			lastError = error;
			await new Promise((resolve) => setTimeout(resolve, 0));
		}
	}
	throw lastError;
}

// --- Fixture helpers ---

type StoredNotificationOverrides = {
	id?: string;
	deviceId?: string;
	title?: string;
	body?: string;
	topic?: string | null;
	sentAt?: string;
	topicId?: string;
	priority?: string;
	attachment?: unknown;
};

type NotificationOverrides = {
	title?: string;
	body?: string;
	topic?: string | null;
	topicId?: string | null;
	sentAt?: string;
	priority?: string;
	id?: string;
};

type NotificationsStoreWithAdd = Pick<
	typeof import('./notifications.svelte').notificationsStore,
	'add'
>;

function seedNotification(overrides: StoredNotificationOverrides = {}): Promise<void> {
	return notificationsRepository.save({
		id: 'n-a',
		deviceId: 'dev-a',
		title: 'Door',
		body: 'Opened',
		topic: 'alerts',
		sentAt: '2026-04-20T09:00:00.000Z',
		...overrides
	});
}

function addNotification(
	store: NotificationsStoreWithAdd,
	overrides: NotificationOverrides = {}
): boolean {
	return store.add(
		overrides.title ?? 'Door',
		overrides.body ?? 'Opened',
		overrides.topic ?? null,
		overrides.topicId ?? null,
		overrides.sentAt ?? '2026-04-20T09:00:00.000Z',
		undefined,
		overrides.priority ?? 'normal',
		overrides.id ?? 'n-a'
	);
}

async function createActiveStore(deviceId = 'dev-a') {
	const { notificationsStore } = await loadStore();
	notificationsStore.activateDevice(deviceId);
	return notificationsStore;
}

describe('notificationsStore', () => {
	beforeEach(async () => {
		vi.restoreAllMocks();
		await clearNotificationStores();
	});

	it('does not hydrate IndexedDB at module load', async () => {
		await seedNotification();

		const { notificationsStore } = await loadStore();

		expect(notificationsStore.activeDeviceId).toBeNull();
		expect(notificationsStore.list).toEqual([]);
	});

	it('activates and loads only the selected device inbox from IndexedDB', async () => {
		await seedNotification();
		await seedNotification({
			id: 'n-b',
			deviceId: 'dev-b',
			title: 'Window',
			body: 'Closed',
			sentAt: '2026-04-20T10:00:00.000Z'
		});

		const { notificationsStore } = await loadStore();

		notificationsStore.activateDevice('dev-a');
		await notificationsStore.loadFromIndexedDB();
		expect(notificationsStore.list.map((notification) => notification.id)).toEqual(['n-a']);

		notificationsStore.activateDevice('dev-b');
		await notificationsStore.loadFromIndexedDB();
		expect(notificationsStore.list.map((notification) => notification.id)).toEqual(['n-b']);
	});

	it('persists new notifications only for the active device in IndexedDB', async () => {
		const setItemSpy = vi.spyOn(Storage.prototype, 'setItem');
		const { notificationsStore } = await loadStore();

		expect(
			notificationsStore.add(
				'Ignored',
				'No device',
				null,
				null,
				'2026-04-20T08:00:00.000Z',
				undefined,
				'normal',
				'n-ignored'
			)
		).toBe(false);

		notificationsStore.activateDevice('dev-a');
		expect(addNotification(notificationsStore, { priority: 'high' })).toBe(true);

		await waitForAssertion(async () => {
			const records = await notificationsRepository.listByDevice('dev-a');
			expect(records).toEqual([expect.objectContaining({ id: 'n-a', title: 'Door' })]);
		});
		expect(await notificationsRepository.listByDevice('dev-b')).toEqual([]);
		expect(setItemSpy).not.toHaveBeenCalled();
	});

	it('reports whether add inserted a new notification', async () => {
		const { notificationsStore } = await loadStore();

		expect(
			notificationsStore.add(
				'Ignored',
				'No device',
				null,
				null,
				'2026-04-20T08:00:00.000Z',
				undefined,
				'normal',
				'n-ignored'
			)
		).toBe(false);

		notificationsStore.activateDevice('dev-a');
		expect(addNotification(notificationsStore)).toBe(true);
		expect(
			addNotification(notificationsStore, {
				body: 'Opened again',
				sentAt: '2026-04-20T09:01:00.000Z'
			})
		).toBe(false);
		const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
		expect(
			addNotification(notificationsStore, {
				title: 'Broken',
				body: 'Invalid date',
				sentAt: 'not-a-date',
				id: 'n-broken'
			})
		).toBe(false);
		consoleSpy.mockRestore();
		expect(notificationsStore.list.map((notification) => notification.id)).toEqual(['n-a']);
	});

	it('clears in-memory state on deactivate without deleting IndexedDB history', async () => {
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore);
		await waitForAssertion(async () => {
			expect(await notificationsRepository.listByDevice('dev-a')).toHaveLength(1);
		});

		notificationsStore.deactivateDevice();

		expect(notificationsStore.activeDeviceId).toBeNull();
		expect(notificationsStore.list).toEqual([]);
		expect(notificationsStore.unreadCount).toBe(0);
		expect(await notificationsRepository.listByDevice('dev-a')).toHaveLength(1);
	});

	it('loads stored topic as null for empty string or null values', async () => {
		await seedNotification({ topic: '' });
		await seedNotification({
			id: 'n-b',
			title: 'Window',
			body: 'Closed',
			topic: null,
			sentAt: '2026-04-20T10:00:00.000Z'
		});

		const { notificationsStore } = await loadStore();
		notificationsStore.activateDevice('dev-a');
		await notificationsStore.loadFromIndexedDB();

		expect(notificationsStore.list.find((n) => n.id === 'n-a')?.topic).toBeNull();
		expect(notificationsStore.list.find((n) => n.id === 'n-b')?.topic).toBeNull();
	});

	it('loads persisted read state from IndexedDB for the active device', async () => {
		await seedNotification();
		await notificationsRepository.saveReadIds('dev-a', ['n-a']);

		const { notificationsStore } = await loadStore();
		notificationsStore.activateDevice('dev-a');
		await notificationsStore.loadFromIndexedDB();

		expect(notificationsStore.unreadCount).toBe(0);
	});

	it('persists read state changes to IndexedDB', async () => {
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore);

		notificationsStore.markAsRead('n-a');
		await waitForAssertion(async () => {
			expect(await notificationsRepository.getState('dev-a')).toEqual(
				expect.objectContaining({ readIds: ['n-a'] })
			);
		});

		notificationsStore.markAsUnread('n-a');
		await waitForAssertion(async () => {
			expect(await notificationsRepository.getState('dev-a')).toEqual(
				expect.objectContaining({ readIds: [] })
			);
		});
	});

	it('persists sync cursor to IndexedDB', async () => {
		const { notificationsStore } = await loadStore();
		notificationsStore.activateDevice('dev-a');

		notificationsStore.syncCursor = 'n-a';

		await waitForAssertion(async () => {
			expect(await notificationsRepository.getState('dev-a')).toEqual(
				expect.objectContaining({ syncCursor: 'n-a' })
			);
		});
	});

	it('persists remove and clearAll to IndexedDB for the active device only', async () => {
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore, { title: 'A', body: '', id: 'n-a' });
		addNotification(notificationsStore, {
			title: 'B',
			body: '',
			sentAt: '2026-04-20T09:01:00.000Z',
			id: 'n-b'
		});
		await seedNotification({
			id: 'n-foreign',
			deviceId: 'dev-b',
			title: 'Foreign',
			body: '',
			topic: '',
			sentAt: '2026-04-20T09:02:00.000Z'
		});

		notificationsStore.remove('n-a');
		await waitForAssertion(async () => {
			expect(
				(await notificationsRepository.listByDevice('dev-a')).map((record) => record.id)
			).toEqual(['n-b']);
		});

		notificationsStore.clearAll();
		await waitForAssertion(async () => {
			expect(await notificationsRepository.listByDevice('dev-a')).toEqual([]);
		});
		expect(await notificationsRepository.listByDevice('dev-b')).toHaveLength(1);
	});

	it('drops unread state when switching the active device', async () => {
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore, { title: 'A', body: '', id: 'n-a' });
		expect(notificationsStore.unreadCount).toBe(1);

		notificationsStore.activateDevice('dev-b');

		expect(notificationsStore.list).toEqual([]);
		expect(notificationsStore.unreadCount).toBe(0);
	});

	it('ignores mutating operations while no device is active and leaves IndexedDB unchanged', async () => {
		const { notificationsStore } = await loadStore();

		addNotification(notificationsStore, { title: 'Ignored', body: '', id: 'n-orphan' });
		notificationsStore.markAsRead('n-orphan');
		notificationsStore.markAsUnread('n-orphan');
		notificationsStore.remove('n-orphan');
		notificationsStore.removeMany(['n-orphan']);
		notificationsStore.clearAll();

		expect(notificationsStore.list).toEqual([]);
		expect(notificationsStore.unreadCount).toBe(0);
		expect(await notificationsRepository.listByDevice('dev-a')).toEqual([]);
	});

	it('does not reset state when activating the already-active device', async () => {
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore);
		expect(notificationsStore.list).toHaveLength(1);
		expect(notificationsStore.unreadCount).toBe(1);

		notificationsStore.markAsRead('n-a');
		expect(notificationsStore.unreadCount).toBe(0);

		notificationsStore.activateDevice('dev-a');
		expect(notificationsStore.list).toHaveLength(1);
		expect(notificationsStore.unreadCount).toBe(0);
	});

	it('loads IndexedDB records without deleting them', async () => {
		await seedNotification();

		const { notificationsStore } = await loadStore();

		await notificationsStore.loadFromIndexedDB();
		expect(notificationsStore.list).toEqual([]);

		notificationsStore.activateDevice('dev-a');
		await notificationsStore.loadFromIndexedDB();

		expect(notificationsStore.list.map((notification) => notification.id)).toEqual(['n-a']);
		expect(await notificationsRepository.listByDevice('dev-a')).toHaveLength(1);
	});

	it('keeps in-memory notification when IndexedDB persistence hits quota', async () => {
		const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.doMock('$lib/services/notifications-repository', async () => {
			const actual = await vi.importActual<typeof import('$lib/services/notifications-repository')>(
				'$lib/services/notifications-repository'
			);
			return {
				...actual,
				notificationsRepository: {
					...actual.notificationsRepository,
					save: vi.fn().mockRejectedValueOnce(new StorageQuotaError())
				}
			};
		});
		const notificationsStore = await createActiveStore();

		expect(addNotification(notificationsStore)).toBe(true);

		await waitForAssertion(() => {
			expect(notificationsStore.storageError).toContain('Browser storage is full');
		});
		expect(notificationsStore.list.map((notification) => notification.id)).toEqual(['n-a']);
		consoleSpy.mockRestore();
		vi.doUnmock('$lib/services/notifications-repository');
	});

	it('merges IndexedDB records into memory without duplicating existing notifications', async () => {
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore);
		await waitForAssertion(async () => {
			expect(await notificationsRepository.listByDevice('dev-a')).toHaveLength(1);
		});

		await seedNotification();
		await seedNotification({
			id: 'n-b',
			title: 'Window',
			body: 'Closed',
			sentAt: '2026-04-20T10:00:00.000Z'
		});

		await notificationsStore.loadFromIndexedDB();

		expect(notificationsStore.list.map((n) => n.id)).toEqual(['n-b', 'n-a']);
		expect(notificationsStore.list.filter((n) => n.id === 'n-a')).toHaveLength(1);
		const stored = await notificationsRepository.listByDevice('dev-a');
		expect(stored.map((r) => r.id).sort()).toEqual(['n-a', 'n-b']);
	});

	it('persists bulk read state changes to IndexedDB', async () => {
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore, { title: 'A', body: '', id: 'n-a' });
		addNotification(notificationsStore, {
			title: 'B',
			body: '',
			sentAt: '2026-04-20T09:01:00.000Z',
			id: 'n-b'
		});

		await waitForAssertion(async () => {
			const state = await notificationsRepository.getState('dev-a');
			expect(state.readIds).toEqual([]);
		});

		notificationsStore.markManyAsRead(['n-a', 'n-b']);

		await waitForAssertion(async () => {
			const state = await notificationsRepository.getState('dev-a');
			expect(state.readIds.sort()).toEqual(['n-a', 'n-b']);
		});

		notificationsStore.markManyAsUnread(['n-a']);

		await waitForAssertion(async () => {
			const state = await notificationsRepository.getState('dev-a');
			expect(state.readIds).toEqual(['n-b']);
		});
	});

	it('persists removeMany to IndexedDB for the active device only', async () => {
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore, { title: 'A', body: '', id: 'n-a' });
		addNotification(notificationsStore, {
			title: 'B',
			body: '',
			sentAt: '2026-04-20T09:01:00.000Z',
			id: 'n-b'
		});
		addNotification(notificationsStore, {
			title: 'C',
			body: '',
			sentAt: '2026-04-20T09:02:00.000Z',
			id: 'n-c'
		});
		await seedNotification({
			id: 'n-foreign',
			deviceId: 'dev-b',
			title: 'Foreign',
			body: '',
			topic: '',
			sentAt: '2026-04-20T09:03:00.000Z'
		});

		await waitForAssertion(async () => {
			expect(await notificationsRepository.listByDevice('dev-a')).toHaveLength(3);
		});

		notificationsStore.markAsRead('n-a');
		notificationsStore.markAsRead('n-c');
		await waitForAssertion(async () => {
			const state = await notificationsRepository.getState('dev-a');
			expect(state.readIds.sort()).toEqual(['n-a', 'n-c']);
		});

		notificationsStore.removeMany(['n-a', 'n-c']);

		await waitForAssertion(async () => {
			const devA = (await notificationsRepository.listByDevice('dev-a')).map((r) => r.id);
			expect(devA).toEqual(['n-b']);
		});

		const devB = (await notificationsRepository.listByDevice('dev-b')).map((r) => r.id);
		expect(devB).toEqual(['n-foreign']);

		await waitForAssertion(async () => {
			const state = await notificationsRepository.getState('dev-a');
			expect(state.readIds).toEqual([]);
		});
	});

	it('sets storageError when read-state persistence hits quota', async () => {
		const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.doMock('$lib/services/notifications-repository', async () => {
			const actual = await vi.importActual<typeof import('$lib/services/notifications-repository')>(
				'$lib/services/notifications-repository'
			);
			const mockSaveReadIds = vi
				.fn()
				.mockImplementationOnce((deviceId: string, readIds: string[]) =>
					actual.notificationsRepository.saveReadIds(deviceId, readIds)
				)
				.mockRejectedValueOnce(new StorageQuotaError());
			return {
				...actual,
				notificationsRepository: {
					...actual.notificationsRepository,
					saveReadIds: mockSaveReadIds
				}
			};
		});
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore);
		await waitForAssertion(async () => {
			expect(await notificationsRepository.listByDevice('dev-a')).toHaveLength(1);
		});

		notificationsStore.markAsRead('n-a');

		await waitForAssertion(() => {
			expect(notificationsStore.storageError).toContain('Browser storage is full');
		});
		expect(notificationsStore.list.map((n) => n.id)).toEqual(['n-a']);
		expect(notificationsStore.unreadCount).toBe(0);
		consoleSpy.mockRestore();
		vi.doUnmock('$lib/services/notifications-repository');
	});

	it('sets storageError when sync cursor persistence hits quota', async () => {
		const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.doMock('$lib/services/notifications-repository', async () => {
			const actual = await vi.importActual<typeof import('$lib/services/notifications-repository')>(
				'$lib/services/notifications-repository'
			);
			return {
				...actual,
				notificationsRepository: {
					...actual.notificationsRepository,
					saveSyncCursor: vi.fn().mockRejectedValueOnce(new StorageQuotaError())
				}
			};
		});
		const notificationsStore = await createActiveStore();

		notificationsStore.syncCursor = 'n-a';

		await waitForAssertion(() => {
			expect(notificationsStore.storageError).toContain('Browser storage is full');
		});
		expect(notificationsStore.syncCursor).toBe('n-a');
		consoleSpy.mockRestore();
		vi.doUnmock('$lib/services/notifications-repository');
	});

	it('sets storageError when clearAll persistence hits quota', async () => {
		const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.doMock('$lib/services/notifications-repository', async () => {
			const actual = await vi.importActual<typeof import('$lib/services/notifications-repository')>(
				'$lib/services/notifications-repository'
			);
			return {
				...actual,
				notificationsRepository: {
					...actual.notificationsRepository,
					clearByDevice: vi.fn().mockRejectedValueOnce(new StorageQuotaError())
				}
			};
		});
		const notificationsStore = await createActiveStore();

		addNotification(notificationsStore);
		await waitForAssertion(async () => {
			expect(await notificationsRepository.listByDevice('dev-a')).toHaveLength(1);
		});

		notificationsStore.clearAll();

		await waitForAssertion(() => {
			expect(notificationsStore.storageError).toContain('Browser storage is full');
		});
		expect(notificationsStore.list).toEqual([]);
		expect(notificationsStore.unreadCount).toBe(0);
		consoleSpy.mockRestore();
		vi.doUnmock('$lib/services/notifications-repository');
	});
});
