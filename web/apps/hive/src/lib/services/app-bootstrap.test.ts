import { describe, expect, it } from 'vitest';
import { bootstrapAppShell } from './app-bootstrap';

describe('bootstrapAppShell', () => {
	it('activates the current device before attaching the bridge and draining IndexedDB', async () => {
		const calls: string[] = [];
		const registration = { scope: '/' };

		const result = await bootstrapAppShell({
			registerServiceWorker: () => {
				calls.push('register');
				return Promise.resolve(registration);
			},
			checkPaired: () => {
				calls.push('checkPaired');
				return Promise.resolve(true);
			},
			getDeviceId: () => {
				calls.push('getDeviceId');
				return Promise.resolve('dev-a');
			},
			activateNotifications: (deviceId) => {
				calls.push(`activate:${deviceId}`);
			},
			attachServiceWorkerListeners: () => {
				calls.push('attach');
			},
			migrateLegacyNotifications: (deviceId) => {
				calls.push(`migrate:${deviceId}`);
				return Promise.resolve();
			},
			loadPersistedNotifications: (phase) => {
				calls.push(`load:${phase}`);
				return Promise.resolve();
			},
			runPostPairingChecks: () => {
				calls.push('postChecks');
				return Promise.resolve();
			}
		});

		expect(result).toEqual({ registration, isPaired: true, deviceId: 'dev-a' });
		expect(calls).toEqual([
			'register',
			'checkPaired',
			'getDeviceId',
			'activate:dev-a',
			'attach',
			'migrate:dev-a',
			'load:initial',
			'postChecks',
			'load:final'
		]);
	});

	it('skips notification activation and drains when the device is not paired', async () => {
		const calls: string[] = [];

		const result = await bootstrapAppShell({
			registerServiceWorker: () => {
				calls.push('register');
				return Promise.resolve({ scope: '/' });
			},
			checkPaired: () => {
				calls.push('checkPaired');
				return Promise.resolve(false);
			},
			getDeviceId: () => {
				calls.push('getDeviceId');
				return Promise.resolve('dev-a');
			},
			activateNotifications: (deviceId) => {
				calls.push(`activate:${deviceId}`);
			},
			attachServiceWorkerListeners: () => {
				calls.push('attach');
			},
			migrateLegacyNotifications: (deviceId) => {
				calls.push(`migrate:${deviceId}`);
				return Promise.resolve();
			},
			loadPersistedNotifications: (phase) => {
				calls.push(`load:${phase}`);
				return Promise.resolve();
			},
			runPostPairingChecks: () => {
				calls.push('postChecks');
				return Promise.resolve();
			}
		});

		expect(result.isPaired).toBe(false);
		expect(result.deviceId).toBeNull();
		expect(calls).toEqual(['register', 'checkPaired']);
	});

	it('leaves notification history inactive when paired credentials are missing', async () => {
		const calls: string[] = [];

		const result = await bootstrapAppShell({
			registerServiceWorker: () => {
				calls.push('register');
				return Promise.resolve({ scope: '/' });
			},
			checkPaired: () => {
				calls.push('checkPaired');
				return Promise.resolve(true);
			},
			getDeviceId: () => {
				calls.push('getDeviceId');
				return Promise.resolve(null);
			},
			activateNotifications: (deviceId) => {
				calls.push(`activate:${deviceId}`);
			},
			attachServiceWorkerListeners: () => {
				calls.push('attach');
			},
			migrateLegacyNotifications: (deviceId) => {
				calls.push(`migrate:${deviceId}`);
				return Promise.resolve();
			},
			loadPersistedNotifications: (phase) => {
				calls.push(`load:${phase}`);
				return Promise.resolve();
			},
			runPostPairingChecks: () => {
				calls.push('postChecks');
				return Promise.resolve();
			}
		});

		expect(result).toEqual({ registration: { scope: '/' }, isPaired: true, deviceId: null });
		expect(calls).toEqual(['register', 'checkPaired', 'getDeviceId']);
	});

	it('skips the final drain when post-pairing checks fail but keeps the listener attached', async () => {
		const calls: string[] = [];
		const failure = new Error('post-checks failed');

		await expect(
			bootstrapAppShell({
				registerServiceWorker: () => {
					calls.push('register');
					return Promise.resolve({ scope: '/' });
				},
				checkPaired: () => {
					calls.push('checkPaired');
					return Promise.resolve(true);
				},
				getDeviceId: () => {
					calls.push('getDeviceId');
					return Promise.resolve('dev-a');
				},
				activateNotifications: (deviceId) => {
					calls.push(`activate:${deviceId}`);
				},
				attachServiceWorkerListeners: () => {
					calls.push('attach');
				},
				migrateLegacyNotifications: (deviceId) => {
					calls.push(`migrate:${deviceId}`);
					return Promise.resolve();
				},
				loadPersistedNotifications: (phase) => {
					calls.push(`load:${phase}`);
					return Promise.resolve();
				},
				runPostPairingChecks: () => {
					calls.push('postChecks');
					return Promise.reject(failure);
				}
			})
		).rejects.toBe(failure);

		expect(calls).toEqual([
			'register',
			'checkPaired',
			'getDeviceId',
			'activate:dev-a',
			'attach',
			'migrate:dev-a',
			'load:initial',
			'postChecks'
		]);
	});
});
