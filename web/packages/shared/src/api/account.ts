// Account management API endpoints (tokens, devices, webhooks).
import { api } from './client';
import { WEBHOOK_URL } from '../config';

export interface AccountUsageDay {
	date: string;
	notifications_created: number;
	delivery_attempts: number;
	deliveries_succeeded: number;
	deliveries_failed: number;
	devices_lost: number;
	notifications_with_attachment: number;
	attachment_bytes_total: number;
	notifications_server_trusted: number;
	notifications_e2e: number;
	sources_cli: number;
	sources_webhook: number;
	sources_api: number;
	sources_internal: number;
}

export interface AccountUsage {
	data: AccountUsageDay[];
}

export type AccountStatus = 'pending' | 'active' | 'blocked';

const ACCOUNT_STATUS = {
	PENDING: 'pending',
	ACTIVE: 'active',
	BLOCKED: 'blocked'
} as const;

export interface AccountStatusAware {
	account_status: AccountStatus;
}

export function userStatusLabel(u: AccountStatusAware): string {
	switch (u.account_status) {
		case ACCOUNT_STATUS.PENDING:
			return 'Pending';
		case ACCOUNT_STATUS.ACTIVE:
			return 'Active';
		case ACCOUNT_STATUS.BLOCKED:
			return 'Blocked';
		default:
			return u.account_status;
	}
}

export function userStatusBadgeClass(u: AccountStatusAware): string {
	switch (u.account_status) {
		case ACCOUNT_STATUS.PENDING:
			return 'badge-warning';
		case ACCOUNT_STATUS.ACTIVE:
			return 'badge-success';
		case ACCOUNT_STATUS.BLOCKED:
			return 'badge-error';
		default:
			return 'badge-ghost';
	}
}

export function userActionInfo(u: AccountStatusAware & { is_admin: boolean }): {
	label: string;
	action: 'approve' | 'block' | 'reactivate' | null;
	class: string;
} {
	if (u.is_admin) return { label: 'Admin', action: null, class: '' };
	switch (u.account_status) {
		case ACCOUNT_STATUS.PENDING:
			return { label: 'Approve', action: 'approve', class: 'btn-success' };
		case ACCOUNT_STATUS.ACTIVE:
			return { label: 'Block', action: 'block', class: 'btn-error' };
		case ACCOUNT_STATUS.BLOCKED:
			return { label: 'Reactivate', action: 'reactivate', class: 'btn-warning' };
		default:
			return { label: u.account_status, action: null, class: '' };
	}
}

export function userTargetStatusForAction(
	action: 'approve' | 'block' | 'reactivate'
): AccountStatus {
	switch (action) {
		case 'block':
			return ACCOUNT_STATUS.BLOCKED;
		default:
			return ACCOUNT_STATUS.ACTIVE;
	}
}

export interface AuthUser {
	id: string;
	email: string;
	is_admin: boolean;
	account_status: AccountStatus;
	trial_started_at?: string | null;
	created_at: string; // ISO8601 UTC
	updated_at: string; // ISO8601 UTC
}

export interface ApiToken {
	id: string;
	name: string;
	description?: string;
	last_four: string;
	created_at: string;
	expires_at?: string;
	last_used_at?: string;
	is_active: boolean;
	topic_ids?: string[];
}

export interface CreatedApiToken {
	token: string;
	name: string;
}

export type PairingStatus = 'pending' | 'paired' | 'unpaired' | 'subscription_gone';

const PAIRING_STATUS = {
	PENDING: 'pending',
	PAIRED: 'paired',
	UNPAIRED: 'unpaired',
	SUBSCRIPTION_GONE: 'subscription_gone'
} as const;

export const deviceIsPaired = (d: Device): boolean => d.pairing_status === PAIRING_STATUS.PAIRED;

export function deviceStatusLabel(d: Device): string {
	switch (d.pairing_status) {
		case PAIRING_STATUS.PENDING:
			return 'Pairing pending';
		case PAIRING_STATUS.PAIRED:
			return 'Paired';
		case PAIRING_STATUS.SUBSCRIPTION_GONE:
			return 'Subscription lost';
		case PAIRING_STATUS.UNPAIRED:
			return 'Unpaired';
		default:
			return 'Not paired';
	}
}

export function deviceStatusBadgeClass(d: Device): string {
	switch (d.pairing_status) {
		case PAIRING_STATUS.PAIRED:
			return 'badge-success';
		case PAIRING_STATUS.SUBSCRIPTION_GONE:
			return 'badge-error';
		default:
			return 'badge-warning';
	}
}

export interface Device {
	id: string;
	name: string;
	description?: string;
	created_at: string;
	is_active: boolean;
	pairing_status: PairingStatus;
	paired_at?: string;
	age_recipient?: string | null;
	age_recipient_fingerprint?: string | null;
	topic_ids?: string[];
}

export interface CreatedDevice {
	device: Device;
	pairing_code: string;
	pairing_url: string;
	qr_code: string;
	expires_at: string;
}

export interface PairingCodeResponse {
	pairing_code: string;
	pairing_url: string;
	qr_code: string;
	expires_at: string;
}

export interface Webhook {
	id: string;
	name: string;
	description?: string;
	payload_type: 'beebuzz' | 'custom';
	title_path?: string;
	body_path?: string;
	priority: 'normal' | 'high';
	created_at: string;
	last_used_at?: string;
	is_active: boolean;
	topic_ids?: string[];
}

export interface CreatedWebhook {
	id: string;
	token: string;
	name: string;
}

export interface InspectSession {
	token: string;
	url: string;
	status: 'waiting' | 'captured';
	expires_at: string;
}

export interface InspectSessionStatus {
	status: 'waiting' | 'captured';
	payload?: Record<string, unknown>;
	captured_at?: string;
	expires_at: string;
}

/**
 * Account management API namespace.
 */
export const accountApi = {
	/**
	 * Get current authenticated user.
	 * @throws Error if not authenticated
	 */
	me: () => api.get<AuthUser>('/me'),

	/** Fetch account usage stats for the selected range (`0` = all time, `1` = today). */
	getUsage: (days: number = 30) => api.get<AccountUsage>(`/me/usage?days=${days}`),

	// API Tokens
	listApiTokens: async () => {
		const data = await api.get<{ data: ApiToken[] }>('/tokens');
		return data.data || [];
	},

	createApiToken: (name: string, topics: string[], description: string = '') =>
		api.post<CreatedApiToken>('/tokens', { name, description, topics }),

	updateApiToken: (id: string, name: string, description: string, topics: string[]) =>
		api.patch<void>(`/tokens/${id}`, { name, description, topics }),

	deleteApiToken: (id: string) => api.delete<void>(`/tokens/${id}`),

	testApiToken: (id: string) => api.post<{ message: string }>('/tokens/test', { id }),

	// Devices
	listDevices: async () => {
		const data = await api.get<{ data: Device[] }>('/devices');
		return data.data || [];
	},

	createDevice: (name: string, description: string = '', topics: string[]) =>
		api.post<CreatedDevice>('/devices', { name, description, topics }),

	updateDevice: (id: string, name: string, description: string, topics: string[]) =>
		api.patch<void>(`/devices/${id}`, {
			name,
			description,
			topics
		}),

	deleteDevice: (id: string) => api.delete<void>(`/devices/${id}`),

	unpairDevice: (id: string) => api.post<void>(`/devices/${id}/unpair`, {}),

	regeneratePairingCode: (id: string) =>
		api.post<PairingCodeResponse>(`/devices/${id}/pairing-code`, {}),

	// Webhooks
	listWebhooks: async () => {
		const data = await api.get<{ data: Webhook[] }>('/webhooks');
		return data.data || [];
	},

	createWebhook: (
		name: string,
		description: string = '',
		payloadType: 'beebuzz' | 'custom' = 'beebuzz',
		titlePath: string = '',
		bodyPath: string = '',
		priority: 'normal' | 'high' = 'normal',
		topics: string[]
	) => {
		const payload: Record<string, unknown> = {
			name,
			description,
			payload_type: payloadType,
			topics,
			priority
		};
		if (payloadType === 'custom') {
			payload.title_path = titlePath;
			payload.body_path = bodyPath;
		}
		return api.post<CreatedWebhook>('/webhooks', payload);
	},

	updateWebhook: (
		id: string,
		name: string,
		description: string,
		payloadType: 'beebuzz' | 'custom',
		titlePath: string,
		bodyPath: string,
		priority: 'normal' | 'high',
		topics: string[]
	) => {
		const payload: Record<string, unknown> = {
			name,
			description,
			payload_type: payloadType,
			topics,
			priority
		};
		if (payloadType === 'custom') {
			payload.title_path = titlePath;
			payload.body_path = bodyPath;
		}
		return api.patch<void>(`/webhooks/${id}`, payload);
	},

	deleteWebhook: (id: string) => api.delete<void>(`/webhooks/${id}`),

	/**
	 * Regenerates the token for a webhook. Returns the new raw token (one-time reveal).
	 */
	regenerateWebhookToken: (id: string) => api.post<{ token: string }>(`/webhooks/${id}/token`, {}),

	/**
	 * Returns the public URL external services use to POST to a webhook.
	 * Uses the dedicated hook subdomain (e.g. hook.beebuzz.app/{token}).
	 */
	getWebhookReceiveUrl: (token: string): string => `${WEBHOOK_URL}/${token}`,

	// Webhook Inspect
	createInspectSession: (
		name: string,
		description: string = '',
		priority: 'normal' | 'high' = 'normal',
		topics: string[]
	) => api.post<InspectSession>('/webhooks/inspect', { name, description, priority, topics }),

	getInspectSession: () => api.get<InspectSessionStatus>('/webhooks/inspect'),

	finalizeInspect: (titlePath: string, bodyPath: string) =>
		api.post<CreatedWebhook>('/webhooks/inspect/finalize', {
			title_path: titlePath,
			body_path: bodyPath
		})
};
