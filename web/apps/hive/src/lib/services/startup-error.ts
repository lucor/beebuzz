import { ApiError } from '@beebuzz/shared/errors';

const DEFAULT_STARTUP_ERROR =
	'Hive could not finish startup. Reload or return to pairing to reconnect.';

/** Converts a thrown startup error into a user-facing message with the failing step when possible. */
export const formatStartupError = (error: unknown): string => {
	if (error instanceof ApiError) {
		if (error.code === 'NETWORK_ERROR') {
			return 'Hive could not reach the BeeBuzz API. Check the dev server and network, then retry.';
		}

		return `Hive startup failed: ${error.userMessage}`;
	}

	if (error instanceof Error) {
		if (error.message.trim().length > 0) {
			return `Hive startup failed: ${error.message}`;
		}
	}

	return DEFAULT_STARTUP_ERROR;
};
