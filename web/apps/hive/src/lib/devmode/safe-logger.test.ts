import { beforeEach, describe, expect, it, vi } from 'vitest';
import { writable } from 'svelte/store';
import type { HiveLogEntry } from './types';
import { HIVE_DIAGNOSTIC, HIVE_BOUNDARY, HIVE_TRANSPORT } from './types';

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

		safeLogger.log(HIVE_DIAGNOSTIC.PUSH_RECEIVED, 'Push received', {
			route: '/v1/hive/debug-reports',
			status: 201,
			duration_ms: 42,
			ok: true
		});

		expect(appendLog).toHaveBeenCalledWith(
			expect.objectContaining({
				data: {
					boundary: HIVE_BOUNDARY.INBOUND,
					transport: HIVE_TRANSPORT.WEB_PUSH,
					route: '/v1/hive/debug-reports',
					status: 201,
					duration_ms: 42,
					ok: true
				}
			})
		);
		expect(appendLog.mock.calls[0]?.[0]).not.toHaveProperty('kind');
	});

	it('drops unknown and wrong-type diagnostic fields', async () => {
		const { safeLogger } = await import('./safe-logger');

		safeLogger.log(HIVE_DIAGNOSTIC.PUSH_RECEIVED, 'Push received', {
			route: '/v1/hive/debug-reports',
			status: '201',
			token: 'secret'
		} as never);

		expect(appendLog.mock.calls[0]?.[0]?.data).toMatchObject({
			boundary: HIVE_BOUNDARY.INBOUND,
			transport: HIVE_TRANSPORT.WEB_PUSH,
			route: '/v1/hive/debug-reports'
		});
	});

	it('omits data when no safe fields remain', async () => {
		const { safeLogger } = await import('./safe-logger');

		safeLogger.log(HIVE_DIAGNOSTIC.PUSH_RECEIVED, 'Push received', {
			status: '201',
			token: 'secret'
		} as never);

		const callArg = appendLog.mock.calls[0]?.[0];
		expect(callArg?.data).toEqual({
			boundary: HIVE_BOUNDARY.INBOUND,
			transport: HIVE_TRANSPORT.WEB_PUSH
		});
	});

	it('applies descriptor boundary and transport defaults', async () => {
		const { safeLogger } = await import('./safe-logger');

		safeLogger.log(HIVE_DIAGNOSTIC.NOTIFICATION_DISPLAYED, 'Shown');

		expect(appendLog.mock.calls[0]?.[0]?.data).toMatchObject({
			boundary: HIVE_BOUNDARY.OUTBOUND,
			transport: HIVE_TRANSPORT.NOTIFICATION_CENTER
		});
	});

	it('allows runtime transport override', async () => {
		const { safeLogger } = await import('./safe-logger');

		safeLogger.log(HIVE_DIAGNOSTIC.OUTBOX_NOTIFICATION_RESOLVE_STARTED, 'Resolving', {
			transport: HIVE_TRANSPORT.CRYPTO
		});

		expect(appendLog.mock.calls[0]?.[0]?.data).toMatchObject({
			boundary: HIVE_BOUNDARY.INTERNAL,
			transport: HIVE_TRANSPORT.CRYPTO
		});
	});
});
