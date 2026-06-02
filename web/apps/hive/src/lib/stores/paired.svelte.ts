// Reactive paired state derived from push subscription + encryption key existence.
import { browser } from '$app/environment';
import { hasPairedIdentity } from '$lib/services/encryption';
import { checkPairedState } from '$lib/services/pairing-state';
import { logger } from '@beebuzz/shared/logger';

/** Creates the reactive paired state singleton. */
const createPairedStore = () => {
	let isPaired = $state(false);
	let loading = $state(true);

	/** Checks push subscription and encryption key to determine paired state. */
	const check = async (): Promise<boolean> => {
		if (!browser) {
			loading = false;
			return false;
		}

		loading = true;

		try {
			if (!('serviceWorker' in navigator)) {
				isPaired = false;
				return false;
			}

			const keyExists = await checkPairedState({
				notificationPermission: Notification.permission,
				getRegistration: async () => navigator.serviceWorker.getRegistration(),
				hasIdentity: hasPairedIdentity
			});
			isPaired = keyExists;
			return keyExists;
		} catch (err) {
			logger.warn('Paired state check failed', { error: String(err) });
			isPaired = false;
			return false;
		} finally {
			loading = false;
		}
	};

	/** Marks the device as unpaired without async checks. */
	const clear = () => {
		isPaired = false;
	};

	return {
		get isPaired() {
			return isPaired;
		},
		get loading() {
			return loading;
		},
		check,
		clear
	};
};

export const paired = createPairedStore();
