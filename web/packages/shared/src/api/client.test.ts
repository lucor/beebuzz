import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../errors';

const loadClient = async () => {
	vi.resetModules();
	vi.stubEnv('VITE_BEEBUZZ_DOMAIN', 'example.test');
	return import('./client');
};

describe('resolveDashboardNavigation', () => {
	beforeEach(() => {
		vi.unstubAllEnvs();
		vi.restoreAllMocks();
		vi.unstubAllGlobals();
	});

	it('uses a full browser navigation when redirecting to dashboard from Hive', async () => {
		const { resolveDashboardNavigation } = await loadClient();

		expect(resolveDashboardNavigation('/auth', 'https://hive.example.test')).toEqual({
			kind: 'external',
			href: 'https://dashboard.example.test/auth'
		});
	});

	it('uses a SvelteKit path when already on dashboard', async () => {
		const { resolveDashboardNavigation } = await loadClient();

		expect(resolveDashboardNavigation('/account', 'https://dashboard.example.test')).toEqual({
			kind: 'internal',
			href: '/account'
		});
	});

	it('lets callers handle 401 responses when auth redirects are disabled', async () => {
		const { request } = await loadClient();
		vi.stubGlobal(
			'fetch',
			vi.fn(() =>
				Promise.resolve(
					new Response(
						JSON.stringify({
							code: 'invalid_pairing_code',
							message: 'Pairing code is invalid or expired'
						}),
						{ status: 401 }
					)
				)
			)
		);

		await expect(request('/pairing', { redirectOnAuthError: false })).rejects.toMatchObject({
			name: 'ApiError',
			code: 'invalid_pairing_code',
			status: 401
		} satisfies Partial<ApiError>);
	});
});
