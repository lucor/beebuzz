import { test, expect } from '@playwright/test';

test.describe('login page', () => {
	test('should display login form', async ({ page }) => {
		await page.goto('/login');

		await expect(page.getByText(/private beta/i)).toBeVisible();
		await expect(page.getByLabel('Email address')).toBeVisible();
		await expect(page.getByRole('button', { name: /continue/i })).toBeVisible();
	});

	test('should have disabled button when email is empty', async ({ page }) => {
		await page.goto('/login');

		const button = page.getByRole('button', { name: /continue/i });
		await expect(button).toBeDisabled();
	});

	test('should enable button when email is filled', async ({ page }) => {
		await page.goto('/login');

		await page.getByLabel('Email address').fill('test@example.com');
		await expect(page.getByRole('button', { name: /continue/i })).toBeEnabled();
	});
});
