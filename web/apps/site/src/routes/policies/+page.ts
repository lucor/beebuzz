import { isSaasMode } from '$lib/config/deployment';

export const prerender = isSaasMode;
export const ssr = true;
