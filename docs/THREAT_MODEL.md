# BeeBuzz Threat Model

This document defines the practical threat model for BeeBuzz as implemented today.

It focuses on current trust boundaries and security claims, not future roadmap work.

## How To Use This Doc

- Canonical for BeeBuzz security goals, attacker classes, trust boundaries, and claim boundaries.
- Read this when reviewing security-sensitive changes or making claims about E2E guarantees, server trust, metadata exposure, or key compromise.
- Update this doc when trust boundaries, attacker assumptions, or security claims change.

## Scope

This document covers:

- BeeBuzz API and attachment storage
- BeeBuzz CLI E2E sending flow
- Hive PWA pairing, key storage, and message decryption
- Web Push delivery of BeeBuzz notifications

This document does not try to model:

- compromise of the underlying OS or browser sandbox
- physical device theft after the device is already unlocked
- a malicious sender who is already authorized to send to the target topic

Implementation details for the current E2E flow live in [E2E_ENCRYPTION.md](E2E_ENCRYPTION.md) and [HIVE_ONBOARDING.md](HIVE_ONBOARDING.md).

## Security Goals

BeeBuzz has two delivery modes with different goals.

### Server-trusted mode

In server-trusted mode, BeeBuzz is allowed to inspect notification content.

Goals:

- authenticated delivery to the correct topic devices
- attachment protection at rest
- safe handling of temporary attachment access tokens

### E2E mode

In E2E mode, BeeBuzz should not need plaintext to deliver messages.

Goals:

- confidentiality of notification content against normal server-side access
- confidentiality of device private keys against server and DB compromise
- local-only decryption on paired Hive devices

Non-goal:

- protection against an actively malicious BeeBuzz server

## Assets

The main assets are:

- notification plaintext
- attachment plaintext
- device private keys
- device public recipients
- API bearer tokens
- push subscriptions
- attachment retrieval tokens
- metadata such as topic, device mapping, timing, and delivery results

## Trust Boundaries

The system has four main trust boundaries:

1. Sender environment
   The CLI or other sender constructs plaintext before encryption.

2. BeeBuzz server
   The server authenticates senders, resolves target devices, stores blobs, and dispatches Web Push.

3. Web Push provider
   Push services transport the final Web Push payloads.

4. Hive device
   The PWA and service worker fetch encrypted blobs, load the local key, and decrypt locally.

## Attacker Classes

### Passive server-side observer

Examples:

- operator with DB read access
- backup leak
- read-only storage compromise
- log aggregation compromise

Expected outcome today:

- should not recover E2E notification plaintext from stored blobs alone
- should not recover device private keys from the server or DB
- can still learn metadata such as user, topic, device count, timestamps, and whether E2E was used

### Active malicious or compromised server

Examples:

- attacker with write access to BeeBuzz app or API responses
- operator intentionally returning attacker-controlled recipients
- malicious frontend or service worker code served from the BeeBuzz origin

Expected outcome today:

- can break future E2E confidentiality by replacing or injecting recipients before sender encryption
- can exfiltrate plaintext or local secrets through malicious client code
- can deny, delay, replay, or drop messages

This is the main limitation of the current E2E model.

### Network attacker without TLS termination

Examples:

- passive ISP observer
- local network attacker

Expected outcome today:

- should not recover plaintext over the wire if TLS is correctly configured
- may observe traffic timing, sizes, and hostnames

### Compromised device

Examples:

- XSS in Hive origin
- malicious browser extension
- compromised browser runtime
- malware on the endpoint

Expected outcome today:

- can access plaintext at render or decryption time
- can potentially trigger use of the local private key
- may exfiltrate pairing state, attachment tokens, and notifications

E2E does not protect against a compromised endpoint.

### Attachment token leakage

Examples:

- token copied into logs
- token leaked via debugging tools
- token leaked by browser history or telemetry

Expected outcome today:

- leaked token allows retrieval of the encrypted blob until expiry
- leaked token alone should not reveal plaintext without the device private key in E2E mode

## Security Claims

Accurate current claims:

- BeeBuzz supports E2E delivery where the server stores opaque ciphertext instead of plaintext message bodies.
- A server DB compromise alone should not reveal device private keys or stored E2E message plaintext.
- BeeBuzz E2E protects content, not metadata.

Claims to avoid:

- BeeBuzz eliminates trust in the server.
- BeeBuzz protects against a malicious server.
- BeeBuzz provides secure-messenger-grade guarantees.

## Related Documents

- [E2E_ENCRYPTION.md](E2E_ENCRYPTION.md)
- [MESSAGING.md](MESSAGING.md)
- [HIVE_ONBOARDING.md](HIVE_ONBOARDING.md)

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- change trust boundaries (e.g., what the server can or cannot see in E2E mode)
- change security claims or E2E guarantees
- change key storage model or device key lifecycle
- change attachment token security properties or expiry behavior
