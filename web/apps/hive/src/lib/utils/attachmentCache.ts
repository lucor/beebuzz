import { fetchBlob } from '@beebuzz/shared/api';
import { decryptBinary } from '$lib/services/encryption';

// In-memory cache for decrypted attachments (survives navigation within session)
const attachmentCache = new Map<string, CachedAttachment>();

export interface CachedAttachment {
	dataUrl: string;
	mimeType: string;
	timestamp: number;
}

/** Convert blob to data URL */
function blobToDataUrl(blob: Blob): Promise<string> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.onload = () => resolve(reader.result as string);
		reader.onerror = () => reject(new Error('Failed to read blob'));
		reader.readAsDataURL(blob);
	});
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

	const dataUrl = await blobToDataUrl(finalBlob);
	const result: CachedAttachment = {
		dataUrl,
		mimeType: finalBlob.type || mimeType,
		timestamp: Date.now()
	};

	attachmentCache.set(url, result);
	return result;
}

/** Returns true if the MIME type represents an image. */
export function isImageMime(mime: string): boolean {
	return mime.startsWith('image/');
}

/** Clear attachment cache */
export function clearAttachmentCache(): void {
	attachmentCache.clear();
}
