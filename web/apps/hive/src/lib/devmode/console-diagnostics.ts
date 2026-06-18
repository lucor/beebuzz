import type { HiveConsoleDiagnosticEntry, HiveConsoleDiagnosticSource } from './types';
import { addConsoleDiagnostic } from './storage';
import { developerSettings } from './settings';
import { get } from 'svelte/store';
import { HIVE_CONSOLE_DIAGNOSTICS_DEDUP_MS } from './constants';

let installed = false;
let isCapturing = false;

const recentFingerprints = new Map<string, number>();

type ConsoleDiagnosticCallback = (entry: HiveConsoleDiagnosticEntry) => void;
const liveCallbacks: Set<ConsoleDiagnosticCallback> = new Set();

function emitLive(entry: HiveConsoleDiagnosticEntry): void {
	for (const cb of liveCallbacks) {
		try {
			cb(entry);
		} catch {
			// ignore callback errors
		}
	}
}

function hashFingerprint(
	level: string,
	normalizedMessage: string,
	firstStackFrame: string
): string {
	let hash = 0;
	const input = level + '::' + normalizedMessage + '::' + firstStackFrame;
	for (let i = 0; i < input.length; i++) {
		hash = (hash * 31 + input.charCodeAt(i)) >>> 0;
	}
	return hash.toString(36);
}

function isDuplicate(fingerprint: string): boolean {
	const now = Date.now();
	const last = recentFingerprints.get(fingerprint);
	if (last !== undefined && now - last < HIVE_CONSOLE_DIAGNOSTICS_DEDUP_MS) {
		return true;
	}
	recentFingerprints.set(fingerprint, now);
	if (recentFingerprints.size > 500) {
		const cutoff = now - 10000;
		for (const [key, ts] of recentFingerprints) {
			if (ts < cutoff) recentFingerprints.delete(key);
		}
	}
	return false;
}

function extractMessage(input: unknown): string {
	if (input instanceof Error) return input.message.slice(0, 4096);
	if (input === null) return 'null';
	if (input === undefined) return 'undefined';
	if (typeof input === 'string') return input.slice(0, 4096);
	if (typeof input === 'number' || typeof input === 'boolean' || typeof input === 'bigint') {
		return input.toString();
	}
	if (typeof input === 'symbol') return input.description ?? '[symbol]';
	if (typeof input === 'function') return input.name ? `[function ${input.name}]` : '[function]';
	if (Array.isArray(input)) {
		return input
			.slice(0, 3)
			.map((v) => extractMessage(v))
			.join(', ');
	}
	if (typeof input === 'object') {
		try {
			const s = JSON.stringify(input);
			return s.slice(0, 1024);
		} catch {
			return '[unserializable]';
		}
	}
	return '[unknown]';
}

function extractStack(error: unknown): string[] {
	if (error instanceof Error && error.stack) {
		const lines = error.stack
			.split('\n')
			.map((l) => l.trim())
			.filter((l) => l.length > 0)
			.slice(0, 20);
		return lines.map((l) => l.slice(0, 500));
	}
	return [];
}

function extractConsoleArgs(args: unknown[]): { message: string; stack: string[] } {
	const error = args.find((arg): arg is Error => arg instanceof Error);
	return {
		message: args.map((arg) => extractMessage(arg)).join(' '),
		stack: error ? extractStack(error) : []
	};
}

function shortId(): string {
	return crypto.randomUUID().slice(0, 12);
}

function makeEntry(
	level: 'warn' | 'error',
	source: HiveConsoleDiagnosticSource,
	message: string,
	stack: string[],
	fingerprint: string
): HiveConsoleDiagnosticEntry {
	return {
		id: shortId(),
		ts: new Date().toISOString(),
		level,
		source,
		message,
		stack: stack.length > 0 ? stack : null,
		fingerprint
	};
}

async function capture(
	level: 'warn' | 'error',
	source: HiveConsoleDiagnosticSource,
	message: string,
	stack: string[]
): Promise<void> {
	if (isCapturing) return;
	if (!get(developerSettings).enabled) return;

	const normalizedMsg = message.slice(0, 2048);
	const firstFrame = stack[0] ?? '';
	const fp = hashFingerprint(level, normalizedMsg, firstFrame);
	if (isDuplicate(fp)) return;

	isCapturing = true;
	try {
		const entry = makeEntry(level, source, normalizedMsg, stack, fp);
		await addConsoleDiagnostic(entry);
		emitLive(entry);
	} finally {
		isCapturing = false;
	}
}

export function subscribeToConsoleDiagnostics(callback: ConsoleDiagnosticCallback): () => void {
	liveCallbacks.add(callback);
	return () => liveCallbacks.delete(callback);
}

let originalConsoleWarn: typeof console.warn | null = null;
let originalConsoleError: typeof console.error | null = null;
let windowErrorHandler: ((event: ErrorEvent) => void) | null = null;
let rejectionHandler: ((event: PromiseRejectionEvent) => void) | null = null;

export function startConsoleDiagnosticsCapture(): void {
	if (installed) return;
	if (typeof window === 'undefined') return;
	installed = true;

	// window.onerror
	windowErrorHandler = (event: ErrorEvent) => {
		const msg = event.message ?? 'Script error';
		const stack = event.error instanceof Error ? extractStack(event.error) : [];
		void capture('error', 'window_error', msg, stack);
	};
	window.addEventListener('error', windowErrorHandler);

	// unhandledrejection
	rejectionHandler = (event: PromiseRejectionEvent) => {
		const msg = extractMessage(event.reason);
		const stack = event.reason instanceof Error ? extractStack(event.reason) : [];
		void capture('error', 'unhandled_rejection', msg, stack);
	};
	window.addEventListener('unhandledrejection', rejectionHandler);

	// console.warn patch
	originalConsoleWarn = console.warn;
	console.warn = function (...args: unknown[]) {
		originalConsoleWarn!.apply(console, args);
		const { message, stack } = extractConsoleArgs(args);
		void capture('warn', 'console', message, stack);
	};

	// console.error patch
	originalConsoleError = console.error;
	console.error = function (...args: unknown[]) {
		originalConsoleError!.apply(console, args);
		const { message, stack } = extractConsoleArgs(args);
		void capture('error', 'console', message, stack);
	};
}

export function stopConsoleDiagnosticsCapture(): void {
	if (!installed) return;
	installed = false;

	if (windowErrorHandler) {
		window.removeEventListener('error', windowErrorHandler);
		windowErrorHandler = null;
	}
	if (rejectionHandler) {
		window.removeEventListener('unhandledrejection', rejectionHandler);
		rejectionHandler = null;
	}
	if (originalConsoleWarn) {
		console.warn = originalConsoleWarn;
		originalConsoleWarn = null;
	}
	if (originalConsoleError) {
		console.error = originalConsoleError;
		originalConsoleError = null;
	}
	recentFingerprints.clear();
}
