export const HIVE_DEV_DB_NAME = 'BeeBuzzDeveloper';
export const HIVE_DEV_DB_VERSION = 1;

export const HIVE_DIAGNOSTIC_RETENTION_MS = 24 * 60 * 60 * 1000;

export const HIVE_DIAGNOSTIC_MAX_EVENTS = 1000;

export const HIVE_BROWSER_CONSOLE_OUTPUT = false;

export const HIVE_DEV_SETTINGS_STORE = 'developer_settings';
export const HIVE_DEV_LOGS_STORE = 'developer_logs';
export const HIVE_DEV_SNAPSHOTS_STORE = 'developer_error_snapshots';

export const HIVE_EVENTS_MAIN = [
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
] as const;

export const HIVE_EVENTS_DEVELOPER = [
	'payload.detected_encrypted',
	'payload.detected_plain',
	'notification.persist_started',
	'notification.persist_failed',
	'clients.match_failed',
	'push.subscription_changed'
] as const;
