import { createHmac } from 'node:crypto';
import { test, expect, type Page } from '@playwright/test';

const BILLING_E2E_ENABLED = process.env.BEEBUZZ_BILLING_E2E === '1';
const MAILPIT_API = process.env.MAILPIT_API || 'http://localhost:8025/api/v1';
const CREEM_API_BASE_URL =
	process.env.BEEBUZZ_BILLING_CREEM_API_BASE_URL || 'https://test-api.creem.io';
const CREEM_E2E_API_KEY = process.env.BEEBUZZ_BILLING_E2E_CREEM_API_KEY;
const CREEM_WEBHOOK_SECRET = process.env.BEEBUZZ_BILLING_CREEM_WEBHOOK_SECRET;
const CREEM_PRODUCT_ID = process.env.BEEBUZZ_BILLING_CREEM_PRODUCT_ID || 'prod_e2e';
const BILLING_TEST_EMAIL = `billing-e2e-${Date.now()}@beebuzz.local`;
const CARD_NUMBER = '4242424242424242';
const CARD_EXPIRY = '12/30';
const CARD_CVC = '123';

type MailpitMessage = {
	ID?: string;
	To?: Array<{ Address?: string }>;
	Text?: string;
};

async function readOTP(email: string): Promise<string> {
	for (let attempt = 0; attempt < 30; attempt++) {
		const response = await fetch(`${MAILPIT_API}/messages`);
		if (response.ok) {
			const payload = (await response.json()) as { messages?: MailpitMessage[] };
			const message = payload.messages?.find((candidate) =>
				candidate.To?.some((recipient) => recipient.Address === email)
			);
			if (message?.ID) {
				const detailResponse = await fetch(`${MAILPIT_API}/message/${message.ID}`);
				if (detailResponse.ok) {
					const detail = (await detailResponse.json()) as MailpitMessage;
					const otp = detail.Text?.match(/\b\d{6}\b/)?.[0];
					if (otp) return otp;
				}
			}
		}

		await new Promise((resolve) => setTimeout(resolve, 500));
	}

	throw new Error(`OTP for ${email} not found in Mailpit`);
}

async function login(page: Page, email: string): Promise<void> {
	await page.goto('/auth');
	await page.getByLabel('Email address').fill(email);
	await page.getByRole('button', { name: /continue/i }).click();
	await expect(page).toHaveURL('/auth/verify');

	await page.getByLabel('One-time code').fill(await readOTP(email));
	await page.getByRole('button', { name: /verify code/i }).click();
	await expect(page).toHaveURL('/account/overview', { timeout: 15_000 });
}

async function callSubscriptionAction(
	subscriptionID: string,
	action: 'cancel' | 'pause' | 'resume',
	body?: object
) {
	const response = await fetch(
		`${CREEM_API_BASE_URL}/v1/subscriptions/${subscriptionID}/${action}`,
		{
			method: 'POST',
			headers: {
				'x-api-key': CREEM_E2E_API_KEY!,
				'content-type': 'application/json'
			},
			body: JSON.stringify(body ?? {})
		}
	);
	if (!response.ok) {
		throw new Error(`Creem subscription ${action} failed with HTTP ${response.status}`);
	}
}

async function cancelSubscriptionAtPeriodEnd(subscriptionID: string): Promise<void> {
	await callSubscriptionAction(subscriptionID, 'cancel', {
		mode: 'scheduled',
		onExecute: 'cancel'
	});
}

async function waitForBillingText(page: Page, text: RegExp): Promise<void> {
	await expect
		.poll(
			async () => {
				await page.goto('/account/billing', { waitUntil: 'networkidle' });
				try {
					await expect(page.getByRole('heading', { name: /plan & billing/i })).toBeVisible({
						timeout: 10_000
					});
					const matches = page.getByText(text);
					return (await matches.count()) > 0 && (await matches.first().isVisible());
				} catch {
					return false;
				}
			},
			{ timeout: 60_000, intervals: [2_000, 3_000, 5_000] }
		)
		.toBe(true);
}

async function sendSubscriptionWebhook(
	page: Page,
	subscriptionID: string,
	userID: string,
	customerID: string,
	eventType:
		| 'subscription.past_due'
		| 'subscription.expired'
		| 'subscription.canceled'
		| 'subscription.active'
): Promise<void> {
	if (!CREEM_WEBHOOK_SECRET) {
		throw new Error('BEEBUZZ_BILLING_CREEM_WEBHOOK_SECRET is required for webhook E2E tests');
	}

	const payload = JSON.stringify({
		id: `evt_billing_e2e_${Date.now()}_${eventType.replace('.', '_')}`,
		eventType,
		created_at: Date.now(),
		object: {
			id: subscriptionID,
			product: { id: CREEM_PRODUCT_ID },
			customer: { id: customerID },
			metadata: { beebuzz_user_id: userID },
			status: eventType.replace('subscription.', ''),
			current_period_end_date: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString()
		}
	});
	const signature = createHmac('sha256', CREEM_WEBHOOK_SECRET).update(payload).digest('hex');
	const dashboardURL = new URL(page.url());
	dashboardURL.hostname = dashboardURL.hostname.replace(/^dashboard\./, 'api.');
	const response = await page.request.post(`${dashboardURL.origin}/v1/billing/webhooks/creem`, {
		data: payload,
		headers: {
			'content-type': 'application/json',
			'creem-signature': signature
		}
	});
	expect(response.status()).toBe(204);
}

async function readCustomerID(subscriptionID: string): Promise<string> {
	const response = await fetch(
		`${CREEM_API_BASE_URL}/v1/subscriptions?subscription_id=${encodeURIComponent(subscriptionID)}`,
		{ headers: { 'x-api-key': CREEM_E2E_API_KEY! } }
	);
	expect(response.ok).toBeTruthy();
	const subscription = (await response.json()) as {
		customer?: string | { id?: string };
	};
	const customerID =
		typeof subscription.customer === 'string' ? subscription.customer : subscription.customer?.id;
	expect(customerID).toBeTruthy();
	return customerID!;
}

async function readUserID(page: Page): Promise<string> {
	const dashboardURL = new URL(page.url());
	dashboardURL.hostname = dashboardURL.hostname.replace(/^dashboard\./, 'api.');
	const response = await page.request.get(`${dashboardURL.origin}/v1/me`);
	expect(response.ok()).toBe(true);
	const user = (await response.json()) as { id?: string };
	expect(user.id).toBeTruthy();
	return user.id!;
}

async function readSubscriptionID(checkoutID: string): Promise<string> {
	const response = await fetch(
		`${CREEM_API_BASE_URL}/v1/checkouts?checkout_id=${encodeURIComponent(checkoutID)}`,
		{
			headers: { 'x-api-key': CREEM_E2E_API_KEY! }
		}
	);
	if (!response.ok) {
		throw new Error(`Creem checkout lookup failed with HTTP ${response.status}`);
	}

	const checkout = (await response.json()) as {
		subscription?: string | { id?: string };
	};
	const subscriptionID =
		typeof checkout.subscription === 'string' ? checkout.subscription : checkout.subscription?.id;
	if (!subscriptionID) {
		throw new Error('Creem checkout response did not include a subscription ID');
	}
	return subscriptionID;
}

test.describe.configure({ mode: 'serial' });

test.describe('Creem billing checkout', () => {
	test.skip(!BILLING_E2E_ENABLED, 'set BEEBUZZ_BILLING_E2E=1 to run Creem sandbox tests');

	test('upgrades through Creem, confirms Hosted, and opens the billing portal', async ({
		page
	}) => {
		test.setTimeout(240_000);
		await login(page, BILLING_TEST_EMAIL);

		await page.goto('/account/billing');
		await expect(page.getByRole('heading', { name: /plan & billing/i })).toBeVisible();
		await expect(page.getByText('Free', { exact: true })).toBeVisible();
		await page.getByRole('button', { name: /upgrade to hosted/i }).click();
		await page.waitForURL(/creem\.io\/test\/checkout/, { timeout: 20_000 });
		const checkoutID = new URL(page.url()).pathname.split('/').pop();
		expect(checkoutID).toMatch(/^ch_/);

		await page.getByPlaceholder('johndoe@example.com').fill(BILLING_TEST_EMAIL);
		await page.getByPlaceholder('John Doe').fill('BeeBuzz Billing E2E');
		await page.locator('select').selectOption('IT');
		await page.getByRole('button', { name: /continue to payment/i }).click();

		await expect
			.poll(() => page.frames().some((frame) => frame.url().includes('sdk-web-card')), {
				timeout: 15_000
			})
			.toBe(true);
		const cardFrame = page.frames().find((frame) => frame.url().includes('sdk-web-card'));
		expect(cardFrame).toBeDefined();
		await cardFrame!.locator('[autocomplete="cc-number"]').fill(CARD_NUMBER);
		await cardFrame!.locator('[autocomplete="cc-exp"]').fill(CARD_EXPIRY);
		await cardFrame!.locator('[autocomplete="cc-csc"]').fill(CARD_CVC);
		await page.getByRole('textbox', { name: /cardholder name/i }).fill('BeeBuzz Billing E2E');
		await page.getByRole('button', { name: /pay €29/i }).click();

		await page.waitForURL(/\/account\/billing\?checkout=success/, { timeout: 45_000 });
		await expect(page.getByText('Hosted is active')).toBeVisible({ timeout: 45_000 });
		await expect(page.getByText('Hosted', { exact: true })).toBeVisible();

		// Subscription read/cancel requires a dedicated Creem API key with subscription scopes.
		if (CREEM_E2E_API_KEY) {
			const subscriptionID = await readSubscriptionID(checkoutID!);
			if (CREEM_WEBHOOK_SECRET) {
				const userID = await readUserID(page);
				const customerID = await readCustomerID(subscriptionID);
				await sendSubscriptionWebhook(
					page,
					subscriptionID,
					userID,
					customerID,
					'subscription.past_due'
				);
				await waitForBillingText(page, /payment issue/i);
				await waitForBillingText(page, /grace period/i);

				await sendSubscriptionWebhook(
					page,
					subscriptionID,
					userID,
					customerID,
					'subscription.expired'
				);
				await waitForBillingText(page, /grace period/i);

				await sendSubscriptionWebhook(
					page,
					subscriptionID,
					userID,
					customerID,
					'subscription.canceled'
				);
				await waitForBillingText(page, /your hosted plan has ended/i);

				await sendSubscriptionWebhook(
					page,
					subscriptionID,
					userID,
					customerID,
					'subscription.active'
				);
				await waitForBillingText(page, /100,000 messages\/month fair use/i);
			}
			await callSubscriptionAction(subscriptionID, 'pause');
			await waitForBillingText(page, /your hosted plan has ended/i);

			await callSubscriptionAction(subscriptionID, 'resume');
			await waitForBillingText(page, /100,000 messages\/month fair use/i);

			await cancelSubscriptionAtPeriodEnd(subscriptionID);
			await waitForBillingText(page, /until the current billing period ends/i);
			await expect(page.getByText(/active until/i)).toBeVisible();
		}
		await page.getByRole('button', { name: /manage billing/i }).click();
		await page.waitForURL(/creem\.io\/test/, { timeout: 20_000 });
	});
});
