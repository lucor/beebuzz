import { deleteAllKeys } from '$lib/services/encryption';
import { unsubscribeFromPush } from '$lib/services/push';
import { logger } from '@beebuzz/shared/logger';

/** Clears stale local pairing state so the next pairing starts from a clean browser state. */
export const cleanupStalePairingState = async (): Promise<void> => {
	try {
		await unsubscribeFromPush();
	} catch (error: unknown) {
		logger.warn('Failed to unsubscribe stale push subscription', { error: String(error) });
	}

	try {
		await deleteAllKeys();
	} catch (error: unknown) {
		logger.warn('Failed to clear stale encryption keys', { error: String(error) });
	}
};
