import type { PushMessage } from '@beebuzz/shared/types';
import { HIVE_BOUNDARY, HIVE_TRANSPORT, HIVE_DIAGNOSTIC } from './lib/devmode/types';
import type { HiveDiagnosticDescriptor, HiveLogData } from './lib/devmode/types';
import {
	DeviceIdentityIntegrityError,
	MissingDeviceIdentityError
} from './lib/services/encryption';
import { recordNotificationReceived } from './lib/services/runtime-metadata-repository';

export type NotificationAttachmentEnvelope = {
	data?: string;
	mime?: string;
	token?: string;
	filename?: string;
};

export type NotificationPayload = {
	id: string;
	title: string;
	body?: string;
	topic_id?: string;
	topic?: string;
	tag?: string;
	sent_at: string;
	priority?: string;
	attachment?: NotificationAttachmentEnvelope;
};

type E2EEnvelope = {
	beebuzz?: {
		id: string;
		token: string;
		sent_at: string;
	};
};

type PushPayloadErrorKind = 'encrypted' | 'plain';

class PushPayloadError extends Error {
	kind: PushPayloadErrorKind;
	cause?: unknown;

	constructor(kind: PushPayloadErrorKind, message: string, cause?: unknown) {
		super(message);
		this.kind = kind;
		this.cause = cause;
	}
}

type WorkerClient = {
	url: string;
	navigate?: (url: string) => Promise<WorkerClient | null | undefined>;
	focus?: () => Promise<WorkerClient>;
	postMessage: (message: PushMessage) => void;
};

type SavedNotification = {
	id: string;
	deviceId: string;
	title: string;
	body: string;
	topic: string;
	sentAt: string;
	topicId?: string;
	attachment?: NotificationAttachmentEnvelope;
	priority?: string;
};

export type ServiceWorkerRuntimeDeps = {
	debug: boolean;
	locationOrigin: string;
	beebuzzDomain?: string;
	now: () => number;
	showNotification: (title: string, options?: NotificationOptions) => Promise<void>;
	saveNotification: (notification: SavedNotification) => Promise<void>;
	matchWindowClients: (includeUncontrolled: boolean) => Promise<WorkerClient[]>;
	openWindow: (url: string) => Promise<WorkerClient | null | undefined>;
	claimClients: () => Promise<void>;
	skipWaiting: () => void | Promise<void>;
	getPushSubscription: () => Promise<{
		endpoint: string;
		toJSON: () => {
			keys?: {
				p256dh?: string;
				auth?: string;
			};
		};
	} | null>;
	getDeviceCredentials: () => Promise<{ deviceId: string } | null>;
	decryptPayload: (data: ArrayBuffer) => Promise<string>;
	fetch: typeof fetch;
	warmupHiveDB?: () => Promise<void>;
	recordDiagnostic: (
		diagnostic: HiveDiagnosticDescriptor,
		message: string,
		data?: HiveLogData
	) => void;
};

export type PushEventLike = {
	data?: {
		arrayBuffer: () => ArrayBuffer;
	} | null;
	waitUntil: (promise: Promise<void>) => void;
};

export type NotificationEventLike = {
	notification: {
		data?: Record<string, unknown>;
		close: () => void;
	};
	waitUntil: (promise: Promise<void>) => void;
};

export type ExtendableEventLike = {
	waitUntil: (promise: Promise<void>) => void;
};

export type ExtendableMessageEventLike = {
	data?: {
		type?: string;
	};
};

const AGE_HEADER = new TextEncoder().encode('age-encryption.org/v1\n');
const NOTIFICATION_ICON = '/assets/manifest-icon-192.maskable.png';

function newPushTraceId(): string {
	return crypto.randomUUID().slice(0, 12);
}

/** Returns true if bytes starts with the given prefix. */
function startsWith(bytes: Uint8Array, prefix: Uint8Array): boolean {
	if (bytes.length < prefix.length) return false;
	for (let i = 0; i < prefix.length; i++) {
		if (bytes[i] !== prefix[i]) return false;
	}
	return true;
}

/** Decodes bytes as UTF-8 and parses as JSON. */
function parseJsonBytes(bytes: Uint8Array): unknown {
	return JSON.parse(new TextDecoder().decode(bytes));
}

function buildAttachmentURL(deps: ServiceWorkerRuntimeDeps, token: string): string {
	if (deps.beebuzzDomain) {
		return `https://api.${deps.beebuzzDomain}/v1/attachments/${encodeURIComponent(token)}`;
	}

	const url = new URL(deps.locationOrigin);
	if (url.hostname.startsWith('hive.')) {
		url.hostname = `api.${url.hostname.slice('hive.'.length)}`;
	}
	url.pathname = `/v1/attachments/${encodeURIComponent(token)}`;
	url.search = '';
	url.hash = '';

	return url.toString();
}

function isE2EEnvelope(value: unknown): value is E2EEnvelope {
	if (!value || typeof value !== 'object') return false;

	const envelope = value as E2EEnvelope;
	return (
		typeof envelope.beebuzz?.id === 'string' &&
		envelope.beebuzz.id.length > 0 &&
		typeof envelope.beebuzz.token === 'string' &&
		envelope.beebuzz.token.length > 0 &&
		typeof envelope.beebuzz.sent_at === 'string' &&
		envelope.beebuzz.sent_at.length > 0
	);
}

function validateNotificationPayload(value: unknown): NotificationPayload {
	if (!value || typeof value !== 'object') {
		throw new Error('invalid notification payload');
	}

	const payload = value as NotificationPayload;
	if (typeof payload.id !== 'string' || payload.id.length === 0) {
		throw new Error('notification id is required');
	}
	if (typeof payload.title !== 'string' || payload.title.length === 0) {
		throw new Error('notification title is required');
	}
	if (typeof payload.sent_at !== 'string' || payload.sent_at.length === 0) {
		throw new Error('notification sent_at is required');
	}

	return {
		...payload,
		body: typeof payload.body === 'string' ? payload.body : ''
	};
}

function validateDecryptedMessagePayload(
	value: unknown
): Omit<NotificationPayload, 'id'> & { id?: string } {
	if (!value || typeof value !== 'object') {
		throw new Error('invalid decrypted message payload');
	}

	const payload = value as NotificationPayload;
	if (typeof payload.title !== 'string' || payload.title.length === 0) {
		throw new Error('notification title is required');
	}

	return {
		...payload,
		body: typeof payload.body === 'string' ? payload.body : ''
	};
}

async function loadE2EPayload(
	deps: ServiceWorkerRuntimeDeps,
	id: string,
	token: string,
	envelopeSentAt: string
): Promise<NotificationPayload> {
	const response = await deps.fetch(buildAttachmentURL(deps, token));
	if (!response.ok) {
		throw new Error(`failed to fetch encrypted payload: ${response.status}`);
	}

	const encryptedPayload = await response.arrayBuffer();
	const decrypted = await deps.decryptPayload(encryptedPayload);
	const payload = validateDecryptedMessagePayload(JSON.parse(decrypted));
	if (payload.id && payload.id !== id) {
		throw new Error('notification id mismatch');
	}

	return {
		...payload,
		id,
		sent_at: envelopeSentAt
	};
}

function buildNotificationOptions(
	deps: ServiceWorkerRuntimeDeps,
	data: NotificationPayload,
	pushTraceId: string,
	deviceId?: string
): NotificationOptions {
	return {
		body: data.body,
		icon: NOTIFICATION_ICON,
		badge: NOTIFICATION_ICON,
		tag: data.tag || `beebuzz-${deps.now()}`,
		data: {
			id: data.id,
			title: data.title,
			body: data.body,
			topic: data.topic,
			topicId: data.topic_id,
			deviceId,
			sentAt: data.sent_at,
			priority: data.priority,
			attachment: data.attachment,
			pushTraceId
		}
	};
}

function buildNotificationClickedMessage(notificationData?: Record<string, unknown>): PushMessage {
	const attachment =
		notificationData?.attachment &&
		typeof notificationData.attachment === 'object' &&
		!Array.isArray(notificationData.attachment)
			? notificationData.attachment
			: undefined;

	return {
		type: 'NOTIFICATION_CLICKED',
		notification: {
			id: typeof notificationData?.id === 'string' ? notificationData.id : undefined,
			title: typeof notificationData?.title === 'string' ? notificationData.title : undefined,
			body: typeof notificationData?.body === 'string' ? notificationData.body : undefined,
			topic:
				typeof notificationData?.topic === 'string' || notificationData?.topic === null
					? notificationData.topic
					: null,
			topicId:
				typeof notificationData?.topicId === 'string' || notificationData?.topicId === null
					? notificationData.topicId
					: null,
			deviceId:
				typeof notificationData?.deviceId === 'string' ? notificationData.deviceId : undefined,
			sentAt: typeof notificationData?.sentAt === 'string' ? notificationData.sentAt : undefined,
			priority:
				typeof notificationData?.priority === 'string' ? notificationData.priority : undefined,
			attachment,
			pushTraceId:
				typeof notificationData?.pushTraceId === 'string' ? notificationData.pushTraceId : undefined
		}
	};
}

async function resolvePushPayload(
	deps: ServiceWorkerRuntimeDeps,
	payloadArray: ArrayBuffer,
	pushTraceId: string
): Promise<NotificationPayload> {
	const payloadBytes = new Uint8Array(payloadArray);

	deps.recordDiagnostic(
		HIVE_DIAGNOSTIC.PAYLOAD_RESOLVE,
		`Payload size: ${payloadArray.byteLength} bytes`,
		{ push_trace_id: pushTraceId }
	);

	let parsed: unknown;
	if (startsWith(payloadBytes, AGE_HEADER)) {
		deps.recordDiagnostic(
			HIVE_DIAGNOSTIC.PAYLOAD_DETECTED_ENCRYPTED,
			'Age-encrypted payload detected, decrypting',
			{ push_trace_id: pushTraceId }
		);
		try {
			const decrypted = await deps.decryptPayload(payloadArray);
			parsed = JSON.parse(decrypted) as unknown;
		} catch (error) {
			throw new PushPayloadError(
				'encrypted',
				error instanceof Error ? error.message : 'unknown encrypted error',
				error
			);
		}
	} else {
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.PAYLOAD_DETECTED_PLAIN, 'Plain JSON payload detected', {
			push_trace_id: pushTraceId
		});
		try {
			parsed = parseJsonBytes(payloadBytes);
		} catch (error) {
			throw new PushPayloadError(
				'plain',
				error instanceof Error ? error.message : 'unknown payload error',
				error
			);
		}
	}

	// E2E envelopes (BeeBuzz default for E2E delivery) require fetching the
	// opaque ciphertext attachment and decrypting it. Any failure here is an
	// encryption-side problem (missing key, fetch failure, decrypt failure),
	// not a parse error: classify it as 'encrypted' so the user sees a useful
	// notification body instead of "could not be parsed".
	if (isE2EEnvelope(parsed)) {
		const { beebuzz: envelope } = parsed;
		if (!envelope) {
			throw new PushPayloadError('plain', 'missing encrypted notification envelope');
		}
		try {
			return await loadE2EPayload(deps, envelope.id, envelope.token, envelope.sent_at);
		} catch (error) {
			throw new PushPayloadError(
				'encrypted',
				error instanceof Error ? error.message : 'unknown encrypted error',
				error
			);
		}
	}

	try {
		return validateNotificationPayload(parsed);
	} catch (error) {
		throw new PushPayloadError(
			'plain',
			error instanceof Error ? error.message : 'invalid notification payload',
			error
		);
	}
}

function getEncryptedPayloadFailureMessage(error: unknown): string {
	if (
		error instanceof MissingDeviceIdentityError ||
		error instanceof DeviceIdentityIntegrityError
	) {
		return 'Device key missing or invalid. Open BeeBuzz to re-pair.';
	}

	return 'Received an encrypted notification that could not be decrypted';
}

/** Handles push events and persists notifications before showing them. */
export async function handlePushEvent(
	deps: ServiceWorkerRuntimeDeps,
	event: PushEventLike
): Promise<void> {
	const pushTraceId = newPushTraceId();
	const pushStartTime = performance.now();
	deps.recordDiagnostic(HIVE_DIAGNOSTIC.PUSH_RECEIVED, 'Push event received', {
		push_trace_id: pushTraceId,
		boundary: HIVE_BOUNDARY.INBOUND,
		transport: HIVE_TRANSPORT.WEB_PUSH
	});

	if (!event.data) {
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.PUSH_EMPTY_PAYLOAD, 'Push received with no data', {
			push_trace_id: pushTraceId,
			boundary: HIVE_BOUNDARY.INBOUND,
			transport: HIVE_TRANSPORT.WEB_PUSH
		});
		await deps.showNotification('BeeBuzz', {
			body: 'Received notification without data',
			icon: NOTIFICATION_ICON
		});
		return;
	}

	let data: NotificationPayload;
	try {
		const payloadArray = event.data.arrayBuffer();
		data = await resolvePushPayload(deps, payloadArray, pushTraceId);
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown payload error';
		if (error instanceof PushPayloadError && error.kind === 'encrypted') {
			deps.recordDiagnostic(HIVE_DIAGNOSTIC.PAYLOAD_DECRYPT_FAILED, message, {
				push_trace_id: pushTraceId
			});
			await deps.showNotification('BeeBuzz Notification', {
				body: getEncryptedPayloadFailureMessage(error.cause),
				icon: NOTIFICATION_ICON
			});
			return;
		}
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.PAYLOAD_INVALID, message, {
			push_trace_id: pushTraceId
		});
		await deps.showNotification('BeeBuzz Notification', {
			body: 'Received a notification that could not be parsed',
			icon: NOTIFICATION_ICON
		});
		return;
	}

	deps.recordDiagnostic(
		HIVE_DIAGNOSTIC.PUSH_RESOLVED,
		`Payload resolved in ${(performance.now() - pushStartTime).toFixed(0)}ms`,
		{ notification_id: data.id, push_trace_id: pushTraceId }
	);

	// Record metadata before any client communication so the developer
	// page refresh sees the updated data.
	await recordNotificationReceived({ via: 'push' });

	let deviceId: string | undefined;
	try {
		deviceId = (await deps.getDeviceCredentials())?.deviceId;
	} catch {
		deps.recordDiagnostic(
			HIVE_DIAGNOSTIC.STORAGE_CREDENTIALS_FAILED,
			'Failed to read device credentials',
			{
				push_trace_id: pushTraceId
			}
		);
	}

	if (deviceId) {
		deps.recordDiagnostic(
			HIVE_DIAGNOSTIC.NOTIFICATION_PERSIST_STARTED,
			'Saving notification to storage',
			{ notification_id: data.id, push_trace_id: pushTraceId }
		);
		try {
			await deps.saveNotification({
				id: data.id,
				deviceId,
				title: data.title,
				body: data.body ?? '',
				topic: data.topic || '',
				sentAt: data.sent_at,
				topicId: data.topic_id,
				attachment: data.attachment,
				priority: data.priority
			});
			deps.recordDiagnostic(
				HIVE_DIAGNOSTIC.NOTIFICATION_PERSISTED,
				'Notification saved to IndexedDB',
				{
					notification_id: data.id,
					push_trace_id: pushTraceId,
					boundary: HIVE_BOUNDARY.INTERNAL,
					transport: HIVE_TRANSPORT.INDEXEDDB
				}
			);
		} catch (error) {
			const message = error instanceof Error ? error.message : 'unknown storage error';
			deps.recordDiagnostic(HIVE_DIAGNOSTIC.NOTIFICATION_PERSIST_FAILED, message, {
				notification_id: data.id,
				push_trace_id: pushTraceId
			});
		}
	}

	await deps.showNotification(
		data.title,
		buildNotificationOptions(deps, data, pushTraceId, deviceId)
	);
	deps.recordDiagnostic(HIVE_DIAGNOSTIC.NOTIFICATION_DISPLAYED, 'Notification shown', {
		notification_id: data.id,
		push_trace_id: pushTraceId,
		boundary: HIVE_BOUNDARY.OUTBOUND,
		transport: HIVE_TRANSPORT.NOTIFICATION_CENTER
	});

	if (!deviceId) {
		return;
	}

	const windowClients = await deps.matchWindowClients(true);
	if (windowClients.length === 0) {
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.CLIENTS_MATCH_FAILED, 'No window clients found', {
			push_trace_id: pushTraceId
		});
	}
	for (const client of windowClients) {
		try {
			client.postMessage({
				type: 'PUSH_RECEIVED',
				id: data.id,
				deviceId,
				title: data.title,
				body: data.body ?? '',
				topicId: data.topic_id,
				topic: data.topic ?? null,
				attachment: data.attachment,
				sentAt: data.sent_at,
				priority: data.priority,
				pushTraceId
			});
			deps.recordDiagnostic(HIVE_DIAGNOSTIC.CLIENTS_NOTIFIED, 'Window client notified', {
				notification_id: data.id,
				push_trace_id: pushTraceId,
				boundary: HIVE_BOUNDARY.INTERNAL,
				transport: HIVE_TRANSPORT.POST_MESSAGE
			});
		} catch (error) {
			const message = error instanceof Error ? error.message : 'unknown postMessage error';
			deps.recordDiagnostic(HIVE_DIAGNOSTIC.CLIENTS_POST_MESSAGE_FAILED, message, {
				notification_id: data.id,
				push_trace_id: pushTraceId,
				boundary: HIVE_BOUNDARY.INTERNAL,
				transport: HIVE_TRANSPORT.POST_MESSAGE
			});
		}
	}
}

/**
 * Best-effort: persist the clicked notification to IndexedDB so the app shell
 * can recover it after launch even when the original push-time persistence was
 * skipped (e.g. credentials weren't readable yet) or the postMessage to the
 * newly opened window was dropped (common on iOS / WebKit).
 */
async function persistClickedNotificationBestEffort(
	deps: ServiceWorkerRuntimeDeps,
	notificationData?: Record<string, unknown>
): Promise<void> {
	if (!notificationData) return;

	const id = typeof notificationData.id === 'string' ? notificationData.id : undefined;
	const title = typeof notificationData.title === 'string' ? notificationData.title : undefined;
	const body = typeof notificationData.body === 'string' ? notificationData.body : '';
	const sentAt = typeof notificationData.sentAt === 'string' ? notificationData.sentAt : undefined;
	if (!id || !title || !sentAt) return;
	const pushTraceId =
		typeof notificationData.pushTraceId === 'string'
			? notificationData.pushTraceId
			: newPushTraceId();

	let deviceId =
		typeof notificationData.deviceId === 'string' ? notificationData.deviceId : undefined;
	if (!deviceId) {
		try {
			deviceId = (await deps.getDeviceCredentials())?.deviceId;
		} catch {
			// Ignore — without a deviceId we cannot scope the record.
		}
	}
	if (!deviceId) return;

	const attachment =
		notificationData.attachment &&
		typeof notificationData.attachment === 'object' &&
		!Array.isArray(notificationData.attachment)
			? (notificationData.attachment as NotificationAttachmentEnvelope)
			: undefined;

	try {
		await deps.saveNotification({
			id,
			deviceId,
			title,
			body,
			topic: typeof notificationData.topic === 'string' ? notificationData.topic : '',
			sentAt,
			topicId: typeof notificationData.topicId === 'string' ? notificationData.topicId : undefined,
			attachment,
			priority:
				typeof notificationData.priority === 'string' ? notificationData.priority : undefined
		});
		deps.recordDiagnostic(
			HIVE_DIAGNOSTIC.NOTIFICATION_PERSISTED,
			'Clicked notification saved to IndexedDB',
			{
				notification_id: id,
				push_trace_id: pushTraceId,
				boundary: HIVE_BOUNDARY.INTERNAL,
				transport: HIVE_TRANSPORT.INDEXEDDB
			}
		);
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown storage error';
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.NOTIFICATION_PERSIST_FAILED, message, {
			notification_id: id,
			push_trace_id: pushTraceId
		});
	}
}

function isSameOriginClient(deps: ServiceWorkerRuntimeDeps, client: WorkerClient): boolean {
	try {
		return new URL(client.url).origin === deps.locationOrigin;
	} catch {
		return false;
	}
}

function hiveInboxUrl(deps: ServiceWorkerRuntimeDeps): string {
	return deps.locationOrigin || '/';
}

async function navigateClientToHiveInboxBestEffort(
	deps: ServiceWorkerRuntimeDeps,
	client: WorkerClient
): Promise<WorkerClient | undefined> {
	if (!client.navigate) return client;

	try {
		return (await client.navigate(hiveInboxUrl(deps))) ?? client;
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown navigate error';
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.CLIENTS_OPEN_WINDOW_FAILED, message);
		return undefined;
	}
}

async function focusClientBestEffort(
	deps: ServiceWorkerRuntimeDeps,
	client: WorkerClient
): Promise<WorkerClient | undefined> {
	try {
		return client.focus ? await client.focus() : client;
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown focus error';
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.CLIENTS_FOCUS_FAILED, message);
		return undefined;
	}
}

async function openHiveWindowBestEffort(
	deps: ServiceWorkerRuntimeDeps
): Promise<WorkerClient | undefined> {
	try {
		return (await deps.openWindow(hiveInboxUrl(deps))) ?? undefined;
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown openWindow error';
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.CLIENTS_OPEN_WINDOW_FAILED, message);
		return undefined;
	}
}

function postClickMessageBestEffort(
	deps: ServiceWorkerRuntimeDeps,
	client: WorkerClient,
	notificationData?: Record<string, unknown>
): void {
	try {
		client.postMessage(buildNotificationClickedMessage(notificationData));
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown postMessage error';
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.CLIENTS_POST_MESSAGE_FAILED, message);
	}
}

/** Handles notification clicks by focusing or opening Hive and sending a fallback payload. */
export async function handleNotificationClickEvent(
	deps: ServiceWorkerRuntimeDeps,
	event: NotificationEventLike
): Promise<void> {
	event.notification.close();
	const notificationData = event.notification.data;
	const notificationId = typeof notificationData?.id === 'string' ? notificationData.id : undefined;
	const pushTraceId =
		typeof notificationData?.pushTraceId === 'string'
			? notificationData.pushTraceId
			: newPushTraceId();
	deps.recordDiagnostic(HIVE_DIAGNOSTIC.NOTIFICATION_CLICKED, 'Notification clicked', {
		notification_id: notificationId,
		push_trace_id: pushTraceId,
		boundary: HIVE_BOUNDARY.INBOUND,
		transport: HIVE_TRANSPORT.NOTIFICATION_CENTER
	});

	let focused: WorkerClient | undefined;
	try {
		const windows = await deps.matchWindowClients(false);
		for (const windowClient of windows) {
			if (!isSameOriginClient(deps, windowClient)) continue;
			const inboxClient = await navigateClientToHiveInboxBestEffort(deps, windowClient);
			if (!inboxClient) continue;
			focused = await focusClientBestEffort(deps, inboxClient);
			if (focused) break;
		}
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown matchAll error';
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.CLIENTS_MATCH_FAILED, message);
	}

	if (!focused) {
		focused = await openHiveWindowBestEffort(deps);
	}

	// Keep Android's activation path as close as possible to the pre-fix
	// behavior: do not touch IndexedDB until after focus/openWindow finished.
	// Once a client exists, persist before the fallback postMessage so iOS can
	// still recover from a dropped click message during cold launch.
	try {
		await persistClickedNotificationBestEffort(deps, event.notification.data);
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown persistence error';
		deps.recordDiagnostic(HIVE_DIAGNOSTIC.NOTIFICATION_PERSIST_FAILED, message);
	}

	if (focused) {
		postClickMessageBestEffort(deps, focused, event.notification.data);
	}
}

/** Claims clients when the worker activates. */
export async function handleActivateEvent(deps: ServiceWorkerRuntimeDeps): Promise<void> {
	deps.recordDiagnostic(HIVE_DIAGNOSTIC.SERVICE_WORKER_ACTIVATED, 'Service Worker activated');
	try {
		await deps.warmupHiveDB?.();
	} catch {
		// Activation should not fail because optional diagnostics storage is unavailable.
	}
	await deps.claimClients();
}

/** Broadcasts subscription changes to open clients. */
export async function handlePushSubscriptionChangeEvent(
	deps: ServiceWorkerRuntimeDeps
): Promise<void> {
	deps.recordDiagnostic(HIVE_DIAGNOSTIC.PUSH_SUBSCRIPTION_CHANGED, 'Push subscription changed');

	const clients = await deps.matchWindowClients(true);
	for (const client of clients) {
		client.postMessage({ type: 'SUBSCRIPTION_CHANGED' });
	}
}

/** Promotes a waiting worker when requested by the app shell. */
export async function handleMessageEvent(
	deps: ServiceWorkerRuntimeDeps,
	event: ExtendableMessageEventLike
): Promise<void> {
	if (event.data?.type !== 'SKIP_WAITING') {
		return;
	}

	deps.recordDiagnostic(HIVE_DIAGNOSTIC.SERVICE_WORKER_SKIP_WAITING, 'SKIP_WAITING received');
	await deps.skipWaiting();
}
