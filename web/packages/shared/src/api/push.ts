// Push notifications and VAPID key API endpoints.
import type { PairingStatus } from './account';
import { api, request } from './client';

/**
 * Push notifications API namespace.
 */
export interface PairDeviceResponse {
	deviceId: string;
	deviceToken: string;
}

export const pushApi = {
	/**
	 * Fetch VAPID public key for push subscriptions.
	 */
	fetchVapidKey: async () => {
		const data = await api.get<{ key: string }>('/vapid-public-key');
		return data.key;
	},

	/**
	 * Pair device with pairing code and return the canonical backend device ID.
	 */
	pairWithCode: async (
		pairingCode: string,
		subscription: PushSubscription,
		ageRecipient: string
	): Promise<PairDeviceResponse> => {
		const subscriptionJson = subscription.toJSON();
		const response = await api.post<{ device_id: string; device_token: string }>('/pairing', {
			pairing_code: pairingCode,
			endpoint: subscriptionJson.endpoint,
			p256dh: subscriptionJson.keys?.p256dh,
			auth: subscriptionJson.keys?.auth,
			age_recipient: ageRecipient
		});
		return { deviceId: response.device_id, deviceToken: response.device_token };
	},

	/**
	 * Check the pairing status for a device using its device token.
	 */
	checkPairingStatus: async (
		deviceId: string,
		deviceToken: string
	): Promise<{ deviceId: string; pairingStatus: PairingStatus }> => {
		const response = await request<{ device_id: string; pairing_status: PairingStatus }>(
			`/pairing/${deviceId}`,
			{
				method: 'GET',
				headers: {
					Authorization: `Bearer ${deviceToken}`
				}
			}
		);
		return { deviceId: response.device_id, pairingStatus: response.pairing_status };
	}
};
