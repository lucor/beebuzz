import { beforeEach, describe, expect, it, vi } from 'vitest';
import { writable } from 'svelte/store';
import type { HiveLogEntry } from './types';

const settingsStore = writable({ enabled: true });
const appendLog = vi.fn<(entry: HiveLogEntry) => Promise<void>>(() => Promise.resolve());

vi.mock('./settings', () => ({
	developerSettings: settingsStore
}));

vi.mock('./storage', () => ({
	appendLog
}));

describe('safeLogger', () => {
	beforeEach(() => {
		appendLog.mockClear();
		settingsStore.set({ enabled: true });
	});

	it('keeps multiple safe diagnostic fields', async () => {
		const { safeLogger } = await import('./safe-logger');

		safeLogger.main('push', 'push.received', 'Push received', {
			route: '/v1/hive/debug-reports',
			status: 201,
			duration_ms: 42,
			ok: true
		});

		expect(appendLog).toHaveBeenCalledWith(
			expect.objectContaining({
				data: {
					route: '/v1/hive/debug-reports',
					status: 201,
					duration_ms: 42,
					ok: true
				}
			})
		);
	});

	it('drops unknown and wrong-type diagnostic fields', async () => {
		const { safeLogger } = await import('./safe-logger');

		safeLogger.main('push', 'push.received', 'Push received', {
			route: '/v1/hive/debug-reports',
			status: '201',
			token: 'secret'
		} as never);

		expect(appendLog).toHaveBeenCalledWith(
			expect.objectContaining({
				data: {
					route: '/v1/hive/debug-reports'
				}
			})
		);
	});

	it('omits data when no safe fields remain', async () => {
		const { safeLogger } = await import('./safe-logger');

		safeLogger.main('push', 'push.received', 'Push received', {
			status: '201',
			token: 'secret'
		} as never);

		const callArg = appendLog.mock.calls[0]?.[0];
		expect(callArg?.data).toBeUndefined();
	});
});
