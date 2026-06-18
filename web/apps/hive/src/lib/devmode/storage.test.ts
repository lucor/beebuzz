import { describe, expect, it } from 'vitest';
import {
	addConsoleDiagnostic,
	appendLog,
	appendSnapshot,
	deleteDeveloperDatabase,
	listConsoleDiagnostics,
	listLogs,
	listSnapshots,
	loadDeveloperSettings,
	setDeveloperModeEnabled
} from './storage';
import type { HiveLogEntry, HiveErrorSnapshot, HiveConsoleDiagnosticEntry } from './types';

function makeLog(over: Partial<HiveLogEntry> = {}): HiveLogEntry {
	return {
		id: crypto.randomUUID(),
		ts: new Date().toISOString(),
		scope: 'app',
		event: 'app.started',
		message: 'test',
		...over
	};
}

function makeSnapshot(over: Partial<HiveErrorSnapshot> = {}): HiveErrorSnapshot {
	return {
		id: crypto.randomUUID(),
		ts: new Date().toISOString(),
		scope: 'app',
		event: 'app.started',
		message: 'test snapshot',
		severity: 'error',
		context: { browser: 'Chrome', os: 'mac' },
		related_logs: [],
		...over
	} as HiveErrorSnapshot;
}

function makeDiagnostic(
	over: Partial<HiveConsoleDiagnosticEntry> = {}
): HiveConsoleDiagnosticEntry {
	return {
		id: crypto.randomUUID().slice(0, 12),
		ts: new Date().toISOString(),
		level: 'error',
		source: 'console',
		message: 'test diagnostic',
		stack: null,
		fingerprint: 'abc',
		...over
	};
}

describe('deleteDeveloperDatabase', () => {
	it('returns disabled settings after deletion', async () => {
		await setDeveloperModeEnabled(true);
		const before = await loadDeveloperSettings();
		expect(before.enabled).toBe(true);

		await deleteDeveloperDatabase();

		const after = await loadDeveloperSettings();
		expect(after.enabled).toBe(false);
	});

	it('clears logs after deletion', async () => {
		await appendLog(makeLog({ id: 'log1' }));
		await deleteDeveloperDatabase();

		const logs = await listLogs();
		expect(logs).toEqual([]);
	});

	it('clears snapshots after deletion', async () => {
		await appendSnapshot(makeSnapshot({ id: 'ss1' }));
		await deleteDeveloperDatabase();

		const snaps = await listSnapshots();
		expect(snaps).toEqual([]);
	});

	it('clears console diagnostics after deletion', async () => {
		await addConsoleDiagnostic(makeDiagnostic({ id: 'diag1' }));
		await deleteDeveloperDatabase();

		const diags = await listConsoleDiagnostics();
		expect(diags).toEqual([]);
	});

	it('does not fail when called on a non-existent database', async () => {
		await deleteDeveloperDatabase();
		await deleteDeveloperDatabase();

		const logs = await listLogs();
		expect(logs).toEqual([]);
	});

	it('allows re-initialization after deletion', async () => {
		await appendLog(makeLog({ id: 'log_a' }));
		await deleteDeveloperDatabase();

		await appendLog(makeLog({ id: 'log_b' }));
		const logs = await listLogs();
		expect(logs.length).toBe(1);
		expect(logs[0].id).toBe('log_b');
	});
});
