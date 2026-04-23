// Authentication API endpoints.
import { api } from './client';

export interface OtpVerifyResponse {
	message: string;
}

/**
 * Authentication API namespace.
 */
export const authApi = {
	/**
	 * Login authentication with email.
	 */
	login: (email: string, state: string, reason?: string) => {
		return api.post<void>('/auth/login', { email, state, reason });
	},

	/**
	 * Verify OTP code.
	 */
	verifyOtp: (otp: string, state: string) =>
		api.post<OtpVerifyResponse>('/auth/otp/verify', { otp, state }),

	/**
	 * Logout current user.
	 */
	logout: () => api.post<void>('/auth/logout')
};
