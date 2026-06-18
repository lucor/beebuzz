import type { HiveLogScope, HiveDiagnosticEvent, HiveLogData, HiveLogEntry } from './types';
import type { HiveDiagnosticDescriptor } from './types';

function shortId(): string {
	return crypto.randomUUID().slice(0, 12);
}
import { appendLog } from './storage';
import { developerSettings } from './settings';
import { get } from 'svelte/store';

type LiveCallback = (entry: HiveLogEntry) => void;

const liveCallbacks: Set<LiveCallback> = new Set();
const SAFE_DATA_SCHEMA = {
	status: 'number',
	duration_ms: 'number',
	route: 'string',
	method: 'string',
	error_code: 'string',
	notification_id: 'string',
	push_trace_id: 'string',
	boundary: 'string',
	transport: 'string',
	endpoint: 'string',
	delivery_mode: 'string',
	sync_cursor: 'string',
	item_count: 'number',
	page_count: 'number',
	imported_count: 'number',
	ok: 'boolean'
} satisfies Record<keyof HiveLogData, 'string' | 'number' | 'boolean'>;

export function subscribeToLogs(callback: LiveCallback): () => void {
	liveCallbacks.add(callback);
	return () => liveCallbacks.delete(callback);
}

function emitLive(entry: HiveLogEntry): void {
	for (const cb of liveCallbacks) {
		try {
			cb(entry);
		} catch {
			// ignore callback errors
		}
	}
}

function validateData(data?: HiveLogData): HiveLogData | undefined {
	if (!data) return undefined;
	const result: HiveLogData = {};
	for (const [key, value] of Object.entries(data) as [
		keyof HiveLogData,
		HiveLogData[keyof HiveLogData]
	][]) {
		if (value === undefined) continue;
		if (typeof value !== SAFE_DATA_SCHEMA[key]) continue;
		result[key] = value as never;
	}
	return Object.keys(result).length > 0 ? result : undefined;
}

function mergeDiagnosticData(
	descriptor: HiveDiagnosticDescriptor,
	data?: HiveLogData
): HiveLogData | undefined {
	const merged: HiveLogData = {
		boundary: descriptor.boundary,
		transport: descriptor.transport,
		...data
	};
	return validateData(merged);
}

function createLogEntry(
	scope: HiveLogScope,
	event: HiveDiagnosticEvent,
	message: string,
	data?: HiveLogData
): HiveLogEntry {
	const validatedData = validateData(data);
	return {
		id: shortId(),
		ts: new Date().toISOString(),
		scope,
		event,
		message,
		data: validatedData
	};
}

async function persist(entry: HiveLogEntry): Promise<void> {
	const settings = get(developerSettings);
	if (!settings.enabled) return;

	try {
		await appendLog(entry);
	} catch {
		// silently fail
	}

	emitLive(entry);
}

export const safeLogger = {
	log(diagnostic: HiveDiagnosticDescriptor, message: string, data?: HiveLogData): void {
		const settings = get(developerSettings);
		if (!settings.enabled) return;

		const entry = createLogEntry(
			diagnostic.scope,
			diagnostic.event,
			message,
			mergeDiagnosticData(diagnostic, data)
		);
		void persist(entry);
	}
};
