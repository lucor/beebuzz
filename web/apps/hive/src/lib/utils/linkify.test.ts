import { describe, expect, it } from 'vitest';
import { parseHttpsLinkSegments } from './linkify';

describe('parseHttpsLinkSegments', () => {
	it('keeps plain text untouched when there is no safe https link', () => {
		const segments = parseHttpsLinkSegments(
			'Visit www.example.com, http://example.com, mailto:test@example.com, and custom://value.'
		);

		expect(segments).toEqual([
			{
				kind: 'text',
				text: 'Visit'
			},
			{
				kind: 'text',
				text: ' '
			},
			{
				kind: 'text',
				text: 'www.example.com,'
			},
			{
				kind: 'text',
				text: ' '
			},
			{
				kind: 'text',
				text: 'http://example.com,'
			},
			{
				kind: 'text',
				text: ' '
			},
			{
				kind: 'text',
				text: 'mailto:test@example.com,'
			},
			{
				kind: 'text',
				text: ' '
			},
			{
				kind: 'text',
				text: 'and'
			},
			{
				kind: 'text',
				text: ' '
			},
			{
				kind: 'text',
				text: 'custom://value.'
			}
		]);
	});

	it('linkifies only tokens that start with https and parse as safe URLs', () => {
		const segments = parseHttpsLinkSegments(
			'Docs: https://example.com/path?q=1 and https://example.com/space%20here'
		);

		expect(segments).toEqual([
			{
				kind: 'text',
				text: 'Docs:'
			},
			{
				kind: 'text',
				text: ' '
			},
			{
				kind: 'link',
				text: 'https://example.com/path?q=1',
				href: 'https://example.com/path?q=1'
			},
			{
				kind: 'text',
				text: ' '
			},
			{
				kind: 'text',
				text: 'and'
			},
			{
				kind: 'text',
				text: ' '
			},
			{
				kind: 'link',
				text: 'https://example.com/space%20here',
				href: 'https://example.com/space%20here'
			}
		]);
	});

	it('rejects tokens that fail URL parsing or do not use https', () => {
		const segments = parseHttpsLinkSegments('https://% https://example.com\nhttp://example.com');

		expect(segments).toEqual([
			{
				kind: 'text',
				text: 'https://%'
			},
			{
				kind: 'text',
				text: ' '
			},
			{
				kind: 'link',
				text: 'https://example.com',
				href: 'https://example.com'
			},
			{
				kind: 'text',
				text: '\n'
			},
			{
				kind: 'text',
				text: 'http://example.com'
			}
		]);
	});
});
