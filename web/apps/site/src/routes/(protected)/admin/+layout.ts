import { redirect } from '@sveltejs/kit';
import { accountApi, type AuthUser } from '@beebuzz/shared/api';
import { auth } from '@beebuzz/shared/stores';
import type { LayoutLoad } from './$types';

export const prerender = false;
export const ssr = false;

/** Fetches current user and guards admin routes. Redirects to /login on 401, to /account if not admin. */
export const load: LayoutLoad = async () => {
	let user: AuthUser;

	try {
		user = await accountApi.me();
	} catch (error: unknown) {
		const status = (error as { status?: number }).status;
		if (status === 403) {
			redirect(302, '/account');
		}
		redirect(302, '/login');
	}

	auth.set(user);

	if (!user.is_admin) {
		redirect(302, '/account');
	}

	return { user };
};
