import type { HiveLogEntry, HiveErrorSnapshot, HiveConsoleDiagnosticEntry } from './types';
import {
	HIVE_DEV_DB_NAME,
	HIVE_DEV_DB_VERSION,
	HIVE_DIAGNOSTIC_MAX_EVENTS,
	HIVE_DIAGNOSTIC_RETENTION_MS,
	HIVE_CONSOLE_DIAGNOSTICS_MAX_ENTRIES,
	HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE,
	HIVE_DEV_LOGS_STORE,
	HIVE_DEV_SETTINGS_STORE,
	HIVE_DEV_SNAPSHOTS_STORE
} from './constants';

let dbPromise: Promise<IDBDatabase> | null = null;
const HIVE_DEV_STORES = [
	HIVE_DEV_SETTINGS_STORE,
	HIVE_DEV_LOGS_STORE,
	HIVE_DEV_SNAPSHOTS_STORE,
	HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE
] as const;

function openDB(): Promise<IDBDatabase> {
	if (dbPromise) return dbPromise;
	dbPromise = new Promise((resolve, reject) => {
		const request = indexedDB.open(HIVE_DEV_DB_NAME, HIVE_DEV_DB_VERSION);
		request.onupgradeneeded = () => {
			const db = request.result;
			if (!db.objectStoreNames.contains(HIVE_DEV_SETTINGS_STORE)) {
				db.createObjectStore(HIVE_DEV_SETTINGS_STORE, { keyPath: 'key' });
			}
			if (!db.objectStoreNames.contains(HIVE_DEV_LOGS_STORE)) {
				const logStore = db.createObjectStore(HIVE_DEV_LOGS_STORE, { keyPath: 'id' });
				logStore.createIndex('by-ts', 'ts', { unique: false });
			}
			if (!db.objectStoreNames.contains(HIVE_DEV_SNAPSHOTS_STORE)) {
				const snapStore = db.createObjectStore(HIVE_DEV_SNAPSHOTS_STORE, { keyPath: 'id' });
				snapStore.createIndex('by-ts', 'ts', { unique: false });
			}
			if (!db.objectStoreNames.contains(HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE)) {
				const consoleStore = db.createObjectStore(HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE, {
					keyPath: 'id'
				});
				consoleStore.createIndex('by-ts', 'ts', { unique: false });
			}
		};
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => {
			dbPromise = null;
			reject(new Error('Failed to open database'));
		};
	});
	return dbPromise;
}

export async function closeDatabase(): Promise<void> {
	if (!dbPromise) return;
	try {
		const db = await dbPromise;
		db.close();
	} catch {
		// ignore
	}
	dbPromise = null;
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
			const tx = db.transaction(HIVE_DEV_SETTINGS_STORE, 'readonly');
			const store = tx.objectStore(HIVE_DEV_SETTINGS_STORE);
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
		await new Promise<void>((resolve, reject) => {
			const tx = db.transaction(HIVE_DEV_SETTINGS_STORE, 'readwrite');
			const store = tx.objectStore(HIVE_DEV_SETTINGS_STORE);
			store.put({ key: 'developer_mode', enabled, updated_at: new Date().toISOString() });
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error('Failed to store developer settings'));
			tx.onabort = () => reject(new Error('Developer settings transaction aborted'));
		});
	} catch {
		// silently fail
	}
}

export async function appendLog(entry: HiveLogEntry): Promise<void> {
	try {
		const db = await openDB();
		const tx = db.transaction(HIVE_DEV_LOGS_STORE, 'readwrite');
		const store = tx.objectStore(HIVE_DEV_LOGS_STORE);
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

export async function listLogs(): Promise<HiveLogEntry[]> {
	try {
		const db = await openDB();
		const cutoff = new Date(Date.now() - HIVE_DIAGNOSTIC_RETENTION_MS).toISOString();
		return new Promise((resolve) => {
			const tx = db.transaction(HIVE_DEV_LOGS_STORE, 'readonly');
			const store = tx.objectStore(HIVE_DEV_LOGS_STORE);
			const req = store.getAll();
			req.onsuccess = () => {
				const entries = (req.result as HiveLogEntry[]).filter((e) => e.ts >= cutoff);
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
		const tx = db.transaction(HIVE_DEV_LOGS_STORE, 'readwrite');
		tx.objectStore(HIVE_DEV_LOGS_STORE).clear();
	} catch {
		// silently fail
	}
}

export async function appendSnapshot(snapshot: HiveErrorSnapshot): Promise<void> {
	try {
		const db = await openDB();
		const tx = db.transaction(HIVE_DEV_SNAPSHOTS_STORE, 'readwrite');
		const store = tx.objectStore(HIVE_DEV_SNAPSHOTS_STORE);
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
			const tx = db.transaction(HIVE_DEV_SNAPSHOTS_STORE, 'readonly');
			const store = tx.objectStore(HIVE_DEV_SNAPSHOTS_STORE);
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
		const tx = db.transaction(HIVE_DEV_SNAPSHOTS_STORE, 'readwrite');
		tx.objectStore(HIVE_DEV_SNAPSHOTS_STORE).clear();
	} catch {
		// silently fail
	}
}

export async function clearAllDeveloperDiagnostics(): Promise<void> {
	await Promise.all([clearLogs(), clearSnapshots(), clearConsoleDiagnostics()]);
}

async function clearDeveloperDatabaseData(): Promise<void> {
	const db = await openDB();
	const stores = HIVE_DEV_STORES.filter((store) => db.objectStoreNames.contains(store));
	if (stores.length === 0) return;

	await new Promise<void>((resolve, reject) => {
		const tx = db.transaction(stores, 'readwrite');
		for (const store of stores) {
			tx.objectStore(store).clear();
		}
		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(new Error('Failed to clear developer database data'));
		tx.onabort = () => reject(new Error('Developer database clear transaction aborted'));
	});
}

export async function deleteDeveloperDatabase(): Promise<void> {
	if (typeof indexedDB === 'undefined') return;
	await clearDeveloperDatabaseData();
	await closeDatabase();
	await new Promise<void>((resolve, reject) => {
		const request = indexedDB.deleteDatabase(HIVE_DEV_DB_NAME);
		let settled = false;
		request.onerror = () => {
			settled = true;
			reject(new Error('Failed to delete developer database'));
		};
		request.onblocked = () => {
			settled = true;
			reject(new Error('Developer database deletion blocked by active connection'));
		};
		request.onsuccess = () => {
			settled = true;
			resolve();
		};
		setTimeout(() => {
			if (!settled) {
				settled = true;
				reject(new Error('Timed out deleting developer database'));
			}
		}, 2000);
	});
}

export async function addConsoleDiagnostic(entry: HiveConsoleDiagnosticEntry): Promise<void> {
	try {
		const db = await openDB();
		await new Promise<void>((resolve, reject) => {
			const tx = db.transaction(HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE, 'readwrite');
			const store = tx.objectStore(HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE);
			store.add(entry);

			const countReq = store.count();
			countReq.onsuccess = () => {
				if (countReq.result > HIVE_CONSOLE_DIAGNOSTICS_MAX_ENTRIES) {
					const allReq = store.getAll();
					allReq.onsuccess = () => {
						const items = allReq.result as { ts: string; id: IDBValidKey }[];
						enforceMaxEvents(store, items, HIVE_CONSOLE_DIAGNOSTICS_MAX_ENTRIES);
					};
				}
			};
			tx.oncomplete = () => resolve();
			tx.onerror = () => reject(new Error('Failed to store console diagnostic'));
			tx.onabort = () => reject(new Error('Console diagnostic transaction aborted'));
		});
	} catch {
		// silently fail
	}
}

export async function listConsoleDiagnostics(
	limit?: number
): Promise<HiveConsoleDiagnosticEntry[]> {
	try {
		const db = await openDB();
		const cutoff = new Date(Date.now() - HIVE_DIAGNOSTIC_RETENTION_MS).toISOString();
		return new Promise((resolve) => {
			const tx = db.transaction(HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE, 'readonly');
			const store = tx.objectStore(HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE);
			const req = store.getAll();
			req.onsuccess = () => {
				let entries = req.result as HiveConsoleDiagnosticEntry[];
				entries = entries.filter((e) => e.ts >= cutoff);
				entries.sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime());
				const cap = limit ?? entries.length;
				resolve(entries.slice(0, cap));
			};
			req.onerror = () => resolve([]);
		});
	} catch {
		return [];
	}
}

export async function clearConsoleDiagnostics(): Promise<void> {
	try {
		const db = await openDB();
		const tx = db.transaction(HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE, 'readwrite');
		tx.objectStore(HIVE_DEV_CONSOLE_DIAGNOSTICS_STORE).clear();
	} catch {
		// silently fail
	}
}
