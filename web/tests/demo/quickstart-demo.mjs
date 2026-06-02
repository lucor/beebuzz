/* global console, crypto, document, Event, fetch, navigator, process, requestAnimationFrame, URL, window */
import { chromium, expect } from '@playwright/test';
import { mkdir, rename, rm } from 'node:fs/promises';
import path from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const demoEmail = process.env.DEMO_EMAIL || 'demo-quickstart@beebuzz.local';
const domain = mustEnv('BEEBUZZ_DOMAIN');
const siteURL = `https://${domain}`;
const hiveURL = `https://hive.${domain}`;
const apiURL = `https://api.${domain}`;
const mailpitAPI = process.env.MAILPIT_API || 'http://localhost:8025/api/v1';
const outputDir = process.env.DEMO_OUTPUT_DIR || 'docs/assets/readme';
const userDataRoot = '/tmp/beebuzz-quickstart-demo/browser';
const actionDelayMs = Number(process.env.DEMO_ACTION_DELAY_MS || 700);
const typingDelayMs = Number(process.env.DEMO_TYPING_DELAY_MS || 55);
const siteVideoDir = path.join(outputDir, 'site-video');
const hiveVideoDir = path.join(outputDir, 'hive-video');
const siteVideoOutput = path.join(siteVideoDir, 'quickstart-demo-site.webm');
const hiveVideoOutput = path.join(hiveVideoDir, 'quickstart-demo-hive.webm');

function mustEnv(name) {
	const value = process.env[name];
	if (!value) {
		throw new Error(`${name} is required`);
	}
	return value;
}

async function waitForHTTP(url, label) {
	for (let attempt = 0; attempt < 90; attempt += 1) {
		try {
			const response = await fetch(url);
			if (response.ok || response.status < 500) return;
		} catch {
			// keep polling
		}
		await delay(1000);
	}
	throw new Error(`${label} did not become ready: ${url}`);
}

async function latestOTP() {
	for (let attempt = 0; attempt < 60; attempt += 1) {
		const messagesResponse = await fetch(`${mailpitAPI}/messages`);
		if (!messagesResponse.ok) {
			await delay(500);
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
			const searchable = `${detail.To?.map?.((to) => to.Address).join(' ') ?? ''}\n${detail.Text ?? ''}`;
			if (!searchable.includes(demoEmail)) continue;

			const otp = String(detail.Text ?? '').match(/\b\d{6}\b/)?.[0];
			if (otp) return otp;
		}

		await delay(500);
	}

	throw new Error(`OTP for ${demoEmail} not found in Mailpit`);
}

async function launchWindow(name, x, userDataDir) {
	await rm(userDataDir, { force: true, recursive: true });
	await rm(path.join(outputDir, `${name}-video`), { force: true, recursive: true });

	return chromium.launchPersistentContext(userDataDir, {
		headless: false,
		slowMo: 120,
		acceptDownloads: false,
		ignoreHTTPSErrors: true,
		recordVideo: {
			dir: path.join(outputDir, `${name}-video`),
			size: { width: 820, height: 920 }
		},
		viewport: { width: 820, height: 920 },
		args: [
			`--window-position=${x},80`,
			'--window-size=820,980',
			'--touch-events=enabled',
			'--disable-features=Translate,AutomationControlled',
			'--no-first-run'
		]
	});
}

async function persistVideo(video, destination) {
	if (!video) {
		throw new Error(`Missing video handle for ${destination}`);
	}

	const source = await video.path();
	await mkdir(path.dirname(destination), { recursive: true });
	await rm(destination, { force: true });
	await rename(source, destination);
}

async function pause(multiplier = 1) {
	await delay(actionDelayMs * multiplier);
}

function formatError(err) {
	return err instanceof Error ? err.message : String(err);
}

async function showClick(locator) {
	await locator.scrollIntoViewIfNeeded();
	await locator.evaluate((element) => {
		const rect = element.getBoundingClientRect();
		const x = rect.left + rect.width / 2;
		const y = rect.top + rect.height / 2;
		const ring = document.createElement('div');
		ring.setAttribute('aria-hidden', 'true');
		ring.style.position = 'fixed';
		ring.style.left = `${x}px`;
		ring.style.top = `${y}px`;
		ring.style.width = '28px';
		ring.style.height = '28px';
		ring.style.marginLeft = '-14px';
		ring.style.marginTop = '-14px';
		ring.style.border = '3px solid #f59e0b';
		ring.style.borderRadius = '9999px';
		ring.style.background = 'rgba(245, 158, 11, 0.22)';
		ring.style.boxShadow = '0 0 0 8px rgba(245, 158, 11, 0.14)';
		ring.style.zIndex = '2147483647';
		ring.style.pointerEvents = 'none';
		ring.style.transform = 'scale(0.7)';
		ring.style.transition = 'transform 260ms ease-out, opacity 420ms ease-out';
		document.body.appendChild(ring);
		requestAnimationFrame(() => {
			ring.style.transform = 'scale(1.9)';
			ring.style.opacity = '0';
		});
		window.setTimeout(() => ring.remove(), 520);
	});
	await delay(120);
}

async function humanFill(locator, value) {
	await showClick(locator);
	await locator.click();
	await locator.press(process.platform === 'darwin' ? 'Meta+A' : 'Control+A');
	await locator.pressSequentially(value, { delay: typingDelayMs });
	await pause(0.5);
}

async function humanClick(locator) {
	await pause(0.5);
	await showClick(locator);
	await locator.click();
	await pause();
}

async function readPairingCode(pairingDialog) {
	// Two separate persistent contexts don't share a clipboard reliably,
	// so read the code straight from the visible dialog. Target the
	// element that contains exactly six digits to avoid matching any
	// other numeric content that might appear in the dialog.
	const codeNode = pairingDialog.getByText(/^\s*\d{6}\s*$/).first();
	await expect(codeNode).toBeVisible();

	const pairingCode = (await codeNode.textContent())?.replace(/\D/g, '');
	if (pairingCode?.length === 6) return pairingCode;

	throw new Error('Pairing code was not visible in the dialog');
}

async function pastePairingCode(pairingCode, hivePage) {
	const pairingInput = hivePage.getByLabel('Pairing code');
	const connectButton = hivePage.getByRole('button', { name: /connect device/i });

	await showClick(pairingInput);
	await pause(0.5);

	// The hive pairing input is visually hidden (opacity-0) and listens
	// for the `input` event to update its bound state. Synthetic Meta+V
	// across separate Chromium contexts is unreliable, so simulate the
	// paste directly: set the value and dispatch the input event.
	await pairingInput.evaluate((input, value) => {
		const target = /** @type {HTMLInputElement} */ (input);
		target.focus();
		target.value = value;
		target.dispatchEvent(new Event('input', { bubbles: true }));
	}, pairingCode);

	// Verify Svelte state updated (button becomes enabled when length === 6).
	// Fall back to per-character typing if the synthetic event didn't take.
	try {
		await expect(connectButton).toBeEnabled({ timeout: 3000 });
	} catch {
		await humanFill(pairingInput, pairingCode);
		await expect(connectButton).toBeEnabled({ timeout: 3000 });
	}
}

/**
 * Creates a push injector that delivers push messages directly to a Hive
 * service worker via Chrome DevTools Protocol, bypassing the real push
 * transport (FCM / VAPID).
 */
async function createPushInjector(page, origin) {
	const context = page.context();
	const browser = context.browser();
	if (!browser) {
		throw new Error('Page must be attached to a browser');
	}

	let cdp;
	try {
		cdp = await context.newCDPSession(page);
		await cdp.send('ServiceWorker.enable');
	} catch {
		cdp = await browser.newBrowserCDPSession();
		await cdp.send('ServiceWorker.enable');
	}

	/** @type {Map<string, { scopeURL: string }>} */
	const registrations = new Map();

	cdp.on('ServiceWorker.workerRegistrationUpdated', (params) => {
		for (const reg of params.registrations) {
			registrations.set(reg.registrationId, { scopeURL: reg.scopeURL });
		}
	});

	await page.waitForTimeout(500);

	let registrationId;
	for (const [id, reg] of registrations) {
		if (reg.scopeURL === origin || reg.scopeURL === origin + '/') {
			registrationId = id;
			break;
		}
	}

	if (!registrationId) {
		// The service worker may not have reported yet; nudge it.
		await page.evaluate(() => navigator.serviceWorker.ready);
		await page.waitForTimeout(500);

		for (const [id, reg] of registrations) {
			if (reg.scopeURL === origin || reg.scopeURL === origin + '/') {
				registrationId = id;
				break;
			}
		}
	}

	if (!registrationId) {
		const known = [...registrations.values()].map((r) => r.scopeURL).join(', ');
		throw new Error(
			`No service worker registration found for ${origin}. Known scopes: ${known || '(none)'}`
		);
	}

	return {
		/** @param {string} data */
		async deliver(data) {
			await cdp.send('ServiceWorker.deliverPushMessage', {
				origin,
				registrationId,
				data
			});
		}
	};
}

function logStep(message) {
	console.log(`[quickstart-demo] ${message}`);
}

async function useHiveBrowserFallback(page) {
	const pairingInput = page.getByLabel('Pairing code');
	const fallbackButtons = [
		page.getByRole('button', { name: /can't install/i }),
		page.getByRole('button', { name: /continue in browser/i }),
		page.getByText(/continue in browser/i)
	];

	for (let attempt = 0; attempt < 60; attempt += 1) {
		if (await pairingInput.isVisible().catch(() => false)) {
			return;
		}

		for (const fallbackButton of fallbackButtons) {
			if (await fallbackButton.isVisible().catch(() => false)) {
				await humanClick(fallbackButton);
				break;
			}
		}

		if (
			await page
				.getByText(/not supported|unable to pair|something went wrong/i)
				.isVisible()
				.catch(() => false)
		) {
			throw new Error('Hive did not reach the browser pairing screen');
		}

		await pause(0.5);
	}

	await expect(pairingInput).toBeVisible({ timeout: 15000 });
}

await waitForHTTP(siteURL, 'site');
await waitForHTTP(hiveURL, 'Hive');
await waitForHTTP(`${apiURL}/health`, 'API');
await waitForHTTP(`${mailpitAPI}/messages`, 'Mailpit');

const siteContext = await launchWindow('site', 40, path.join(userDataRoot, 'site'));
const hiveContext = await launchWindow('hive', 900, path.join(userDataRoot, 'hive'));

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

await siteContext.grantPermissions(['clipboard-read', 'clipboard-write'], { origin: siteURL });
await hiveContext.grantPermissions(['clipboard-read', 'clipboard-write', 'notifications'], {
	origin: hiveURL
});

const sitePage = siteContext.pages()[0] ?? (await siteContext.newPage());
const hivePage = hiveContext.pages()[0] ?? (await hiveContext.newPage());

// Diagnostics: surface API failures from the Hive page so we can debug
// pairing/push issues that only show up in this Playwright environment.
hivePage.on('console', (message) => {
	if (message.type() === 'error' || message.type() === 'warning') {
		console.log(`[hive console:${message.type()}] ${message.text()}`);
	}
});
hivePage.on('response', async (response) => {
	const url = response.url();
	if (!url.includes('/v1/') || response.ok()) return;
	let body = '';
	try {
		body = await response.text();
	} catch {
		// ignore
	}
	console.log(`[hive api error] ${response.status()} ${url} ${body}`.trim());
});
hivePage.on('request', (request) => {
	if (!request.url().includes('/v1/pairing')) return;
	const method = request.method();
	if (method !== 'POST') return;
	const post = request.postData();
	if (!post) return;
	try {
		const parsed = JSON.parse(post);
		const safe = {
			pairing_code: parsed.pairing_code ? '<redacted>' : null,
			endpoint: parsed.endpoint,
			p256dh_len: parsed.p256dh?.length,
			auth_len: parsed.auth?.length,
			age_recipient: parsed.age_recipient
		};
		console.log(`[hive pair request] ${JSON.stringify(safe)}`);
	} catch {
		console.log(`[hive pair request] (unparseable body, ${post.length} bytes)`);
	}
});

try {
	logStep('opening Site and Hive');
	await Promise.all([sitePage.goto(`${siteURL}/login`), hivePage.goto(`${hiveURL}/pair`)]);
	await pause(1.5);

	logStep('switching Hive to browser fallback');
	await useHiveBrowserFallback(hivePage);

	logStep('signing in on Site');
	await humanFill(sitePage.getByLabel('Email address'), demoEmail);
	await humanClick(sitePage.getByRole('button', { name: /continue/i }));

	await expect(sitePage).toHaveURL(/\/verify/);
	const otp = await latestOTP();
	await pause();
	logStep('entering OTP');
	await humanFill(sitePage.getByLabel('One-time code'), otp);
	await humanClick(sitePage.getByRole('button', { name: /verify code/i }));
	await expect(sitePage).toHaveURL(/\/account\/overview/, { timeout: 15000 });

	logStep('creating pairing code');
	await pause(1.2);
	await sitePage.goto(`${siteURL}/account/devices`);
	await pause(1.2);
	await humanClick(sitePage.getByRole('button', { name: /add device/i }));
	await humanFill(sitePage.getByLabel('Device Name'), 'Demo Chrome Hive');
	await humanClick(sitePage.getByRole('button', { name: /generate pairing code/i }));

	const pairingDialog = sitePage.getByRole('dialog', { name: /pairing code generated/i });
	await expect(pairingDialog).toBeVisible();
	await pause();
	await humanClick(pairingDialog.getByTitle('Copy code'));
	const pairingCode = await readPairingCode(pairingDialog);
	console.log(`[quickstart-demo] pairing code = ${pairingCode}`);

	logStep('pasting pairing code in Hive');
	await pastePairingCode(pairingCode, hivePage);
	await humanClick(hivePage.getByRole('button', { name: /connect device/i }));
	await expect(hivePage.getByText(/no notifications yet/i)).toBeVisible({ timeout: 30000 });
	console.log('[quickstart-demo] Hive paired and ready');

	await pause(1.2);
	await humanClick(pairingDialog.getByRole('button', { name: /done/i }));
	await expect(sitePage.getByText(/pairing done/i)).toBeVisible({ timeout: 15000 });
	await humanClick(sitePage.getByRole('button', { name: /go to api tokens/i }));

	logStep('creating API token');
	await expect(sitePage).toHaveURL(/\/account\/api-tokens/);
	await pause(1.2);
	await humanClick(sitePage.getByRole('button', { name: /create token/i }).first());
	await humanFill(sitePage.getByLabel('Token Name'), 'Quickstart demo token');

	// The submit button lives outside the <form> (in the dialog footer) and is
	// linked via form="create-token-form". Scope the click to the dialog.
	const createDialog = sitePage.getByRole('dialog', { name: /create api token/i });
	await expect(createDialog).toBeVisible();
	const submitButton = createDialog.getByRole('button', { name: /^create token$/i });
	await expect(submitButton).toBeEnabled({ timeout: 5000 });
	console.log('[quickstart-demo] submit button is enabled');
	await humanClick(submitButton);

	const tokenDialog = sitePage.getByRole('dialog', { name: /api token created/i });
	await expect(tokenDialog).toBeVisible();
	console.log('[quickstart-demo] API token created');
	await pause(1.5);

	logStep('creating push injector for CDP delivery');
	const injector = await createPushInjector(hivePage, hiveURL);
	console.log('[quickstart-demo] Push injector ready');

	logStep('sending test notification');
	await humanClick(tokenDialog.getByRole('button', { name: /send test notification now/i }));
	await expect(tokenDialog.getByText(/test sent to/i)).toBeVisible({ timeout: 30000 });
	console.log('[quickstart-demo] Test notification sent via API');

	logStep('polling push stub bridge');
	/** @type {{ data: string } | undefined} */
	let pushEvent;
	for (let attempt = 0; attempt < 10; attempt += 1) {
		const stubResponse = await fetch(`${apiURL}/_stub/push/next`);
		console.log(`[quickstart-demo] push stub attempt ${attempt + 1}: ${stubResponse.status}`);
		if (stubResponse.status === 204) {
			await delay(500);
			continue;
		}
		if (!stubResponse.ok) {
			throw new Error(`Push stub bridge returned ${stubResponse.status}`);
		}
		pushEvent = await stubResponse.json();
		console.log(
			`[quickstart-demo] push event captured: device_id=${pushEvent.device_id}, data_len=${pushEvent.data?.length}`
		);
		break;
	}
	if (!pushEvent) {
		throw new Error('No push event captured from stub bridge after multiple attempts');
	}

	logStep('injecting push via CDP');
	await injector.deliver(pushEvent.data);
	console.log('[quickstart-demo] Push delivered via CDP');

	logStep('waiting for notification in Hive');
	await expect(hivePage.getByText('BeeBuzz test notification')).toBeVisible({ timeout: 10000 });
	console.log('[quickstart-demo] Notification visible in Hive');

	await pause(2);
	await sitePage.screenshot({ path: path.join(outputDir, 'quickstart-demo-site-final.png') });
	await hivePage.screenshot({ path: path.join(outputDir, 'quickstart-demo-hive-final.png') });
	await delay(2500);
} catch (err) {
	console.error(`[quickstart-demo] FAILED: ${formatError(err)}`);
	throw err;
} finally {
	console.log('[quickstart-demo] closing browser contexts');
	const siteVideo = sitePage.video();
	const hiveVideo = hivePage.video();
	await siteContext
		.close()
		.catch((e) => console.error(`[quickstart-demo] site close error: ${formatError(e)}`));
	await hiveContext
		.close()
		.catch((e) => console.error(`[quickstart-demo] hive close error: ${formatError(e)}`));
	await persistVideo(siteVideo, siteVideoOutput).catch((e) =>
		console.error(`[quickstart-demo] site video move error: ${formatError(e)}`)
	);
	await persistVideo(hiveVideo, hiveVideoOutput).catch((e) =>
		console.error(`[quickstart-demo] hive video move error: ${formatError(e)}`)
	);
}
