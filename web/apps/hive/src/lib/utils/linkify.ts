export type BodySegment =
	| {
			kind: 'text';
			text: string;
	  }
	| {
			kind: 'link';
			text: string;
			href: string;
	  };

function isSafeHttpsUrl(candidate: string): boolean {
	try {
		const url = new URL(candidate);
		return url.protocol === 'https:';
	} catch {
		return false;
	}
}

/** Split plain text into text and safe HTTPS link segments. */
export function parseHttpsLinkSegments(text: string): BodySegment[] {
	if (!text) return [];

	return text
		.split(/(\s+)/)
		.filter((segment) => segment.length > 0)
		.map((segment) => {
			if (/\s+/.test(segment)) {
				return { kind: 'text', text: segment } as const;
			}

			if (!segment.startsWith('https://')) {
				return { kind: 'text', text: segment } as const;
			}

			if (!isSafeHttpsUrl(segment)) {
				return { kind: 'text', text: segment } as const;
			}

			return { kind: 'link', text: segment, href: segment } as const;
		});
}
