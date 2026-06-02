import { test, expect } from '@playwright/test';

const ADMIN_EMAIL = 'admin@beebuzz.local';
const MAILPIT_API = 'http://localhost:8025/api/v1';

test.describe('auth flow', () => {
	test('full login flow with OTP', async ({ page }) => {
		await page.goto('/login');

		await page.getByLabel('Email address').fill(ADMIN_EMAIL);
		await page.getByRole('button', { name: /continue/i }).click();

		await expect(page).toHaveURL('/verify');

		const messages = await fetch(`${MAILPIT_API}/messages`).then((r) => r.json());
		const latestMsg = messages.messages[0];

		const msg = await fetch(`${MAILPIT_API}/message/${latestMsg.ID}`).then((r) => r.json());
		const otp = msg.Text.match(/\d{6}/)?.[0];
		expect(otp).toHaveLength(6);

		await page.getByLabel('One-time code').fill(otp!);
		await page.getByRole('button', { name: /verify code/i }).click();

		await expect(page).toHaveURL('/account/overview', { timeout: 15000 });

		await expect(page.getByRole('heading', { name: /overview/i })).toBeVisible();
	});
});
