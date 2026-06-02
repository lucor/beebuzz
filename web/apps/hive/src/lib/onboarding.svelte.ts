// Onboarding state machine for Hive PWA.
// Drives the install → permission → pairing flow.
import {
	detectCapabilities,
	isSupported,
	hasBaseSupport,
	requiresStandaloneForPush,
	isStandalone,
	isLikelySafariMacOS,
	isLikelyFirefox
} from '$lib/services/capability';
import {
	initInstallListener,
	canInstallNatively,
	triggerInstall,
	wasInstalled,
	wasInstalledPreviously
} from '$lib/services/install';
import {
	requestNotificationPermission,
	getReadyRegistration,
	subscribeToPush,
	getVapidKey,
	pairDevice,
	registerServiceWorker
} from '$lib/services/push';
import {
	checkPairingStatus,
	PAIRING_STATUS_CHECK_REASON,
	PAIRING_STATUS_CHECK_STATUS,
	RECONNECT_REQUIRED_REASON,
	type ReconnectRequiredReason
} from '$lib/services/pairing-check';
import {
	finalizePendingStoredKeyPair,
	generateAndStorePendingKeyPair,
	hasPairedIdentity
} from '$lib/services/encryption';
import { cleanupStalePairingState } from '$lib/services/startup-recovery';
import { deviceKeysRepository } from '$lib/services/device-keys-repository';
import { paired } from '$lib/stores/paired.svelte';
import { notificationsStore } from '$lib/stores/notifications.svelte';
import { formatStartupError } from '$lib/services/startup-error';
import { withTimeout } from '$lib/utils/async';
import { logger } from '@beebuzz/shared/logger';
import { ApiError } from '@beebuzz/shared/errors';
import type { CapabilityResult } from '$lib/services/capability';

export const PUSH_STATE_STATUS = {
	OK: 'ok',
	RECONNECT_REQUIRED: 'reconnect-required',
	LOCAL_REPAIR_REQUIRED: 'local-repair-required',
	TRANSIENT_BACKEND_ERROR: 'transient-backend-error'
} as const;

export const LOCAL_REPAIR_REQUIRED_REASON = {
	SUBSCRIPTION_LOST: 'subscription_lost',
	PERMISSION_REVOKED: 'permission_revoked',
	KEYS_LOST: 'keys_lost'
} as const;

export type OnboardingState =
	| 'checking'
	| 'unsupported'
	| 'not-installed'
	| 'ready'
	| 'permission-prompt'
	| 'permission-blocked'
	| 'pairing'
	| 'paired'
	| 'error';

export type InstallPlatform =
	| 'ios'
	| 'safari-macos'
	| 'chromium'
	| 'firefox'
	| 'already-installed'
	| 'browser-fallback';

const STARTUP_TIMEOUT_MS = 10000;

/** Pairing code stashed across permission-blocked state so retry can resume pairing. */
let savedPairingCode = '';

/** Determines which install platform variant to show. */
const detectInstallPlatform = (cap: CapabilityResult): InstallPlatform => {
	if (wasInstalledPreviously()) return 'already-installed';
	if (cap.iosWebKit) return 'ios';
	if (isLikelySafariMacOS()) return 'safari-macos';
	if (canInstallNatively()) return 'chromium';
	if (isLikelyFirefox()) return 'firefox';
	return 'browser-fallback';
};

/**
 * Checks push subscription state on startup for paired devices.
 * Does NOT auto-resubscribe — re-pairing is required if subscription is lost.
 */
export type PushStateResult =
	| { status: typeof PUSH_STATE_STATUS.OK; reason: null }
	| { status: typeof PUSH_STATE_STATUS.RECONNECT_REQUIRED; reason: ReconnectRequiredReason }
	| {
			status: typeof PUSH_STATE_STATUS.LOCAL_REPAIR_REQUIRED;
			reason:
				| typeof LOCAL_REPAIR_REQUIRED_REASON.SUBSCRIPTION_LOST
				| typeof LOCAL_REPAIR_REQUIRED_REASON.PERMISSION_REVOKED
				| typeof LOCAL_REPAIR_REQUIRED_REASON.KEYS_LOST;
	  }
	| {
			status: typeof PUSH_STATE_STATUS.TRANSIENT_BACKEND_ERROR;
			reason: typeof PAIRING_STATUS_CHECK_REASON.BACKEND_UNREACHABLE;
	  };

export const reconcilePushState = async (): Promise<PushStateResult> => {
	if (Notification.permission !== 'granted') {
		return {
			status: PUSH_STATE_STATUS.LOCAL_REPAIR_REQUIRED,
			reason: LOCAL_REPAIR_REQUIRED_REASON.PERMISSION_REVOKED
		};
	}

	const registration = await navigator.serviceWorker.ready;
	const subscription = await registration.pushManager.getSubscription();

	if (!subscription) {
		return {
			status: PUSH_STATE_STATUS.LOCAL_REPAIR_REQUIRED,
			reason: LOCAL_REPAIR_REQUIRED_REASON.SUBSCRIPTION_LOST
		};
	}

	if (!(await hasPairedIdentity())) {
		return {
			status: PUSH_STATE_STATUS.LOCAL_REPAIR_REQUIRED,
			reason: LOCAL_REPAIR_REQUIRED_REASON.KEYS_LOST
		};
	}

	const credentials = await deviceKeysRepository.getDeviceCredentials();
	if (!credentials) {
		return {
			status: PUSH_STATE_STATUS.RECONNECT_REQUIRED,
			reason: RECONNECT_REQUIRED_REASON.MISSING_DEVICE_TOKEN
		};
	}

	const statusResult = await checkPairingStatus(credentials.deviceId, credentials.deviceToken);
	if (statusResult.status === PAIRING_STATUS_CHECK_STATUS.OK) {
		return { status: PUSH_STATE_STATUS.OK, reason: null };
	}
	if (statusResult.status === PAIRING_STATUS_CHECK_STATUS.RECONNECT_REQUIRED) {
		return { status: PUSH_STATE_STATUS.RECONNECT_REQUIRED, reason: statusResult.reason };
	}

	return {
		status: PUSH_STATE_STATUS.TRANSIENT_BACKEND_ERROR,
		reason: PAIRING_STATUS_CHECK_REASON.BACKEND_UNREACHABLE
	};
};

/** Creates the reactive onboarding state machine singleton. */
const createOnboarding = () => {
	let state = $state<OnboardingState>('checking');
	let errorMessage = $state<string | null>(null);
	let capabilities = $state<CapabilityResult | null>(null);
	let installPlatform = $state<InstallPlatform>('chromium');

	/** Initializes capability detection and determines initial state. */
	const init = async () => {
		state = 'checking';
		errorMessage = null;

		try {
			initInstallListener(
				() => {
					// beforeinstallprompt arrived — upgrade to native install if still on install screen
					// Don't override already-installed: user has the app, no need for native prompt
					if (state === 'not-installed' && installPlatform !== 'already-installed') {
						installPlatform = 'chromium';
					}
				},
				() => {
					// appinstalled fired — transition to ready if still on install screen
					if (state === 'not-installed') {
						pollForStandalone();
					}
				}
			);

			const cap = await withTimeout(
				detectCapabilities(),
				STARTUP_TIMEOUT_MS,
				'Capability detection'
			);
			capabilities = cap;
			installPlatform = detectInstallPlatform(cap);

			// iOS WebKit in-browser: Push/Notification APIs only exist in standalone mode.
			// Check base support (secure + SW + encryption) and route to install screen.
			if (requiresStandaloneForPush(cap)) {
				if (!hasBaseSupport(cap)) {
					state = 'unsupported';
					return;
				}
				state = 'not-installed';
				return;
			}

			if (!isSupported(cap)) {
				state = 'unsupported';
				return;
			}

			await withTimeout(registerServiceWorker(), STARTUP_TIMEOUT_MS, 'Service worker registration');

			// Check if already paired (push subscription + encryption key exist)
			const alreadyPaired = await withTimeout(
				paired.check(),
				STARTUP_TIMEOUT_MS,
				'Paired device check'
			);
			if (alreadyPaired) {
				const pushState = await withTimeout(
					reconcilePushState(),
					STARTUP_TIMEOUT_MS,
					'Push state validation'
				);
				if (
					pushState.status === PUSH_STATE_STATUS.OK ||
					pushState.status === PUSH_STATE_STATUS.RECONNECT_REQUIRED ||
					pushState.status === PUSH_STATE_STATUS.TRANSIENT_BACKEND_ERROR
				) {
					state = 'paired';
					return;
				}

				logger.warn('Push state degraded on startup, forcing re-pair', {
					push_state: pushState.status,
					reason: pushState.reason
				});
				await cleanupStalePairingState();
				paired.clear();
			}

			// Already standalone — skip install step
			if (cap.standalone) {
				state = 'ready';
				return;
			}

			// Firefox has no install support — go straight to pairing
			if (installPlatform === 'firefox') {
				state = 'ready';
				return;
			}

			state = 'not-installed';
		} catch (error: unknown) {
			errorMessage = formatStartupError(error);
			state = 'error';
			logger.error('Hive onboarding init failed', { error: String(error) });
		}
	};

	/** Called when the user clicks the native install button (Chromium). */
	const handleNativeInstall = async () => {
		const accepted = await triggerInstall();
		if (accepted || wasInstalled()) {
			// User installed — tell them to reopen from installed app
			// The state stays 'not-installed' until standalone is detected
			// Poll for standalone mode after install
			pollForStandalone();
		}
	};

	/** Polls for standalone mode after install event. */
	const pollForStandalone = () => {
		let attempts = 0;
		const MAX_ATTEMPTS = 60; // 30 seconds
		const INTERVAL_MS = 500;

		const check = () => {
			attempts++;
			if (isStandalone()) {
				// Re-run init to validate push APIs in standalone context (critical for iOS)
				void init();
				return;
			}
			if (attempts < MAX_ATTEMPTS) {
				setTimeout(check, INTERVAL_MS);
			}
		};

		setTimeout(check, INTERVAL_MS);
	};

	/** Skips installation (fallback for "Can't install?" link). */
	const skipInstall = () => {
		state = 'ready';
	};

	/** Starts the pairing flow: requests permission, then subscribes + pairs. */
	const startPairing = async (pairingCode: string) => {
		if (state === 'pairing' || state === 'permission-prompt') return;

		errorMessage = null;
		savedPairingCode = pairingCode;

		// Request notification permission
		state = 'permission-prompt';
		const permission = await requestNotificationPermission();

		if (permission === 'denied') {
			state = 'permission-blocked';
			return;
		}

		if (permission === 'default') {
			// Dismissed — go back to ready to retry
			state = 'ready';
			return;
		}

		// Permission granted — proceed with pairing
		state = 'pairing';

		try {
			const registration = await getReadyRegistration();
			const vapidKey = await getVapidKey();
			const subscription = await subscribeToPush(registration, vapidKey);

			const ageRecipient = await generateAndStorePendingKeyPair();
			const { deviceId, deviceToken } = await pairDevice(
				pairingCode.trim(),
				subscription,
				ageRecipient
			);
			await finalizePendingStoredKeyPair(deviceId);
			await deviceKeysRepository.storeDeviceCredentials(deviceId, deviceToken);
			notificationsStore.activateDevice(deviceId);

			await paired.check();
			await notificationsStore.loadFromIndexedDB();

			state = 'paired';
		} catch (err) {
			await cleanupStalePairingState();
			if (err instanceof ApiError) {
				errorMessage = err.userMessage;
				logger.error('Pairing failed (API)', { code: err.code, message: err.message });
			} else {
				errorMessage = 'Unable to pair device. Please try again.';
				logger.error('Pairing failed', { error: String(err) });
			}
			state = 'error';
		}
	};

	/** Retries from error state — goes back to ready. */
	const retry = () => {
		errorMessage = null;
		state = 'ready';
	};

	/** Retries from permission-blocked state.
	 * Calls requestPermission() instead of reading the cached Notification.permission
	 * property, which can be stale on some browsers (notably Safari macOS) after the
	 * user changes notification settings at the OS level.
	 * If granted, resumes pairing automatically with the previously entered code. */
	const retryPermission = async () => {
		const permission = await Notification.requestPermission();
		if (permission !== 'granted') return;
		await startPairing(savedPairingCode);
	};

	return {
		get state() {
			return state;
		},
		get errorMessage() {
			return errorMessage;
		},
		get capabilities() {
			return capabilities;
		},
		get installPlatform() {
			return installPlatform;
		},
		init,
		handleNativeInstall,
		skipInstall,
		startPairing,
		retry,
		retryPermission
	};
};

export const onboarding = createOnboarding();
