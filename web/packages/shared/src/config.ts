declare global {
	interface Window {
		readonly __BEEBUZZ_CONFIG__?: {
			readonly domain?: string;
		};
	}
}

const RUNTIME_DOMAIN =
	typeof window === 'undefined' ? undefined : window.__BEEBUZZ_CONFIG__?.domain;
const DOMAIN = RUNTIME_DOMAIN;

if (typeof window !== 'undefined' && !DOMAIN) {
	throw new Error(
		'BeeBuzz domain is not configured. Set BEEBUZZ_DOMAIN in the web container environment.'
	);
}

export const PUBLIC_SITE_URL = DOMAIN ? `https://${DOMAIN}` : '';
export const DASHBOARD_URL = DOMAIN ? `https://dashboard.${DOMAIN}` : '';
export const API_URL = DOMAIN ? `https://api.${DOMAIN}` : '';
export const PUSH_URL = DOMAIN ? `https://push.${DOMAIN}` : '';
export const HIVE_URL = DOMAIN ? `https://hive.${DOMAIN}` : '';
export const WEBHOOK_URL = DOMAIN ? `https://hook.${DOMAIN}` : '';
