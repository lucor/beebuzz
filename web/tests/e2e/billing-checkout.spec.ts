import { createHmac } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { test, expect, type Page } from '@playwright/test';

const BILLING_E2E_ENABLED = process.env.BEEBUZZ_BILLING_E2E === '1';
const MAILPIT_API = process.env.MAILPIT_API || 'http://localhost:8025/api/v1';
const CREEM_API_BASE_URL =
	process.env.BEEBUZZ_BILLING_CREEM_API_BASE_URL || 'https://test-api.creem.io';
const CREEM_E2E_API_KEY = process.env.BEEBUZZ_BILLING_E2E_CREEM_API_KEY;
const CREEM_WEBHOOK_SECRET = process.env.BEEBUZZ_BILLING_CREEM_WEBHOOK_SECRET;
const CREEM_PRODUCT_ID = process.env.BEEBUZZ_BILLING_CREEM_PRODUCT_ID || 'prod_e2e';
const BILLING_E2E_LOG_FILE = process.env.BEEBUZZ_BILLING_E2E_LOG_FILE;
const BILLING_TEST_EMAIL = `billing-e2e-${Date.now()}@beebuzz.local`;
const CARD_NUMBER = '4242424242424242';
const CARD_EXPIRY = '12/30';
const CARD_CVC = '123';
// Yuno's cardholder input accepts letters/spaces only in the sandbox.
const CARDHOLDER_NAME = 'BeeBuzz Test';

type MailpitMessage = {
	ID?: string;
	To?: Array<{ Address?: string }>;
	Subject?: string;
	Text?: string;
};

async function waitForBillingEmail(subject: RegExp): Promise<void> {
	await expect
		.poll(
			async () => {
				const response = await fetch(`${MAILPIT_API}/messages`);
				if (!response.ok) return false;
				const payload = (await response.json()) as { messages?: MailpitMessage[] };
				for (const message of payload.messages ?? []) {
					if (!message.To?.some((recipient) => recipient.Address === BILLING_TEST_EMAIL)) continue;
					if (message.Subject && subject.test(message.Subject)) return true;
					if (!message.ID) continue;
					const detailResponse = await fetch(`${MAILPIT_API}/message/${message.ID}`);
					if (!detailResponse.ok) continue;
					const detail = (await detailResponse.json()) as MailpitMessage;
					if (detail.Subject && subject.test(detail.Subject)) return true;
				}
				return false;
			},
			{ timeout: 30_000, intervals: [2_000, 3_000, 5_000] }
		)
		.toBe(true);
}

async function waitForWebhookLog(eventType: string, subscriptionID: string): Promise<string> {
	if (!BILLING_E2E_LOG_FILE) {
		throw new Error('BEEBUZZ_BILLING_E2E_LOG_FILE is required for webhook log assertions');
	}

	let eventID = '';
	await expect
		.poll(
			async () => {
				try {
					const contents = await readFile(BILLING_E2E_LOG_FILE, 'utf8');
					for (const line of contents.split('\n').reverse()) {
						if (
							!line.includes(eventType) ||
							!line.includes(subscriptionID) ||
							(!line.includes('outcome=applied') && !line.includes('"outcome":"applied"'))
						) {
							continue;
						}
						const match = line.match(/event_id(?:=|":")([^,\s"]+)/);
						if (match?.[1]) {
							eventID = match[1];
							return true;
						}
					}
				} catch {
					return false;
				}
				return false;
			},
			{ timeout: 30_000, intervals: [2_000, 3_000, 5_000] }
		)
		.toBe(true);
	return eventID;
}

async function waitForPageText(page: Page, text: RegExp): Promise<void> {
	await expect
		.poll(
			async () => {
				const matches = page.getByText(text);
				return (await matches.count()) > 0 && (await matches.first().isVisible());
			},
			{ timeout: 30_000 }
		)
		.toBe(true);
}

async function returnFromPortal(page: Page, portalPage: Page): Promise<void> {
	await portalPage.close();
	await page.bringToFront();
	// Closing a popup does not consistently emit focus in headless Chromium.
	// Dispatch the same browser event that the dashboard listens for in production.
	await page.evaluate(() => window.dispatchEvent(new Event('focus')));
}

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

async function waitForBillingProjection(
	page: Page,
	predicate: (user: { plan?: string; subscription_status?: string }) => boolean
): Promise<void> {
	const dashboardURL = new URL(page.url());
	dashboardURL.hostname = dashboardURL.hostname.replace(/^dashboard\./, 'api.');
	await expect
		.poll(
			async () => {
				const response = await page.request.get(`${dashboardURL.origin}/v1/me`);
				if (!response.ok()) return false;
				return predicate(
					(await response.json()) as { plan?: string; subscription_status?: string }
				);
			},
			{ timeout: 10_000 }
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
		| 'subscription.unpaid'
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
	expect(response.status()).toBe(200);
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
		test.setTimeout(180_000);
		await login(page, BILLING_TEST_EMAIL);

		await page.goto('/account/billing');
		await expect(page.getByRole('heading', { name: /plan & billing/i })).toBeVisible();
		await expect(page.getByText('Free', { exact: true })).toBeVisible();
		await page.getByRole('button', { name: /upgrade to hosted/i }).click();
		await page.waitForURL(/creem\.io\/test\/checkout/, { timeout: 15_000 });
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
		const cardNumber = cardFrame!.locator('[autocomplete="cc-number"]');
		const cardExpiry = cardFrame!.locator('[autocomplete="cc-exp"]');
		const cardCVC = cardFrame!.locator('[autocomplete="cc-csc"]');
		await cardNumber.fill(CARD_NUMBER);
		await cardExpiry.fill(CARD_EXPIRY);
		await cardCVC.fill(CARD_CVC);
		await expect(cardCVC).toHaveValue(CARD_CVC);
		const cardholder = page.getByRole('textbox', { name: 'Cardholder Name', exact: true });
		await expect(cardholder).toBeVisible({ timeout: 10_000 });
		await cardholder.click();
		await cardholder.pressSequentially(CARDHOLDER_NAME, { delay: 20 });
		await expect(cardholder).toHaveValue(CARDHOLDER_NAME);
		await page.getByRole('button', { name: /pay €29/i }).click();

		try {
			await page.waitForURL(
				(url) =>
					url.pathname === '/account/billing' && url.searchParams.get('checkout') === 'success',
				{ timeout: 30_000 }
			);
		} catch (error) {
			console.error(`Creem payment did not return to BeeBuzz; current URL: ${page.url()}`);
			throw error;
		}
		await expect(page.getByText('Hosted is active')).toBeVisible({ timeout: 30_000 });
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
				await waitForBillingProjection(
					page,
					(user) => user.plan === 'hosted' && user.subscription_status === 'past_due'
				);

				await sendSubscriptionWebhook(
					page,
					subscriptionID,
					userID,
					customerID,
					'subscription.unpaid'
				);
				await waitForBillingProjection(
					page,
					(user) => user.plan === 'hosted' && user.subscription_status === 'past_due'
				);

				await sendSubscriptionWebhook(
					page,
					subscriptionID,
					userID,
					customerID,
					'subscription.expired'
				);
				await waitForBillingProjection(
					page,
					(user) => user.plan === 'hosted' && user.subscription_status === 'past_due'
				);

				await sendSubscriptionWebhook(
					page,
					subscriptionID,
					userID,
					customerID,
					'subscription.canceled'
				);
				await waitForBillingProjection(
					page,
					(user) => user.plan === 'free' && user.subscription_status === 'canceled'
				);

				await sendSubscriptionWebhook(
					page,
					subscriptionID,
					userID,
					customerID,
					'subscription.active'
				);
				await waitForBillingProjection(
					page,
					(user) => user.plan === 'hosted' && user.subscription_status === 'active'
				);
			}
			// Open the real Creem Customer Portal, change the subscription in the sandbox,
			// then close it. Returning focus to this page must refresh billing state.
			const scheduledPortalPromise = page.waitForEvent('popup');
			await page.getByRole('button', { name: /manage billing/i }).click();
			const scheduledPortal = await scheduledPortalPromise;
			await expect(scheduledPortal).toHaveURL(/creem\.io\/test/, { timeout: 20_000 });
			await cancelSubscriptionAtPeriodEnd(subscriptionID);
			const scheduledEventID = await waitForWebhookLog(
				'subscription.scheduled_cancel',
				subscriptionID
			);
			await returnFromPortal(page, scheduledPortal);
			await waitForPageText(page, /^Active until$/i);
			await expect(
				page.getByText(/Hosted remains active until the end of your billing period/i)
			).toBeVisible();
			await expect(page.getByText('Hosted is active')).toHaveCount(0);
			await waitForBillingEmail(/scheduled to end/i);
			expect(scheduledEventID).toBeTruthy();

			const resumedPortalPromise = page.waitForEvent('popup');
			await page.getByRole('button', { name: /manage billing/i }).click();
			const resumedPortal = await resumedPortalPromise;
			await expect(resumedPortal).toHaveURL(/creem\.io\/test/, { timeout: 20_000 });
			await callSubscriptionAction(subscriptionID, 'resume');
			const resumedEventID = await waitForWebhookLog('subscription.active', subscriptionID);
			await returnFromPortal(page, resumedPortal);
			await waitForPageText(page, /^Renews or expires$/i);
			await waitForBillingEmail(/active again/i);
			expect(resumedEventID).toBeTruthy();
			expect(resumedEventID).not.toBe(scheduledEventID);
		} else {
			const portalPagePromise = page.waitForEvent('popup');
			await page.getByRole('button', { name: /manage billing/i }).click();
			const portalPage = await portalPagePromise;
			await expect(portalPage).toHaveURL(/creem\.io\/test/, { timeout: 20_000 });
		}
	});
});
