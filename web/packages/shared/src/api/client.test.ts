import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../errors';

const mockGoto = vi.fn();
vi.mock('$app/navigation', () => ({
	goto: mockGoto
}));

const mockClear = vi.fn();
vi.mock('../stores', () => ({
	auth: {
		clear: mockClear,
		get user() {
			return null;
		},
		set: vi.fn()
	}
}));

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

describe('request', () => {
	beforeEach(() => {
		vi.unstubAllEnvs();
		vi.restoreAllMocks();
		vi.unstubAllGlobals();
	});

	it('redirects to /auth and clears auth on 401 with default redirectOnAuthError', async () => {
		const { request } = await loadClient();

		// Set location to match DASHBOARD_URL so redirectToDashboard uses goto
		Object.defineProperty(window, 'location', {
			value: { origin: 'https://dashboard.example.test', assign: vi.fn() },
			configurable: true,
			writable: true
		});

		vi.stubGlobal(
			'fetch',
			vi.fn(() =>
				Promise.resolve(
					new Response(
						JSON.stringify({
							code: 'invalid_session',
							message: 'Invalid or expired session'
						}),
						{ status: 401 }
					)
				)
			)
		);

		await expect(request('/me')).resolves.toBeUndefined();
		expect(mockClear).toHaveBeenCalled();
		expect(mockGoto).toHaveBeenCalledWith('/auth', { replaceState: true });
	});
});
