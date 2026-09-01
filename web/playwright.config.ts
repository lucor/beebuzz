import { defineConfig, devices } from '@playwright/test';

const domain = process.env.BEEBUZZ_DOMAIN;
const baseURL = process.env.PLAYWRIGHT_BASE_URL || (domain && `https://dashboard.${domain}`);

if (!baseURL) {
	throw new Error('Set PLAYWRIGHT_BASE_URL or BEEBUZZ_DOMAIN before running Playwright.');
}

export default defineConfig({
	testDir: './tests/e2e',
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 2 : 0,
	workers: process.env.CI ? 1 : undefined,
	reporter: 'list',
	use: {
		baseURL,
		trace: 'on-first-retry'
	},
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'] }
		}
	]
});
