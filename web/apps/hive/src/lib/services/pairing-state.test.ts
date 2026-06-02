import { describe, expect, it, vi } from 'vitest';
import { checkPairedState } from './pairing-state';

describe('checkPairedState', () => {
	it('returns false when notifications are not granted', async () => {
		const getRegistration = vi.fn();
		const hasIdentity = vi.fn();

		await expect(
			checkPairedState({
				notificationPermission: 'default',
				getRegistration,
				hasIdentity
			})
		).resolves.toBe(false);

		expect(getRegistration).not.toHaveBeenCalled();
		expect(hasIdentity).not.toHaveBeenCalled();
	});

	it('returns false when no service worker registration exists', async () => {
		const hasIdentity = vi.fn();

		await expect(
			checkPairedState({
				notificationPermission: 'granted',
				getRegistration: vi.fn(() => Promise.resolve(undefined)),
				hasIdentity
			})
		).resolves.toBe(false);

		expect(hasIdentity).not.toHaveBeenCalled();
	});

	it('returns false when the registration has no push subscription', async () => {
		const hasIdentity = vi.fn();
		const registration = {
			pushManager: {
				getSubscription: vi.fn(() => Promise.resolve(null))
			}
		} as unknown as ServiceWorkerRegistration;

		await expect(
			checkPairedState({
				notificationPermission: 'granted',
				getRegistration: vi.fn(() => Promise.resolve(registration)),
				hasIdentity
			})
		).resolves.toBe(false);

		expect(hasIdentity).not.toHaveBeenCalled();
	});

	it('returns true only when subscription and local identity both exist', async () => {
		const registration = {
			pushManager: {
				getSubscription: vi.fn(() => Promise.resolve({ endpoint: 'https://push.example.test/sub' }))
			}
		} as unknown as ServiceWorkerRegistration;
		const hasIdentity = vi.fn(() => Promise.resolve(true));

		await expect(
			checkPairedState({
				notificationPermission: 'granted',
				getRegistration: vi.fn(() => Promise.resolve(registration)),
				hasIdentity
			})
		).resolves.toBe(true);

		expect(hasIdentity).toHaveBeenCalledTimes(1);
	});

	it('returns false when local identity is missing', async () => {
		const registration = {
			pushManager: {
				getSubscription: vi.fn(() => Promise.resolve({ endpoint: 'https://push.example.test/sub' }))
			}
		} as unknown as ServiceWorkerRegistration;
		const hasIdentity = vi.fn(() => Promise.resolve(false));

		await expect(
			checkPairedState({
				notificationPermission: 'granted',
				getRegistration: vi.fn(() => Promise.resolve(registration)),
				hasIdentity
			})
		).resolves.toBe(false);
	});

	it('rejects when service worker registration lookup fails', async () => {
		await expect(
			checkPairedState({
				notificationPermission: 'granted',
				getRegistration: vi.fn(() => Promise.reject(new Error('registration failed'))),
				hasIdentity: vi.fn()
			})
		).rejects.toThrow('registration failed');
	});

	it('rejects when push subscription lookup fails', async () => {
		const registration = {
			pushManager: {
				getSubscription: vi.fn(() => Promise.reject(new Error('subscription failed')))
			}
		} as unknown as ServiceWorkerRegistration;

		await expect(
			checkPairedState({
				notificationPermission: 'granted',
				getRegistration: vi.fn(() => Promise.resolve(registration)),
				hasIdentity: vi.fn()
			})
		).rejects.toThrow('subscription failed');
	});

	it('rejects when identity lookup fails', async () => {
		const registration = {
			pushManager: {
				getSubscription: vi.fn(() => Promise.resolve({ endpoint: 'https://push.example.test/sub' }))
			}
		} as unknown as ServiceWorkerRegistration;

		await expect(
			checkPairedState({
				notificationPermission: 'granted',
				getRegistration: vi.fn(() => Promise.resolve(registration)),
				hasIdentity: vi.fn(() => Promise.reject(new Error('identity failed')))
			})
		).rejects.toThrow('identity failed');
	});
});
