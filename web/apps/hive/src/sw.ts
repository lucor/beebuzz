import { Decrypter } from 'age-encryption';
import { getDeviceIdentity, MissingDeviceIdentityError } from './lib/services/encryption';
import { deviceKeysRepository } from './lib/services/device-keys-repository';
import { notificationsRepository } from './lib/services/notifications-repository';
import { openHiveDB } from './lib/services/hive-db';
import type { HiveDiagnosticDescriptor, HiveLogData } from './lib/devmode/types';
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
const CACHE_PREFIX = 'beebuzz-hive-';
const APP_CACHE = `${CACHE_PREFIX}${import.meta.env.VITE_BEEBUZZ_VERSION || 'dev'}`;
const CORE_ASSETS = ['/', '/manifest.json', '/assets/manifest-icon-192.maskable.png'];

function isSameOrigin(url: URL): boolean {
	return url.origin === self.location.origin;
}

function isCacheableResponse(response: Response): boolean {
	return response.ok && (response.type === 'basic' || response.type === 'default');
}

function isImmutableAsset(url: URL): boolean {
	return url.pathname.startsWith('/_app/immutable/');
}

function isRuntimeAsset(request: Request, url: URL): boolean {
	if (url.pathname.startsWith('/assets/')) return true;
	if (url.pathname === '/manifest.json') return true;
	return ['script', 'style', 'font', 'image', 'manifest'].includes(request.destination);
}

async function cacheResponse(request: Request, response: Response): Promise<Response> {
	if (isCacheableResponse(response)) {
		const cache = await caches.open(APP_CACHE);
		await cache.put(request, response.clone());
	}
	return response;
}

async function cacheFirst(request: Request): Promise<Response> {
	const cache = await caches.open(APP_CACHE);
	const cached = await cache.match(request);
	if (cached) return cached;
	return cacheResponse(request, await fetch(request));
}

async function networkFirst(request: Request): Promise<Response> {
	const cache = await caches.open(APP_CACHE);
	try {
		return await cacheResponse(request, await fetch(request));
	} catch (error) {
		const cached = await cache.match(request);
		if (cached) return cached;
		throw error;
	}
}

async function navigationResponse(request: Request): Promise<Response> {
	const cache = await caches.open(APP_CACHE);
	try {
		return await cacheResponse(request, await fetch(request));
	} catch (error) {
		const cachedPage = await cache.match(request);
		if (cachedPage) return cachedPage;
		const appShell = await cache.match('/');
		if (appShell) return appShell;
		throw error;
	}
}

async function deleteOldCaches(): Promise<void> {
	const keys = await caches.keys();
	await Promise.all(
		keys.filter((k) => k.startsWith(CACHE_PREFIX) && k !== APP_CACHE).map((k) => caches.delete(k))
	);
}

async function precacheCoreAssets(): Promise<void> {
	const cache = await caches.open(APP_CACHE);
	await Promise.all(
		CORE_ASSETS.map(async (asset) => {
			try {
				const response = await fetch(asset, { cache: 'reload' });
				if (isCacheableResponse(response)) {
					await cache.put(asset, response);
				}
			} catch {
				// Core cache is best-effort; push handling must not depend on it.
			}
		})
	);
}

async function handleFetchEvent(event: FetchEvent): Promise<Response> {
	const request = event.request;
	if (request.method !== 'GET') return fetch(request);

	const url = new URL(request.url);
	if (!isSameOrigin(url)) return fetch(request);

	if (request.mode === 'navigate') {
		return navigationResponse(request);
	}

	if (isImmutableAsset(url)) {
		return cacheFirst(request);
	}

	if (isRuntimeAsset(request, url)) {
		return networkFirst(request);
	}

	return fetch(request);
}

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
	_diagnostic: HiveDiagnosticDescriptor,
	_message: string,
	_data?: HiveLogData
): void {
	// Diagnostics are recorded by the app shell. The service worker
	// forwards events through postMessage instead of writing directly
	// to IndexedDB to avoid lock contention. The app shell's message
	// handler is responsible for recording these events if Developer
	// Mode is enabled.
	try {
		void self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
			for (const client of clients) {
				const data: HiveLogData = {
					boundary: _diagnostic.boundary,
					transport: _diagnostic.transport,
					..._data
				};
				client.postMessage({
					type: 'DIAGNOSTIC_EVENT',
					scope: _diagnostic.scope,
					event: _diagnostic.event,
					message: _message,
					data
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
	warmupHiveDB: async () => {
		const db = await openHiveDB();
		db.close();
	},
	recordDiagnostic
};

self.addEventListener('push', (event: PushEvent) => {
	event.waitUntil(handlePushEvent(runtimeDeps, event));
});

self.addEventListener('notificationclick', (event: NotificationEvent) => {
	event.waitUntil(handleNotificationClickEvent(runtimeDeps, event));
});

self.addEventListener('install', (event: ExtendableEvent) => {
	event.waitUntil(precacheCoreAssets());
});

self.addEventListener('activate', (event: ExtendableEvent) => {
	event.waitUntil(Promise.all([handleActivateEvent(runtimeDeps), deleteOldCaches()]));
});

self.addEventListener('pushsubscriptionchange', (event: ExtendableEvent) => {
	event.waitUntil(handlePushSubscriptionChangeEvent(runtimeDeps));
});

self.addEventListener('message', (event: ExtendableMessageEvent) => {
	event.waitUntil(handleMessageEvent(runtimeDeps, event));
});

self.addEventListener('fetch', (event: FetchEvent) => {
	event.respondWith(handleFetchEvent(event));
});
