import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getStoragePersistenceStatus, requestStoragePersistence } from './storage-persistence';

const REQUEST_ATTEMPT_KEY = 'beebuzz.hive.storage-persistence-requested';
const originalStorageDescriptor = Object.getOwnPropertyDescriptor(navigator, 'storage');

function setStorageManager(storage: Partial<StorageManager> | undefined): void {
	Object.defineProperty(navigator, 'storage', {
		configurable: true,
		value: storage
	});
}

describe('storage persistence', () => {
	beforeEach(() => {
		localStorage.removeItem(REQUEST_ATTEMPT_KEY);
	});

	afterEach(() => {
		if (originalStorageDescriptor) {
			Object.defineProperty(navigator, 'storage', originalStorageDescriptor);
		} else {
			Reflect.deleteProperty(navigator, 'storage');
		}
	});

	it('reports persistent storage when the browser has granted it', async () => {
		setStorageManager({ persisted: vi.fn().mockResolvedValue(true) });

		expect(await getStoragePersistenceStatus()).toBe('persistent');
	});

	it('requests persistence once and reports the resulting browser state', async () => {
		const persisted = vi.fn().mockResolvedValueOnce(false).mockResolvedValueOnce(true);
		const persist = vi.fn().mockResolvedValue(true);
		setStorageManager({ persisted, persist });

		expect(await requestStoragePersistence()).toBe('persistent');
		expect(persist).toHaveBeenCalledOnce();
		expect(localStorage.getItem(REQUEST_ATTEMPT_KEY)).toBe('true');
	});

	it('does not automatically repeat an unsuccessful request but allows an explicit retry', async () => {
		const persisted = vi.fn().mockResolvedValue(false);
		const persist = vi.fn().mockResolvedValue(false);
		setStorageManager({ persisted, persist });

		expect(await requestStoragePersistence()).toBe('best-effort');
		expect(await requestStoragePersistence()).toBe('best-effort');
		expect(persist).toHaveBeenCalledOnce();

		expect(await requestStoragePersistence({ retry: true })).toBe('best-effort');
		expect(persist).toHaveBeenCalledTimes(2);
	});

	it('reports unsupported when the browser has no StorageManager API', async () => {
		setStorageManager(undefined);

		expect(await getStoragePersistenceStatus()).toBe('unsupported');
		expect(await requestStoragePersistence()).toBe('unsupported');
	});

	it('reports unavailable when the persistence API rejects', async () => {
		setStorageManager({
			persisted: vi.fn().mockResolvedValue(false),
			persist: vi.fn().mockRejectedValue(new Error('Storage request failed'))
		});

		expect(await requestStoragePersistence()).toBe('unavailable');
	});
});
