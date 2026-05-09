// Authentication service layer. Owns business logic and logging for auth flows.
import { authApi, type AuthUser } from '../api';
import { auth } from '../stores';
import { logger } from '../logger';
import { ApiError } from '../errors';
import { STORAGE_KEY_STATE, STORAGE_KEY_EMAIL } from '../constants/auth';
import { me } from '../services/account';

/** Clears auth-related browser storage entries used by auth flows. */
const clearAuthStorage = () => {
	localStorage.removeItem(STORAGE_KEY_STATE);
	sessionStorage.removeItem(STORAGE_KEY_EMAIL);
};

/** Requests authentication for the given email. Stores state and email in localStorage. */
export const login = async (
	email: string,
	reason?: string,
	referralCode?: string
): Promise<void> => {
	const state = crypto.randomUUID();
	localStorage.setItem(STORAGE_KEY_STATE, state);

	try {
		await authApi.login(email, state, reason, referralCode);
		sessionStorage.setItem(STORAGE_KEY_EMAIL, email);
		logger.debug('auth request sent');
	} catch (error: unknown) {
		if (error instanceof ApiError) {
			logger.error('login failed', { code: error.code, message: error.message });
		}
		throw error;
	}
};

/** Verifies an OTP code, fetches the user, sets auth state, and clears storage. */
export const verifyOtp = async (otp: string, state: string): Promise<AuthUser> => {
	try {
		await authApi.verifyOtp(otp, state);
		const user = await me();
		clearAuthStorage();
		logger.info('OTP verified', { user_id: user.id });
		return user;
	} catch (error: unknown) {
		if (error instanceof ApiError) {
			logger.error('OTP verification failed', { code: error.code, message: error.message });
		}
		throw error;
	}
};

/** Logs out the current user, clears all auth state. */
export const logout = async (): Promise<void> => {
	try {
		await authApi.logout();
		auth.clear();
		clearAuthStorage();
		logger.info('user logged out');
	} catch (error: unknown) {
		if (error instanceof ApiError) {
			logger.error('logout failed', { code: error.code, message: error.message });
		}
		throw error;
	}
};
