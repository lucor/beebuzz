// Account service layer. Owns business logic and logging for account operations.
import { accountApi, type AuthUser } from '../api';
import { auth } from '../stores';
import { logger } from '../logger';
import { ApiError } from '../errors';

/** Fetches current authenticated user from the backend and sets auth state. */
export const me = async (): Promise<AuthUser> => {
	try {
		const user = await accountApi.me();
		auth.set(user);
		logger.debug('current user fetched', { user_id: user.id });
		return user;
	} catch (error: unknown) {
		if (error instanceof ApiError) {
			logger.error('fetch user failed', { code: error.code, message: error.message });
		}
		throw error;
	}
};
