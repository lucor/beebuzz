import { fetchBlob } from '@beebuzz/shared/api';
import { decryptBinary } from '$lib/services/encryption';

// In-memory cache for decrypted attachments (survives navigation within session)
const attachmentCache = new Map<string, CachedAttachment>();

export interface CachedAttachment {
	blob: Blob;
	mimeType: string;
	timestamp: number;
}

const PREVIEWABLE_IMAGE_MIME_TYPES = new Set([
	'image/jpeg',
	'image/png',
	'image/webp',
	'image/gif'
]);

const PREVIEWABLE_VIDEO_MIME_TYPES = new Set(['video/mp4']);

function normalizeMime(mime: string): string {
	return mime.split(';', 1)[0]?.trim().toLowerCase() ?? '';
}

/**
 * Fetch, optionally decrypt, and cache an attachment.
 * @param url - URL to fetch (internal paths starting with "/" use the API client)
 * @param mimeType - MIME type of the attachment
 * @param encrypted - Whether the response is age-encrypted
 */
export async function fetchAndCacheAttachment(
	url: string,
	mimeType = 'application/octet-stream',
	encrypted = false
): Promise<CachedAttachment> {
	const cached = attachmentCache.get(url);
	if (cached) {
		return cached;
	}

	const blob = url.startsWith('/')
		? await fetchBlob(url)
		: await fetch(url).then((r) => {
				if (!r.ok) throw new Error(`HTTP ${r.status}`);
				return r.blob();
			});

	let finalBlob: Blob;
	if (encrypted) {
		const ciphertext = new Uint8Array(await blob.arrayBuffer());
		const plaintext = await decryptBinary(ciphertext);
		finalBlob = new Blob([plaintext.buffer as ArrayBuffer], { type: mimeType });
	} else {
		finalBlob = blob;
	}

	const result: CachedAttachment = {
		blob: finalBlob,
		mimeType: finalBlob.type || mimeType,
		timestamp: Date.now()
	};

	attachmentCache.set(url, result);
	return result;
}

/** Returns true if the MIME type represents an image. */
export function isImageMime(mime: string): boolean {
	return PREVIEWABLE_IMAGE_MIME_TYPES.has(normalizeMime(mime));
}

/** Returns true if the MIME type represents a previewable video. */
export function isVideoMime(mime: string): boolean {
	return PREVIEWABLE_VIDEO_MIME_TYPES.has(normalizeMime(mime));
}

/** Clear attachment cache */
export function clearAttachmentCache(): void {
	attachmentCache.clear();
}
