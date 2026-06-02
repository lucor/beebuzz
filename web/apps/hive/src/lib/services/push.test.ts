import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
	checkPairingStatus,
	PAIRING_STATUS_CHECK_REASON,
	PAIRING_STATUS_CHECK_STATUS,
	RECONNECT_REQUIRED_REASON
} from './pairing-check';

vi.mock('@beebuzz/shared/config', () => ({
	API_URL: 'https://api.example.test'
}));

describe('checkPairingStatus', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('maps paired responses to ok', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(() =>
				Promise.resolve(new Response(JSON.stringify({ pairing_status: 'paired' }), { status: 200 }))
			)
		);

		await expect(checkPairingStatus('dev-1', 'tok-1')).resolves.toEqual({
			status: PAIRING_STATUS_CHECK_STATUS.OK,
			pairingStatus: 'paired'
		});
	});

	it('maps 401 responses to reconnect-required invalid_device_token', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(() => Promise.resolve(new Response(null, { status: 401 })))
		);

		await expect(checkPairingStatus('dev-1', 'tok-1')).resolves.toEqual({
			status: PAIRING_STATUS_CHECK_STATUS.RECONNECT_REQUIRED,
			reason: RECONNECT_REQUIRED_REASON.INVALID_DEVICE_TOKEN
		});
	});

	it('maps degraded pairing status to reconnect-required', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(() =>
				Promise.resolve(
					new Response(JSON.stringify({ pairing_status: 'subscription_gone' }), { status: 200 })
				)
			)
		);

		await expect(checkPairingStatus('dev-1', 'tok-1')).resolves.toEqual({
			status: PAIRING_STATUS_CHECK_STATUS.RECONNECT_REQUIRED,
			reason: RECONNECT_REQUIRED_REASON.SUBSCRIPTION_GONE
		});
	});

	it('maps network failures to transient-backend-error', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(() => Promise.reject(new Error('network failed')))
		);

		await expect(checkPairingStatus('dev-1', 'tok-1')).resolves.toEqual({
			status: PAIRING_STATUS_CHECK_STATUS.TRANSIENT_BACKEND_ERROR,
			reason: PAIRING_STATUS_CHECK_REASON.BACKEND_UNREACHABLE
		});
	});
});
