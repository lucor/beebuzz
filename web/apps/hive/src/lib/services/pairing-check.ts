// Pairing-status check logic. No SvelteKit dependencies — safe for vitest.
import { API_URL } from '@beebuzz/shared/config';
import type { PairingStatus } from '@beebuzz/shared/api';

export const RECONNECT_REQUIRED_REASON = {
	SUBSCRIPTION_GONE: 'subscription_gone',
	UNPAIRED: 'unpaired',
	PENDING: 'pending',
	INVALID_DEVICE_TOKEN: 'invalid_device_token',
	MISSING_DEVICE_TOKEN: 'missing_device_token'
} as const;

export const PAIRING_STATUS_CHECK_STATUS = {
	OK: 'ok',
	RECONNECT_REQUIRED: 'reconnect-required',
	TRANSIENT_BACKEND_ERROR: 'transient-backend-error'
} as const;

export const PAIRING_STATUS_CHECK_REASON = {
	BACKEND_UNREACHABLE: 'backend_unreachable'
} as const;

type ValueOf<T> = T[keyof T];

export type ReconnectRequiredReason = ValueOf<typeof RECONNECT_REQUIRED_REASON>;
type RemoteReconnectRequiredReason = Exclude<
	ReconnectRequiredReason,
	typeof RECONNECT_REQUIRED_REASON.MISSING_DEVICE_TOKEN
>;
export type PairingStatusResult =
	| { status: typeof PAIRING_STATUS_CHECK_STATUS.OK; pairingStatus: 'paired' }
	| {
			status: typeof PAIRING_STATUS_CHECK_STATUS.RECONNECT_REQUIRED;
			reason: RemoteReconnectRequiredReason;
	  }
	| {
			status: typeof PAIRING_STATUS_CHECK_STATUS.TRANSIENT_BACKEND_ERROR;
			reason: typeof PAIRING_STATUS_CHECK_REASON.BACKEND_UNREACHABLE;
	  };

/** Checks pairing status with the backend. Uses raw fetch to avoid shared client's 401 redirect. */
export const checkPairingStatus = async (
	deviceId: string,
	deviceToken: string
): Promise<PairingStatusResult> => {
	try {
		const response = await fetch(`${API_URL}/v1/pairing/${deviceId}`, {
			method: 'GET',
			headers: {
				Authorization: `Bearer ${deviceToken}`
			}
		});

		if (response.status === 401) {
			return {
				status: PAIRING_STATUS_CHECK_STATUS.RECONNECT_REQUIRED,
				reason: RECONNECT_REQUIRED_REASON.INVALID_DEVICE_TOKEN
			};
		}

		if (response.status >= 500) {
			return {
				status: PAIRING_STATUS_CHECK_STATUS.TRANSIENT_BACKEND_ERROR,
				reason: PAIRING_STATUS_CHECK_REASON.BACKEND_UNREACHABLE
			};
		}

		if (!response.ok) {
			return {
				status: PAIRING_STATUS_CHECK_STATUS.TRANSIENT_BACKEND_ERROR,
				reason: PAIRING_STATUS_CHECK_REASON.BACKEND_UNREACHABLE
			};
		}

		const data = (await response.json()) as { pairing_status?: PairingStatus };
		const status = data.pairing_status;

		if (status === 'paired') {
			return { status: PAIRING_STATUS_CHECK_STATUS.OK, pairingStatus: status };
		}

		if (
			status === RECONNECT_REQUIRED_REASON.SUBSCRIPTION_GONE ||
			status === RECONNECT_REQUIRED_REASON.UNPAIRED ||
			status === RECONNECT_REQUIRED_REASON.PENDING
		) {
			return { status: PAIRING_STATUS_CHECK_STATUS.RECONNECT_REQUIRED, reason: status };
		}

		return {
			status: PAIRING_STATUS_CHECK_STATUS.TRANSIENT_BACKEND_ERROR,
			reason: PAIRING_STATUS_CHECK_REASON.BACKEND_UNREACHABLE
		};
	} catch {
		return {
			status: PAIRING_STATUS_CHECK_STATUS.TRANSIENT_BACKEND_ERROR,
			reason: PAIRING_STATUS_CHECK_REASON.BACKEND_UNREACHABLE
		};
	}
};
