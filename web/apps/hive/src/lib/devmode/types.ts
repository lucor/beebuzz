export type HiveDiagnosticKind = 'main' | 'developer';

export const HIVE_DIAGNOSTIC_KINDS = ['main', 'developer'] as const;

export type HiveLogScope =
	| 'app'
	| 'push'
	| 'payload'
	| 'notification'
	| 'service_worker'
	| 'storage'
	| 'pairing'
	| 'network'
	| 'encryption';

export const HIVE_LOG_SCOPES = [
	'app',
	'push',
	'payload',
	'notification',
	'service_worker',
	'storage',
	'pairing',
	'network',
	'encryption'
] as const;

export type HiveDiagnosticEvent =
	| 'app.started'
	| 'app.bootstrap_failed'
	| 'service_worker.registered'
	| 'service_worker.activated'
	| 'service_worker.skip_waiting'
	| 'pairing.reconnect_required'
	| 'push.received'
	| 'push.empty_payload'
	| 'push.resolved'
	| 'push.subscription_changed'
	| 'payload.resolve'
	| 'payload.detected_encrypted'
	| 'payload.detected_plain'
	| 'payload.decrypt_failed'
	| 'payload.invalid'
	| 'storage.credentials_failed'
	| 'notification.persist_started'
	| 'notification.persist_failed'
	| 'notification.displayed'
	| 'clients.match_failed'
	| 'clients.focus_failed'
	| 'clients.open_window_failed'
	| 'clients.post_message_failed'
	| 'debug_report.missing_device_token'
	| 'debug_report.submit_failed';

export const HIVE_DIAGNOSTIC_EVENTS = [
	'app.started',
	'app.bootstrap_failed',
	'service_worker.registered',
	'service_worker.activated',
	'service_worker.skip_waiting',
	'pairing.reconnect_required',
	'push.received',
	'push.empty_payload',
	'push.resolved',
	'push.subscription_changed',
	'payload.resolve',
	'payload.detected_encrypted',
	'payload.detected_plain',
	'payload.decrypt_failed',
	'payload.invalid',
	'storage.credentials_failed',
	'notification.persist_started',
	'notification.persist_failed',
	'notification.displayed',
	'clients.match_failed',
	'clients.focus_failed',
	'clients.open_window_failed',
	'clients.post_message_failed',
	'debug_report.missing_device_token',
	'debug_report.submit_failed'
] as const satisfies readonly HiveDiagnosticEvent[];

export function isHiveDiagnosticKind(value: string): value is HiveDiagnosticKind {
	return (HIVE_DIAGNOSTIC_KINDS as readonly string[]).includes(value);
}

export function isHiveLogScope(value: string): value is HiveLogScope {
	return (HIVE_LOG_SCOPES as readonly string[]).includes(value);
}

export function isHiveDiagnosticEvent(value: string): value is HiveDiagnosticEvent {
	return (HIVE_DIAGNOSTIC_EVENTS as readonly string[]).includes(value);
}

export type HiveLogData = {
	status?: number;
	duration_ms?: number;
	route?: string;
	method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
	error_code?: string;
	ok?: boolean;
};

export type HiveLogEntry = {
	id: string;
	ts: string;
	kind: HiveDiagnosticKind;
	scope: HiveLogScope;
	event: HiveDiagnosticEvent;
	message: string;
	data?: HiveLogData;
};

export type HiveSafeContext = {
	app_version: string;
	build_id: string;
	browser_family: string;
	browser_version_major: string;
	os: string;
	display_mode: string;
	notification_permission: string;
	service_worker_supported: boolean;
	service_worker_state: string;
	push_supported: boolean;
	push_present: boolean;
	webcrypto_supported: boolean;
	x25519_supported: boolean;
	indexeddb_available: boolean;
	network_online: boolean;
	last_push_received_at: string | null;
	last_notification_displayed_at: string | null;
};

export type HiveErrorSnapshot = {
	id: string;
	ts: string;
	scope: HiveLogScope;
	event: HiveDiagnosticEvent;
	severity: 'warn' | 'error';
	message: string;
	error: {
		name: string;
		code: string | null;
		stack: string[] | null;
	} | null;
	context: HiveSafeContext;
	related_logs: HiveLogEntry[];
};

export type HiveDebugReport = {
	schema_version: number;
	source: 'hive';
	report_type: 'manual_error_report';
	created_at: string;
	snapshot: HiveErrorSnapshot;
};

export type HiveDebugReportResponse = {
	report_id: string;
	created_at: string;
};
