import { beforeEach, describe, expect, it } from 'vitest';
import { NOTIFICATION_STATE_STORE, NOTIFICATIONS_STORE, openHiveDB } from './hive-db';
import { notificationsRepository, type StoredNotificationRecord } from './notifications-repository';

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

describe('notificationsRepository', () => {
	beforeEach(async () => {
		await clearNotificationStores();
	});

	it('saves records with a device ID', async () => {
		const record: StoredNotificationRecord = {
			id: 'n-a',
			deviceId: 'dev-a',
			title: 'Door',
			body: 'Opened',
			topic: 'alerts',
			sentAt: '2026-04-20T09:00:00.000Z'
		};

		await notificationsRepository.save(record);
		const list = await notificationsRepository.listByDevice('dev-a');

		expect(list).toEqual([record]);
	});

	it('lists only records for the requested device', async () => {
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

		const list = await notificationsRepository.listByDevice('dev-a');

		expect(list).toHaveLength(1);
		expect(list[0].id).toBe('n-a');
	});

	it('deletes only the imported IDs requested by the caller', async () => {
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

		await notificationsRepository.deleteMany(['n-a']);

		const devAList = await notificationsRepository.listByDevice('dev-a');
		const devBList = await notificationsRepository.listByDevice('dev-b');

		expect(devAList).toHaveLength(0);
		expect(devBList).toHaveLength(1);
	});

	it('clears only records for the requested device', async () => {
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

		await notificationsRepository.clearByDevice('dev-a');

		expect(await notificationsRepository.listByDevice('dev-a')).toEqual([]);
		expect(await notificationsRepository.listByDevice('dev-b')).toHaveLength(1);
	});

	it('persists read IDs and sync cursor per device', async () => {
		await notificationsRepository.saveReadIds('dev-a', ['n-a', 'n-a', 'n-b']);
		await notificationsRepository.saveSyncCursor('dev-a', 'n-b');
		await notificationsRepository.saveSyncCursor('dev-b', 'n-z');

		expect(await notificationsRepository.getState('dev-a')).toEqual(
			expect.objectContaining({
				deviceId: 'dev-a',
				readIds: ['n-a', 'n-b'],
				syncCursor: 'n-b'
			})
		);
		expect(await notificationsRepository.getState('dev-b')).toEqual(
			expect.objectContaining({
				deviceId: 'dev-b',
				readIds: [],
				syncCursor: 'n-z'
			})
		);
	});

	it('preserves both fields when read IDs and sync cursor are written concurrently', async () => {
		await Promise.all([
			notificationsRepository.saveReadIds('dev-a', ['n-a', 'n-b']),
			notificationsRepository.saveSyncCursor('dev-a', 'n-b')
		]);

		expect(await notificationsRepository.getState('dev-a')).toEqual(
			expect.objectContaining({
				deviceId: 'dev-a',
				readIds: ['n-a', 'n-b'],
				syncCursor: 'n-b'
			})
		);
	});

	it('clears notification state for one device', async () => {
		await notificationsRepository.saveReadIds('dev-a', ['n-a']);
		await notificationsRepository.saveSyncCursor('dev-a', 'n-a');

		await notificationsRepository.clearState('dev-a');

		expect(await notificationsRepository.getState('dev-a')).toEqual(
			expect.objectContaining({
				deviceId: 'dev-a',
				readIds: [],
				syncCursor: null
			})
		);
	});
});
