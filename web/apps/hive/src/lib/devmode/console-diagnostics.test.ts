import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest';
import { addConsoleDiagnostic, listConsoleDiagnostics, clearConsoleDiagnostics } from './storage';
import { HIVE_CONSOLE_DIAGNOSTICS_MAX_ENTRIES } from './constants';
import {
	startConsoleDiagnosticsCapture,
	stopConsoleDiagnosticsCapture
} from './console-diagnostics';
import { developerSettings } from './settings';
import type { HiveConsoleDiagnosticEntry } from './types';

function makeEntry(over: Partial<HiveConsoleDiagnosticEntry> = {}): HiveConsoleDiagnosticEntry {
	return {
		id: crypto.randomUUID().slice(0, 12),
		ts: new Date().toISOString(),
		level: 'error',
		source: 'console',
		message: 'test message',
		stack: null,
		fingerprint: 'abc123',
		...over
	};
}

async function waitForDiagnostics(count: number): Promise<HiveConsoleDiagnosticEntry[]> {
	for (let attempt = 0; attempt < 20; attempt += 1) {
		const entries = await listConsoleDiagnostics();
		if (entries.length >= count) return entries;
		await new Promise((resolve) => setTimeout(resolve, 10));
	}
	return listConsoleDiagnostics();
}

beforeEach(async () => {
	stopConsoleDiagnosticsCapture();
	developerSettings.set({ enabled: true });
	await clearConsoleDiagnostics();
});

afterEach(() => {
	stopConsoleDiagnosticsCapture();
	vi.restoreAllMocks();
});

describe('console diagnostics storage', () => {
	it('stores and retrieves entries', async () => {
		const entry = makeEntry({ id: 'e1', message: 'test error' });
		await addConsoleDiagnostic(entry);

		const result = await listConsoleDiagnostics();
		expect(result.length).toBe(1);
		expect(result[0].id).toBe('e1');
		expect(result[0].message).toBe('test error');
	});

	it('returns entries in reverse chronological order', async () => {
		await addConsoleDiagnostic(
			makeEntry({ id: 'a', ts: new Date(Date.now() - 1000).toISOString() })
		);
		await addConsoleDiagnostic(makeEntry({ id: 'b', ts: new Date().toISOString() }));

		const result = await listConsoleDiagnostics();
		expect(result[0].id).toBe('b');
		expect(result[1].id).toBe('a');
	});

	it('respects limit parameter', async () => {
		await addConsoleDiagnostic(makeEntry({ id: 'a' }));
		await addConsoleDiagnostic(makeEntry({ id: 'b' }));
		await addConsoleDiagnostic(makeEntry({ id: 'c' }));

		const result = await listConsoleDiagnostics(2);
		expect(result.length).toBe(2);
	});

	it('caps at max entries (oldest removed)', async () => {
		const now = Date.now();
		for (let i = 0; i < HIVE_CONSOLE_DIAGNOSTICS_MAX_ENTRIES + 20; i++) {
			await addConsoleDiagnostic(makeEntry({ id: `e${i}`, ts: new Date(now + i).toISOString() }));
		}

		const result = await listConsoleDiagnostics();
		expect(result.length).toBeLessThanOrEqual(HIVE_CONSOLE_DIAGNOSTICS_MAX_ENTRIES);
		expect(result.some((entry) => entry.id === 'e0')).toBe(false);
	});

	it('returns empty array when no data', async () => {
		const result = await listConsoleDiagnostics();
		expect(result).toEqual([]);
	});

	it('clears all entries', async () => {
		await addConsoleDiagnostic(makeEntry({ id: 'e1' }));
		await clearConsoleDiagnostics();
		const result = await listConsoleDiagnostics();
		expect(result).toEqual([]);
	});

	it('captures console.error context, error message, and stack from all args', async () => {
		vi.spyOn(console, 'error').mockImplementation(() => {});
		startConsoleDiagnosticsCapture();

		const error = new Error('boom');
		console.error('Failed:', error);

		const result = await waitForDiagnostics(1);
		expect(result.length).toBe(1);
		expect(result[0].level).toBe('error');
		expect(result[0].message).toBe('Failed: boom');
		expect(result[0].stack?.[0]).toContain('boom');
	});
});
