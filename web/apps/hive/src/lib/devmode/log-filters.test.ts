import { describe, expect, it } from 'vitest';
import type { HiveLogEntry } from './types';
import { filterHiveLogs, listHiveLogFilterOptions } from './log-filters';

const logEntry = (entry: Partial<HiveLogEntry> & Pick<HiveLogEntry, 'id'>): HiveLogEntry => ({
	ts: '2026-06-18T10:00:00.000Z',
	scope: 'app',
	event: 'app.started',
	message: 'Hive app ready',
	...entry
});

const emptyFilters = {
	scope: '',
	event: '',
	trace: '',
	notification: '',
	search: ''
};

const allSentinelFilters = {
	scope: 'all',
	event: 'all',
	trace: 'all',
	notification: 'all',
	search: ''
};

describe('developer log filters', () => {
	const storedLogs: HiveLogEntry[] = [
		logEntry({
			id: 'stored-app',
			ts: '2026-06-18T10:00:01.000Z',
			scope: 'app',
			event: 'app.started',
			message: 'Hive app ready'
		}),
		logEntry({
			id: 'stored-outbox',
			ts: '2026-06-18T10:00:03.000Z',
			scope: 'outbox',
			event: 'outbox.request_started',
			message: 'Requesting outbox notifications',
			data: {
				notification_id: 'notification-1',
				push_trace_id: 'trace-1',
				method: 'GET',
				endpoint: '/v1/devices/{device_id}/notifications'
			}
		})
	];

	const liveLogs: HiveLogEntry[] = [
		logEntry({
			id: 'live-push',
			ts: '2026-06-18T10:00:02.000Z',
			scope: 'push',
			event: 'push.received',
			message: 'Push event received',
			data: {
				notification_id: 'notification-2',
				push_trace_id: 'trace-2'
			}
		})
	];

	it('combines live and stored logs in timestamp order without mutating inputs', () => {
		const storedBefore = storedLogs.map((log) => log.id);

		const result = filterHiveLogs(storedLogs, liveLogs, true, emptyFilters);

		expect(result.map((log) => log.id)).toEqual(['stored-app', 'live-push', 'stored-outbox']);
		expect(storedLogs.map((log) => log.id)).toEqual(storedBefore);
	});

	it('excludes live logs when live mode is off', () => {
		const result = filterHiveLogs(storedLogs, liveLogs, false, emptyFilters);

		expect(result.map((log) => log.id)).toEqual(['stored-app', 'stored-outbox']);
	});

	it('filters by scope and event together', () => {
		const result = filterHiveLogs(storedLogs, liveLogs, true, {
			...emptyFilters,
			scope: 'outbox',
			event: 'outbox.request_started'
		});

		expect(result.map((log) => log.id)).toEqual(['stored-outbox']);
	});

	it('treats empty filter values as no-op', () => {
		const result = filterHiveLogs(storedLogs, liveLogs, true, emptyFilters);
		expect(result.map((log) => log.id)).toEqual(['stored-app', 'live-push', 'stored-outbox']);
	});

	it('treats all sentinel filter values as no-op', () => {
		const result = filterHiveLogs(storedLogs, liveLogs, true, allSentinelFilters);
		expect(result.map((log) => log.id)).toEqual(['stored-app', 'live-push', 'stored-outbox']);
	});

	it('clears one select filter while keeping other filters active', () => {
		expect(
			filterHiveLogs(storedLogs, liveLogs, true, {
				...emptyFilters,
				scope: 'all',
				event: 'outbox.request_started'
			}).map((log) => log.id)
		).toEqual(['stored-outbox']);
		expect(
			filterHiveLogs(storedLogs, liveLogs, true, {
				...emptyFilters,
				scope: 'outbox',
				event: 'all'
			}).map((log) => log.id)
		).toEqual(['stored-outbox']);
	});

	it('filters by trace and notification metadata', () => {
		const result = filterHiveLogs(storedLogs, liveLogs, true, {
			...emptyFilters,
			trace: 'trace-1',
			notification: 'notification-1'
		});

		expect(result.map((log) => log.id)).toEqual(['stored-outbox']);
	});

	it('searches event, message, scope, notification id, and trace id', () => {
		expect(
			filterHiveLogs(storedLogs, liveLogs, true, { ...emptyFilters, search: 'push.received' }).map(
				(log) => log.id
			)
		).toEqual(['live-push']);
		expect(
			filterHiveLogs(storedLogs, liveLogs, true, { ...emptyFilters, search: 'Requesting' }).map(
				(log) => log.id
			)
		).toEqual(['stored-outbox']);
		expect(
			filterHiveLogs(storedLogs, liveLogs, true, { ...emptyFilters, search: 'notification-2' }).map(
				(log) => log.id
			)
		).toEqual(['live-push']);
		expect(
			filterHiveLogs(storedLogs, liveLogs, true, { ...emptyFilters, search: 'trace-1' }).map(
				(log) => log.id
			)
		).toEqual(['stored-outbox']);
	});

	it('keeps search active when select filters are all sentinels', () => {
		const result = filterHiveLogs(storedLogs, liveLogs, true, {
			...allSentinelFilters,
			search: 'push'
		});

		expect(result.map((log) => log.id)).toEqual(['live-push']);
	});

	it('does not throw when older stored logs are missing typed fields at runtime', () => {
		const malformedLog = {
			id: 'stored-legacy',
			ts: '2026-06-18T10:00:04.000Z',
			scope: 'app',
			message: 'Legacy diagnostic row'
		} as unknown as HiveLogEntry;

		expect(() =>
			filterHiveLogs([...storedLogs, malformedLog], liveLogs, true, emptyFilters)
		).not.toThrow();
		expect(
			filterHiveLogs([...storedLogs, malformedLog], liveLogs, true, {
				...emptyFilters,
				search: 'legacy'
			}).map((log) => log.id)
		).toEqual(['stored-legacy']);
	});

	it('lists filter options from live and stored logs', () => {
		const options = listHiveLogFilterOptions(storedLogs, liveLogs);

		expect(options.scopes).toEqual(['app', 'outbox', 'push']);
		expect(options.events).toEqual(['app.started', 'outbox.request_started', 'push.received']);
		expect(options.notifications).toEqual(['notification-1', 'notification-2']);
		expect(options.traces).toEqual(['trace-2', 'trace-1']);
	});
});
