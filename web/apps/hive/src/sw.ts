import { Decrypter } from 'age-encryption';
import { getDeviceIdentity, MissingDeviceIdentityError } from './lib/services/encryption';
import { deviceKeysRepository } from './lib/services/device-keys-repository';
import { notificationsRepository } from './lib/services/notifications-repository';
import type { HiveDiagnosticEvent, HiveDiagnosticKind, HiveLogScope } from './lib/devmode/types';
import {
	handleActivateEvent,
	handleMessageEvent,
	handleNotificationClickEvent,
	handlePushEvent,
	handlePushSubscriptionChangeEvent,
	type NotificationAttachmentEnvelope,
	type ServiceWorkerRuntimeDeps
} from './sw-runtime';

declare const self: ServiceWorkerGlobalScope;

const BEEBUZZ_DOMAIN = import.meta.env.VITE_BEEBUZZ_DOMAIN as string | undefined;

async function decryptPayload(data: ArrayBuffer): Promise<string> {
	const identity = await getDeviceIdentity();
	if (!identity) {
		throw new MissingDeviceIdentityError(
			'No encryption key found in IndexedDB - device may not be paired correctly'
		);
	}

	const d = new Decrypter();
	d.addIdentity(identity);
	const decrypted = await d.decrypt(new Uint8Array(data), 'text');

	return decrypted;
}

function saveNotificationToStorage(input: {
	id: string;
	deviceId: string;
	title: string;
	body: string;
	topic: string;
	sentAt: string;
	topicId?: string;
	attachment?: NotificationAttachmentEnvelope;
	priority?: string;
}): Promise<void> {
	return notificationsRepository.save(input);
}

function recordDiagnostic(
	_kind: HiveDiagnosticKind,
	_scope: HiveLogScope,
	_event: HiveDiagnosticEvent,
	_message: string
): void {
	// Diagnostics are recorded by the app shell. The service worker
	// forwards events through postMessage instead of writing directly
	// to IndexedDB to avoid lock contention. The app shell's message
	// handler is responsible for recording these events if Developer
	// Mode is enabled.
	try {
		void self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
			for (const client of clients) {
				client.postMessage({
					type: 'DIAGNOSTIC_EVENT',
					kind: _kind,
					scope: _scope,
					event: _event,
					message: _message
				});
			}
		});
	} catch {
		// silently fail
	}
}

const runtimeDeps: ServiceWorkerRuntimeDeps = {
	debug: false,
	locationOrigin: self.location.origin,
	beebuzzDomain: BEEBUZZ_DOMAIN,
	now: () => Date.now(),
	showNotification: (title: string, options?: NotificationOptions) =>
		self.registration.showNotification(title, options),
	saveNotification: saveNotificationToStorage,
	matchWindowClients: async (includeUncontrolled: boolean) =>
		(await self.clients.matchAll({
			type: 'window',
			includeUncontrolled
		})) as WindowClient[],
	openWindow: (url: string) => self.clients.openWindow(url),
	claimClients: () => self.clients.claim(),
	skipWaiting: () => self.skipWaiting(),
	getPushSubscription: () => self.registration.pushManager.getSubscription(),
	getDeviceCredentials: () => deviceKeysRepository.getDeviceCredentials(),
	decryptPayload,
	fetch: (input, init) => self.fetch(input, init),
	recordDiagnostic
};

self.addEventListener('push', (event: PushEvent) => {
	event.waitUntil(handlePushEvent(runtimeDeps, event));
});

self.addEventListener('notificationclick', (event: NotificationEvent) => {
	event.waitUntil(handleNotificationClickEvent(runtimeDeps, event));
});

self.addEventListener('install', () => {
	// Service worker installing — no action required.
});

self.addEventListener('activate', (event: ExtendableEvent) => {
	event.waitUntil(handleActivateEvent(runtimeDeps));
});

self.addEventListener('pushsubscriptionchange', (event: ExtendableEvent) => {
	event.waitUntil(handlePushSubscriptionChangeEvent(runtimeDeps));
});

self.addEventListener('message', (event: ExtendableMessageEvent) => {
	event.waitUntil(handleMessageEvent(runtimeDeps, event));
});
