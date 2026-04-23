// Derives all application URLs from the single VITE_BEEBUZZ_DOMAIN env var.
const DOMAIN = import.meta.env.VITE_BEEBUZZ_DOMAIN;

if (!DOMAIN) {
	throw new Error('VITE_BEEBUZZ_DOMAIN is not set. Define it in your .env file.');
}

export const SITE_URL = `https://${DOMAIN}`;
export const API_URL = `https://api.${DOMAIN}`;
export const PUSH_URL = `https://push.${DOMAIN}`;
export const HIVE_URL = `https://hive.${DOMAIN}`;
export const WEBHOOK_URL = `https://hook.${DOMAIN}`;
