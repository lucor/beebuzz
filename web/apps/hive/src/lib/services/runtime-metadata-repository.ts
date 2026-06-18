import { RUNTIME_METADATA_STORE, openExistingHiveDB, openHiveDB } from './hive-db';

const NOTIFICATION_RUNTIME_KEY = 'notification_runtime';

export type NotificationReceivedVia = 'push' | 'outbox';

export type NotificationRuntimeMetadata = {
	key: typeof NOTIFICATION_RUNTIME_KEY;
	lastNotificationReceivedAt: string | null;
	lastNotificationReceivedVia: NotificationReceivedVia | null;
};

export type RecordNotificationReceivedInput = {
	via: NotificationReceivedVia;
	receivedAt?: string;
};

const DEFAULT_NOTIFICATION_RUNTIME: NotificationRuntimeMetadata = {
	key: NOTIFICATION_RUNTIME_KEY,
	lastNotificationReceivedAt: null,
	lastNotificationReceivedVia: null
};

function hasRuntimeStore(db: IDBDatabase): boolean {
	return db.objectStoreNames.contains(RUNTIME_METADATA_STORE);
}

/**
 * Reads the technical notification runtime metadata for debug reports.
 *
 * Runs on the main thread (report/context path), so it may force the schema
 * upgrade that creates the store. Returns defaults if the store is missing or
 * the read fails.
 */
export async function getNotificationRuntimeMetadata(): Promise<NotificationRuntimeMetadata> {
	try {
		const db = await openHiveDB();
		return await new Promise((resolve, reject) => {
			const tx = db.transaction(RUNTIME_METADATA_STORE, 'readonly');
			const req = tx.objectStore(RUNTIME_METADATA_STORE).get(NOTIFICATION_RUNTIME_KEY);
			req.onsuccess = () =>
				resolve({
					...DEFAULT_NOTIFICATION_RUNTIME,
					...(req.result as Partial<NotificationRuntimeMetadata> | undefined)
				});
			req.onerror = () => reject(new Error(req.error?.message ?? 'Runtime metadata fetch failed'));
		});
	} catch {
		return DEFAULT_NOTIFICATION_RUNTIME;
	}
}

/**
 * Records the latest received notification metadata, best-effort.
 *
 * Uses a non-upgrading open and no-ops when the store does not exist yet, so
 * push delivery never depends on an IndexedDB schema upgrade. Any failure is
 * swallowed.
 */
export async function recordNotificationReceived({
	via,
	receivedAt = new Date().toISOString()
}: RecordNotificationReceivedInput): Promise<void> {
	try {
		const db = await openExistingHiveDB();
		if (!hasRuntimeStore(db)) return;

		await new Promise<void>((resolve, reject) => {
			const tx = db.transaction(RUNTIME_METADATA_STORE, 'readwrite');
			const store = tx.objectStore(RUNTIME_METADATA_STORE);
			const req = store.get(NOTIFICATION_RUNTIME_KEY);

			req.onsuccess = () => {
				const current: NotificationRuntimeMetadata = {
					...DEFAULT_NOTIFICATION_RUNTIME,
					...(req.result as Partial<NotificationRuntimeMetadata> | undefined)
				};
				store.put({
					...current,
					lastNotificationReceivedAt: receivedAt,
					lastNotificationReceivedVia: via
				});
			};
			req.onerror = () => reject(new Error(req.error?.message ?? 'Runtime metadata read failed'));

			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Runtime metadata write failed'));
			tx.onabort = () => reject(new Error(tx.error?.message ?? 'Runtime metadata write aborted'));
		});
	} catch {
		// Best-effort: push delivery must not depend on metadata writes.
	}
}

/**
 * Records the timestamp for the last received notification, best-effort.
 */
export async function recordNotificationRuntimeTimestamp(
	ts = new Date().toISOString()
): Promise<void> {
	try {
		const db = await openExistingHiveDB();
		if (!hasRuntimeStore(db)) return;

		await new Promise<void>((resolve, reject) => {
			const tx = db.transaction(RUNTIME_METADATA_STORE, 'readwrite');
			const store = tx.objectStore(RUNTIME_METADATA_STORE);
			const req = store.get(NOTIFICATION_RUNTIME_KEY);

			req.onsuccess = () => {
				const current: NotificationRuntimeMetadata = {
					...DEFAULT_NOTIFICATION_RUNTIME,
					...(req.result as Partial<NotificationRuntimeMetadata> | undefined)
				};
				store.put({ ...current, lastNotificationReceivedAt: ts });
			};
			req.onerror = () => reject(new Error(req.error?.message ?? 'Runtime metadata read failed'));

			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Runtime metadata write failed'));
			tx.onabort = () => reject(new Error(tx.error?.message ?? 'Runtime metadata write aborted'));
		});
	} catch {
		// Best-effort: push delivery must not depend on metadata writes.
	}
}

/**
 * Records the delivery method for the last received notification, best-effort.
 */
export async function recordNotificationReceivedVia(via: NotificationReceivedVia): Promise<void> {
	try {
		const db = await openExistingHiveDB();
		if (!hasRuntimeStore(db)) return;

		await new Promise<void>((resolve, reject) => {
			const tx = db.transaction(RUNTIME_METADATA_STORE, 'readwrite');
			const store = tx.objectStore(RUNTIME_METADATA_STORE);
			const req = store.get(NOTIFICATION_RUNTIME_KEY);

			req.onsuccess = () => {
				const current: NotificationRuntimeMetadata = {
					...DEFAULT_NOTIFICATION_RUNTIME,
					...(req.result as Partial<NotificationRuntimeMetadata> | undefined)
				};
				store.put({ ...current, lastNotificationReceivedVia: via });
			};
			req.onerror = () => reject(new Error(req.error?.message ?? 'Runtime metadata read failed'));

			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Runtime metadata write failed'));
			tx.onabort = () => reject(new Error(tx.error?.message ?? 'Runtime metadata write aborted'));
		});
	} catch {
		// Best-effort: push delivery must not depend on metadata writes.
	}
}

/** Clears the notification runtime metadata (e.g. on device disconnect). */
export async function clearNotificationRuntimeMetadata(): Promise<void> {
	const db = await openHiveDB();
	if (!hasRuntimeStore(db)) return;

	await new Promise<void>((resolve, reject) => {
		const tx = db.transaction(RUNTIME_METADATA_STORE, 'readwrite');
		tx.objectStore(RUNTIME_METADATA_STORE).delete(NOTIFICATION_RUNTIME_KEY);
		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(new Error(tx.error?.message ?? 'Runtime metadata clear failed'));
	});
}
