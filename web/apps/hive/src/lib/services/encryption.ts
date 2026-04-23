import * as age from 'age-encryption';
import { computeRecipientFingerprint } from './fingerprint';
import { deviceKeysRepository, type KeyMetadata } from './device-keys-repository';
const PENDING_PAIRING_KEY_ID = '__pending_pairing__';
const X25519_ALGORITHM = 'X25519';
const X25519_USAGES: KeyUsage[] = ['deriveBits'];
const AES_GCM_ALGORITHM = 'AES-GCM';
const AES_GCM_KEY_LENGTH = 256;
const AES_GCM_IV_LENGTH = 12;
const AES_GCM_USAGES: KeyUsage[] = ['wrapKey', 'unwrapKey'];

export interface StoredDeviceKey {
	id: string;
	recipient: string;
	fingerprint: string;
	createdAt: string;
}

export class MissingDeviceIdentityError extends Error {
	constructor(message = 'No encryption key found for this device') {
		super(message);
		this.name = 'MissingDeviceIdentityError';
	}
}

export class DeviceIdentityIntegrityError extends Error {
	constructor(message = 'Stored device key is missing or invalid') {
		super(message);
		this.name = 'DeviceIdentityIntegrityError';
	}
}

/**
 * BeeBuzz uses one storage strategy everywhere:
 * - Persist an AES-GCM wrapping key as a non-extractable CryptoKey
 * - Wrap the extractable X25519 private key once during pairing
 * - Store the wrapped PKCS8 bytes with AES-GCM metadata in IndexedDB
 * - Unwrap the stored bytes back into a non-extractable runtime key when needed
 *
 * Why: Safari does not reliably deserialize persisted X25519 CryptoKeys, but it
 * does reliably persist AES-GCM CryptoKeys. Using a single wrapped-key flow keeps
 * the implementation consistent across browsers and avoids maintaining two code
 * paths that would have to stay in sync between the app and the service worker.
 */

/** Generates a non-extractable AES-GCM wrapping key used to protect the age key at rest. */
async function generateWrappingKey(): Promise<CryptoKey> {
	return crypto.subtle.generateKey(
		{ name: AES_GCM_ALGORITHM, length: AES_GCM_KEY_LENGTH },
		false,
		AES_GCM_USAGES
	);
}

/** Generates the X25519 identity and exports it once so it can be wrapped for storage. */
async function generateExportableIdentity(): Promise<CryptoKeyPair> {
	return (await crypto.subtle.generateKey(
		{ name: X25519_ALGORITHM },
		true,
		X25519_USAGES
	)) as CryptoKeyPair;
}

/** Returns a plain ArrayBuffer copy suitable for WebCrypto and IndexedDB. */
function toArrayBuffer(bytes: Uint8Array): ArrayBuffer {
	const copy = new Uint8Array(bytes.byteLength);
	copy.set(bytes);
	return copy.buffer;
}

/** Creates a fresh AES-GCM IV for wrapping/unwrapping the PKCS8 payload. */
function createWrappingIv(): ArrayBuffer {
	return toArrayBuffer(crypto.getRandomValues(new Uint8Array(AES_GCM_IV_LENGTH)));
}

/** Wraps the private key using the persisted AES-GCM wrapping key. */
async function wrapPrivateKey(
	wrappingKey: CryptoKey,
	recipient: string,
	privateKey: CryptoKey
): Promise<{ id: string; wrappedKey: ArrayBuffer; iv: ArrayBuffer; format: 'pkcs8' }> {
	const iv = createWrappingIv();
	const wrappedKey = await crypto.subtle.wrapKey('pkcs8', privateKey, wrappingKey, {
		name: AES_GCM_ALGORITHM,
		iv,
		additionalData: toArrayBuffer(new TextEncoder().encode(`recipient:${recipient}`))
	});

	return {
		id: '',
		wrappedKey,
		iv,
		format: 'pkcs8'
	};
}

/** Unwraps the stored private key with the wrapping key. */
async function unwrapPrivateKey(
	wrappingKey: CryptoKey,
	record: { wrappedKey: ArrayBuffer; iv: ArrayBuffer; format: 'pkcs8' },
	recipient: string
): Promise<CryptoKey> {
	return crypto.subtle.unwrapKey(
		'pkcs8',
		record.wrappedKey,
		wrappingKey,
		{
			name: AES_GCM_ALGORITHM,
			iv: record.iv,
			additionalData: toArrayBuffer(new TextEncoder().encode(`recipient:${recipient}`))
		},
		{ name: X25519_ALGORITHM },
		false,
		X25519_USAGES
	);
}

/** Renames the locally persisted wrapped identity after the backend returns the canonical device ID. */
export async function renameStoredKeyPair(currentKeyId: string, nextKeyId: string): Promise<void> {
	return deviceKeysRepository.rename(currentKeyId, nextKeyId);
}

async function loadStoredIdentity(metadata: KeyMetadata): Promise<CryptoKey | null> {
	const { wrappingKey, wrappedPrivateKey } = await deviceKeysRepository.getStoredMaterial(
		metadata.id
	);

	if (!wrappingKey || !wrappedPrivateKey) {
		return null;
	}

	try {
		const identity = await unwrapPrivateKey(wrappingKey, wrappedPrivateKey, metadata.recipient);
		const derivedRecipient = await age.identityToRecipient(identity);
		if (derivedRecipient !== metadata.recipient) {
			throw new DeviceIdentityIntegrityError('Stored device key does not match recipient metadata');
		}

		return identity;
	} catch (error) {
		if (error instanceof DeviceIdentityIntegrityError) {
			throw error;
		}

		throw new DeviceIdentityIntegrityError(
			error instanceof Error ? error.message : 'Stored device key could not be loaded'
		);
	}
}

/** Generates a wrapped X25519 identity and stores it in IndexedDB. */
export async function generateAndStoreKeyPair(deviceId: string): Promise<string> {
	const wrappingKey = await generateWrappingKey();
	const { privateKey } = await generateExportableIdentity();
	const recipient = await age.identityToRecipient(privateKey);
	const fingerprint = computeRecipientFingerprint(recipient);
	const metadata = {
		id: deviceId,
		recipient,
		fingerprint,
		createdAt: new Date().toISOString()
	};

	const wrappedPrivateKey = await wrapPrivateKey(wrappingKey, recipient, privateKey);
	await deviceKeysRepository.storeWrappedIdentity(
		deviceId,
		metadata,
		wrappingKey,
		wrappedPrivateKey
	);

	const storedIdentity = await loadStoredIdentity(metadata);
	if (!storedIdentity) {
		throw new DeviceIdentityIntegrityError('Stored device key could not be reloaded after write');
	}

	return recipient;
}

/** Creates the temporary local key used while backend pairing is still in progress. */
export async function generateAndStorePendingKeyPair(): Promise<string> {
	return generateAndStoreKeyPair(PENDING_PAIRING_KEY_ID);
}

/** Promotes the temporary pairing key to the canonical backend device ID. */
export async function finalizePendingStoredKeyPair(deviceId: string): Promise<void> {
	return renameStoredKeyPair(PENDING_PAIRING_KEY_ID, deviceId);
}

/** Loads, unwraps, and re-imports the first device identity as a non-extractable CryptoKey. */
export async function getDeviceIdentity(): Promise<CryptoKey | null> {
	const metadata = await deviceKeysRepository.getFirstMetadata();

	if (!metadata) {
		return null;
	}

	return loadStoredIdentity(metadata);
}

/** Returns true if at least one encryption key exists in IndexedDB. */
export async function hasIdentity(): Promise<boolean> {
	const metadata = await deviceKeysRepository.getFirstMetadata();
	return metadata !== null;
}

/** Returns true only when the stored key belongs to a fully paired backend device. */
export async function hasPairedIdentity(): Promise<boolean> {
	const metadata = await deviceKeysRepository.getFirstMetadata();
	if (!metadata) {
		return false;
	}

	if (metadata.id === PENDING_PAIRING_KEY_ID) {
		return false;
	}

	try {
		return (await loadStoredIdentity(metadata)) !== null;
	} catch {
		return false;
	}
}

/** Returns the first stored device public key metadata, if available. */
export async function getStoredDeviceKey(): Promise<StoredDeviceKey | null> {
	const metadata = await deviceKeysRepository.getFirstMetadata();
	if (!metadata) {
		return null;
	}

	return {
		id: metadata.id,
		recipient: metadata.recipient,
		fingerprint: metadata.fingerprint ?? computeRecipientFingerprint(metadata.recipient),
		createdAt: metadata.createdAt
	};
}

/** Deletes all stored encryption material from IndexedDB. */
export async function deleteAllKeys(): Promise<void> {
	return deviceKeysRepository.clearAll();
}

/** Decrypts age-encrypted binary data and returns the plaintext as Uint8Array. */
export async function decryptBinary(ciphertext: Uint8Array): Promise<Uint8Array> {
	const identity = await getDeviceIdentity();
	if (!identity) {
		throw new MissingDeviceIdentityError();
	}

	const decrypter = new age.Decrypter();
	decrypter.addIdentity(identity);
	return decrypter.decrypt(ciphertext);
}
