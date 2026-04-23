import { beforeEach, describe, expect, it, vi } from 'vitest';

const unsubscribeFromPush = vi.fn();
const deleteAllKeys = vi.fn();
const warn = vi.fn();

vi.mock('$lib/services/push', () => ({
	unsubscribeFromPush
}));

vi.mock('$lib/services/encryption', () => ({
	deleteAllKeys
}));

vi.mock('@beebuzz/shared/logger', () => ({
	logger: {
		warn
	}
}));

describe('cleanupStalePairingState', () => {
	beforeEach(() => {
		vi.resetModules();
		unsubscribeFromPush.mockReset();
		deleteAllKeys.mockReset();
		warn.mockReset();
	});

	it('cleans both push subscription and stored keys', async () => {
		const { cleanupStalePairingState } = await import('./startup-recovery');

		await cleanupStalePairingState();

		expect(unsubscribeFromPush).toHaveBeenCalledTimes(1);
		expect(deleteAllKeys).toHaveBeenCalledTimes(1);
		expect(warn).not.toHaveBeenCalled();
	});

	it('continues clearing keys when push unsubscribe fails', async () => {
		unsubscribeFromPush.mockRejectedValueOnce(new Error('unsubscribe failed'));
		const { cleanupStalePairingState } = await import('./startup-recovery');

		await cleanupStalePairingState();

		expect(deleteAllKeys).toHaveBeenCalledTimes(1);
		expect(warn).toHaveBeenCalledWith('Failed to unsubscribe stale push subscription', {
			error: 'Error: unsubscribe failed'
		});
	});

	it('logs key cleanup failures without throwing', async () => {
		deleteAllKeys.mockRejectedValueOnce(new Error('clear keys failed'));
		const { cleanupStalePairingState } = await import('./startup-recovery');

		await expect(cleanupStalePairingState()).resolves.toBeUndefined();

		expect(warn).toHaveBeenCalledWith('Failed to clear stale encryption keys', {
			error: 'Error: clear keys failed'
		});
	});

	it('logs both failures when unsubscribe and key cleanup fail in the same run', async () => {
		unsubscribeFromPush.mockRejectedValueOnce(new Error('unsubscribe failed'));
		deleteAllKeys.mockRejectedValueOnce(new Error('clear keys failed'));
		const { cleanupStalePairingState } = await import('./startup-recovery');

		await expect(cleanupStalePairingState()).resolves.toBeUndefined();

		expect(warn).toHaveBeenNthCalledWith(1, 'Failed to unsubscribe stale push subscription', {
			error: 'Error: unsubscribe failed'
		});
		expect(warn).toHaveBeenNthCalledWith(2, 'Failed to clear stale encryption keys', {
			error: 'Error: clear keys failed'
		});
	});
});
