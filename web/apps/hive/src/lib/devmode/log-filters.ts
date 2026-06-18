import type { HiveLogEntry } from './types';

export type HiveLogFilters = {
	scope: string;
	event: string;
	trace: string;
	notification: string;
	search: string;
};

export type HiveLogFilterOptions = {
	scopes: string[];
	events: HiveLogEntry['event'][];
	traces: string[];
	notifications: string[];
};

const byTimestampAscending = (a: HiveLogEntry, b: HiveLogEntry): number =>
	new Date(a.ts).getTime() - new Date(b.ts).getTime();

const isActiveFilter = (value: string): boolean =>
	value !== '' && value !== 'all' && value !== '__all__';

const asText = (value: unknown): string => (typeof value === 'string' ? value : '');

export function filterHiveLogs(
	storedLogs: readonly HiveLogEntry[],
	liveLogs: readonly HiveLogEntry[],
	liveEnabled: boolean,
	filters: HiveLogFilters
): HiveLogEntry[] {
	let items = liveEnabled ? [...liveLogs, ...storedLogs] : [...storedLogs];
	items.sort(byTimestampAscending);

	if (isActiveFilter(filters.scope)) {
		items = items.filter((log) => asText(log.scope) === filters.scope);
	}
	if (isActiveFilter(filters.event)) {
		items = items.filter((log) => asText(log.event) === filters.event);
	}
	if (isActiveFilter(filters.trace)) {
		items = items.filter((log) => log.data?.push_trace_id === filters.trace);
	}
	if (isActiveFilter(filters.notification)) {
		items = items.filter((log) => log.data?.notification_id === filters.notification);
	}
	if (filters.search) {
		const query = filters.search.toLowerCase();
		items = items.filter(
			(log) =>
				asText(log.event).toLowerCase().includes(query) ||
				asText(log.message).toLowerCase().includes(query) ||
				asText(log.scope).toLowerCase().includes(query) ||
				Boolean(log.data?.notification_id?.toLowerCase().includes(query)) ||
				Boolean(log.data?.push_trace_id?.toLowerCase().includes(query))
		);
	}

	return items;
}

export function listHiveLogFilterOptions(
	storedLogs: readonly HiveLogEntry[],
	liveLogs: readonly HiveLogEntry[]
): HiveLogFilterOptions {
	const all = [...liveLogs, ...storedLogs];
	return {
		scopes: [...new Set(all.map((log) => asText(log.scope)).filter(Boolean))].sort(),
		events: [
			...new Set(
				all
					.map((log) => asText(log.event))
					.filter((event): event is HiveLogEntry['event'] => Boolean(event))
			)
		].sort(),
		traces: [
			...new Set(
				all.map((log) => log.data?.push_trace_id).filter((id): id is string => Boolean(id))
			)
		]
			.sort()
			.reverse(),
		notifications: [
			...new Set(
				all.map((log) => log.data?.notification_id).filter((id): id is string => Boolean(id))
			)
		].sort()
	};
}
