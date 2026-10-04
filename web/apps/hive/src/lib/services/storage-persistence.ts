const REQUEST_ATTEMPT_KEY = 'beebuzz.hive.storage-persistence-requested';

export type StoragePersistenceStatus = 'persistent' | 'best-effort' | 'unsupported' | 'unavailable';

let requestInFlight: Promise<StoragePersistenceStatus> | null = null;

function getStorageManager(): StorageManager | null {
	if (typeof navigator === 'undefined' || !('storage' in navigator)) return null;
	try {
		return navigator.storage;
	} catch {
		return null;
	}
}

/** Reads the current persistence state without requesting it. */
export async function getStoragePersistenceStatus(): Promise<StoragePersistenceStatus> {
	const storage = getStorageManager();
	if (!storage?.persisted) return 'unsupported';

	try {
		return (await storage.persisted()) ? 'persistent' : 'best-effort';
	} catch {
		return 'unavailable';
	}
}

/** Requests persistence once automatically, or again when explicitly requested. */
export function requestStoragePersistence(
	options: { retry?: boolean } = {}
): Promise<StoragePersistenceStatus> {
	if (requestInFlight) return requestInFlight;

	requestInFlight = requestStoragePersistenceOnce(options.retry ?? false).finally(() => {
		requestInFlight = null;
	});
	return requestInFlight;
}

async function requestStoragePersistenceOnce(retry: boolean): Promise<StoragePersistenceStatus> {
	const status = await getStoragePersistenceStatus();
	if (status !== 'best-effort') return status;

	const storage = getStorageManager();
	if (!storage?.persist) return 'unsupported';

	if (!retry) {
		try {
			if (localStorage.getItem(REQUEST_ATTEMPT_KEY) === 'true') return 'best-effort';
			localStorage.setItem(REQUEST_ATTEMPT_KEY, 'true');
		} catch {
			// Continue with the request if browser storage for the marker is unavailable.
		}
	}

	try {
		await storage.persist();
		return await getStoragePersistenceStatus();
	} catch {
		return 'unavailable';
	}
}
