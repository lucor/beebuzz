import { error, redirect } from '@sveltejs/kit';
import { isSaasMode } from '$lib/config/deployment';

export const prerender = isSaasMode;

/** Redirects docs root to the first docs page. */
export function load() {
	if (!isSaasMode) {
		error(404, 'Not found');
	}

	redirect(308, '/docs/quickstart');
}
