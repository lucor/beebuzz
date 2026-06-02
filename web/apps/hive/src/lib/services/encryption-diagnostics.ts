import * as age from 'age-encryption';
import { logger } from '@beebuzz/shared/logger';
import type {
	EncryptionProbeResult,
	EncryptionProbeScenarioResult,
	EncryptionProbeStepResult,
	KeyPersistenceProbeResult,
	WrappingKeyProbeResult
} from '$lib/types/encryption';

const PROBE_DB_NAME = 'BeeBuzzEncryptionProbe';
const PROBE_DB_VERSION = 1;
const KEY_ONLY_STORE = 'key_only';
const DERIVE_BITS_USAGE = 'deriveBits';
const X25519_ALGORITHM = 'X25519';
const AES_GCM_ALGORITHM = 'AES-GCM';
const AES_GCM_KEY_LENGTH = 256;
const AES_GCM_IV_LENGTH = 12;
const AES_GCM_USAGES: KeyUsage[] = ['wrapKey', 'unwrapKey'];
const AES_GCM_WRAP_USAGES: KeyUsage[] = ['wrapKey', 'unwrapKey'];

interface ProbeScenario {
	id: string;
	label: string;
	extractable: boolean;
}

const PROBE_SCENARIOS: ProbeScenario[] = [
	{
		id: 'non_extractable',
		label: 'Non-extractable CryptoKey',
		extractable: false
	}
];

/** Opens the dedicated IndexedDB database used by the encryption probe. */
const openProbeDB = (): Promise<IDBDatabase> => {
	return new Promise((resolve, reject) => {
		const request = indexedDB.open(PROBE_DB_NAME, PROBE_DB_VERSION);

		request.onupgradeneeded = () => {
			const db = request.result;

			if (!db.objectStoreNames.contains(KEY_ONLY_STORE)) {
				db.createObjectStore(KEY_ONLY_STORE);
			}
		};

		request.onerror = () => {
			reject(new Error(request.error?.message ?? 'IndexedDB open failed'));
		};
		request.onsuccess = () => resolve(request.result);
	});
};

/** Clears the probe stores before each scenario run. */
const clearProbeStores = async (db: IDBDatabase): Promise<void> => {
	await new Promise<void>((resolve, reject) => {
		const tx = db.transaction([KEY_ONLY_STORE], 'readwrite');
		tx.objectStore(KEY_ONLY_STORE).clear();
		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(new Error(tx.error?.message ?? 'Probe clear failed'));
	});
};

/** Writes a single key directly as the object store value. */
const putKeyOnly = async (db: IDBDatabase, keyId: string, key: CryptoKey): Promise<void> => {
	await new Promise<void>((resolve, reject) => {
		const tx = db.transaction(KEY_ONLY_STORE, 'readwrite');
		tx.objectStore(KEY_ONLY_STORE).put(key, keyId);
		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(new Error(tx.error?.message ?? 'Key-only put failed'));
	});
};

/** Reads a key previously stored as a direct object store value. */
const getKeyOnly = async (db: IDBDatabase, keyId: string): Promise<CryptoKey | null> => {
	return new Promise((resolve, reject) => {
		const tx = db.transaction(KEY_ONLY_STORE, 'readonly');
		const request = tx.objectStore(KEY_ONLY_STORE).get(keyId);
		request.onsuccess = () => resolve((request.result as CryptoKey | undefined) ?? null);
		request.onerror = () => reject(new Error(request.error?.message ?? 'Key-only get failed'));
	});
};

/** Returns a concise description of the CryptoKey that came back from a probe step. */
const describeKey = (key: CryptoKey | null): string => {
	if (!key) {
		return 'No key returned';
	}

	const usages = key.usages.join(', ') || 'none';
	return `type=${key.type}, algorithm=${key.algorithm.name}, extractable=${String(key.extractable)}, usages=${usages}`;
};

/** Records a successful probe step. */
const createSuccessStep = (
	id: string,
	label: string,
	detail: string
): EncryptionProbeStepResult => {
	return { id, label, ok: true, detail };
};

/** Records a failed probe step. */
const createFailureStep = (
	id: string,
	label: string,
	detail: string
): EncryptionProbeStepResult => {
	return { id, label, ok: false, detail };
};

/** Normalizes thrown values into a readable diagnostic message. */
const formatError = (error: unknown): string => {
	if (error instanceof Error) {
		return `${error.name}: ${error.message}`;
	}

	return String(error);
};

/** Verifies that the loaded key still behaves as an age identity. */
const verifyRecipient = async (key: CryptoKey): Promise<string> => {
	return age.identityToRecipient(key);
};

/** Copies a Uint8Array into a plain ArrayBuffer accepted by WebCrypto typings. */
const toArrayBuffer = (bytes: Uint8Array): ArrayBuffer => {
	const copy = new Uint8Array(bytes.byteLength);
	copy.set(bytes);
	return copy.buffer;
};

/** Creates a deterministic test payload for the wrapping-key probe. */
/** Creates a fixed IV for AES-GCM diagnostics. */
const createProbeIv = (): ArrayBuffer => {
	return toArrayBuffer(crypto.getRandomValues(new Uint8Array(AES_GCM_IV_LENGTH)));
};

/** Appends a step result while preserving execution flow for later diagnostics. */
const pushStepResult = (
	steps: EncryptionProbeStepResult[],
	result: EncryptionProbeStepResult
): void => {
	steps.push(result);
};

/**
 * This probe is intentionally narrow: it keeps only the checks that justified the
 * production design. We only need to know that direct X25519 CryptoKey persistence
 * is broken, not every possible broken storage shape in WebKit.
 */
const runScenario = async (
	db: IDBDatabase,
	scenario: ProbeScenario
): Promise<EncryptionProbeScenarioResult> => {
	const steps: EncryptionProbeStepResult[] = [];
	const keyId = `${scenario.id}_${Date.now()}`;
	let scenarioOk = true;

	await clearProbeStores(db);

	const keyPair = (await crypto.subtle.generateKey(
		{ name: X25519_ALGORITHM },
		scenario.extractable,
		[DERIVE_BITS_USAGE]
	)) as CryptoKeyPair;
	const originalKey = keyPair.privateKey;

	steps.push(
		createSuccessStep('generate', 'Generate key', `Generated ${describeKey(originalKey)}`)
	);

	const originalRecipient = await verifyRecipient(originalKey);
	steps.push(
		createSuccessStep(
			'original_recipient',
			'Use generated key with age',
			`Recipient ${originalRecipient}`
		)
	);

	try {
		await putKeyOnly(db, keyId, originalKey);
		pushStepResult(
			steps,
			createSuccessStep('put_key_only', 'Store key as direct value', 'IndexedDB put succeeded')
		);
	} catch (error) {
		scenarioOk = false;
		pushStepResult(
			steps,
			createFailureStep('put_key_only', 'Store key as direct value', formatError(error))
		);
	}

	let loadedKey: CryptoKey | null = null;
	if (steps.every((step) => step.id !== 'put_key_only' || step.ok)) {
		try {
			loadedKey = await getKeyOnly(db, keyId);
			pushStepResult(
				steps,
				createSuccessStep('get_key_only', 'Load direct key value', describeKey(loadedKey))
			);
		} catch (error) {
			scenarioOk = false;
			pushStepResult(
				steps,
				createFailureStep('get_key_only', 'Load direct key value', formatError(error))
			);
		}

		if (!loadedKey) {
			scenarioOk = false;
			pushStepResult(
				steps,
				createFailureStep('use_loaded_key_only', 'Use loaded direct key', 'Loaded key is null')
			);
		} else {
			try {
				const loadedRecipient = await verifyRecipient(loadedKey);
				pushStepResult(
					steps,
					createSuccessStep(
						'use_loaded_key_only',
						'Use loaded direct key with age',
						`Recipient ${loadedRecipient}`
					)
				);
			} catch (error) {
				scenarioOk = false;
				pushStepResult(
					steps,
					createFailureStep(
						'use_loaded_key_only',
						'Use loaded direct key with age',
						formatError(error)
					)
				);
			}
		}
	}

	return {
		id: scenario.id,
		label: scenario.label,
		ok: scenarioOk,
		recipient: originalRecipient,
		steps
	};
};

/** Probes whether structuredClone itself preserves a CryptoKey before touching IndexedDB. */
const runStructuredCloneProbe = async (): Promise<EncryptionProbeStepResult> => {
	try {
		const keyPair = (await crypto.subtle.generateKey({ name: X25519_ALGORITHM }, false, [
			DERIVE_BITS_USAGE
		])) as CryptoKeyPair;
		const clonedKey = structuredClone(keyPair.privateKey);

		return createSuccessStep(
			'structured_clone',
			'Structured clone CryptoKey',
			describeKey(clonedKey)
		);
	} catch (error) {
		return createFailureStep('structured_clone', 'Structured clone CryptoKey', formatError(error));
	}
};

/**
 * This probe keeps only the wrapping-key checks that matter for BeeBuzz:
 * - the AES-GCM key must persist as a non-extractable CryptoKey
 * - that persisted key must unwrap the stored X25519 private key
 * - a fresh wrap-capable AES-GCM key must support wrapKey/unwrapKey directly
 */
const runWrappingKeyProbe = async (db: IDBDatabase): Promise<WrappingKeyProbeResult> => {
	const steps: EncryptionProbeStepResult[] = [];
	let probeOk = true;
	const keyId = `wrapping_${Date.now()}`;

	await clearProbeStores(db);

	let wrappingKey: CryptoKey;
	try {
		wrappingKey = await crypto.subtle.generateKey(
			{ name: AES_GCM_ALGORITHM, length: AES_GCM_KEY_LENGTH },
			false,
			AES_GCM_USAGES
		);
		pushStepResult(
			steps,
			createSuccessStep(
				'generate_wrapping_key',
				'Generate non-extractable AES-GCM key',
				describeKey(wrappingKey)
			)
		);
	} catch (error) {
		return {
			ok: false,
			steps: [
				createFailureStep(
					'generate_wrapping_key',
					'Generate non-extractable AES-GCM key',
					formatError(error)
				)
			]
		};
	}

	try {
		await putKeyOnly(db, keyId, wrappingKey);
		pushStepResult(
			steps,
			createSuccessStep(
				'put_wrapping_key_only',
				'Store AES-GCM key as direct value',
				'IndexedDB put succeeded'
			)
		);
	} catch (error) {
		probeOk = false;
		pushStepResult(
			steps,
			createFailureStep(
				'put_wrapping_key_only',
				'Store AES-GCM key as direct value',
				formatError(error)
			)
		);
	}

	let loadedWrappingKey: CryptoKey | null = null;
	if (steps.every((step) => step.id !== 'put_wrapping_key_only' || step.ok)) {
		try {
			loadedWrappingKey = await getKeyOnly(db, keyId);
			pushStepResult(
				steps,
				createSuccessStep(
					'get_wrapping_key_only',
					'Load AES-GCM key as direct value',
					describeKey(loadedWrappingKey)
				)
			);
		} catch (error) {
			probeOk = false;
			pushStepResult(
				steps,
				createFailureStep(
					'get_wrapping_key_only',
					'Load AES-GCM key as direct value',
					formatError(error)
				)
			);
		}
	}

	if (!loadedWrappingKey) {
		pushStepResult(
			steps,
			createFailureStep(
				'unwrap_wrapped_key',
				'Unwrap X25519 key with persisted AES-GCM key',
				'No persisted AES-GCM key available'
			)
		);
		return { ok: false, steps };
	}

	try {
		const x25519KeyPair = (await crypto.subtle.generateKey({ name: X25519_ALGORITHM }, true, [
			DERIVE_BITS_USAGE
		])) as CryptoKeyPair;
		const wrapIv = createProbeIv();
		const wrappedKey = await crypto.subtle.wrapKey(
			'pkcs8',
			x25519KeyPair.privateKey,
			loadedWrappingKey,
			{ name: AES_GCM_ALGORITHM, iv: wrapIv }
		);
		const unwrappedKey = await crypto.subtle.unwrapKey(
			'pkcs8',
			wrappedKey,
			loadedWrappingKey,
			{ name: AES_GCM_ALGORITHM, iv: wrapIv },
			{ name: X25519_ALGORITHM },
			false,
			[DERIVE_BITS_USAGE]
		);
		const recipient = await verifyRecipient(unwrappedKey);
		pushStepResult(
			steps,
			createSuccessStep(
				'unwrap_wrapped_key',
				'Unwrap X25519 key with persisted AES-GCM key',
				`${describeKey(unwrappedKey)} recipient=${recipient}`
			)
		);
	} catch (error) {
		probeOk = false;
		pushStepResult(
			steps,
			createFailureStep(
				'unwrap_wrapped_key',
				'Unwrap X25519 key with persisted AES-GCM key',
				formatError(error)
			)
		);
	}

	try {
		const wrapApiKey = await crypto.subtle.generateKey(
			{ name: AES_GCM_ALGORITHM, length: AES_GCM_KEY_LENGTH },
			false,
			AES_GCM_WRAP_USAGES
		);
		pushStepResult(
			steps,
			createSuccessStep(
				'generate_wrap_api_key',
				'Generate AES-GCM key for wrapKey/unwrapKey',
				describeKey(wrapApiKey)
			)
		);

		const x25519KeyPair = (await crypto.subtle.generateKey({ name: X25519_ALGORITHM }, true, [
			DERIVE_BITS_USAGE
		])) as CryptoKeyPair;
		const wrapIv = createProbeIv();
		const wrappedKey = await crypto.subtle.wrapKey('pkcs8', x25519KeyPair.privateKey, wrapApiKey, {
			name: AES_GCM_ALGORITHM,
			iv: wrapIv
		});
		pushStepResult(
			steps,
			createSuccessStep(
				'wrap_key',
				'Wrap X25519 key with fresh AES-GCM key',
				`Wrapped ${wrappedKey.byteLength} bytes`
			)
		);

		const unwrappedKey = await crypto.subtle.unwrapKey(
			'pkcs8',
			wrappedKey,
			wrapApiKey,
			{ name: AES_GCM_ALGORITHM, iv: wrapIv },
			{ name: X25519_ALGORITHM },
			false,
			[DERIVE_BITS_USAGE]
		);
		const recipient = await verifyRecipient(unwrappedKey);
		pushStepResult(
			steps,
			createSuccessStep(
				'unwrap_key',
				'Unwrap X25519 key as non-extractable',
				`${describeKey(unwrappedKey)} recipient=${recipient}`
			)
		);
	} catch (error) {
		probeOk = false;
		pushStepResult(
			steps,
			createFailureStep(
				'unwrap_key',
				'Wrap/unwrap X25519 key as non-extractable',
				formatError(error)
			)
		);
	}

	return { ok: probeOk, steps };
};

/** Runs the full IndexedDB probe used to diagnose Safari CryptoKey persistence issues. */
export const runEncryptionProbe = async (): Promise<EncryptionProbeResult> => {
	const structuredClone = await runStructuredCloneProbe();
	const db = await openProbeDB();

	try {
		const scenarios: EncryptionProbeScenarioResult[] = [];

		for (const scenario of PROBE_SCENARIOS) {
			scenarios.push(await runScenario(db, scenario));
		}

		const keyPersistence: KeyPersistenceProbeResult = {
			ok: structuredClone.ok && scenarios.every((scenario) => scenario.ok),
			structuredClone,
			scenarios
		};

		const wrappingKey = await runWrappingKeyProbe(db);

		return {
			runAt: new Date().toISOString(),
			userAgent: navigator.userAgent,
			keyPersistence,
			wrappingKey
		};
	} catch (error) {
		logger.error('Encryption probe failed unexpectedly', { error: String(error) });
		throw error;
	} finally {
		db.close();
	}
};
