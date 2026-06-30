import {
	NOTIFICATIONS_BY_DEVICE_INDEX,
	NOTIFICATION_STATE_STORE,
	NOTIFICATIONS_STORE,
	openExistingHiveDB,
	openHiveDB
} from './hive-db';

export interface StoredNotificationRecord {
	id: string;
	deviceId: string;
	title: string;
	body: string;
	topic: string | null;
	sentAt: string;
	topicId?: string;
	attachment?: unknown;
	priority?: string;
}

export interface NotificationStateRecord {
	deviceId: string;
	readIds: string[];
	syncCursor: string | null;
	updatedAt: string;
}

export class StorageQuotaError extends Error {
	constructor(message = 'Browser storage quota exceeded') {
		super(message);
		this.name = 'StorageQuotaError';
	}
}

function isQuotaExceeded(error: unknown): boolean {
	if (!error || typeof error !== 'object') return false;
	const name = 'name' in error ? String((error as { name?: unknown }).name) : '';
	const code = 'code' in error ? Number((error as { code?: unknown }).code) : 0;
	return name === 'QuotaExceededError' || name === 'NS_ERROR_DOM_QUOTA_REACHED' || code === 22;
}

function storageError(error: unknown, fallback: string): Error {
	if (isQuotaExceeded(error)) {
		return new StorageQuotaError();
	}
	if (error instanceof Error) {
		return error;
	}
	return new Error(fallback);
}

function transactionError(tx: IDBTransaction, fallback: string): Error {
	return storageError(tx.error, tx.error?.message ?? fallback);
}

function defaultNotificationState(deviceId: string): NotificationStateRecord {
	return {
		deviceId,
		readIds: [],
		syncCursor: null,
		updatedAt: new Date(0).toISOString()
	};
}

async function getNotificationState(
	db: IDBDatabase,
	deviceId: string
): Promise<NotificationStateRecord> {
	return new Promise((resolve, reject) => {
		const tx = db.transaction(NOTIFICATION_STATE_STORE, 'readonly');
		const request = tx.objectStore(NOTIFICATION_STATE_STORE).get(deviceId);
		request.onsuccess = () => {
			resolve({
				...defaultNotificationState(deviceId),
				...(request.result as Partial<NotificationStateRecord> | undefined)
			});
		};
		request.onerror = () =>
			reject(
				storageError(request.error, request.error?.message ?? 'Notification state fetch failed')
			);
	});
}

/**
 * Reads, mutates, and writes a device's notification state inside a single
 * readwrite transaction. IndexedDB serializes overlapping readwrite
 * transactions, so keeping the read-modify-write in one transaction keeps
 * concurrent `readIds` and `syncCursor` updates from overwriting each other.
 */
async function updateNotificationState(
	db: IDBDatabase,
	deviceId: string,
	mutate: (state: NotificationStateRecord) => NotificationStateRecord
): Promise<void> {
	return new Promise((resolve, reject) => {
		const tx = db.transaction(NOTIFICATION_STATE_STORE, 'readwrite');
		const store = tx.objectStore(NOTIFICATION_STATE_STORE);
		const request = store.get(deviceId);
		request.onsuccess = () => {
			const current: NotificationStateRecord = {
				...defaultNotificationState(deviceId),
				...(request.result as Partial<NotificationStateRecord> | undefined)
			};
			store.put({
				...mutate(current),
				deviceId,
				updatedAt: new Date().toISOString()
			});
		};
		request.onerror = () =>
			reject(
				storageError(request.error, request.error?.message ?? 'Notification state fetch failed')
			);
		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(transactionError(tx, 'Notification state transaction failed'));
		tx.onabort = () => reject(transactionError(tx, 'Notification state transaction aborted'));
	});
}

export const notificationsRepository = {
	/** Persists one notification record to IndexedDB. */
	async save(input: StoredNotificationRecord): Promise<void> {
		const db = await openExistingHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
			const request = tx.objectStore(NOTIFICATIONS_STORE).put(input);
			request.onerror = () =>
				reject(storageError(request.error, request.error?.message ?? 'Notification write failed'));
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(transactionError(tx, 'Notification transaction failed'));
			tx.onabort = () => reject(transactionError(tx, 'Notification transaction aborted'));
		});
	},

	/** Loads persisted notification records for one backend device ID. */
	async listByDevice(deviceId: string): Promise<StoredNotificationRecord[]> {
		const db = await openHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readonly');
			const index = tx.objectStore(NOTIFICATIONS_STORE).index(NOTIFICATIONS_BY_DEVICE_INDEX);
			const request = index.getAll(deviceId);
			request.onsuccess = () => resolve(request.result as StoredNotificationRecord[]);
			request.onerror = () =>
				reject(storageError(request.error, request.error?.message ?? 'Notifications fetch failed'));
		});
	},

	/** Deletes persisted notification records by id. */
	async deleteMany(ids: string[]): Promise<void> {
		if (ids.length === 0) {
			return;
		}

		const db = await openHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
			const store = tx.objectStore(NOTIFICATIONS_STORE);
			for (const id of ids) {
				store.delete(id);
			}
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(transactionError(tx, 'Notifications delete failed'));
			tx.onabort = () => reject(transactionError(tx, 'Notifications delete aborted'));
		});
	},

	/** Deletes all notification records for one backend device ID. */
	async clearByDevice(deviceId: string): Promise<void> {
		const db = await openHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
			const index = tx.objectStore(NOTIFICATIONS_STORE).index(NOTIFICATIONS_BY_DEVICE_INDEX);
			const request = index.getAllKeys(deviceId);
			request.onsuccess = () => {
				const store = tx.objectStore(NOTIFICATIONS_STORE);
				for (const key of request.result) {
					store.delete(key);
				}
			};
			request.onerror = () =>
				reject(
					storageError(request.error, request.error?.message ?? 'Notification keys fetch failed')
				);
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(transactionError(tx, 'Notifications clear failed'));
			tx.onabort = () => reject(transactionError(tx, 'Notifications clear aborted'));
		});
	},

	/** Loads read-state and sync cursor metadata for one backend device ID. */
	async getState(deviceId: string): Promise<NotificationStateRecord> {
		const db = await openHiveDB();
		return getNotificationState(db, deviceId);
	},

	/** Persists read notification IDs for one backend device ID. */
	async saveReadIds(deviceId: string, readIds: string[]): Promise<void> {
		const db = await openHiveDB();
		await updateNotificationState(db, deviceId, (state) => ({
			...state,
			readIds: [...new Set(readIds)]
		}));
	},

	/** Persists the outbox sync cursor for one backend device ID. */
	async saveSyncCursor(deviceId: string, syncCursor: string | null): Promise<void> {
		const db = await openHiveDB();
		await updateNotificationState(db, deviceId, (state) => ({
			...state,
			syncCursor
		}));
	},

	/** Clears notification state metadata for one backend device ID. */
	async clearState(deviceId: string): Promise<void> {
		const db = await openHiveDB();
		return new Promise((resolve, reject) => {
			const tx = db.transaction(NOTIFICATION_STATE_STORE, 'readwrite');
			tx.objectStore(NOTIFICATION_STATE_STORE).delete(deviceId);
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(transactionError(tx, 'Notification state clear failed'));
			tx.onabort = () => reject(transactionError(tx, 'Notification state clear aborted'));
		});
	}
};
