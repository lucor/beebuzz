# E2E Encryption Architecture

This document describes the end-to-end encryption model currently implemented in BeeBuzz.

## How To Use This Doc

- Canonical for the E2E key model, encryption flow, BeeBuzz envelope, and client key storage strategy.
- Read this when changing pairing, key storage, CLI encryption, Hive decryption, or any claim about what BeeBuzz can and cannot see in E2E mode.
- Update this doc when the key lifecycle, ciphertext flow, envelope shape, or stated security properties change.

See [MESSAGING.md](MESSAGING.md) for the full messaging system overview (endpoints, request formats, attachments, and limits). This document focuses on the encryption model used in E2E mode.

## Key Model

Each paired device has an age X25519 keypair:

- the **recipient** (public key) is sent to BeeBuzz during pairing and stored in `push_subscriptions.age_recipient`
- the **identity** (private key) never leaves the device and is stored locally by Hive

BeeBuzz stores only public recipients.

## Current E2E Flow

### Pairing

Hive generates a local age X25519 keypair and sends the public recipient to BeeBuzz during device pairing. The private identity never leaves the device. See [HIVE_ONBOARDING.md](HIVE_ONBOARDING.md) for the full client-side pairing flow.

### Sending

1. The BeeBuzz CLI fetches current recipients from `GET /v1/push/keys`
2. The CLI builds a plaintext JSON payload locally
3. The CLI encrypts that payload with age for all cached recipients
4. The CLI sends the ciphertext to `POST /v1/push/{topic}` with `Content-Type: application/octet-stream`
5. BeeBuzz stores the opaque ciphertext as a temporary attachment blob
6. BeeBuzz creates a tiny JSON envelope containing only the stored attachment token
7. BeeBuzz delivers that JSON envelope through Web Push

The CLI treats the configured BeeBuzz API host as trusted for an existing profile. A send-time `--api-url` override may adjust path details on the same host, but it cannot silently switch a saved profile to a different host; changing servers requires an explicit `beebuzz connect`.

### Receiving

1. Hive receives the JSON push envelope in the service worker
2. Hive extracts the attachment token from the envelope
3. Hive fetches the stored blob from `GET /v1/attachments/{token}`
4. Hive decrypts the blob locally
5. Hive renders the final notification content

## BeeBuzz Envelope

After storing the opaque ciphertext, BeeBuzz creates this JSON envelope for Web Push delivery:

```json
{
  "beebuzz": {
    "id": "uuid-v7",
    "token": "attachment-token",
    "sent_at": "2026-04-17T12:00:00Z"
  }
}
```

The envelope contains:

- `id`: Unique notification identifier (UUID v7)
- `token`: Attachment token to fetch the encrypted blob
- `sent_at`: **Server-authored timestamp** representing when the notification was accepted

The server can route and store this envelope, but it does not know the original encrypted payload contents. The `sent_at` field provides the authoritative notification time and travels **outside** the encrypted blob by design.

## CLI Payload

Before age encryption, the CLI constructs a plaintext payload like:

```json
{
  "title": "Build failed",
  "body": "Deployment exited with code 1",
  "topic": "alerts",
  "attachment": {
    "data": "<base64>",
    "mime": "image/png",
    "filename": "image.png"
  }
}
```

That JSON is encrypted locally with age and sent as raw `application/octet-stream`.

## Server Responsibilities in E2E Mode

In E2E mode BeeBuzz still:

- authenticates API tokens
- resolves the target topic and paired devices
- stores the opaque encrypted blob temporarily
- creates the minimal token envelope
- encrypts the envelope per device
- sends Web Push notifications

BeeBuzz does **not** inspect or decrypt the E2E request body.

## Security Properties

E2E mode provides these properties:

- BeeBuzz cannot read the original encrypted notification body
- BeeBuzz stores only public recipients, not private identities
- a DB compromise does not reveal device private keys

It does **not** eliminate all trust:

- a compromised server could still serve malicious frontend code
- a compromised device/browser can still expose the local identity at runtime

## Client Key Storage

Browser support is not only about whether `X25519` exists. Hive also has to persist the device identity reliably across browser restarts.

### Current storage strategy

Hive uses a single wrapped-key flow on every supported browser:

1. generate a non-extractable `AES-GCM` wrapping key
2. generate an `X25519` private key for age
3. wrap the `X25519` private key once with Web Crypto `wrapKey('pkcs8', ...)`
4. bind the wrapped bytes to the stored recipient with AES-GCM authenticated metadata
5. store in IndexedDB:
   - metadata
   - the wrapping key as `CryptoKey`
   - the wrapped private-key payload + IV
6. when decryption is needed, unwrap the stored payload back into an `extractable: false` `X25519` key and verify that it still derives the stored recipient before use

### Why BeeBuzz does this

Direct persistence of `X25519 CryptoKey` values in IndexedDB was not reliable on affected Safari/WebKit builds.

Observed behavior:

- `X25519` key generation works
- `age.identityToRecipient()` works
- direct IndexedDB `put()` may succeed
- later reads can return `null`
- nested-record storage for `X25519` can also be unreliable

The wrapped-key approach avoids Safari-specific code paths while keeping one storage model across browsers. Hive still treats local key storage as recoverable browser state rather than durable backup material: if browser storage is evicted, the device must re-pair.

### Tradeoff

This is not the same as persisting the `X25519` key directly as a non-extractable `CryptoKey`, because Hive still relies on a browser-managed wrapping key and stored wrapped bytes instead of persisting the `X25519` `CryptoKey` object directly.

The practical security model is still acceptable for Hive:

- the stored private material is encrypted with a browser-managed non-extractable wrapping key
- the runtime key used by age is always unwrapped as `extractable: false`
- the re-imported identity is checked against the stored recipient before Hive uses it

## Current Limitations

- Hive currently uses the first available local identity for decryption
- encrypted blobs are temporary, not permanent
- BeeBuzz still stores metadata such as topics, device recipients, and attachment records

## Related Documents

- [MESSAGING.md](MESSAGING.md)
- [HIVE_ONBOARDING.md](HIVE_ONBOARDING.md)

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- change the age keypair generation, storage, or wrapped-key flow in Hive encryption services
- change the CLI encryption or key-fetching flow in `cmd/beebuzz/`
- change the BeeBuzz envelope shape or fields
- change how the server handles E2E `application/octet-stream` requests
- change the service worker decryption or blob-fetch flow
