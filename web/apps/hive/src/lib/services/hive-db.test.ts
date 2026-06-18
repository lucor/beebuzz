import { beforeEach, describe, expect, it } from 'vitest';
import {
	ENCRYPTED_KEY_STORE,
	ENCRYPTION_METADATA_STORE,
	HIVE_DB_NAME,
	NOTIFICATIONS_BY_DEVICE_INDEX,
	NOTIFICATIONS_STORE,
	RUNTIME_METADATA_STORE,
	WRAPPING_KEY_STORE,
	openExistingHiveDB,
	openHiveDB
} from './hive-db';

function deleteHiveDB(): Promise<void> {
	return new Promise((resolve, reject) => {
		const request = indexedDB.deleteDatabase(HIVE_DB_NAME);
		request.onsuccess = () => resolve();
		request.onerror = () => reject(new Error(request.error?.message ?? 'Delete failed'));
		request.onblocked = () => reject(new Error('Delete blocked'));
	});
}

function createV1HiveDB(): Promise<IDBDatabase> {
	return new Promise((resolve, reject) => {
		const request = indexedDB.open(HIVE_DB_NAME, 1);
		request.onupgradeneeded = () => {
			const db = request.result;
			db.createObjectStore(NOTIFICATIONS_STORE, { keyPath: 'id' });
			db.createObjectStore(ENCRYPTION_METADATA_STORE, { keyPath: 'id' });
			db.createObjectStore(WRAPPING_KEY_STORE);
			db.createObjectStore(ENCRYPTED_KEY_STORE, { keyPath: 'id' });
		};
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => reject(new Error(request.error?.message ?? 'Open failed'));
	});
}

function createV2HiveDBWithNotification(): Promise<IDBDatabase> {
	return new Promise((resolve, reject) => {
		const request = indexedDB.open(HIVE_DB_NAME, 2);
		request.onupgradeneeded = () => {
			const db = request.result;
			const notifications = db.createObjectStore(NOTIFICATIONS_STORE, { keyPath: 'id' });
			notifications.createIndex(NOTIFICATIONS_BY_DEVICE_INDEX, 'deviceId');
			db.createObjectStore(ENCRYPTION_METADATA_STORE, { keyPath: 'id' });
			db.createObjectStore(WRAPPING_KEY_STORE);
			db.createObjectStore(ENCRYPTED_KEY_STORE, { keyPath: 'id' });
			notifications.put({
				id: 'n-published',
				deviceId: 'dev-a',
				title: 'Door',
				body: 'Opened',
				topic: 'alerts',
				sentAt: '2026-04-20T09:00:00.000Z'
			});
		};
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => reject(new Error(request.error?.message ?? 'Open failed'));
	});
}

describe('hive-db', () => {
	beforeEach(async () => {
		await deleteHiveDB();
	});

	it('opens an existing v1 database without forcing the v2 upgrade', async () => {
		const v1DB = await createV1HiveDB();
		v1DB.close();

		const existing = await openExistingHiveDB();

		expect(existing.version).toBe(1);
		expect(existing.objectStoreNames.contains(ENCRYPTION_METADATA_STORE)).toBe(true);
		existing.close();
	});

	it('creates an empty v1 database when no Hive database exists yet', async () => {
		const existing = await openExistingHiveDB();

		expect(existing.version).toBe(1);
		expect(existing.objectStoreNames.length).toBe(0);
		existing.close();
	});

	it('creates missing stores when upgrading a sparse existing database to the latest version', async () => {
		const sparseDB = await openExistingHiveDB();
		expect(sparseDB.version).toBe(1);
		sparseDB.close();

		const upgraded = await openHiveDB();

		expect(upgraded.version).toBe(3);
		expect(upgraded.objectStoreNames.contains(NOTIFICATIONS_STORE)).toBe(true);
		const tx = upgraded.transaction(NOTIFICATIONS_STORE, 'readonly');
		expect(
			tx.objectStore(NOTIFICATIONS_STORE).indexNames.contains(NOTIFICATIONS_BY_DEVICE_INDEX)
		).toBe(true);
		upgraded.close();
	});

	it('creates the runtime metadata store in the latest schema', async () => {
		const db = await openHiveDB();

		expect(db.version).toBe(3);
		expect(db.objectStoreNames.contains(RUNTIME_METADATA_STORE)).toBe(true);
		db.close();
	});

	it('upgrades published v2 databases to v3 without clearing existing notifications', async () => {
		const v2DB = await createV2HiveDBWithNotification();
		expect(v2DB.version).toBe(2);
		expect(v2DB.objectStoreNames.contains(RUNTIME_METADATA_STORE)).toBe(false);
		v2DB.close();

		const upgraded = await openHiveDB();

		expect(upgraded.version).toBe(3);
		expect(upgraded.objectStoreNames.contains(RUNTIME_METADATA_STORE)).toBe(true);
		const tx = upgraded.transaction(NOTIFICATIONS_STORE, 'readonly');
		const request = tx.objectStore(NOTIFICATIONS_STORE).get('n-published');
		await new Promise<void>((resolve, reject) => {
			request.onsuccess = () => resolve();
			request.onerror = () => reject(new Error(request.error?.message ?? 'Read failed'));
		});
		expect(request.result).toMatchObject({ id: 'n-published', deviceId: 'dev-a' });
		upgraded.close();
	});
});
