import { beforeEach, describe, expect, it } from 'vitest';
import { NOTIFICATIONS_STORE, openHiveDB } from './hive-db';
import { notificationsRepository, type StoredNotificationRecord } from './notifications-repository';

async function clearNotificationsStore(): Promise<void> {
	const db = await openHiveDB();
	return new Promise((resolve, reject) => {
		const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
		const request = tx.objectStore(NOTIFICATIONS_STORE).clear();
		request.onsuccess = () => resolve();
		request.onerror = () => reject(new Error(request.error?.message ?? 'Clear failed'));
	});
}

describe('notificationsRepository', () => {
	beforeEach(async () => {
		await clearNotificationsStore();
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

	it('migrates legacy records without deviceId and removes foreign ones', async () => {
		// Seed a legacy record (no deviceId) directly via raw IndexedDB.
		const db = await openHiveDB();
		await new Promise<void>((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
			const store = tx.objectStore(NOTIFICATIONS_STORE);
			store.put({
				id: 'n-legacy',
				title: 'Legacy',
				body: 'No deviceId',
				topic: 'alerts',
				sentAt: '2026-04-20T08:00:00.000Z'
			});
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Seed failed'));
		});

		await notificationsRepository.save({
			id: 'n-current',
			deviceId: 'dev-a',
			title: 'Current',
			body: 'Already scoped',
			topic: 'alerts',
			sentAt: '2026-04-20T09:00:00.000Z'
		});
		await notificationsRepository.save({
			id: 'n-foreign',
			deviceId: 'dev-b',
			title: 'Foreign',
			body: 'Wrong device',
			topic: 'alerts',
			sentAt: '2026-04-20T10:00:00.000Z'
		});

		await notificationsRepository.migrateLegacyNotifications('dev-a');

		const devAList = await notificationsRepository.listByDevice('dev-a');
		const devBList = await notificationsRepository.listByDevice('dev-b');

		expect(devAList.map((r) => r.id).sort()).toEqual(['n-current', 'n-legacy']);
		expect(devBList).toHaveLength(0);
	});

	it('normalizes snake_case fields during legacy migration', async () => {
		const db = await openHiveDB();
		await new Promise<void>((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
			const store = tx.objectStore(NOTIFICATIONS_STORE);
			store.put({
				id: 'n-snake',
				title: 'Snake',
				body: 'Case',
				topic: 'alerts',
				sent_at: '2026-04-20T08:00:00.000Z',
				topic_id: 't-1'
			});
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Seed failed'));
		});

		await notificationsRepository.migrateLegacyNotifications('dev-a');

		const list = await notificationsRepository.listByDevice('dev-a');
		expect(list).toHaveLength(1);
		const migrated = list[0] as unknown as Record<string, unknown>;
		expect(migrated.sentAt).toBe('2026-04-20T08:00:00.000Z');
		expect(migrated.sent_at).toBeUndefined();
		expect(migrated.topicId).toBe('t-1');
		expect(migrated.topic_id).toBeUndefined();
	});
});
