import { expect, test } from '@playwright/test';

test.describe('public beta docs', () => {
	test('redirects docs root to the quickstart', async ({ page }) => {
		await page.goto('/docs');

		await expect(page).toHaveURL(/\/docs\/quickstart$/);
		await expect(page.getByRole('heading', { name: 'Quickstart' })).toBeVisible();
	});

	test('shows the first-send quickstart path', async ({ page }) => {
		await page.goto('/docs/quickstart');

		await expect(page.getByRole('heading', { name: 'Quickstart' })).toBeVisible();
		await expect(page.getByText('pair one device')).toBeVisible();
		await expect(page.getByText('create one API token')).toBeVisible();
		await expect(page.getByText('beebuzz connect')).toBeVisible();
		await expect(page.getByText('beebuzz send "Hello from BeeBuzz"')).toBeVisible();
		await expect(page.getByRole('link', { name: 'Webhooks' })).toBeVisible();
	});

	test('links from quickstart to webhooks', async ({ page }) => {
		await page.goto('/docs/quickstart');

		await page.getByRole('link', { name: 'Webhooks' }).click();

		await expect(page).toHaveURL(/\/docs\/webhooks$/);
		await expect(page.getByRole('heading', { name: 'Webhooks' })).toBeVisible();
	});

	test('documents webhook and Home Assistant setup', async ({ page }) => {
		await page.goto('/docs/webhooks');

		await expect(page.getByRole('heading', { name: 'Webhooks' })).toBeVisible();
		await expect(page.getByText('BeeBuzz Payload Mode')).toBeVisible();
		await expect(page.getByText('Custom Payload Mode')).toBeVisible();
		await expect(page.getByText('Home Assistant Example')).toBeVisible();
		await expect(page.getByText('rest_command:')).toBeVisible();
		await expect(page.getByText('hook.beebuzz.app/v1/webhooks/YOUR_WEBHOOK_TOKEN')).toBeVisible();
	});

	test('documents browser requirements for Hive', async ({ page }) => {
		await page.goto('/docs/browser-support');

		await expect(page.getByRole('heading', { name: 'Browser Support' })).toBeVisible();
		await expect(page.getByRole('cell', { name: 'Safari (iPhone/iPad)' })).toBeVisible();
		await expect(page.getByRole('cell', { name: 'Required' })).toBeVisible();
		await expect(
			page.getByText('Web Push only works from the Home Screen installed app')
		).toBeVisible();
	});
});
