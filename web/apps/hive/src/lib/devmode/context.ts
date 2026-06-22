import type { HiveSafeContext } from './types';
import { getNotificationRuntimeMetadata } from '$lib/services/runtime-metadata-repository';

export async function collectHiveSafeContext(): Promise<HiveSafeContext> {
	const nav = navigator;
	const ua = nav.userAgent;

	let browserFamily = 'unknown';
	let browserVersionMajor = '0';
	if (ua.includes('Chrome/') && !ua.includes('Edg/')) {
		browserFamily = 'Chrome';
		const match = ua.match(/Chrome\/(\d+)/);
		if (match) browserVersionMajor = match[1];
	} else if (ua.includes('Safari/') && !ua.includes('Chrome/')) {
		browserFamily = 'Safari';
		const match = ua.match(/Version\/(\d+)/);
		if (match) browserVersionMajor = match[1];
	} else if (ua.includes('Firefox/')) {
		browserFamily = 'Firefox';
		const match = ua.match(/Firefox\/(\d+)/);
		if (match) browserVersionMajor = match[1];
	} else if (ua.includes('Edg/')) {
		browserFamily = 'Edge';
		const match = ua.match(/Edg\/(\d+)/);
		if (match) browserVersionMajor = match[1];
	}

	let os: HiveSafeContext['os'] = 'unknown';
	if (ua.includes('Windows')) os = 'Windows';
	else if (ua.includes('Mac OS X') || ua.includes('macOS')) os = 'macOS';
	else if (ua.includes('Linux')) os = 'Linux';
	else if (ua.includes('Android')) os = 'Android';
	else if (ua.includes('iPhone') || ua.includes('iPad')) os = 'iOS';

	let displayMode: HiveSafeContext['display_mode'] = 'browser';
	if (typeof window !== 'undefined') {
		if (window.matchMedia('(display-mode: standalone)').matches) displayMode = 'standalone';
		else if (window.matchMedia('(display-mode: minimal-ui)').matches) displayMode = 'minimal-ui';
		else if (window.matchMedia('(display-mode: fullscreen)').matches) displayMode = 'fullscreen';
	}

	let notificationPermission: HiveSafeContext['notification_permission'] = 'not-supported';
	if ('Notification' in window) {
		const raw = Notification.permission;
		if (raw === 'granted' || raw === 'denied' || raw === 'default') {
			notificationPermission = raw;
		}
	}

	const swSupport = 'serviceWorker' in navigator;
	let swState: HiveSafeContext['service_worker_state'] = 'not-supported';
	if (swSupport) {
		try {
			const reg = await navigator.serviceWorker.getRegistration();
			swState = reg?.active
				? 'active'
				: reg?.waiting
					? 'waiting'
					: reg?.installing
						? 'installing'
						: 'no-registration';
		} catch {
			swState = 'error';
		}
	}

	let pushSupport = false;
	let pushPresent = false;
	if ('PushManager' in window && swSupport) {
		pushSupport = true;
		try {
			const reg = await navigator.serviceWorker.getRegistration();
			if (reg) {
				const sub = await reg.pushManager.getSubscription();
				pushPresent = sub !== null;
			}
		} catch {
			// ignore
		}
	}

	const webCryptoSupported = typeof crypto !== 'undefined' && !!crypto.subtle;
	const x25519Supported = webCryptoSupported
		? await crypto.subtle
				.generateKey({ name: 'X25519' } as EcKeyGenParams, false, ['deriveBits'])
				.then(() => true)
				.catch(() => false)
		: false;

	const idbAvailable = (() => {
		try {
			return typeof indexedDB !== 'undefined' && !!indexedDB.open;
		} catch {
			return false;
		}
	})();

	const runtimeMetadata = await getNotificationRuntimeMetadata();

	return {
		app_version: String(import.meta.env.VITE_BEEBUZZ_VERSION || 'dev'),
		browser_family: browserFamily,
		browser_version_major: browserVersionMajor,
		os,
		display_mode: displayMode,
		notification_permission: notificationPermission,
		service_worker_supported: swSupport,
		service_worker_state: swState,
		push_supported: pushSupport,
		push_present: pushPresent,
		webcrypto_supported: webCryptoSupported,
		x25519_supported: x25519Supported,
		indexeddb_available: idbAvailable,
		network_online: navigator.onLine,
		last_notification_received_at: runtimeMetadata.lastNotificationReceivedAt,
		last_notification_received_via: runtimeMetadata.lastNotificationReceivedVia
	};
}
