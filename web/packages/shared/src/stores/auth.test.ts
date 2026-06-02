import { beforeEach, describe, expect, it } from 'vitest';
import type { AuthUser } from '../api';
import { auth } from './auth.svelte';

const buildUser = (): AuthUser => ({
	id: 'user_1',
	email: 'user@example.com',
	is_admin: false,
	created_at: '2026-01-01T00:00:00Z',
	updated_at: '2026-01-01T00:00:00Z'
});

describe('auth store', () => {
	beforeEach(() => {
		auth.clear();
	});

	it('starts unauthenticated', () => {
		expect(auth.user).toBeNull();
	});

	it('sets the current user', () => {
		const user = buildUser();
		auth.set(user);

		expect(auth.user).toEqual(user);
	});

	it('clears the current user', () => {
		auth.set(buildUser());
		auth.clear();

		expect(auth.user).toBeNull();
	});
});
