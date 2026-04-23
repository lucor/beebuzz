import { describe, expect, it } from 'vitest';
import { parseDeploymentMode } from './deployment';

describe('parseDeploymentMode', () => {
	it('accepts saas explicitly', () => {
		expect(parseDeploymentMode('saas')).toBe('saas');
	});

	it('defaults unknown values to self_hosted', () => {
		expect(parseDeploymentMode('true')).toBe('self_hosted');
		expect(parseDeploymentMode('')).toBe('self_hosted');
		expect(parseDeploymentMode(undefined)).toBe('self_hosted');
	});
});
