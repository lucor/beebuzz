import { error } from '@sveltejs/kit';
import { isSaasMode } from '$lib/config/deployment';

export const prerender = isSaasMode;
export const ssr = true;

export function load({ data }: { data?: { metadata?: Record<string, unknown> } }) {
	if (!isSaasMode) {
		error(404, 'Not found');
	}

	return {
		metadata: data?.metadata || {}
	};
}
