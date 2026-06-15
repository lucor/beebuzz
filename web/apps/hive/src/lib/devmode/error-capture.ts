import { collectHiveSafeContext } from './context';
import { listLogs, appendSnapshot } from './storage';
import { developerSettings } from './settings';
import { get } from 'svelte/store';
import type { HiveErrorSnapshot, HiveLogScope, HiveDiagnosticEvent } from './types';

type CaptureHiveErrorInput = {
	scope: HiveLogScope;
	event: HiveDiagnosticEvent;
	message: string;
	error?: unknown;
	severity?: 'warn' | 'error';
};

function shortId(): string {
	return crypto.randomUUID().slice(0, 16);
}

function normalizeStackTrace(stack: string | undefined): string[] {
	if (!stack) return [];
	const lines = stack.split('\n').filter((line) => line.trim().length > 0);
	return lines.slice(0, 8).map((line) => {
		return line.trim().slice(0, 180);
	});
}

function getSafeErrorName(error: unknown): string {
	if (error instanceof Error && error.name.trim().length > 0) {
		return error.name;
	}
	return 'Error';
}

function getSafeErrorCode(error: unknown): string | null {
	if (!error || typeof error !== 'object') {
		return null;
	}
	const code = (error as { code?: unknown }).code;
	return typeof code === 'string' && /^[A-Za-z0-9_:-]{1,64}$/i.test(code) ? code : null;
}

function getSafeErrorStack(error: unknown): string[] | null {
	if (!(error instanceof Error)) {
		return null;
	}
	const stack = normalizeStackTrace(error.stack);
	return stack.length > 0 ? stack : null;
}

export async function captureHiveError(
	input: CaptureHiveErrorInput
): Promise<HiveErrorSnapshot | null> {
	const settings = get(developerSettings);
	if (!settings.enabled) return null;

	const context = await collectHiveSafeContext();
	const relatedLogs = await listLogs();
	const recentLogs = relatedLogs.slice(0, 20);
	const error =
		input.error === undefined
			? null
			: {
					name: getSafeErrorName(input.error),
					code: getSafeErrorCode(input.error),
					stack: getSafeErrorStack(input.error)
				};

	const snapshot: HiveErrorSnapshot = {
		id: shortId(),
		ts: new Date().toISOString(),
		scope: input.scope,
		event: input.event,
		severity: input.severity ?? 'error',
		message: input.message,
		error,
		context,
		related_logs: recentLogs
	};

	try {
		await appendSnapshot(snapshot);
	} catch {
		// silently fail
	}

	return snapshot;
}
