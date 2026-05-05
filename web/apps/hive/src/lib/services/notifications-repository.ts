import {
	NOTIFICATIONS_BY_DEVICE_INDEX,
	NOTIFICATIONS_STORE,
	openExistingHiveDB,
	openHiveDB
} from './hive-db';

export interface StoredNotificationRecord {
	id: string;
	deviceId: string;
	title: string;
	body: string;
	topic: string;
	sentAt: string;
	topicId?: string;
	attachment?: unknown;
	priority?: string;
}

/**
 * Normalizes a legacy record by converting snake_case fields to camelCase.
 */
function normalizeLegacyRecord(record: Record<string, unknown>): Record<string, unknown> {
	const normalized: Record<string, unknown> = { ...record };
	if ('sent_at' in record && !('sentAt' in record)) {
		normalized.sentAt = record.sent_at;
		delete normalized.sent_at;
	}
	if ('topic_id' in record && !('topicId' in record)) {
		normalized.topicId = record.topic_id;
		delete normalized.topic_id;
	}
	return normalized;
}

export const notificationsRepository = {
	/** Persists one notification record to IndexedDB. */
	async save(input: StoredNotificationRecord): Promise<void> {
		const db = await openExistingHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
			const request = tx.objectStore(NOTIFICATIONS_STORE).put(input);
			request.onerror = () =>
				reject(new Error(request.error?.message ?? 'Notification write failed'));
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Notification transaction failed'));
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
				reject(new Error(request.error?.message ?? 'Notifications fetch failed'));
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
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Notifications delete failed'));
		});
	},

	/**
	 * Migrates legacy notification records that lack a deviceId.
	 *
	 * - Records without a deviceId are stamped with the current deviceId and
	 *   re-saved so they become queryable via the by-device index.
	 * - Records that already belong to a different deviceId are deleted.
	 */
	async migrateLegacyNotifications(deviceId: string): Promise<void> {
		const db = await openHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
			const store = tx.objectStore(NOTIFICATIONS_STORE);
			const request = store.getAll();

			request.onsuccess = () => {
				const records = request.result as Array<Record<string, unknown>>;
				for (const record of records) {
					const recordDeviceId = record.deviceId;
					if (typeof recordDeviceId !== 'string') {
						// Legacy record: normalize fields and stamp with current deviceId.
						store.put({ ...normalizeLegacyRecord(record), deviceId });
					} else if (recordDeviceId !== deviceId) {
						// Belongs to a different device: remove.
						store.delete(record.id as string);
					}
				}
			};
			tx.oncomplete = () => resolve();
			tx.onerror = () =>
				reject(new Error(tx.error?.message ?? 'Legacy migration transaction failed'));
		});
	}
};
