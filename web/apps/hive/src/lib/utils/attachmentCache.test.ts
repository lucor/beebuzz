import { beforeEach, describe, expect, it, vi } from 'vitest';

const fetchBlobMock = vi.fn();
const decryptBinaryMock = vi.fn();

vi.mock('@beebuzz/shared/api', () => ({
	fetchBlob: fetchBlobMock
}));

vi.mock('$lib/services/encryption', () => ({
	decryptBinary: decryptBinaryMock
}));

describe('attachmentCache', () => {
	beforeEach(() => {
		vi.resetModules();
		vi.clearAllMocks();
	});

	it('decrypts encrypted attachments into a cached Blob result', async () => {
		const ciphertext = new Uint8Array([1, 2, 3]);
		const plaintext = new Uint8Array([7, 8, 9]);
		fetchBlobMock.mockResolvedValue(new Blob([ciphertext], { type: 'application/octet-stream' }));
		decryptBinaryMock.mockResolvedValue(plaintext);

		const { clearAttachmentCache, fetchAndCacheAttachment } = await import('./attachmentCache');
		clearAttachmentCache();

		const first = await fetchAndCacheAttachment('/attachments/token-1', 'video/mp4', true);
		const second = await fetchAndCacheAttachment('/attachments/token-1', 'video/mp4', true);

		expect(fetchBlobMock).toHaveBeenCalledTimes(1);
		expect(decryptBinaryMock).toHaveBeenCalledTimes(1);
		expect(first).toBe(second);
		expect(first.mimeType).toBe('video/mp4');
		expect(await first.blob.arrayBuffer()).toEqual(
			await new Blob([plaintext], { type: 'video/mp4' }).arrayBuffer()
		);
	});

	it('matches only the supported preview MIME types', async () => {
		const { isImageMime, isVideoMime } = await import('./attachmentCache');

		expect(isImageMime('image/png')).toBe(true);
		expect(isImageMime('image/svg+xml')).toBe(false);
		expect(isImageMime('image/webp; charset=utf-8')).toBe(true);
		expect(isVideoMime('video/mp4')).toBe(true);
		expect(isVideoMime('video/webm')).toBe(false);
		expect(isVideoMime('video/mp4; codecs=avc1.42E01E,mp4a.40.2')).toBe(true);
	});
});
