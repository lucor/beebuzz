import { API_URL } from '@beebuzz/shared/config';
import { pushApi, type DeviceNotificationSyncItem } from '@beebuzz/shared/api';
import { logger } from '@beebuzz/shared/logger';
import { decryptBinary } from '$lib/services/encryption';
import { notificationsStore } from '$lib/stores/notifications.svelte';
import type { Attachment, NotificationPriority } from '@beebuzz/shared/types';

type TrustedNotificationPayload = {
	id?: unknown;
	title?: unknown;
	body?: unknown;
	topic_id?: unknown;
	topic?: unknown;
	sent_at?: unknown;
	priority?: unknown;
	attachment?: unknown;
};

type E2EEnvelope = {
	beebuzz?: {
		id?: unknown;
		token?: unknown;
		sent_at?: unknown;
	};
};

type NormalizedNotification = {
	id: string;
	title: string;
	body: string;
	topic: string | null;
	topicId: string | null;
	sentAt: string;
	priority?: NotificationPriority;
	attachment?: Attachment;
};

const SYNC_LIMIT = 50;

function asAttachment(value: unknown): Attachment | undefined {
	if (!value || typeof value !== 'object' || Array.isArray(value)) {
		return undefined;
	}

	const record = value as Record<string, unknown>;
	return {
		data: typeof record.data === 'string' ? record.data : undefined,
		mime: typeof record.mime === 'string' ? record.mime : undefined,
		token: typeof record.token === 'string' ? record.token : undefined,
		filename: typeof record.filename === 'string' ? record.filename : undefined
	};
}

function normalizeTrustedPayload(payload: TrustedNotificationPayload): NormalizedNotification {
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
		id: payload.id,
		title: payload.title,
		body: typeof payload.body === 'string' ? payload.body : '',
		topic: typeof payload.topic === 'string' ? payload.topic : null,
		topicId: typeof payload.topic_id === 'string' ? payload.topic_id : null,
		sentAt: payload.sent_at,
		priority:
			payload.priority === 'high' || payload.priority === 'normal' ? payload.priority : undefined,
		attachment: asAttachment(payload.attachment)
	};
}

function isE2EEnvelope(payload: unknown): payload is E2EEnvelope {
	if (!payload || typeof payload !== 'object') return false;
	const envelope = payload as E2EEnvelope;
	return (
		typeof envelope.beebuzz?.id === 'string' &&
		typeof envelope.beebuzz.token === 'string' &&
		typeof envelope.beebuzz.sent_at === 'string'
	);
}

async function resolveE2EPayload(
	item: DeviceNotificationSyncItem
): Promise<NormalizedNotification> {
	if (!isE2EEnvelope(item.payload)) {
		throw new Error('invalid e2e notification envelope');
	}

	const envelope = item.payload.beebuzz;
	if (!envelope) {
		throw new Error('invalid e2e notification envelope');
	}
	const { id: envId, token, sent_at: envSentAt } = envelope;
	if (typeof envId !== 'string' || typeof token !== 'string' || typeof envSentAt !== 'string') {
		throw new Error('invalid e2e notification envelope');
	}

	const response = await fetch(`${API_URL}/v1/attachments/${encodeURIComponent(token)}`);
	if (!response.ok) {
		throw new Error(`failed to fetch encrypted payload: ${response.status}`);
	}

	const ciphertext = new Uint8Array(await response.arrayBuffer());
	const plaintext = await decryptBinary(ciphertext);
	const decrypted = JSON.parse(new TextDecoder().decode(plaintext)) as TrustedNotificationPayload;
	const normalized = normalizeTrustedPayload({
		...decrypted,
		id: envId,
		sent_at: envSentAt
	});
	return normalized;
}

async function normalizeSyncItem(
	item: DeviceNotificationSyncItem
): Promise<NormalizedNotification> {
	if (item.delivery_mode === 'e2e') {
		return resolveE2EPayload(item);
	}
	return normalizeTrustedPayload(item.payload as TrustedNotificationPayload);
}

/** Pulls recently missed notifications over HTTPS and merges them into the active store. */
export async function syncRecentNotifications(
	deviceId: string,
	deviceToken: string
): Promise<void> {
	let after = notificationsStore.syncCursor ?? undefined;
	let gap = false;

	while (true) {
		const response = await pushApi.syncDeviceNotifications(
			deviceId,
			deviceToken,
			after,
			SYNC_LIMIT
		);
		if (response.gap) {
			gap = true;
		}

		for (const item of response.notifications) {
			try {
				const notification = await normalizeSyncItem(item);
				notificationsStore.add(
					notification.title,
					notification.body,
					notification.topic,
					notification.topicId,
					notification.sentAt,
					notification.attachment,
					notification.priority,
					notification.id
				);
			} catch (error) {
				logger.warn('Skipped synced notification', {
					id: item.id,
					error: error instanceof Error ? error.message : String(error)
				});
			}
		}

		if (response.notifications.length === 0) break;
		if (!response.next_cursor) break;
		after = response.next_cursor;
	}

	// Persist the last synced id as the cursor for next sync.
	const lastId = notificationsStore.latestNotificationId;
	if (lastId) {
		notificationsStore.syncCursor = lastId;
	}

	if (gap) {
		logger.warn('Notification sync gap detected', { deviceId });
	}
}
