import { NOTIFICATIONS_STORE, openHiveDB } from './hive-db';

export interface StoredNotificationRecord {
	id: string;
	title: string;
	body: string;
	topic: string;
	sentAt: string;
	topicId?: string;
	attachment?: unknown;
	priority?: string;
}

export const notificationsRepository = {
	/** Persists one notification record to IndexedDB. */
	async save(input: StoredNotificationRecord): Promise<void> {
		const db = await openHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readwrite');
			const request = tx.objectStore(NOTIFICATIONS_STORE).put(input);
			request.onerror = () =>
				reject(new Error(request.error?.message ?? 'Notification write failed'));
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Notification transaction failed'));
		});
	},

	/** Loads all persisted notification records from IndexedDB. */
	async list(): Promise<Record<string, unknown>[]> {
		const db = await openHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(NOTIFICATIONS_STORE, 'readonly');
			const request = tx.objectStore(NOTIFICATIONS_STORE).getAll();
			request.onsuccess = () => resolve(request.result as Record<string, unknown>[]);
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
	}
};
