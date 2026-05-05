import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
	handleActivateEvent,
	handleMessageEvent,
	handleNotificationClickEvent,
	handlePushEvent,
	handlePushSubscriptionChangeEvent,
	type NotificationEventLike,
	type NotificationPayload,
	type PushEventLike,
	type ServiceWorkerRuntimeDeps
} from './sw-runtime';
import { MissingDeviceIdentityError } from './lib/services/encryption';

type NotificationOptionsWithData = NotificationOptions & {
	data?: Record<string, unknown>;
};

function createPushEvent(payload: NotificationPayload): PushEventLike {
	const bytes = new TextEncoder().encode(JSON.stringify(payload));
	const payloadBuffer = new ArrayBuffer(bytes.byteLength);
	new Uint8Array(payloadBuffer).set(bytes);

	return {
		data: {
			arrayBuffer: () => payloadBuffer
		},
		waitUntil: () => {}
	};
}

function createNotificationClickEvent(data?: Record<string, unknown>): NotificationEventLike {
	return {
		notification: {
			data,
			close: vi.fn()
		},
		waitUntil: () => {}
	};
}

function createDeps(overrides: Partial<ServiceWorkerRuntimeDeps> = {}): ServiceWorkerRuntimeDeps {
	return {
		debug: false,
		locationOrigin: 'https://hive.beebuzz.test',
		beebuzzDomain: 'beebuzz.test',
		now: () => 1700000000000,
		showNotification: vi.fn(() => Promise.resolve()),
		saveNotification: vi.fn(() => Promise.resolve()),
		matchWindowClients: vi.fn(() => Promise.resolve([])),
		openWindow: vi.fn(() => Promise.resolve(null)),
		claimClients: vi.fn(() => Promise.resolve()),
		skipWaiting: vi.fn(() => Promise.resolve()),
		getPushSubscription: vi.fn(() => Promise.resolve(null)),
		getDeviceCredentials: vi.fn(() => Promise.resolve({ deviceId: 'dev-a' })),
		decryptPayload: vi.fn(() => Promise.resolve('')),
		fetch: vi.fn(() => Promise.resolve(new Response('{}', { status: 200 }))),
		...overrides
	};
}

describe('service worker runtime', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('persists the push notification before showing it to the OS', async () => {
		const order: string[] = [];
		const deps = createDeps({
			saveNotification: vi.fn(() => {
				order.push('save');
				return Promise.resolve();
			}),
			showNotification: vi.fn(() => {
				order.push('show');
				return Promise.resolve();
			})
		});

		await handlePushEvent(
			deps,
			createPushEvent({
				id: 'n-1',
				title: 'Door',
				body: 'Front door opened',
				topic: 'alerts',
				topic_id: 'topic-1',
				sent_at: '2026-04-20T09:00:00.000Z',
				priority: 'high'
			})
		);

		expect(order).toEqual(['save', 'show']);
		expect(deps.saveNotification).toHaveBeenCalledWith({
			id: 'n-1',
			deviceId: 'dev-a',
			title: 'Door',
			body: 'Front door opened',
			topic: 'alerts',
			topicId: 'topic-1',
			sentAt: '2026-04-20T09:00:00.000Z',
			priority: 'high',
			attachment: undefined
		});
		const showNotificationCall = vi.mocked(deps.showNotification).mock.calls[0];
		const notificationOptions: NotificationOptionsWithData | undefined = showNotificationCall?.[1];
		expect(showNotificationCall?.[0]).toBe('Door');
		expect(notificationOptions?.body).toBe('Front door opened');
		expect(notificationOptions?.data).toMatchObject({
			id: 'n-1',
			sentAt: '2026-04-20T09:00:00.000Z'
		});
	});

	it('notifies open clients after persistence and ignores postMessage failures', async () => {
		const healthyClient = {
			url: 'https://hive.beebuzz.test/inbox',
			postMessage: vi.fn()
		};
		const failingClient = {
			url: 'https://hive.beebuzz.test/other',
			postMessage: vi.fn(() => {
				throw new Error('frozen client');
			})
		};
		const deps = createDeps({
			matchWindowClients: vi.fn(() => Promise.resolve([healthyClient, failingClient]))
		});

		await expect(
			handlePushEvent(
				deps,
				createPushEvent({
					id: 'n-2',
					title: 'Motion',
					body: 'Garage motion detected',
					topic: 'security',
					sent_at: '2026-04-20T10:00:00.000Z'
				})
			)
		).resolves.toBeUndefined();

		expect(deps.matchWindowClients).toHaveBeenCalledWith(true);
		expect(healthyClient.postMessage).toHaveBeenCalledWith({
			type: 'PUSH_RECEIVED',
			id: 'n-2',
			deviceId: 'dev-a',
			title: 'Motion',
			body: 'Garage motion detected',
			topicId: undefined,
			topic: 'security',
			attachment: undefined,
			sentAt: '2026-04-20T10:00:00.000Z',
			priority: undefined
		});
		expect(failingClient.postMessage).toHaveBeenCalledTimes(1);
	});

	it('focuses an existing Hive window and sends the notification fallback payload on click', async () => {
		const focusedClient = {
			url: 'https://hive.beebuzz.test/',
			postMessage: vi.fn(),
			focus: vi.fn(function (this: typeof focusedClient) {
				return Promise.resolve(this);
			})
		};
		const deps = createDeps({
			matchWindowClients: vi.fn(() => Promise.resolve([focusedClient]))
		});
		const event = createNotificationClickEvent({
			id: 'n-3',
			deviceId: 'dev-a',
			title: 'Bell',
			body: 'Someone rang the bell',
			topic: 'door',
			topicId: 'topic-3',
			sentAt: '2026-04-20T11:00:00.000Z',
			priority: 'normal'
		});

		await handleNotificationClickEvent(deps, event);

		expect(event.notification.close).toHaveBeenCalledTimes(1);
		expect(deps.matchWindowClients).toHaveBeenCalledWith(false);
		expect(focusedClient.focus).toHaveBeenCalledTimes(1);
		expect(focusedClient.postMessage).toHaveBeenCalledWith({
			type: 'NOTIFICATION_CLICKED',
			notification: {
				id: 'n-3',
				deviceId: 'dev-a',
				title: 'Bell',
				body: 'Someone rang the bell',
				topic: 'door',
				topicId: 'topic-3',
				sentAt: '2026-04-20T11:00:00.000Z',
				priority: 'normal',
				attachment: undefined
			}
		});
	});

	it('opens a new Hive window when no client exists and sends the same fallback payload', async () => {
		const openedClient = {
			url: 'https://hive.beebuzz.test/',
			postMessage: vi.fn()
		};
		const deps = createDeps({
			matchWindowClients: vi.fn(() => Promise.resolve([])),
			openWindow: vi.fn(() => Promise.resolve(openedClient))
		});

		await handleNotificationClickEvent(
			deps,
			createNotificationClickEvent({
				id: 'n-4',
				deviceId: 'dev-a',
				title: 'Alarm',
				body: 'Window opened',
				sentAt: '2026-04-20T12:00:00.000Z'
			})
		);

		expect(deps.openWindow).toHaveBeenCalledWith('https://hive.beebuzz.test');
		expect(openedClient.postMessage).toHaveBeenCalledWith({
			type: 'NOTIFICATION_CLICKED',
			notification: {
				id: 'n-4',
				deviceId: 'dev-a',
				title: 'Alarm',
				body: 'Window opened',
				topic: null,
				topicId: null,
				sentAt: '2026-04-20T12:00:00.000Z',
				priority: undefined,
				attachment: undefined
			}
		});
	});

	it('claims clients during activation', async () => {
		const deps = createDeps();

		await handleActivateEvent(deps);

		expect(deps.claimClients).toHaveBeenCalledTimes(1);
	});

	it('triggers skipWaiting only for the activation message', async () => {
		const deps = createDeps();

		await handleMessageEvent(deps, { data: { type: 'OTHER' } });
		await handleMessageEvent(deps, { data: { type: 'SKIP_WAITING' } });

		expect(deps.skipWaiting).toHaveBeenCalledTimes(1);
	});

	it('shows a fallback notification when event.data is null', async () => {
		const deps = createDeps();
		const event: PushEventLike = {
			data: null,
			waitUntil: () => {}
		};

		await handlePushEvent(deps, event);

		expect(deps.showNotification).toHaveBeenCalledWith('BeeBuzz', {
			body: 'Received notification without data',
			icon: '/assets/manifest-icon-192.maskable.png'
		});
		expect(deps.saveNotification).not.toHaveBeenCalled();
	});

	it('still shows the notification when saveNotification fails', async () => {
		const deps = createDeps({
			saveNotification: vi.fn(() => Promise.reject(new Error('IDB write failed')))
		});
		const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

		await handlePushEvent(
			deps,
			createPushEvent({
				id: 'n-storage-fail',
				title: 'Sensor',
				body: 'Temperature alert',
				sent_at: '2026-04-20T13:00:00.000Z'
			})
		);

		expect(deps.showNotification).toHaveBeenCalledWith(
			'Sensor',
			expect.objectContaining({ body: 'Temperature alert' })
		);
		expect(consoleSpy).toHaveBeenCalledWith(
			'[PUSH] Failed to persist notification',
			expect.objectContaining({ error: 'IDB write failed' })
		);
		consoleSpy.mockRestore();
	});

	it('shows the OS notification without importable UI history when credentials are missing', async () => {
		const client = { url: 'https://hive.beebuzz.test/inbox', postMessage: vi.fn() };
		const deps = createDeps({
			getDeviceCredentials: vi.fn(() => Promise.resolve(null)),
			matchWindowClients: vi.fn(() => Promise.resolve([client]))
		});

		await handlePushEvent(
			deps,
			createPushEvent({
				id: 'n-no-credentials',
				title: 'Sensor',
				body: 'Temperature alert',
				sent_at: '2026-04-20T13:10:00.000Z'
			})
		);

		expect(deps.saveNotification).not.toHaveBeenCalled();
		const showNotificationCall = vi.mocked(deps.showNotification).mock.calls[0];
		const notificationOptions: NotificationOptionsWithData | undefined = showNotificationCall?.[1];
		expect(showNotificationCall?.[0]).toBe('Sensor');
		expect(notificationOptions?.body).toBe('Temperature alert');
		expect(notificationOptions?.data).toMatchObject({
			id: 'n-no-credentials',
			deviceId: undefined
		});
		expect(deps.matchWindowClients).not.toHaveBeenCalled();
		expect(client.postMessage).not.toHaveBeenCalled();
	});

	it('shows a decryption-specific fallback for encrypted payload failures', async () => {
		const ageHeader = new TextEncoder().encode('age-encryption.org/v1\ncorrupted-data');
		const buffer = ageHeader.buffer.slice(
			ageHeader.byteOffset,
			ageHeader.byteOffset + ageHeader.byteLength
		);
		const deps = createDeps({
			decryptPayload: vi.fn(() => Promise.reject(new Error('decryption failed')))
		});
		const event: PushEventLike = {
			data: { arrayBuffer: () => buffer },
			waitUntil: () => {}
		};

		await handlePushEvent(deps, event);

		expect(deps.showNotification).toHaveBeenCalledWith('BeeBuzz Notification', {
			body: 'Received an encrypted notification that could not be decrypted',
			icon: '/assets/manifest-icon-192.maskable.png'
		});
		expect(deps.saveNotification).not.toHaveBeenCalled();
	});

	it('resolves an E2E envelope by fetching and decrypting the stored payload', async () => {
		const envelopeBytes = new TextEncoder().encode(
			JSON.stringify({
				beebuzz: {
					id: 'n-e2e-1',
					token: 'attachment-token',
					sent_at: '2026-04-20T13:30:00.000Z'
				}
			})
		);
		const buffer = envelopeBytes.buffer.slice(
			envelopeBytes.byteOffset,
			envelopeBytes.byteOffset + envelopeBytes.byteLength
		);
		const fetchSpy = vi.fn(() =>
			Promise.resolve(
				new Response('ciphertext', {
					status: 200
				})
			)
		);
		const deps = createDeps({
			fetch: fetchSpy,
			decryptPayload: vi.fn(() =>
				Promise.resolve(
					JSON.stringify({
						title: 'Encrypted',
						body: 'Secret message',
						topic: 'alerts'
					})
				)
			)
		});
		const event: PushEventLike = {
			data: { arrayBuffer: () => buffer },
			waitUntil: () => {}
		};

		await handlePushEvent(deps, event);

		expect(fetchSpy).toHaveBeenCalledWith(
			'https://api.beebuzz.test/v1/attachments/attachment-token'
		);
		expect(deps.saveNotification).toHaveBeenCalledWith({
			id: 'n-e2e-1',
			deviceId: 'dev-a',
			title: 'Encrypted',
			body: 'Secret message',
			topic: 'alerts',
			sentAt: '2026-04-20T13:30:00.000Z',
			topicId: undefined,
			attachment: undefined,
			priority: undefined
		});
		expect(deps.showNotification).toHaveBeenCalledWith(
			'Encrypted',
			expect.objectContaining({ body: 'Secret message' })
		);
	});

	it('tells the user to re-pair when the device key is missing', async () => {
		const ageHeader = new TextEncoder().encode('age-encryption.org/v1\ncorrupted-data');
		const buffer = ageHeader.buffer.slice(
			ageHeader.byteOffset,
			ageHeader.byteOffset + ageHeader.byteLength
		);
		const deps = createDeps({
			decryptPayload: vi.fn(() => Promise.reject(new MissingDeviceIdentityError()))
		});
		const event: PushEventLike = {
			data: { arrayBuffer: () => buffer },
			waitUntil: () => {}
		};

		await handlePushEvent(deps, event);

		expect(deps.showNotification).toHaveBeenCalledWith('BeeBuzz Notification', {
			body: 'Device key missing or invalid. Open BeeBuzz to re-pair.',
			icon: '/assets/manifest-icon-192.maskable.png'
		});
		expect(deps.saveNotification).not.toHaveBeenCalled();
	});

	it('shows a parse-specific fallback for invalid plain JSON payloads', async () => {
		const garbage = new TextEncoder().encode('not-json!!!');
		const buffer = garbage.buffer.slice(
			garbage.byteOffset,
			garbage.byteOffset + garbage.byteLength
		);
		const deps = createDeps();
		const event: PushEventLike = {
			data: { arrayBuffer: () => buffer },
			waitUntil: () => {}
		};
		const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

		await handlePushEvent(deps, event);

		expect(deps.showNotification).toHaveBeenCalledWith('BeeBuzz Notification', {
			body: 'Received a notification that could not be parsed',
			icon: '/assets/manifest-icon-192.maskable.png'
		});
		expect(deps.saveNotification).not.toHaveBeenCalled();
		consoleSpy.mockRestore();
	});

	it('broadcasts SUBSCRIPTION_CHANGED to all open clients', async () => {
		const client1 = { url: 'https://hive.beebuzz.test/', postMessage: vi.fn() };
		const client2 = { url: 'https://hive.beebuzz.test/device', postMessage: vi.fn() };
		const deps = createDeps({
			matchWindowClients: vi.fn(() => Promise.resolve([client1, client2]))
		});

		await handlePushSubscriptionChangeEvent(deps);

		expect(deps.matchWindowClients).toHaveBeenCalledWith(true);
		expect(client1.postMessage).toHaveBeenCalledWith({ type: 'SUBSCRIPTION_CHANGED' });
		expect(client2.postMessage).toHaveBeenCalledWith({ type: 'SUBSCRIPTION_CHANGED' });
	});

	it('does not match a client with a different origin on notification click', async () => {
		const foreignClient = {
			url: 'https://evil.beebuzz.test/',
			postMessage: vi.fn(),
			focus: vi.fn()
		};
		const deps = createDeps({
			matchWindowClients: vi.fn(() => Promise.resolve([foreignClient])),
			openWindow: vi.fn(() => Promise.resolve(null))
		});

		await handleNotificationClickEvent(
			deps,
			createNotificationClickEvent({
				id: 'n-origin',
				title: 'Test',
				sentAt: '2026-04-20T14:00:00.000Z'
			})
		);

		expect(foreignClient.focus).not.toHaveBeenCalled();
		expect(deps.openWindow).toHaveBeenCalledWith('https://hive.beebuzz.test');
	});
});
