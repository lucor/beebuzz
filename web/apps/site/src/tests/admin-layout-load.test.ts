import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '@beebuzz/shared/errors';
import type { AuthUser } from '@beebuzz/shared/api';

const { meMock, authSetMock } = vi.hoisted(() => ({
	meMock: vi.fn(),
	authSetMock: vi.fn()
}));

vi.mock('@beebuzz/shared/api', () => ({
	accountApi: {
		me: meMock
	}
}));

vi.mock('@beebuzz/shared/stores', () => ({
	auth: {
		set: authSetMock
	}
}));

import { load } from '../routes/(protected)/admin/+layout';

const buildUser = (isAdmin: boolean): AuthUser => ({
	id: 'user_1',
	email: 'user@example.com',
	is_admin: isAdmin,
	account_status: 'active',
	created_at: '2026-01-01T00:00:00Z',
	updated_at: '2026-01-01T00:00:00Z'
});

const callLoad = () => load({} as Parameters<typeof load>[0]);

describe('admin layout load guard', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('returns user data for admin users', async () => {
		const user = buildUser(true);
		meMock.mockResolvedValueOnce(user);

		await expect(callLoad()).resolves.toEqual({ user });
		expect(authSetMock).toHaveBeenCalledWith(user);
	});

	it('redirects non-admin users to /account', async () => {
		const user = buildUser(false);
		meMock.mockResolvedValueOnce(user);

		await expect(callLoad()).rejects.toMatchObject({ status: 302, location: '/account' });
		expect(authSetMock).toHaveBeenCalledWith(user);
	});

	it('redirects unauthenticated users to /login', async () => {
		meMock.mockRejectedValueOnce(new ApiError('invalid_session', 401, 'unauthorized'));

		await expect(callLoad()).rejects.toMatchObject({ status: 302, location: '/login' });
		expect(authSetMock).not.toHaveBeenCalled();
	});

	it('redirects forbidden users to /account', async () => {
		meMock.mockRejectedValueOnce(new ApiError('forbidden', 403, 'forbidden'));

		await expect(callLoad()).rejects.toMatchObject({ status: 302, location: '/account' });
		expect(authSetMock).not.toHaveBeenCalled();
	});
});
