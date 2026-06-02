import { test, expect } from '@playwright/test';
import { createPushInjector } from './helpers/push-injector';

const domain = process.env.BEEBUZZ_DOMAIN || 'localhost';
const siteURL = `https://${domain}`;
const hiveURL = `https://hive.${domain}`;
const apiURL = `https://api.${domain}`;
const mailpitAPI = process.env.MAILPIT_API || 'http://localhost:8025/api/v1';
const demoEmail = process.env.DEMO_EMAIL || 'test-push-stub@beebuzz.local';

async function latestOTP() {
	for (let attempt = 0; attempt < 60; attempt++) {
		const messagesResponse = await fetch(`${mailpitAPI}/messages`);
		if (!messagesResponse.ok) {
			await new Promise((r) => setTimeout(r, 500));
			continue;
		}

		const messagesBody = await messagesResponse.json();
		const messages = Array.isArray(messagesBody.messages) ? messagesBody.messages : [];

		for (const message of messages) {
			const id = message.ID ?? message.Id ?? message.id;
			if (!id) continue;

			const detailResponse = await fetch(`${mailpitAPI}/message/${id}`);
			if (!detailResponse.ok) continue;

			const detail = await detailResponse.json();
			const toList = Array.isArray(detail.To) ? (detail.To as Array<{ Address: string }>) : [];
			const searchable = `${toList.map((to) => to.Address).join(' ')}\n${detail.Text ?? ''}`;
			if (!searchable.includes(demoEmail)) continue;

			const otp = String(detail.Text ?? '').match(/\b\d{6}\b/)?.[0];
			if (otp) return otp;
		}

		await new Promise((r) => setTimeout(r, 500));
	}

	throw new Error(`OTP for ${demoEmail} not found in Mailpit`);
}

async function waitForHTTP(url: string, label: string) {
	for (let attempt = 0; attempt < 90; attempt++) {
		try {
			const response = await fetch(url);
			if (response.ok || response.status < 500) return;
		} catch {
			// keep polling
		}
		await new Promise((r) => setTimeout(r, 1000));
	}
	throw new Error(`${label} did not become ready: ${url}`);
}

test.describe.configure({ mode: 'serial' });

test.describe('Push Stub E2E', () => {
	test.beforeAll(async () => {
		await waitForHTTP(siteURL, 'site');
		await waitForHTTP(hiveURL, 'Hive');
		await waitForHTTP(`${apiURL}/health`, 'API');
		await waitForHTTP(`${mailpitAPI}/messages`, 'Mailpit');
	});

	test('complete signup, pairing, and receive push via stub bridge', async ({ browser }) => {
		// Hive context
		const hiveContext = await browser.newContext({ ignoreHTTPSErrors: true });

		// Playwright push subscriptions use *.google.com endpoints that the backend
		// rejects as unsupported. Rewrite them to a fake FCM endpoint so pairing
		// succeeds while keeping the real allowlist strict.
		await hiveContext.route('**/v1/pairing', async (route, request) => {
			if (request.method() !== 'POST') {
				await route.continue();
				return;
			}
			const body = JSON.parse(request.postData() || '{}');
			if (body.endpoint) {
				const url = new URL(body.endpoint);
				if (url.hostname.endsWith('.google.com') && url.hostname !== 'fcm.googleapis.com') {
					body.endpoint = `https://fcm.googleapis.com/fcm/send/demo-${crypto.randomUUID()}`;
				}
			}
			await route.continue({ postData: JSON.stringify(body) });
		});

		await hiveContext.grantPermissions(['clipboard-read', 'clipboard-write', 'notifications'], {
			origin: hiveURL
		});
		const hivePage = await hiveContext.newPage();

		// Site context
		const siteContext = await browser.newContext({ ignoreHTTPSErrors: true });
		await siteContext.grantPermissions(['clipboard-read', 'clipboard-write'], { origin: siteURL });
		const sitePage = await siteContext.newPage();

		// Diagnostics
		hivePage.on('console', (message) => {
			if (message.type() === 'error' || message.type() === 'warning') {
				console.log(`[hive console:${message.type()}] ${message.text()}`);
			}
		});

		// Step 1: open Site and Hive
		await Promise.all([sitePage.goto(`${siteURL}/login`), hivePage.goto(`${hiveURL}/pair`)]);

		// Step 2: skip install fallback on Hive
		const fallbackButton = hivePage.getByRole('button', { name: /continue in browser/i });
		if (await fallbackButton.isVisible().catch(() => false)) {
			await fallbackButton.click();
		}

		// Step 3: sign in on Site
		await sitePage.getByLabel('Email address').fill(demoEmail);
		await sitePage.getByRole('button', { name: /continue/i }).click();
		await expect(sitePage).toHaveURL(/\/verify/);

		const otp = await latestOTP();
		await sitePage.getByLabel('One-time code').fill(otp);
		await sitePage.getByRole('button', { name: /verify code/i }).click();
		await expect(sitePage).toHaveURL(/\/account\/overview/, { timeout: 15000 });

		// Step 4: create pairing code
		await sitePage.goto(`${siteURL}/account/devices`);
		await sitePage.getByRole('button', { name: /add device/i }).click();
		await sitePage.getByLabel('Device Name').fill('Push Stub Test Device');
		await sitePage.getByRole('button', { name: /generate pairing code/i }).click();

		const pairingDialog = sitePage.getByRole('dialog', { name: /pairing code generated/i });
		await expect(pairingDialog).toBeVisible();

		const codeNode = pairingDialog.getByText(/^\s*\d{6}\s*$/).first();
		await expect(codeNode).toBeVisible();
		const pairingCode = (await codeNode.textContent())?.replace(/\D/g, '');
		if (!pairingCode || pairingCode.length !== 6) {
			throw new Error('Pairing code was not visible');
		}

		// Step 5: paste pairing code in Hive
		const pairingInput = hivePage.getByLabel('Pairing code');
		await pairingInput.evaluate((input: unknown, value: string) => {
			const target = input as HTMLInputElement;
			target.focus();
			target.value = value;
			target.dispatchEvent(new Event('input', { bubbles: true }));
		}, pairingCode);

		await hivePage.getByRole('button', { name: /connect device/i }).click();
		await expect(hivePage.getByText(/no notifications yet/i)).toBeVisible({ timeout: 30000 });

		await pairingDialog.getByRole('button', { name: /done/i }).click();
		await expect(sitePage.getByText(/pairing done/i)).toBeVisible({ timeout: 15000 });
		await sitePage.getByRole('button', { name: /go to api tokens/i }).click();

		// Step 6: create API token
		await expect(sitePage).toHaveURL(/\/account\/api-tokens/);
		await sitePage
			.getByRole('button', { name: /create token/i })
			.first()
			.click();
		await sitePage.getByLabel('Token Name').fill('Push Stub Test Token');
		await sitePage.getByRole('button', { name: /^create token$/i }).click();

		const tokenDialog = sitePage.getByRole('dialog', { name: /api token created/i });
		await expect(tokenDialog).toBeVisible();

		// Step 7: send test notification via UI
		await sitePage.getByRole('button', { name: /send test notification now/i }).click();
		await expect(tokenDialog.getByText(/test sent to/i)).toBeVisible({ timeout: 30000 });

		// Step 8: poll push stub bridge for the captured payload
		const stubResponse = await fetch(`${apiURL}/_stub/push/next`);
		expect(stubResponse.ok).toBe(true);
		const pushEvent = await stubResponse.json();
		expect(pushEvent.data).toBeDefined();

		// Step 9: inject the push into Hive via CDP
		const injector = await createPushInjector(hivePage, hiveURL);
		await injector.deliver(pushEvent.data);

		// Step 10: assert notification visible in Hive
		await expect(hivePage.getByText('BeeBuzz test notification')).toBeVisible({ timeout: 10000 });

		await siteContext.close();
		await hiveContext.close();
	});
});
