// Browser capability detection for push notification support.
// Pure helpers — no UA sniffing for capability gating.
import * as age from 'age-encryption';
import { logger } from '@beebuzz/shared/logger';

export interface CapabilityResult {
	secure: boolean;
	serviceWorker: boolean;
	pushManager: boolean;
	notification: boolean;
	encryption: boolean;
	standalone: boolean;
	iosWebKit: boolean;
	installPromptAvailable: boolean;
}

let encryptionProbeResult: boolean | null = null;

/**
 * Probes whether the browser supports X25519 key generation and age recipient derivation.
 * Result is cached after first call.
 */
const probeEncryptionSupport = async (): Promise<boolean> => {
	if (encryptionProbeResult !== null) return encryptionProbeResult;

	try {
		const { privateKey } = (await crypto.subtle.generateKey({ name: 'X25519' }, false, [
			'deriveBits'
		])) as CryptoKeyPair;

		try {
			await age.identityToRecipient(privateKey);
		} catch (e) {
			logger.error('Encryption probe: age identityToRecipient failed', {
				error: String(e)
			});
			encryptionProbeResult = false;
			return encryptionProbeResult;
		}

		encryptionProbeResult = true;
	} catch (e) {
		logger.error('Encryption probe: X25519 generateKey failed', { error: String(e) });
		encryptionProbeResult = false;
	}

	return encryptionProbeResult;
};

/** Detects whether the app is running in standalone (installed) mode. */
export const isStandalone = (): boolean => {
	if (typeof window === 'undefined') return false;

	// iOS standalone check
	if ('standalone' in navigator && (navigator as { standalone?: boolean }).standalone === true) {
		return true;
	}

	// Standard display-mode check
	return window.matchMedia('(display-mode: standalone)').matches;
};

/**
 * UA heuristic for iOS/iPadOS WebKit. Used for install-flow routing and UI copy,
 * never used to claim push capability. Named honestly as a heuristic.
 */
export const isLikelyIOSWebKit = (): boolean => {
	if (typeof navigator === 'undefined') return false;

	return (
		/iPad|iPhone|iPod/.test(navigator.userAgent) ||
		// eslint-disable-next-line @typescript-eslint/no-deprecated -- navigator.platform is deprecated but navigator.userAgentData is not universally available; this heuristic is intentionally conservative.
		(navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
	);
};

/**
 * UA heuristic for Safari on macOS (Sonoma+). Used ONLY for install copy.
 * Safari does not fire beforeinstallprompt but supports "Add to Dock".
 */
export const isLikelySafariMacOS = (): boolean => {
	if (typeof navigator === 'undefined') return false;

	const ua = navigator.userAgent;
	// Must be Mac, not iOS (no touch or maxTouchPoints <= 1)
	const isMac = /Macintosh/.test(ua) && navigator.maxTouchPoints <= 1;
	// Safari UA contains "Safari" but not "Chrome" or "Chromium"
	const isSafari = /Safari/.test(ua) && !/Chrome|Chromium/.test(ua);

	return isMac && isSafari;
};

/** Runs all capability checks and returns the result. Encryption probe is async. */
export const detectCapabilities = async (): Promise<CapabilityResult> => {
	if (typeof window === 'undefined') {
		return {
			secure: false,
			serviceWorker: false,
			pushManager: false,
			notification: false,
			encryption: false,
			standalone: false,
			iosWebKit: false,
			installPromptAvailable: false
		};
	}

	const secure = window.isSecureContext;
	const serviceWorker = 'serviceWorker' in navigator;
	const pushManager = 'PushManager' in window;
	const notification = 'Notification' in window;
	const encryption = secure ? await probeEncryptionSupport() : false;
	const standalone = isStandalone();
	const iosWebKit = isLikelyIOSWebKit();

	return {
		secure,
		serviceWorker,
		pushManager,
		notification,
		encryption,
		standalone,
		iosWebKit,
		installPromptAvailable: false // set async via beforeinstallprompt listener
	};
};

/** Returns true if the browser has base capabilities (available even before standalone on iOS). */
export const hasBaseSupport = (cap: CapabilityResult): boolean => {
	return cap.secure && cap.serviceWorker && cap.encryption;
};

/** Returns true if the browser has push-specific APIs (only available in standalone on iOS). */
export const hasPushRuntimeSupport = (cap: CapabilityResult): boolean => {
	return cap.pushManager && cap.notification;
};

/** Returns true if the browser has all required capabilities for BeeBuzz push. */
export const isSupported = (cap: CapabilityResult): boolean => {
	return hasBaseSupport(cap) && hasPushRuntimeSupport(cap);
};

/**
 * iOS WebKit only exposes Push/Notification APIs in standalone mode.
 * Used for install-flow routing only; never used to claim push capability.
 */
export const requiresStandaloneForPush = (cap: CapabilityResult): boolean => {
	return cap.iosWebKit && !cap.standalone;
};

/**
 * UA heuristic for Firefox. Used ONLY for install copy (Firefox has no PWA install support).
 * Firefox UA contains "Firefox" and does not contain "Seamonkey".
 */
export const isLikelyFirefox = (): boolean => {
	if (typeof navigator === 'undefined') return false;
	return /Firefox/.test(navigator.userAgent) && !/Seamonkey/.test(navigator.userAgent);
};
