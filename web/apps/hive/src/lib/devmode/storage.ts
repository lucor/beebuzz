import type { HiveLogEntry, HiveErrorSnapshot } from './types';
import {
	HIVE_DEV_DB_NAME,
	HIVE_DEV_DB_VERSION,
	HIVE_DIAGNOSTIC_MAX_EVENTS,
	HIVE_DIAGNOSTIC_RETENTION_MS
} from './constants';

function openDB(): Promise<IDBDatabase> {
	return new Promise((resolve, reject) => {
		const request = indexedDB.open(HIVE_DEV_DB_NAME, HIVE_DEV_DB_VERSION);
		request.onupgradeneeded = () => {
			const db = request.result;
			if (!db.objectStoreNames.contains('developer_settings')) {
				db.createObjectStore('developer_settings', { keyPath: 'key' });
			}
			if (!db.objectStoreNames.contains('developer_logs')) {
				const logStore = db.createObjectStore('developer_logs', { keyPath: 'id' });
				logStore.createIndex('by-ts', 'ts', { unique: false });
				logStore.createIndex('by-kind', 'kind', { unique: false });
			}
			if (!db.objectStoreNames.contains('developer_error_snapshots')) {
				const snapStore = db.createObjectStore('developer_error_snapshots', { keyPath: 'id' });
				snapStore.createIndex('by-ts', 'ts', { unique: false });
			}
		};
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => reject(new Error('Failed to open database'));
	});
}

function enforceMaxEvents(
	store: IDBObjectStore,
	items: { ts: string; id: IDBValidKey }[],
	max: number
): void {
	if (items.length <= max) return;
	const sorted = [...items].sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime());
	const toRemove = sorted.slice(max);
	for (const entry of toRemove) {
		store.delete(entry.id);
	}
}

export async function loadDeveloperSettings(): Promise<{ enabled: boolean }> {
	try {
		const db = await openDB();
		return new Promise((resolve) => {
			const tx = db.transaction('developer_settings', 'readonly');
			const store = tx.objectStore('developer_settings');
			const req = store.get('developer_mode');
			req.onsuccess = () => {
				const result = req.result as { enabled: boolean } | undefined;
				resolve(result ? { enabled: result.enabled } : { enabled: false });
			};
			req.onerror = () => resolve({ enabled: false });
		});
	} catch {
		return { enabled: false };
	}
}

export async function setDeveloperModeEnabled(enabled: boolean): Promise<void> {
	try {
		const db = await openDB();
		const tx = db.transaction('developer_settings', 'readwrite');
		const store = tx.objectStore('developer_settings');
		store.put({ key: 'developer_mode', enabled, updated_at: new Date().toISOString() });
	} catch {
		// silently fail
	}
}

export async function appendLog(entry: HiveLogEntry): Promise<void> {
	try {
		const db = await openDB();
		const tx = db.transaction('developer_logs', 'readwrite');
		const store = tx.objectStore('developer_logs');
		store.add(entry);

		const countReq = store.count();
		countReq.onsuccess = () => {
			if (countReq.result > HIVE_DIAGNOSTIC_MAX_EVENTS) {
				const allReq = store.getAll();
				allReq.onsuccess = () => {
					const items = allReq.result as { ts: string; id: IDBValidKey }[];
					enforceMaxEvents(store, items, HIVE_DIAGNOSTIC_MAX_EVENTS);
				};
			}
		};
	} catch {
		// silently fail
	}
}

export async function listLogs(kind?: 'main' | 'developer'): Promise<HiveLogEntry[]> {
	try {
		const db = await openDB();
		const cutoff = new Date(Date.now() - HIVE_DIAGNOSTIC_RETENTION_MS).toISOString();
		return new Promise((resolve) => {
			const tx = db.transaction('developer_logs', 'readonly');
			const store = tx.objectStore('developer_logs');
			const req = store.getAll();
			req.onsuccess = () => {
				const entries = (req.result as HiveLogEntry[]).filter((e) => e.ts >= cutoff);
				if (kind) {
					return resolve(
						entries
							.filter((e) => e.kind === kind)
							.sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime())
					);
				}
				resolve(entries.sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime()));
			};
			req.onerror = () => resolve([]);
		});
	} catch {
		return [];
	}
}

export async function clearLogs(): Promise<void> {
	try {
		const db = await openDB();
		const tx = db.transaction('developer_logs', 'readwrite');
		tx.objectStore('developer_logs').clear();
	} catch {
		// silently fail
	}
}

export async function appendSnapshot(snapshot: HiveErrorSnapshot): Promise<void> {
	try {
		const db = await openDB();
		const tx = db.transaction('developer_error_snapshots', 'readwrite');
		const store = tx.objectStore('developer_error_snapshots');
		store.add(snapshot);

		const countReq = store.count();
		countReq.onsuccess = () => {
			if (countReq.result > 100) {
				const allReq = store.getAll();
				allReq.onsuccess = () => {
					const items = allReq.result as { ts: string; id: IDBValidKey }[];
					enforceMaxEvents(store, items, 100);
				};
			}
		};
	} catch {
		// silently fail
	}
}

export async function listSnapshots(): Promise<HiveErrorSnapshot[]> {
	try {
		const db = await openDB();
		const cutoff = new Date(Date.now() - HIVE_DIAGNOSTIC_RETENTION_MS).toISOString();
		return new Promise((resolve) => {
			const tx = db.transaction('developer_error_snapshots', 'readonly');
			const store = tx.objectStore('developer_error_snapshots');
			const req = store.getAll();
			req.onsuccess = () => {
				let entries = req.result as HiveErrorSnapshot[];
				entries = entries.filter((e) => e.ts >= cutoff);
				entries.sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime());
				resolve(entries);
			};
			req.onerror = () => resolve([]);
		});
	} catch {
		return [];
	}
}

export async function clearSnapshots(): Promise<void> {
	try {
		const db = await openDB();
		const tx = db.transaction('developer_error_snapshots', 'readwrite');
		tx.objectStore('developer_error_snapshots').clear();
	} catch {
		// silently fail
	}
}

export async function clearAllDeveloperDiagnostics(): Promise<void> {
	await Promise.all([clearLogs(), clearSnapshots()]);
}
