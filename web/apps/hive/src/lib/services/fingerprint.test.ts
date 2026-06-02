import { describe, expect, it } from 'vitest';
import { computeRecipientFingerprint } from './fingerprint';

describe('computeRecipientFingerprint', () => {
	it('matches the backend fingerprint format for known recipients', () => {
		expect(
			computeRecipientFingerprint('age1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq0d8l7g')
		).toBe('c6856fc8854f2dae');

		expect(
			computeRecipientFingerprint('age1ex4mwjm2g0x8j0l4a9m3r5n5g5j2u4v6k8f3z2k2d9l9t8k2d9sqf9x7j')
		).toBe('a7c0d71489673bbd');
	});

	it('returns a stable 16-character lowercase hex fingerprint', () => {
		const fingerprint = computeRecipientFingerprint(
			'age1q9rj6gr4t6jap0h8j4z0h4g6y0yeg4m2vjlwm4usv29w07e3vqas3q7s7p'
		);

		expect(fingerprint).toMatch(/^[0-9a-f]{16}$/);
		expect(fingerprint).toHaveLength(16);
	});
});
