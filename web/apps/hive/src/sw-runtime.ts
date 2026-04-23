import type { PushMessage } from '@beebuzz/shared/types';
import {
	DeviceIdentityIntegrityError,
	MissingDeviceIdentityError
} from './lib/services/encryption';

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
	focus?: () => Promise<WorkerClient>;
	postMessage: (message: PushMessage) => void;
};

type SavedNotification = {
	id: string;
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
	decryptPayload: (data: ArrayBuffer) => Promise<string>;
	fetch: typeof fetch;
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
	data: NotificationPayload
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
			sentAt: data.sent_at,
			priority: data.priority,
			attachment: data.attachment
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
			sentAt: typeof notificationData?.sentAt === 'string' ? notificationData.sentAt : undefined,
			priority:
				typeof notificationData?.priority === 'string' ? notificationData.priority : undefined,
			attachment
		}
	};
}

async function resolvePushPayload(
	deps: ServiceWorkerRuntimeDeps,
	payloadArray: ArrayBuffer
): Promise<NotificationPayload> {
	const payloadBytes = new Uint8Array(payloadArray);

	if (deps.debug) {
		console.log(`[PUSH] Payload size: ${payloadArray.byteLength} bytes`);
	}

	if (startsWith(payloadBytes, AGE_HEADER)) {
		if (deps.debug) {
			console.log('[PUSH] Detected age-encrypted payload, decrypting...');
		}
		try {
			const decrypted = await deps.decryptPayload(payloadArray);
			const parsed = JSON.parse(decrypted) as unknown;
			if (isE2EEnvelope(parsed)) {
				const envelope = parsed.beebuzz;
				if (!envelope) {
					throw new Error('missing encrypted notification envelope');
				}
				return loadE2EPayload(deps, envelope.id, envelope.token, envelope.sent_at);
			}

			return validateNotificationPayload(parsed);
		} catch (error) {
			throw new PushPayloadError(
				'encrypted',
				error instanceof Error ? error.message : 'unknown encrypted error',
				error
			);
		}
	}

	if (deps.debug) {
		console.log('[PUSH] Plain JSON payload detected');
	}
	try {
		const parsed = parseJsonBytes(payloadBytes);
		if (isE2EEnvelope(parsed)) {
			const envelope = parsed.beebuzz;
			if (!envelope) {
				throw new Error('missing encrypted notification envelope');
			}
			return loadE2EPayload(deps, envelope.id, envelope.token, envelope.sent_at);
		}

		return validateNotificationPayload(parsed);
	} catch (error) {
		throw new PushPayloadError(
			'plain',
			error instanceof Error ? error.message : 'unknown payload error',
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
	const pushStartTime = performance.now();
	if (deps.debug) {
		console.log('Push event received');
	}

	if (!event.data) {
		console.warn('❌ Push received but event.data is null - notification sent without payload');
		try {
			const sub = await deps.getPushSubscription();
			if (sub) {
				const subJson = sub.toJSON();
				const p256dh = subJson.keys?.p256dh ?? '';
				const auth = subJson.keys?.auth ?? '';
				console.warn(`[PUSH NULL] Subscription endpoint: ${sub.endpoint.slice(0, 80)}...`);
				console.warn(`[PUSH NULL] Key lengths: p256dh=${p256dh.length}, auth=${auth.length}`);
			} else {
				console.warn('[PUSH NULL] No active push subscription found');
			}
		} catch (err) {
			console.warn('[PUSH NULL] Failed to get subscription info:', err);
		}
		await deps.showNotification('BeeBuzz', {
			body: 'Received notification without data',
			icon: NOTIFICATION_ICON
		});
		return;
	}

	let data: NotificationPayload;
	try {
		const payloadArray = event.data.arrayBuffer();
		data = await resolvePushPayload(deps, payloadArray);
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown payload error';
		console.error('[PUSH] Failed to parse notification payload', { error: message });
		if (error instanceof PushPayloadError && error.kind === 'encrypted') {
			await deps.showNotification('BeeBuzz Notification', {
				body: getEncryptedPayloadFailureMessage(error.cause),
				icon: NOTIFICATION_ICON
			});
			return;
		}
		await deps.showNotification('BeeBuzz Notification', {
			body: 'Received a notification that could not be parsed',
			icon: NOTIFICATION_ICON
		});
		return;
	}

	const totalDuration = performance.now() - pushStartTime;
	if (deps.debug) {
		console.log(`[PUSH TOTAL] duration=${totalDuration.toFixed(2)}ms`);
	}

	try {
		await deps.saveNotification({
			id: data.id,
			title: data.title,
			body: data.body ?? '',
			topic: data.topic || '',
			sentAt: data.sent_at,
			topicId: data.topic_id,
			attachment: data.attachment,
			priority: data.priority
		});
	} catch (error) {
		const message = error instanceof Error ? error.message : 'unknown storage error';
		console.error('[PUSH] Failed to persist notification', { error: message });
	}

	await deps.showNotification(data.title, buildNotificationOptions(deps, data));

	const windowClients = await deps.matchWindowClients(true);
	for (const client of windowClients) {
		try {
			client.postMessage({
				type: 'PUSH_RECEIVED',
				id: data.id,
				title: data.title,
				body: data.body ?? '',
				topicId: data.topic_id,
				topic: data.topic ?? null,
				attachment: data.attachment,
				sentAt: data.sent_at,
				priority: data.priority
			});
		} catch {
			// Client is frozen or terminated — safe to ignore, persistence already succeeded.
		}
	}
}

/** Handles notification clicks by focusing or opening Hive and sending a fallback payload. */
export async function handleNotificationClickEvent(
	deps: ServiceWorkerRuntimeDeps,
	event: NotificationEventLike
): Promise<void> {
	event.notification.close();

	const windows = await deps.matchWindowClients(false);
	let focused: WorkerClient | undefined;
	for (const windowClient of windows) {
		const clientOrigin = new URL(windowClient.url).origin;
		if (clientOrigin === deps.locationOrigin) {
			focused = windowClient.focus ? await windowClient.focus() : windowClient;
			break;
		}
	}

	if (!focused) {
		focused = (await deps.openWindow(deps.locationOrigin || '/')) ?? undefined;
	}

	if (focused) {
		focused.postMessage(buildNotificationClickedMessage(event.notification.data));
	}
}

/** Claims clients when the worker activates. */
export async function handleActivateEvent(deps: ServiceWorkerRuntimeDeps): Promise<void> {
	if (deps.debug) {
		console.log('Service Worker activating...');
	}
	await deps.claimClients();
}

/** Broadcasts subscription changes to open clients. */
export async function handlePushSubscriptionChangeEvent(
	deps: ServiceWorkerRuntimeDeps
): Promise<void> {
	if (deps.debug) {
		console.log('Push subscription changed');
	}

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

	if (deps.debug) {
		console.log('Service Worker received SKIP_WAITING message');
	}

	await deps.skipWaiting();
}
