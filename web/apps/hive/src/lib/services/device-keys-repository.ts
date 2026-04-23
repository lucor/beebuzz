import {
	ENCRYPTED_KEY_STORE,
	ENCRYPTION_METADATA_STORE,
	WRAPPING_KEY_STORE,
	openHiveDB
} from './hive-db';

const AUTH_STATE_KEY = '__auth_state__';

export interface DeviceCredentials {
	id: string;
	deviceId: string;
	deviceToken: string;
}

export interface KeyMetadata {
	id: string;
	recipient: string;
	fingerprint?: string;
	createdAt: string;
}

export interface WrappedPrivateKeyRecord {
	id: string;
	wrappedKey: ArrayBuffer;
	iv: ArrayBuffer;
	format: 'pkcs8';
}

export interface WrappedIdentityRecord {
	metadata: KeyMetadata;
	wrappingKey: CryptoKey;
	wrappedPrivateKey: WrappedPrivateKeyRecord;
}

function getStoreValue<T>(
	db: IDBDatabase,
	storeName: string,
	key: IDBValidKey,
	errorMessage: string
): Promise<T | null> {
	return new Promise((resolve, reject) => {
		const tx = db.transaction(storeName, 'readonly');
		const request = tx.objectStore(storeName).get(key);
		request.onsuccess = () => resolve((request.result as T | undefined) ?? null);
		request.onerror = () => reject(new Error(request.error?.message ?? errorMessage));
	});
}

/** Writes wrapped identity records into an already-open database. */
function putWrappedIdentity(
	db: IDBDatabase,
	keyId: string,
	metadata: KeyMetadata,
	wrappingKey: CryptoKey,
	wrappedPrivateKey: WrappedPrivateKeyRecord
): Promise<void> {
	return new Promise((resolve, reject) => {
		const tx = db.transaction(
			[ENCRYPTION_METADATA_STORE, WRAPPING_KEY_STORE, ENCRYPTED_KEY_STORE],
			'readwrite'
		);

		tx.objectStore(ENCRYPTION_METADATA_STORE).put(metadata);
		tx.objectStore(WRAPPING_KEY_STORE).put(wrappingKey, keyId);
		tx.objectStore(ENCRYPTED_KEY_STORE).put({
			...wrappedPrivateKey,
			id: keyId
		});

		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(new Error(tx.error?.message ?? 'Transaction failed'));
	});
}

export const deviceKeysRepository = {
	/** Returns the first usable key metadata record, skipping reserved entries. */
	async getFirstMetadata(): Promise<KeyMetadata | null> {
		const db = await openHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(ENCRYPTION_METADATA_STORE, 'readonly');
			const cursorReq = tx.objectStore(ENCRYPTION_METADATA_STORE).openCursor();
			cursorReq.onsuccess = () => {
				const cursor = cursorReq.result;
				if (!cursor) {
					resolve(null);
					return;
				}

				if (cursor.key === AUTH_STATE_KEY) {
					cursor.continue();
					return;
				}

				resolve(cursor.value as KeyMetadata);
			};
			cursorReq.onerror = () => reject(new Error(cursorReq.error?.message ?? 'Cursor failed'));
		});
	},

	/** Returns the stored wrapping key and encrypted PKCS8 blob for a device, if present. */
	async getStoredMaterial(keyId: string): Promise<{
		wrappingKey: CryptoKey | null;
		wrappedPrivateKey: WrappedPrivateKeyRecord | null;
	}> {
		const db = await openHiveDB();
		const [wrappingKey, wrappedPrivateKey] = await Promise.all([
			getStoreValue<CryptoKey>(db, WRAPPING_KEY_STORE, keyId, 'Wrapping key fetch failed'),
			getStoreValue<WrappedPrivateKeyRecord>(
				db,
				ENCRYPTED_KEY_STORE,
				keyId,
				'Wrapped key fetch failed'
			)
		]);

		return { wrappingKey, wrappedPrivateKey };
	},

	/** Stores metadata, wrapping key, and wrapped private key in one transaction. */
	async storeWrappedIdentity(
		keyId: string,
		metadata: KeyMetadata,
		wrappingKey: CryptoKey,
		wrappedPrivateKey: WrappedPrivateKeyRecord
	): Promise<void> {
		const db = await openHiveDB();
		return putWrappedIdentity(db, keyId, metadata, wrappingKey, wrappedPrivateKey);
	},

	/** Renames all stored records from one key id to another. */
	async rename(currentKeyId: string, nextKeyId: string): Promise<void> {
		if (currentKeyId === nextKeyId) {
			return;
		}

		const db = await openHiveDB();
		const [metadata, wrappingKey, wrappedPrivateKey] = await Promise.all([
			getStoreValue<KeyMetadata>(
				db,
				ENCRYPTION_METADATA_STORE,
				currentKeyId,
				'Metadata fetch failed'
			),
			getStoreValue<CryptoKey>(db, WRAPPING_KEY_STORE, currentKeyId, 'Wrapping key fetch failed'),
			getStoreValue<WrappedPrivateKeyRecord>(
				db,
				ENCRYPTED_KEY_STORE,
				currentKeyId,
				'Wrapped key fetch failed'
			)
		]);

		if (!metadata || !wrappingKey || !wrappedPrivateKey) {
			throw new Error('Stored encryption key not found for rename');
		}

		await putWrappedIdentity(
			db,
			nextKeyId,
			{ ...metadata, id: nextKeyId },
			wrappingKey,
			wrappedPrivateKey
		);

		return new Promise((resolve, reject) => {
			const tx = db.transaction(
				[ENCRYPTION_METADATA_STORE, WRAPPING_KEY_STORE, ENCRYPTED_KEY_STORE],
				'readwrite'
			);
			tx.objectStore(ENCRYPTION_METADATA_STORE).delete(currentKeyId);
			tx.objectStore(WRAPPING_KEY_STORE).delete(currentKeyId);
			tx.objectStore(ENCRYPTED_KEY_STORE).delete(currentKeyId);
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Rename cleanup failed'));
		});
	},

	/** Stores device credentials (deviceId + deviceToken) in IndexedDB. */
	async storeDeviceCredentials(deviceId: string, deviceToken: string): Promise<void> {
		const db = await openHiveDB();
		return new Promise((resolve, reject) => {
			const tx = db.transaction(ENCRYPTION_METADATA_STORE, 'readwrite');
			tx.objectStore(ENCRYPTION_METADATA_STORE).put({
				id: AUTH_STATE_KEY,
				deviceId,
				deviceToken
			} satisfies DeviceCredentials);
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Store credentials failed'));
		});
	},

	/** Retrieves stored device credentials, if available. */
	async getDeviceCredentials(): Promise<DeviceCredentials | null> {
		const db = await openHiveDB();
		return getStoreValue<DeviceCredentials>(
			db,
			ENCRYPTION_METADATA_STORE,
			AUTH_STATE_KEY,
			'Credentials fetch failed'
		);
	},

	/** Clears all locally stored device key material. */
	async clearAll(): Promise<void> {
		const db = await openHiveDB();

		return new Promise((resolve, reject) => {
			const tx = db.transaction(
				[ENCRYPTION_METADATA_STORE, WRAPPING_KEY_STORE, ENCRYPTED_KEY_STORE],
				'readwrite'
			);
			tx.objectStore(ENCRYPTION_METADATA_STORE).clear();
			tx.objectStore(WRAPPING_KEY_STORE).clear();
			tx.objectStore(ENCRYPTED_KEY_STORE).clear();
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error(tx.error?.message ?? 'Clear keys failed'));
		});
	}
};
