export const HIVE_DB_NAME = 'BeeBuzz';
export const HIVE_DB_VERSION = 2;
export const NOTIFICATIONS_STORE = 'notifications';
export const NOTIFICATIONS_BY_DEVICE_INDEX = 'by-device';
export const ENCRYPTION_METADATA_STORE = 'encryption_keys';
export const WRAPPING_KEY_STORE = 'wrapping_keys';
export const ENCRYPTED_KEY_STORE = 'encrypted_private_keys';

function attachVersionChangeClose(db: IDBDatabase): IDBDatabase {
	// If another context (page or service worker) needs to upgrade, close
	// this connection so the upgrade is not blocked.
	db.onversionchange = () => {
		db.close();
	};
	return db;
}

/** Opens the shared Hive IndexedDB database and creates or upgrades required stores. */
export function openHiveDB(): Promise<IDBDatabase> {
	return new Promise((resolve, reject) => {
		const request = indexedDB.open(HIVE_DB_NAME, HIVE_DB_VERSION);
		request.onupgradeneeded = (event) => {
			const db = request.result;
			const oldVersion = event.oldVersion;

			// First-time install: create the notifications store with the by-device index.
			if (oldVersion < 1) {
				const notifications = db.createObjectStore(NOTIFICATIONS_STORE, { keyPath: 'id' });
				notifications.createIndex(NOTIFICATIONS_BY_DEVICE_INDEX, 'deviceId');
			}

			// v1 -> v2: per-device scoping was introduced. Add the by-device index to
			// the existing store so legacy rows can be migrated lazily by the app shell.
			if (oldVersion >= 1 && oldVersion < 2) {
				if (!db.objectStoreNames.contains(NOTIFICATIONS_STORE)) {
					const notifications = db.createObjectStore(NOTIFICATIONS_STORE, { keyPath: 'id' });
					notifications.createIndex(NOTIFICATIONS_BY_DEVICE_INDEX, 'deviceId');
				} else {
					const notifications = request.transaction?.objectStore(NOTIFICATIONS_STORE);
					notifications?.createIndex(NOTIFICATIONS_BY_DEVICE_INDEX, 'deviceId');
				}
			}

			if (!db.objectStoreNames.contains(ENCRYPTION_METADATA_STORE)) {
				db.createObjectStore(ENCRYPTION_METADATA_STORE, { keyPath: 'id' });
			}
			if (!db.objectStoreNames.contains(WRAPPING_KEY_STORE)) {
				db.createObjectStore(WRAPPING_KEY_STORE);
			}
			if (!db.objectStoreNames.contains(ENCRYPTED_KEY_STORE)) {
				db.createObjectStore(ENCRYPTED_KEY_STORE, { keyPath: 'id' });
			}
		};
		request.onerror = () => reject(new Error(request.error?.message ?? 'IndexedDB open failed'));
		request.onblocked = () =>
			reject(
				new Error(
					'IndexedDB upgrade blocked by another open BeeBuzz tab or worker — please close other tabs'
				)
			);
		request.onsuccess = () => {
			resolve(attachVersionChangeClose(request.result));
		};
	});
}

/**
 * Opens the current on-disk Hive database without requesting a version upgrade.
 *
 * Service-worker push/click handling uses this for stores that already existed
 * in v1 (device credentials, key material, and notification records). During a
 * staged rollout, an old foreground page can keep a v1 connection open; asking
 * the service worker for v2 in that moment can block decryption. Non-upgrading
 * opens keep notification delivery independent from the by-device index upgrade.
 */
export function openExistingHiveDB(): Promise<IDBDatabase> {
	return new Promise((resolve, reject) => {
		const request = indexedDB.open(HIVE_DB_NAME);
		request.onerror = () => reject(new Error(request.error?.message ?? 'IndexedDB open failed'));
		request.onsuccess = () => {
			resolve(attachVersionChangeClose(request.result));
		};
	});
}
