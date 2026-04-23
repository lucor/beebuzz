import { blake2b } from '@noble/hashes/blake2.js';

const FINGERPRINT_HEX_LENGTH = 16;
const FINGERPRINT_DIGEST_LENGTH = 32;

/** Computes the short BeeBuzz fingerprint for an age public recipient. */
export function computeRecipientFingerprint(recipient: string): string {
	const digest = blake2b(new TextEncoder().encode(recipient), {
		dkLen: FINGERPRINT_DIGEST_LENGTH
	});

	return Array.from(digest, (byte) => byte.toString(16).padStart(2, '0'))
		.join('')
		.slice(0, FINGERPRINT_HEX_LENGTH);
}
