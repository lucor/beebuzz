import type { PushDebugSnapshot } from '$lib/types/encryption';

/** Collects a snapshot of the service worker and push subscription state. */
export async function loadPushDebugSnapshot(): Promise<PushDebugSnapshot> {
	const registration = await navigator.serviceWorker.getRegistration();
	const subscription = registration ? await registration.pushManager.getSubscription() : null;
	const subscriptionJson = subscription?.toJSON();
	const endpointHost = subscription ? new URL(subscription.endpoint).host : null;

	return {
		userAgent: navigator.userAgent,
		controllerScriptURL: navigator.serviceWorker.controller?.scriptURL ?? null,
		controllerState: navigator.serviceWorker.controller?.state ?? null,
		registrationScope: registration?.scope ?? null,
		registrationInstallingState: registration?.installing?.state ?? null,
		registrationWaitingState: registration?.waiting?.state ?? null,
		registrationActiveState: registration?.active?.state ?? null,
		subscriptionEndpointHost: endpointHost,
		subscriptionP256dhLength: subscriptionJson?.keys?.p256dh?.length ?? 0,
		subscriptionAuthLength: subscriptionJson?.keys?.auth?.length ?? 0
	};
}

/** Forces the browser to check for a newer service worker script. */
export async function updateServiceWorkerRegistration(): Promise<void> {
	const registration = await navigator.serviceWorker.getRegistration();
	if (!registration) {
		throw new Error('service worker registration not found');
	}

	await registration.update();
}

/** Promotes a waiting service worker to active by asking it to skip waiting. */
export async function activateWaitingServiceWorker(): Promise<boolean> {
	const registration = await navigator.serviceWorker.getRegistration();
	const waitingWorker = registration?.waiting;
	if (!waitingWorker) {
		return false;
	}

	waitingWorker.postMessage({ type: 'SKIP_WAITING' });
	return true;
}

/** Unregisters the current service worker registration. */
export async function unregisterServiceWorker(): Promise<boolean> {
	const registration = await navigator.serviceWorker.getRegistration();
	if (!registration) {
		return false;
	}

	return registration.unregister();
}
