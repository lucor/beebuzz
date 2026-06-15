import type {
	HiveDiagnosticKind,
	HiveLogScope,
	HiveDiagnosticEvent,
	HiveLogData,
	HiveLogEntry
} from './types';

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

function createLogEntry(
	kind: HiveDiagnosticKind,
	scope: HiveLogScope,
	event: HiveDiagnosticEvent,
	message: string,
	data?: HiveLogData
): HiveLogEntry {
	const validatedData = validateData(data);
	return {
		id: shortId(),
		ts: new Date().toISOString(),
		kind,
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
	main(scope: HiveLogScope, event: HiveDiagnosticEvent, message: string, data?: HiveLogData): void {
		const settings = get(developerSettings);
		if (!settings.enabled) return;

		const entry = createLogEntry('main', scope, event, message, data);
		void persist(entry);
	},

	developer(
		scope: HiveLogScope,
		event: HiveDiagnosticEvent,
		message: string,
		data?: HiveLogData
	): void {
		const settings = get(developerSettings);
		if (!settings.enabled) return;

		const entry = createLogEntry('developer', scope, event, message, data);
		void persist(entry);
	}
};
