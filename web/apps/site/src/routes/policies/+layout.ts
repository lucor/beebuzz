import { error } from '@sveltejs/kit';
import { isSaasMode } from '$lib/config/deployment';

export const prerender = isSaasMode;
export const ssr = true;

/** Hides the hosted policies hub outside SaaS builds. */
export function load() {
	if (!isSaasMode) {
		error(404, 'Not found');
	}
}
