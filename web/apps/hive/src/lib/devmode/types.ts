export type HiveLogScope =
	| 'app'
	| 'push'
	| 'payload'
	| 'notification'
	| 'service_worker'
	| 'storage'
	| 'pairing'
	| 'network'
	| 'outbox'
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
	'outbox',
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
	| 'notification.persisted'
	| 'notification.persist_failed'
	| 'notification.imported'
	| 'notification.clicked'
	| 'notification.displayed'
	| 'clients.notified'
	| 'clients.match_failed'
	| 'clients.focus_failed'
	| 'clients.open_window_failed'
	| 'clients.post_message_failed'
	| 'outbox.sync_started'
	| 'outbox.request_started'
	| 'outbox.response_received'
	| 'outbox.notification_resolve_started'
	| 'outbox.notification_imported'
	| 'outbox.notification_import_failed'
	| 'outbox.cursor_updated'
	| 'outbox.gap_detected'
	| 'outbox.sync_completed'
	| 'outbox.sync_failed'
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
	'notification.persisted',
	'notification.persist_failed',
	'notification.imported',
	'notification.clicked',
	'notification.displayed',
	'clients.notified',
	'clients.match_failed',
	'clients.focus_failed',
	'clients.open_window_failed',
	'clients.post_message_failed',
	'outbox.sync_started',
	'outbox.request_started',
	'outbox.response_received',
	'outbox.notification_resolve_started',
	'outbox.notification_imported',
	'outbox.notification_import_failed',
	'outbox.cursor_updated',
	'outbox.gap_detected',
	'outbox.sync_completed',
	'outbox.sync_failed',
	'debug_report.missing_device_token',
	'debug_report.submit_failed'
] as const satisfies readonly HiveDiagnosticEvent[];

export function isHiveLogScope(value: string): value is HiveLogScope {
	return (HIVE_LOG_SCOPES as readonly string[]).includes(value);
}

export function isHiveDiagnosticEvent(value: string): value is HiveDiagnosticEvent {
	return (HIVE_DIAGNOSTIC_EVENTS as readonly string[]).includes(value);
}

export const HIVE_BOUNDARY = {
	INBOUND: 'inbound',
	OUTBOUND: 'outbound',
	INTERNAL: 'internal'
} as const;

export const HIVE_TRANSPORT = {
	WEB_PUSH: 'web_push',
	HTTPS: 'https',
	POST_MESSAGE: 'post_message',
	INDEXEDDB: 'indexeddb',
	LOCAL_STORAGE: 'local_storage',
	NOTIFICATION_CENTER: 'notification_center',
	SERVICE_WORKER: 'service_worker',
	CRYPTO: 'crypto'
} as const;

export type HiveDiagnosticDescriptor = {
	scope: HiveLogScope;
	event: HiveDiagnosticEvent;
	boundary?: HiveLogData['boundary'];
	transport?: HiveLogData['transport'];
};

export const HIVE_DIAGNOSTIC = {
	APP_STARTED: {
		scope: 'app',
		event: 'app.started'
	},
	APP_BOOTSTRAP_FAILED: {
		scope: 'app',
		event: 'app.bootstrap_failed'
	},
	SERVICE_WORKER_REGISTERED: {
		scope: 'service_worker',
		event: 'service_worker.registered',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	SERVICE_WORKER_ACTIVATED: {
		scope: 'service_worker',
		event: 'service_worker.activated',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	SERVICE_WORKER_SKIP_WAITING: {
		scope: 'service_worker',
		event: 'service_worker.skip_waiting',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	PAIRING_RECONNECT_REQUIRED: {
		scope: 'pairing',
		event: 'pairing.reconnect_required'
	},
	PUSH_RECEIVED: {
		scope: 'push',
		event: 'push.received',
		boundary: HIVE_BOUNDARY.INBOUND,
		transport: HIVE_TRANSPORT.WEB_PUSH
	},
	PUSH_EMPTY_PAYLOAD: {
		scope: 'push',
		event: 'push.empty_payload',
		boundary: HIVE_BOUNDARY.INBOUND,
		transport: HIVE_TRANSPORT.WEB_PUSH
	},
	PUSH_RESOLVED: {
		scope: 'push',
		event: 'push.resolved',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	PUSH_SUBSCRIPTION_CHANGED: {
		scope: 'push',
		event: 'push.subscription_changed',
		boundary: HIVE_BOUNDARY.INBOUND,
		transport: HIVE_TRANSPORT.WEB_PUSH
	},
	PAYLOAD_RESOLVE: {
		scope: 'push',
		event: 'payload.resolve',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	PAYLOAD_DETECTED_ENCRYPTED: {
		scope: 'payload',
		event: 'payload.detected_encrypted',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.CRYPTO
	},
	PAYLOAD_DETECTED_PLAIN: {
		scope: 'payload',
		event: 'payload.detected_plain',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	PAYLOAD_DECRYPT_FAILED: {
		scope: 'payload',
		event: 'payload.decrypt_failed',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.CRYPTO
	},
	PAYLOAD_INVALID: {
		scope: 'payload',
		event: 'payload.invalid',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	STORAGE_CREDENTIALS_FAILED: {
		scope: 'storage',
		event: 'storage.credentials_failed',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.INDEXEDDB
	},
	NOTIFICATION_PERSIST_STARTED: {
		scope: 'notification',
		event: 'notification.persist_started',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.INDEXEDDB
	},
	NOTIFICATION_PERSISTED: {
		scope: 'notification',
		event: 'notification.persisted',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.INDEXEDDB
	},
	NOTIFICATION_PERSIST_FAILED: {
		scope: 'notification',
		event: 'notification.persist_failed',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.INDEXEDDB
	},
	NOTIFICATION_IMPORTED: {
		scope: 'notification',
		event: 'notification.imported',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.POST_MESSAGE
	},
	NOTIFICATION_CLICKED: {
		scope: 'notification',
		event: 'notification.clicked',
		boundary: HIVE_BOUNDARY.INBOUND,
		transport: HIVE_TRANSPORT.NOTIFICATION_CENTER
	},
	NOTIFICATION_DISPLAYED: {
		scope: 'notification',
		event: 'notification.displayed',
		boundary: HIVE_BOUNDARY.OUTBOUND,
		transport: HIVE_TRANSPORT.NOTIFICATION_CENTER
	},
	CLIENTS_NOTIFIED: {
		scope: 'service_worker',
		event: 'clients.notified',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.POST_MESSAGE
	},
	CLIENTS_MATCH_FAILED: {
		scope: 'service_worker',
		event: 'clients.match_failed',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.POST_MESSAGE
	},
	CLIENTS_FOCUS_FAILED: {
		scope: 'service_worker',
		event: 'clients.focus_failed',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	CLIENTS_OPEN_WINDOW_FAILED: {
		scope: 'service_worker',
		event: 'clients.open_window_failed',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	CLIENTS_POST_MESSAGE_FAILED: {
		scope: 'service_worker',
		event: 'clients.post_message_failed',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.POST_MESSAGE
	},
	OUTBOX_SYNC_STARTED: {
		scope: 'outbox',
		event: 'outbox.sync_started',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.HTTPS
	},
	OUTBOX_REQUEST_STARTED: {
		scope: 'outbox',
		event: 'outbox.request_started',
		boundary: HIVE_BOUNDARY.OUTBOUND,
		transport: HIVE_TRANSPORT.HTTPS
	},
	OUTBOX_RESPONSE_RECEIVED: {
		scope: 'outbox',
		event: 'outbox.response_received',
		boundary: HIVE_BOUNDARY.INBOUND,
		transport: HIVE_TRANSPORT.HTTPS
	},
	OUTBOX_NOTIFICATION_RESOLVE_STARTED: {
		scope: 'outbox',
		event: 'outbox.notification_resolve_started',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	OUTBOX_NOTIFICATION_IMPORTED: {
		scope: 'outbox',
		event: 'outbox.notification_imported',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	OUTBOX_NOTIFICATION_IMPORT_FAILED: {
		scope: 'outbox',
		event: 'outbox.notification_import_failed',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.SERVICE_WORKER
	},
	OUTBOX_CURSOR_UPDATED: {
		scope: 'outbox',
		event: 'outbox.cursor_updated',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.LOCAL_STORAGE
	},
	OUTBOX_GAP_DETECTED: {
		scope: 'outbox',
		event: 'outbox.gap_detected',
		boundary: HIVE_BOUNDARY.INBOUND,
		transport: HIVE_TRANSPORT.HTTPS
	},
	OUTBOX_SYNC_COMPLETED: {
		scope: 'outbox',
		event: 'outbox.sync_completed',
		boundary: HIVE_BOUNDARY.INTERNAL,
		transport: HIVE_TRANSPORT.HTTPS
	},
	OUTBOX_SYNC_FAILED: {
		scope: 'outbox',
		event: 'outbox.sync_failed',
		boundary: HIVE_BOUNDARY.OUTBOUND,
		transport: HIVE_TRANSPORT.HTTPS
	},
	DEBUG_REPORT_MISSING_DEVICE_TOKEN: {
		scope: 'network',
		event: 'debug_report.missing_device_token'
	},
	DEBUG_REPORT_SUBMIT_FAILED: {
		scope: 'network',
		event: 'debug_report.submit_failed'
	}
} as const satisfies Record<string, HiveDiagnosticDescriptor>;

export type HiveLogData = {
	status?: number;
	duration_ms?: number;
	route?: string;
	method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
	error_code?: string;
	notification_id?: string;
	push_trace_id?: string;
	boundary?: (typeof HIVE_BOUNDARY)[keyof typeof HIVE_BOUNDARY];
	transport?: (typeof HIVE_TRANSPORT)[keyof typeof HIVE_TRANSPORT];
	endpoint?: string;
	delivery_mode?: string;
	sync_cursor?: string;
	item_count?: number;
	page_count?: number;
	imported_count?: number;
	ok?: boolean;
};

export type HiveLogEntry = {
	id: string;
	ts: string;
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
	last_notification_received_at: string | null;
	last_notification_received_via: string | null;
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
	console_diagnostics?: HiveConsoleDiagnosticEntry[];
};

export type HiveDebugReportResponse = {
	report_id: string;
	created_at: string;
};

export type HiveConsoleDiagnosticSource = 'console' | 'window_error' | 'unhandled_rejection';

export type HiveConsoleDiagnosticLevel = 'warn' | 'error';

export type HiveConsoleDiagnosticEntry = {
	id: string;
	ts: string;
	level: HiveConsoleDiagnosticLevel;
	source: HiveConsoleDiagnosticSource;
	message: string;
	stack: string[] | null;
	fingerprint: string;
};
