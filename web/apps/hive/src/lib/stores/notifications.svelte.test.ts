import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NOTIFICATIONS_STORE, openHiveDB } from '$lib/services/hive-db';
import { notificationsRepository } from '$lib/services/notifications-repository';

async function loadStore() {
	vi.resetModules();
	return import('./notifications.svelte');
}

async function clearNotificationsStore(): Promise<void> {
	const db = await openHiveDB();
	return new Promise((resolve, reject) => {
		const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
		const request = tx.objectStore(NOTIFICATIONS_STORE).clear();
		request.onsuccess = () => resolve();
		request.onerror = () => reject(new Error(request.error?.message ?? 'Clear failed'));
	});
}

describe('notificationsStore', () => {
	beforeEach(async () => {
		localStorage.clear();
		await clearNotificationsStore();
	});

	it('does not hydrate localStorage at module load', async () => {
		localStorage.setItem(
			'notifications:dev-a',
			JSON.stringify([
				{
					id: 'n-a',
					title: 'Door',
					body: 'Opened',
					sentAt: '2026-04-20T09:00:00.000Z'
				}
			])
		);

		const { notificationsStore } = await loadStore();

		expect(notificationsStore.activeDeviceId).toBeNull();
		expect(notificationsStore.list).toEqual([]);
	});

	it('activates and loads only the selected device history', async () => {
		localStorage.setItem(
			'notifications:dev-a',
			JSON.stringify([
				{
					id: 'n-a',
					title: 'Door',
					body: 'Opened',
					sentAt: '2026-04-20T09:00:00.000Z'
				}
			])
		);

		const { notificationsStore } = await loadStore();

		notificationsStore.activateDevice('dev-a');
		expect(notificationsStore.list.map((notification) => notification.id)).toEqual(['n-a']);

		// Re-seed dev-b because activating dev-a garbage-collected stale entries.
		localStorage.setItem(
			'notifications:dev-b',
			JSON.stringify([
				{
					id: 'n-b',
					title: 'Window',
					body: 'Closed',
					sentAt: '2026-04-20T10:00:00.000Z'
				}
			])
		);
		notificationsStore.activateDevice('dev-b');
		expect(notificationsStore.list.map((notification) => notification.id)).toEqual(['n-b']);
	});

	it('persists new notifications only under the active device key', async () => {
		const { notificationsStore } = await loadStore();

		notificationsStore.add(
			'Ignored',
			'No device',
			null,
			null,
			'2026-04-20T08:00:00.000Z',
			undefined,
			'normal',
			'n-ignored'
		);
		expect(localStorage.getItem('notifications')).toBeNull();

		notificationsStore.activateDevice('dev-a');
		notificationsStore.add(
			'Door',
			'Opened',
			null,
			null,
			'2026-04-20T09:00:00.000Z',
			undefined,
			'high',
			'n-a'
		);

		expect(localStorage.getItem('notifications')).toBeNull();
		expect(localStorage.getItem('notifications:dev-b')).toBeNull();
		expect(JSON.parse(localStorage.getItem('notifications:dev-a') ?? '[]')).toEqual([
			expect.objectContaining({ id: 'n-a', title: 'Door' })
		]);
	});

	it('clears in-memory state on deactivate without touching localStorage', async () => {
		const { notificationsStore } = await loadStore();

		notificationsStore.activateDevice('dev-a');
		notificationsStore.add(
			'Door',
			'Opened',
			null,
			null,
			'2026-04-20T09:00:00.000Z',
			undefined,
			'normal',
			'n-a'
		);
		expect(notificationsStore.list).toHaveLength(1);
		expect(notificationsStore.unreadCount).toBe(1);

		notificationsStore.deactivateDevice();

		expect(notificationsStore.activeDeviceId).toBeNull();
		expect(notificationsStore.list).toEqual([]);
		expect(notificationsStore.unreadCount).toBe(0);
		// localStorage retains the per-device payload so a later activate restores history.
		expect(JSON.parse(localStorage.getItem('notifications:dev-a') ?? '[]')).toHaveLength(1);
	});

	it('drops unread state when switching the active device', async () => {
		const { notificationsStore } = await loadStore();

		notificationsStore.activateDevice('dev-a');
		notificationsStore.add(
			'A',
			'',
			null,
			null,
			'2026-04-20T09:00:00.000Z',
			undefined,
			'normal',
			'n-a'
		);
		expect(notificationsStore.unreadCount).toBe(1);

		notificationsStore.activateDevice('dev-b');

		expect(notificationsStore.list).toEqual([]);
		expect(notificationsStore.unreadCount).toBe(0);
	});

	it('ignores mutating operations while no device is active', async () => {
		const { notificationsStore } = await loadStore();

		notificationsStore.add(
			'Ignored',
			'',
			null,
			null,
			'2026-04-20T09:00:00.000Z',
			undefined,
			'normal',
			'n-orphan'
		);
		notificationsStore.markAsRead('n-orphan');
		notificationsStore.markAsUnread('n-orphan');
		notificationsStore.remove('n-orphan');
		notificationsStore.removeMany(['n-orphan']);
		notificationsStore.clearAll();

		expect(notificationsStore.list).toEqual([]);
		expect(notificationsStore.unreadCount).toBe(0);
		expect(localStorage.length).toBe(0);
	});

	it('removes stale localStorage entries for other devices on activate', async () => {
		localStorage.setItem(
			'notifications:dev-a',
			JSON.stringify([
				{
					id: 'n-a',
					title: 'Door',
					body: 'Opened',
					sentAt: '2026-04-20T09:00:00.000Z'
				}
			])
		);
		localStorage.setItem('notifications_read_ids:dev-a', JSON.stringify(['n-a']));
		localStorage.setItem(
			'notifications:dev-b',
			JSON.stringify([
				{
					id: 'n-b',
					title: 'Window',
					body: 'Closed',
					sentAt: '2026-04-20T10:00:00.000Z'
				}
			])
		);
		localStorage.setItem('notifications_read_ids:dev-b', JSON.stringify(['n-b']));

		const { notificationsStore } = await loadStore();
		notificationsStore.activateDevice('dev-a');

		expect(localStorage.getItem('notifications:dev-a')).not.toBeNull();
		expect(localStorage.getItem('notifications_read_ids:dev-a')).not.toBeNull();
		expect(localStorage.getItem('notifications:dev-b')).toBeNull();
		expect(localStorage.getItem('notifications_read_ids:dev-b')).toBeNull();
	});

	it('removes legacy unscoped localStorage keys on activate', async () => {
		localStorage.setItem(
			'notifications',
			JSON.stringify([
				{
					id: 'n-legacy',
					title: 'Old',
					body: 'Unscoped',
					sentAt: '2026-04-20T07:00:00.000Z'
				}
			])
		);
		localStorage.setItem('notifications_read_ids', JSON.stringify(['n-legacy']));

		const { notificationsStore } = await loadStore();
		notificationsStore.activateDevice('dev-a');

		expect(localStorage.getItem('notifications')).toBeNull();
		expect(localStorage.getItem('notifications_read_ids')).toBeNull();
	});

	it('does not reset state when activating the already-active device', async () => {
		const { notificationsStore } = await loadStore();

		notificationsStore.activateDevice('dev-a');
		notificationsStore.add(
			'Door',
			'Opened',
			null,
			null,
			'2026-04-20T09:00:00.000Z',
			undefined,
			'normal',
			'n-a'
		);
		expect(notificationsStore.list).toHaveLength(1);
		expect(notificationsStore.unreadCount).toBe(1);

		notificationsStore.markAsRead('n-a');
		expect(notificationsStore.unreadCount).toBe(0);

		// Re-activating the same device must not reload from localStorage.
		notificationsStore.activateDevice('dev-a');
		expect(notificationsStore.list).toHaveLength(1);
		expect(notificationsStore.unreadCount).toBe(0);
	});

	it('imports and deletes only IndexedDB records for the active device', async () => {
		await notificationsRepository.save({
			id: 'n-a',
			deviceId: 'dev-a',
			title: 'Door',
			body: 'Opened',
			topic: 'alerts',
			sentAt: '2026-04-20T09:00:00.000Z'
		});
		await notificationsRepository.save({
			id: 'n-b',
			deviceId: 'dev-b',
			title: 'Window',
			body: 'Closed',
			topic: 'alerts',
			sentAt: '2026-04-20T10:00:00.000Z'
		});

		const { notificationsStore } = await loadStore();

		await notificationsStore.loadFromIndexedDB();
		expect(notificationsStore.list).toEqual([]);

		notificationsStore.activateDevice('dev-a');
		await notificationsStore.loadFromIndexedDB();

		expect(notificationsStore.list.map((notification) => notification.id)).toEqual(['n-a']);

		// Verify record was deleted from IndexedDB after import.
		const remaining = await notificationsRepository.listByDevice('dev-a');
		expect(remaining).toHaveLength(0);
	});
});
