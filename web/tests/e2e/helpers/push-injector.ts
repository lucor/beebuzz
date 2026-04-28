import type { Page } from '@playwright/test';

export interface PushInjector {
	/** Delivers a push payload to the Hive service worker via CDP. */
	deliver(data: string): Promise<void>;
}

/**
 * Creates a push injector that can deliver push messages directly to a Hive
 * service worker using Chrome DevTools Protocol.
 *
 * This bypasses the real push transport (FCM / VAPID) and is intended for
 * local development and automated end-to-end tests only.
 */
export async function createPushInjector(page: Page, origin: string): Promise<PushInjector> {
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

	const registrations = new Map<string, { scopeURL: string }>();

	cdp.on('ServiceWorker.workerRegistrationUpdated', (params: unknown) => {
		const p = params as { registrations: Array<{ registrationId: string; scopeURL: string }> };
		for (const reg of p.registrations) {
			registrations.set(reg.registrationId, { scopeURL: reg.scopeURL });
		}
	});

	await page.waitForTimeout(500);

	let registrationId: string | undefined;
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
		async deliver(data: string) {
			await cdp.send('ServiceWorker.deliverPushMessage', {
				origin,
				registrationId,
				data
			});
		}
	};
}
