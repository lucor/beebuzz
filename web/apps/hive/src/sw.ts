import { Decrypter } from 'age-encryption';
import { getDeviceIdentity } from './lib/services/encryption';
import { notificationsRepository } from './lib/services/notifications-repository';
import {
	handleActivateEvent,
	handleMessageEvent,
	handleNotificationClickEvent,
	handlePushEvent,
	handlePushSubscriptionChangeEvent,
	type NotificationAttachmentEnvelope
} from './sw-runtime';

declare const self: ServiceWorkerGlobalScope;

const DEBUG = import.meta.env.VITE_BEEBUZZ_DEBUG === true;
const BEEBUZZ_DOMAIN = import.meta.env.VITE_BEEBUZZ_DOMAIN as string | undefined;

async function decryptPayload(data: ArrayBuffer): Promise<string> {
	if (DEBUG) {
		console.log(`[AGE] Loading identity from IndexedDB...`);
	}
	const identity = await getDeviceIdentity();
	if (!identity) {
		throw new Error('❌ No encryption key found in IndexedDB - device may not be paired correctly');
	}
	if (DEBUG) {
		console.log(`[AGE] ✅ Identity loaded, starting decryption...`);
	}

	const ciphertextBytes = new Uint8Array(data).length;
	const startDecrypt = performance.now();

	const d = new Decrypter();
	d.addIdentity(identity);
	const decrypted = await d.decrypt(new Uint8Array(data), 'text');

	const duration = performance.now() - startDecrypt;
	if (DEBUG) {
		console.log(
			`[AGE] ✅ DECRYPTION SUCCESS: duration=${duration.toFixed(2)}ms, ciphertext=${ciphertextBytes}B, plaintext=${decrypted.length}B`
		);
	}

	return decrypted;
}

function saveNotificationToStorage(input: {
	id: string;
	title: string;
	body: string;
	topic: string;
	sentAt: string;
	topicId?: string;
	attachment?: NotificationAttachmentEnvelope;
	priority?: string;
}): Promise<void> {
	return notificationsRepository.save(input).then(
		() => {
			if (DEBUG) {
				console.log('Notification saved to IndexedDB');
			}
		},
		(error) => {
			if (DEBUG) {
				console.error('Failed to add notification to IndexedDB:', error);
			}
			throw error;
		}
	);
}

const runtimeDeps = {
	debug: DEBUG,
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
	decryptPayload,
	fetch
};

self.addEventListener('push', (event: PushEvent) => {
	event.waitUntil(handlePushEvent(runtimeDeps, event));
});

self.addEventListener('notificationclick', (event: NotificationEvent) => {
	event.waitUntil(handleNotificationClickEvent(runtimeDeps, event));
});

self.addEventListener('install', () => {
	if (DEBUG) {
		console.log('Service Worker installing...');
	}
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
