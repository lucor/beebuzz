# BeeBuzz

Simple, private push notifications for servers, scripts, apps, and webhooks.

BeeBuzz is a focused Web Push delivery system for alerts that should reach your own
paired devices without becoming another chat surface. It supports both fast
server-trusted notifications and real end-to-end encrypted delivery, where message
content is encrypted before it reaches BeeBuzz and the server stores ciphertext
instead of plaintext.

## Why BeeBuzz

- **Private alerting**: send high-signal machine-to-person notifications to paired devices.
- **Two delivery modes**: start quickly with trusted delivery, or use E2E mode when content privacy matters.
- **Real E2E push flow**: in E2E mode, the CLI encrypts locally for paired device keys and Hive decrypts locally on the receiving device.
- **Small auditable stack**: Go, SQLite, SvelteKit, Web Push, and a Hive PWA receiver.
- **Focused scope**: BeeBuzz is not a team chat, inbox, or general messaging platform.

## Architecture

BeeBuzz is split into a few small pieces:

- **Server**: Go + SQLite API for accounts, topics, API tokens, devices, attachments, and Web Push dispatch.
- **Site**: SvelteKit web app for sign-in, device pairing, API tokens, webhook setup, and administration.
- **Hive**: PWA receiver that handles Web Push, stores pairing state locally, and decrypts E2E notifications on-device.
- **CLI**: sender for end-to-end encrypted notifications from terminals, scripts, and automation.

## Delivery Modes

### Server-trusted

Use JSON or multipart requests when the sender trusts the BeeBuzz server with the
notification payload.

```text
sender -> BeeBuzz API -> Web Push -> Hive
```

BeeBuzz authenticates the API token, reads and validates the payload, optionally
handles an attachment, then sends a Web Push notification to subscribed devices.
This is the fastest path for tests, simple integrations, and webhooks.

### End-to-end encrypted

Use the CLI or an `application/octet-stream` request when notification content
should stay opaque to BeeBuzz.

```text
CLI -> encrypt locally for paired devices -> BeeBuzz stores ciphertext -> Hive fetches and decrypts locally
```

The CLI fetches paired device public keys, encrypts the notification locally with
age/X25519, and sends only ciphertext to BeeBuzz. The server stores the opaque
blob temporarily and pushes a small envelope containing the notification ID,
attachment token, and server acceptance time. Hive receives the envelope, fetches
the blob, and decrypts the final notification locally.

## Try It

- Read the docs: <https://beebuzz.app/docs>
- Use the hosted BeeBuzz beta: <https://beebuzz.app/docs/quickstart>
- Run BeeBuzz locally for development: <https://beebuzz.app/docs/local-dev>

Install the CLI from a [GitHub release](https://github.com/lucor/beebuzz/releases) (no Go required) or with Go:

```bash
go install lucor.dev/beebuzz/cmd/beebuzz@latest
```

Send an encrypted notification after connecting the CLI:

```bash
beebuzz send "Hello from BeeBuzz"
```

## Security Model

In E2E mode:

- BeeBuzz should not recover notification plaintext from stored blobs alone.
- BeeBuzz stores paired device public recipients, not device private identities.
- A database compromise alone should not reveal stored E2E message plaintext or device private keys.

E2E protects message content, not metadata. BeeBuzz still sees operational metadata
such as users, topics, device mappings, timestamps, delivery results, and whether
E2E mode was used. It also does not protect against a compromised endpoint or an
actively malicious server serving malicious client code or replacing recipient keys.

See [docs/E2E_ENCRYPTION.md](docs/E2E_ENCRYPTION.md) and
[docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) for the full model.

## Project Status

BeeBuzz is currently optimized for two workflows:

1. get approved for the hosted beta and send your first notification in seconds
2. run the stack locally with a fast development loop

Detailed production self-hosting docs will come later.

Hosted access is free during beta. After beta, the hosted service will move to a
single paid plan, priced to keep the project sustainable. Self-hosting remains
free, open source, and available under the AGPL license.

## License

BeeBuzz is licensed under the GNU Affero General Public License v3.0 only.
