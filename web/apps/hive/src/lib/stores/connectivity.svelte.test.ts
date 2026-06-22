import { describe, expect, it, beforeEach, afterEach } from 'vitest';

describe('connectivity store', () => {
	let mockOnline: boolean;

	beforeEach(() => {
		mockOnline = true;
		Object.defineProperty(navigator, 'onLine', {
			configurable: true,
			get: () => mockOnline
		});
	});

	afterEach(async () => {
		const { connectivity } = await import('./connectivity.svelte');
		connectivity.destroy();
		connectivity._resetForTest(true);
	});

	it('starts online by default before init', async () => {
		const { connectivity } = await import('./connectivity.svelte');
		expect(connectivity.online).toBe(true);
		expect(connectivity.tone).toBe('online');
		expect(connectivity.label).toBe('Online');
	});

	it('reads navigator.onLine on init', async () => {
		mockOnline = false;
		const { connectivity } = await import('./connectivity.svelte');
		connectivity.init();
		expect(connectivity.online).toBe(false);
		expect(connectivity.tone).toBe('offline');
		expect(connectivity.label).toBe('Offline');
	});

	it('transitions to offline when offline event fires', async () => {
		const { connectivity } = await import('./connectivity.svelte');
		connectivity.init();
		window.dispatchEvent(new Event('offline'));
		expect(connectivity.online).toBe(false);
		expect(connectivity.tone).toBe('offline');
		expect(connectivity.label).toBe('Offline');
	});

	it('transitions back to online when online event fires', async () => {
		mockOnline = false;
		const { connectivity } = await import('./connectivity.svelte');
		connectivity.init();
		expect(connectivity.online).toBe(false);

		mockOnline = true;
		window.dispatchEvent(new Event('online'));
		expect(connectivity.online).toBe(true);
		expect(connectivity.tone).toBe('online');
		expect(connectivity.label).toBe('Online');
	});

	it('stops listening after destroy', async () => {
		const { connectivity } = await import('./connectivity.svelte');
		connectivity.init();
		connectivity.destroy();

		window.dispatchEvent(new Event('offline'));
		expect(connectivity.online).toBe(true);
	});

	it('tone and label are derived from online', async () => {
		const { connectivity } = await import('./connectivity.svelte');
		connectivity._resetForTest(true);
		expect(connectivity.tone).toBe('online');
		expect(connectivity.label).toBe('Online');

		connectivity._resetForTest(false);
		expect(connectivity.tone).toBe('offline');
		expect(connectivity.label).toBe('Offline');
	});
});
