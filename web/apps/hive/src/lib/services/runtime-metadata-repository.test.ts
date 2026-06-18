import { beforeEach, describe, expect, it } from 'vitest';
import {
	ENCRYPTED_KEY_STORE,
	ENCRYPTION_METADATA_STORE,
	HIVE_DB_NAME,
	NOTIFICATIONS_STORE,
	RUNTIME_METADATA_STORE,
	WRAPPING_KEY_STORE,
	openHiveDB
} from './hive-db';
import {
	clearNotificationRuntimeMetadata,
	getNotificationRuntimeMetadata,
	recordNotificationReceived,
	recordNotificationRuntimeTimestamp,
	recordNotificationReceivedVia
} from './runtime-metadata-repository';

const RECEIVED_TS = '2026-01-01T00:00:00.000Z';

function deleteHiveDB(): Promise<void> {
	return new Promise((resolve, reject) => {
		const request = indexedDB.deleteDatabase(HIVE_DB_NAME);
		request.onsuccess = () => resolve();
		request.onerror = () => reject(new Error(request.error?.message ?? 'Delete failed'));
		request.onblocked = () => reject(new Error('Delete blocked'));
	});
}

function createV2HiveDB(): Promise<IDBDatabase> {
	return new Promise((resolve, reject) => {
		const request = indexedDB.open(HIVE_DB_NAME, 2);
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

describe('runtime metadata repository', () => {
	beforeEach(async () => {
		await deleteHiveDB();
	});

	it('returns empty notification runtime metadata by default', async () => {
		const metadata = await getNotificationRuntimeMetadata();

		expect(metadata).toEqual({
			key: 'notification_runtime',
			lastNotificationReceivedAt: null,
			lastNotificationReceivedVia: null
		});
	});

	it('records notification received timestamp after the store exists', async () => {
		const db = await openHiveDB();
		expect(db.objectStoreNames.contains(RUNTIME_METADATA_STORE)).toBe(true);
		db.close();

		await recordNotificationRuntimeTimestamp(RECEIVED_TS);

		const metadata = await getNotificationRuntimeMetadata();
		expect(metadata).toEqual({
			key: 'notification_runtime',
			lastNotificationReceivedAt: RECEIVED_TS,
			lastNotificationReceivedVia: null
		});
	});

	it('records notification received via after the store exists', async () => {
		const db = await openHiveDB();
		expect(db.objectStoreNames.contains(RUNTIME_METADATA_STORE)).toBe(true);
		db.close();

		await recordNotificationReceivedVia('push');

		const metadata = await getNotificationRuntimeMetadata();
		expect(metadata).toEqual({
			key: 'notification_runtime',
			lastNotificationReceivedAt: null,
			lastNotificationReceivedVia: 'push'
		});
	});

	it('records notification received timestamp and via atomically', async () => {
		const db = await openHiveDB();
		expect(db.objectStoreNames.contains(RUNTIME_METADATA_STORE)).toBe(true);
		db.close();

		await recordNotificationReceived({ receivedAt: RECEIVED_TS, via: 'push' });

		const metadata = await getNotificationRuntimeMetadata();
		expect(metadata).toEqual({
			key: 'notification_runtime',
			lastNotificationReceivedAt: RECEIVED_TS,
			lastNotificationReceivedVia: 'push'
		});
	});

	it('does not throw when recording before the runtime metadata store exists', async () => {
		const db = await createV2HiveDB();
		expect(db.objectStoreNames.contains(RUNTIME_METADATA_STORE)).toBe(false);
		db.close();

		await expect(recordNotificationRuntimeTimestamp(RECEIVED_TS)).resolves.toBe(undefined);
		await expect(recordNotificationReceivedVia('push')).resolves.toBe(undefined);
		await expect(
			recordNotificationReceived({ receivedAt: RECEIVED_TS, via: 'push' })
		).resolves.toBe(undefined);
	});

	it('clears notification runtime metadata', async () => {
		await openHiveDB();
		await recordNotificationRuntimeTimestamp(RECEIVED_TS);
		await recordNotificationReceivedVia('push');

		await clearNotificationRuntimeMetadata();

		const metadata = await getNotificationRuntimeMetadata();
		expect(metadata).toEqual({
			key: 'notification_runtime',
			lastNotificationReceivedAt: null,
			lastNotificationReceivedVia: null
		});
	});
});
