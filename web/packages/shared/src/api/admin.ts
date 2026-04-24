// Admin API endpoints.
import { api } from './client';
import type { AccountStatus } from './account';

export interface AdminUser {
	id: string;
	email: string;
	is_admin: boolean;
	account_status: AccountStatus;
	signup_reason?: string | null;
	trial_started_at?: string | null;
	created_at: string;
	updated_at: string;
}

export interface DailyBreakdown {
	date: string;
	notifications_created: number;
	delivery_attempts: number;
	deliveries_succeeded: number;
	deliveries_failed: number;
	notifications_with_attachment: number;
	attachment_bytes_total: number;
	notifications_server_trusted: number;
	notifications_e2e: number;
	sources_cli: number;
	sources_webhook: number;
	sources_api: number;
	sources_internal: number;
	devices_lost: number;
	updated_at: string;
}

export interface PlatformDashboard {
	notifications_created: number;
	delivery_attempts: number;
	deliveries_succeeded: number;
	deliveries_failed: number;
	delivery_success_rate: number;
	active_users: number;
	notifications_server_trusted: number;
	notifications_e2e: number;
	notifications_with_attachment: number;
	attachment_bytes_total: number;
	sources_cli: number;
	sources_webhook: number;
	sources_api: number;
	sources_internal: number;
	devices_lost: number;
	daily_breakdown: DailyBreakdown[];
}

export interface AdminUsersListResponse {
	data: AdminUser[];
}

export interface SystemNotificationSettings {
	enabled: boolean;
	recipient_user_id?: string;
	topic_id?: string;
	signup_created_enabled: boolean;
	created_at?: string;
	updated_at?: string;
}

export interface UpdateSystemNotificationSettings {
	enabled: boolean;
	topic_id: string;
	signup_created_enabled: boolean;
}

/**
 * Admin API namespace.
 */
export const adminApi = {
	/**
	 * Fetch all users (admin only).
	 */
	listUsers: async () => {
		const data = await api.get<AdminUsersListResponse>('/admin/users');
		return data.data || [];
	},

	updateUserStatus: (userId: string, accountStatus: AccountStatus) =>
		api.patch<{ data: AdminUser }>(`/admin/users/${userId}`, { account_status: accountStatus }),

	/** Fetch platform dashboard data (`0` = all time, `1` = today). */
	getDashboard: (days: number = 30) => api.get<PlatformDashboard>(`/admin/dashboard?days=${days}`),

	getSystemNotificationSettings: () =>
		api.get<SystemNotificationSettings>('/admin/system/notifications'),

	updateSystemNotificationSettings: (settings: UpdateSystemNotificationSettings) =>
		api.patch<SystemNotificationSettings>('/admin/system/notifications', settings)
};
