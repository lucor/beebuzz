// Derives all application URLs from the single BEEBUZZ_DOMAIN env var.
// BEEBUZZ_DOMAIN is provided to Vite's define block as VITE_BEEBUZZ_DOMAIN.
const DOMAIN = import.meta.env.VITE_BEEBUZZ_DOMAIN;

if (!DOMAIN) {
	throw new Error(
		'VITE_BEEBUZZ_DOMAIN is not set. Define BEEBUZZ_DOMAIN in your environment; Vite exposes it to the client as VITE_BEEBUZZ_DOMAIN.'
	);
}

export const PUBLIC_SITE_URL = `https://${DOMAIN}`;
export const DASHBOARD_URL = `https://dashboard.${DOMAIN}`;
export const API_URL = `https://api.${DOMAIN}`;
export const PUSH_URL = `https://push.${DOMAIN}`;
export const HIVE_URL = `https://hive.${DOMAIN}`;
export const WEBHOOK_URL = `https://hook.${DOMAIN}`;
