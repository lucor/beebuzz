// Push notification service layer. Owns business logic and logging for push operations.
import { browser } from '$app/environment';
import { base } from '$app/paths';
import { pushApi } from '@beebuzz/shared/api';
import { logger } from '@beebuzz/shared/logger';
import { ApiError } from '@beebuzz/shared/errors';

export interface PushSupport {
	notifications: boolean;
	serviceWorker: boolean;
	supported: boolean;
	errorMessage: string | null;
}

/** Checks browser support for push notifications. */
export const checkPushSupport = (): PushSupport => {
	if (!browser) {
		return {
			notifications: false,
			serviceWorker: false,
			supported: false,
			errorMessage: 'Not running in browser'
		};
	}

	const notifications = 'Notification' in window;
	const serviceWorker = 'serviceWorker' in navigator;
	const supported = notifications && serviceWorker;

	let errorMessage: string | null = null;
	if (!serviceWorker) {
		errorMessage =
			"Service Workers are required for notifications. Please ensure you're not in private/incognito mode.";
	} else if (!notifications) {
		errorMessage =
			"Your browser doesn't support push notifications. Please use Chrome, Firefox, Safari, or Edge.";
	}

	return { notifications, serviceWorker, supported, errorMessage };
};

let registration: ServiceWorkerRegistration | null = null;
let vapidPublicKey: string | undefined;

/** Registers the service worker. Returns cached registration if already registered. */
export const registerServiceWorker = async (): Promise<ServiceWorkerRegistration> => {
	if (!browser) throw new Error('Not in browser');
	if (registration) return registration;

	// eslint-disable-next-line @typescript-eslint/no-deprecated -- `base` is deprecated but `resolve()` only accepts known SvelteKit routes; `/sw.js` is a static asset, not a route.
	registration = await navigator.serviceWorker.register(`${base}/sw.js`);
	logger.debug('Service Worker registered');
	return registration;
};

/** Returns the ready service worker registration. */
export const getReadyRegistration = async (): Promise<ServiceWorkerRegistration> => {
	if (!browser) throw new Error('Not in browser');
	return navigator.serviceWorker.ready;
};

/** Returns the cached service worker registration. */
export const getRegistration = (): ServiceWorkerRegistration | null => {
	return registration;
};

/** Fetches the VAPID public key from the backend. Caches after first fetch. */
export const getVapidKey = async (): Promise<string> => {
	if (vapidPublicKey) return vapidPublicKey;

	try {
		const key = await pushApi.fetchVapidKey();
		vapidPublicKey = key;
		logger.debug('VAPID public key fetched', { length: key.length });
		return key;
	} catch (error: unknown) {
		if (error instanceof ApiError) {
			logger.error('fetch VAPID key failed', { code: error.code, message: error.message });
		}
		throw error;
	}
};

/** Subscribes to push notifications, replacing any existing subscription with mismatched keys. */
export const subscribeToPush = async (
	reg: ServiceWorkerRegistration,
	vapidKey: string
): Promise<PushSubscription> => {
	const existing = await reg.pushManager.getSubscription();
	if (existing) {
		const existingKey = existing.options?.applicationServerKey;
		const newKey = urlBase64ToUint8Array(vapidKey);
		const keysMatch =
			existingKey &&
			new Uint8Array(existingKey).length === newKey.length &&
			new Uint8Array(existingKey).every((b, i) => b === newKey[i]);

		if (keysMatch) {
			logger.debug('Existing push subscription reused (VAPID keys match)');
			return existing;
		}

		logger.warn('VAPID key mismatch detected, unsubscribing old subscription');
		await existing.unsubscribe();
	}

	const subscription = await reg.pushManager.subscribe({
		userVisibleOnly: true,
		applicationServerKey: urlBase64ToUint8Array(vapidKey) as BufferSource
	});

	logger.debug('New push subscription created', { endpoint: subscription.endpoint });
	return subscription;
};

/** Unsubscribes from push notifications if a registration exists. */
export const unsubscribeFromPush = async (): Promise<void> => {
	if (!browser || !('serviceWorker' in navigator)) {
		return;
	}

	const activeRegistration =
		registration ?? (await navigator.serviceWorker.getRegistration()) ?? null;
	if (!activeRegistration) return;

	const subscription = await activeRegistration.pushManager.getSubscription();
	if (subscription) {
		await subscription.unsubscribe();
	}
};

/** Requests notification permission from the user. */
export const requestNotificationPermission = async (): Promise<NotificationPermission> => {
	if (Notification.permission === 'granted') {
		return 'granted';
	}
	return Notification.requestPermission();
};

/** Pairs a device with the backend using a pairing code and push subscription. */
export const pairDevice = async (
	pairingCode: string,
	subscription: PushSubscription,
	ageRecipient: string
): Promise<{ deviceId: string; deviceToken: string }> => {
	try {
		const response = await pushApi.pairWithCode(pairingCode, subscription, ageRecipient);
		logger.info('Device paired');
		return response;
	} catch (error: unknown) {
		if (error instanceof ApiError) {
			logger.error('device pairing failed', { code: error.code, message: error.message });
		}
		throw error;
	}
};

/** Converts a URL-safe base64 string to a Uint8Array. */
const urlBase64ToUint8Array = (base64String: string): Uint8Array => {
	const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
	const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
	const rawData = window.atob(base64);
	const outputArray = new Uint8Array(rawData.length);

	for (let i = 0; i < rawData.length; ++i) {
		outputArray[i] = rawData.charCodeAt(i);
	}
	return outputArray;
};
