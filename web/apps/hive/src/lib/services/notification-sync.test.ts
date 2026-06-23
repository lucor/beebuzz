import { beforeEach, describe, expect, it, vi } from 'vitest';
import { HIVE_DIAGNOSTIC } from '$lib/devmode/types';

const syncDeviceNotifications = vi.fn();
const loggerWarn = vi.fn();
const safeLoggerLog = vi.fn();
const recordNotificationReceived = vi.fn<(_input: { via: 'push' | 'outbox' }) => Promise<void>>(
	() => Promise.resolve()
);
const notificationsStore = {
	syncCursor: null as string | null,
	latestNotificationId: undefined as string | undefined,
	add: vi.fn(
		(
			_title: string,
			_body: string,
			_topic: string | null,
			_topicId: string | null,
			_sentAt: string,
			_attachment: unknown,
			_priority: string | undefined,
			id: string | undefined
		) => {
			notificationsStore.latestNotificationId = id;
			return true;
		}
	)
};

vi.mock('@beebuzz/shared/api', () => ({
	pushApi: {
		syncDeviceNotifications
	}
}));

vi.mock('@beebuzz/shared/config', () => ({
	API_URL: 'https://api.example.test'
}));

vi.mock('@beebuzz/shared/logger', () => ({
	logger: {
		warn: loggerWarn
	}
}));

vi.mock('$lib/stores/notifications.svelte', () => ({
	notificationsStore
}));

vi.mock('$lib/devmode/safe-logger', () => ({
	safeLogger: {
		log: safeLoggerLog
	}
}));

vi.mock('$lib/services/runtime-metadata-repository', () => ({
	recordNotificationReceived
}));

describe('syncRecentNotifications', () => {
	beforeEach(() => {
		syncDeviceNotifications.mockReset();
		loggerWarn.mockReset();
		safeLoggerLog.mockReset();
		recordNotificationReceived.mockClear();
		notificationsStore.syncCursor = null;
		notificationsStore.latestNotificationId = undefined;
		notificationsStore.add.mockClear();
	});

	it('records the outbox sync lifecycle with a trace per notification', async () => {
		syncDeviceNotifications.mockResolvedValueOnce({
			notifications: [
				{
					id: 'n-sync-1',
					delivery_mode: 'server_trusted',
					payload: {
						id: 'n-sync-1',
						title: 'Door',
						body: 'Opened',
						topic: 'security',
						topic_id: 'topic-security',
						sent_at: '2026-04-20T10:00:00.000Z',
						priority: 'normal'
					},
					sent_at: '2026-04-20T10:00:00.000Z',
					expires_at: '2026-04-20T10:10:00.000Z'
				}
			],
			next_cursor: null,
			gap: false
		});

		const { syncRecentNotifications } = await import('./notification-sync');
		await syncRecentNotifications('dev-a', 'device-token');

		expect(syncDeviceNotifications).toHaveBeenCalledWith('dev-a', 'device-token', undefined, 50);
		expect(notificationsStore.add).toHaveBeenCalledWith(
			'Door',
			'Opened',
			'security',
			'topic-security',
			'2026-04-20T10:00:00.000Z',
			undefined,
			'normal',
			'n-sync-1'
		);
		expect(notificationsStore.syncCursor).toBe('n-sync-1');
		expect(recordNotificationReceived).toHaveBeenCalledWith({ via: 'outbox' });
		expect(safeLoggerLog).toHaveBeenCalledWith(
			HIVE_DIAGNOSTIC.OUTBOX_SYNC_STARTED,
			'Notification outbox sync started',
			expect.objectContaining({
				endpoint: '/v1/devices/{device_id}/notifications',
				method: 'GET'
			})
		);
		expect(safeLoggerLog).toHaveBeenCalledWith(
			HIVE_DIAGNOSTIC.OUTBOX_RESPONSE_RECEIVED,
			'Outbox response received',
			expect.objectContaining({
				item_count: 1,
				ok: true
			})
		);
		expect(safeLoggerLog).toHaveBeenCalledWith(
			HIVE_DIAGNOSTIC.OUTBOX_NOTIFICATION_IMPORTED,
			'Synced notification imported',
			expect.objectContaining({
				notification_id: 'n-sync-1',
				push_trace_id: 'outbox-n-sync-1',
				delivery_mode: 'server_trusted'
			})
		);
		expect(safeLoggerLog).toHaveBeenCalledWith(
			HIVE_DIAGNOSTIC.OUTBOX_CURSOR_UPDATED,
			'Outbox sync cursor updated',
			expect.objectContaining({
				sync_cursor: 'n-sync-1'
			})
		);
		expect(safeLoggerLog).toHaveBeenCalledWith(
			HIVE_DIAGNOSTIC.OUTBOX_SYNC_COMPLETED,
			'Notification outbox sync completed',
			expect.objectContaining({
				imported_count: 1,
				page_count: 1,
				ok: true
			})
		);
	});

	it('does not record outbox delivery when the synced notification is already in the store', async () => {
		notificationsStore.add.mockImplementationOnce(
			(
				_title: string,
				_body: string,
				_topic: string | null,
				_topicId: string | null,
				_sentAt: string,
				_attachment: unknown,
				_priority: string | undefined,
				id: string | undefined
			) => {
				notificationsStore.latestNotificationId = id;
				return false;
			}
		);
		syncDeviceNotifications.mockResolvedValueOnce({
			notifications: [
				{
					id: 'n-sync-1',
					delivery_mode: 'server_trusted',
					payload: {
						id: 'n-sync-1',
						title: 'Door',
						body: 'Opened',
						sent_at: '2026-04-20T10:00:00.000Z'
					},
					sent_at: '2026-04-20T10:00:00.000Z',
					expires_at: '2026-04-20T10:10:00.000Z'
				}
			],
			next_cursor: null,
			gap: false
		});

		const { syncRecentNotifications } = await import('./notification-sync');
		await syncRecentNotifications('dev-a', 'device-token');

		expect(recordNotificationReceived).not.toHaveBeenCalled();
		expect(safeLoggerLog).not.toHaveBeenCalledWith(
			HIVE_DIAGNOSTIC.OUTBOX_NOTIFICATION_IMPORTED,
			'Synced notification imported',
			expect.anything()
		);
		expect(safeLoggerLog).toHaveBeenCalledWith(
			HIVE_DIAGNOSTIC.OUTBOX_SYNC_COMPLETED,
			'Notification outbox sync completed',
			expect.objectContaining({
				imported_count: 0,
				ok: true
			})
		);
	});
});
