// Push notifications and VAPID key API endpoints.
import type { PairingStatus } from './account';
import { api, request } from './client';
import { API_URL } from '../config';

/**
 * Push notifications API namespace.
 */
export interface PairDeviceResponse {
	deviceId: string;
	deviceToken: string;
}

export interface DeviceNotificationSyncItem {
	id: string;
	delivery_mode: 'server_trusted' | 'e2e';
	payload: unknown;
	sent_at: string;
	expires_at: string;
}

export interface DeviceNotificationSyncResponse {
	notifications: DeviceNotificationSyncItem[];
	next_cursor: string | null;
	gap: boolean;
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
	},

	/**
	 * Recover recently missed notifications for a paired Hive device.
	 */
	syncDeviceNotifications: async (
		deviceId: string,
		deviceToken: string,
		after?: string,
		limit = 50
	): Promise<DeviceNotificationSyncResponse> => {
		const params = new URLSearchParams({ limit: String(limit) });
		if (after) {
			params.set('after', after);
		}

		let response: Response;
		try {
			response = await fetch(
				`${API_URL}/v1/devices/${encodeURIComponent(deviceId)}/notifications?${params}`,
				{
					method: 'GET',
					headers: {
						Authorization: `Bearer ${deviceToken}`
					}
				}
			);
		} catch {
			throw new Error('Network error');
		}

		if (!response.ok) {
			throw new Error(`Notification sync failed: ${response.status}`);
		}

		return response.json() as Promise<DeviceNotificationSyncResponse>;
	}
};
