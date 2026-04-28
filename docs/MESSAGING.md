# Messaging System

This document describes the BeeBuzz messaging system as it exists today.

## How To Use This Doc

- Canonical for messaging behavior, delivery-mode semantics, attachment policy, and timestamp rules.
- Read this when changing `/v1/push`, webhook dispatch into notifications, CLI send behavior, Hive receive semantics, or attachment retention.
- Update this doc when delivery flow, payload semantics, limits, or retention policy change.

Use [openapi.yaml](openapi.yaml) for exact request and response schemas. Use [E2E_ENCRYPTION.md](E2E_ENCRYPTION.md) for the cryptographic flow and trust model behind E2E delivery.

## Overview

BeeBuzz supports two delivery modes:

- **Server-trusted**: the server can inspect the notification payload and optionally process attachments.
- **End-to-end (E2E)**: the server stores an opaque encrypted blob and sends a minimal push envelope containing only the notification ID, attachment token, and `sent_at`.

**Timestamp semantics**: All notifications use a single timestamp `sent_at` which represents the **server acceptance time**. This is the only user-facing timestamp and is always authored by the server. UUID v7 is used for ID generation only and is **not** a UI time source. Client-provided event time is not supported.

Push delivery is topic-based. If no topic is provided, BeeBuzz uses the default topic `general`.

Notification analytics use stable source labels:

- `api`: direct Push API requests
- `cli`: BeeBuzz CLI sends
- `webhook`: webhook-triggered sends
- `internal`: BeeBuzz system-generated notifications, such as admin alerts for new signups

## Active Endpoints

### Push API

- `POST /v1/push`
- `POST /v1/push/{topic}`

Authentication uses a Bearer API token.

Accepted request formats:

- `application/json`
- `multipart/form-data`
- `application/octet-stream`

### Device key discovery

- `GET /v1/push/keys`

Returns the current age public keys for all paired devices of the authenticated user.

### Attachment retrieval

- `GET /v1/attachments/{token}`

Returns the stored encrypted attachment blob for a previously issued attachment token.
Access remains token-based, but retrieval is rate-limited per attachment token to reduce abuse if a token leaks.

## Delivery Modes

### System-generated notifications

BeeBuzz can generate server-trusted notifications for operational platform events. System-generated notifications are configured by admins and are dispatched internally through the notification service layer; they do not use stored API tokens or a loopback HTTP call to `/v1/push`.

### 1. Server-trusted delivery

Server-trusted delivery is used for JSON and multipart requests.

The server:

1. Authenticates the API token for the requested topic.
2. Resolves the subscribed devices for that topic.
3. Builds a JSON notification payload with:
   - `id`
   - `title`
   - `body`
   - `topic_id`
   - `topic`
   - `priority`
   - `sent_at`
   - optional `attachment`
4. Sends that JSON payload through Web Push to each subscribed device.

Current response shape:

```json
{
  "status": "partial",
  "sent_count": 2,
  "total_count": 3,
  "failed_count": 1
}
```

Response semantics:

- `200 OK` with `status: delivered` when every device send succeeds
- `200 OK` with `status: partial` when at least one device send succeeds and at least one fails
- `502 Bad Gateway` with code `push_delivery_failed` when all device sends fail

### 2. End-to-end delivery

E2E delivery is used for `application/octet-stream` requests and by the BeeBuzz CLI.
The server does not inspect the encrypted payload. It stores the opaque blob as an attachment and pushes a compact JSON envelope that points the Hive service worker at the stored token. See [E2E_ENCRYPTION.md](E2E_ENCRYPTION.md) for the full encryption flow, CLI plaintext payload, and key management details.

When the request uses E2E mode, the push response also includes the current device keys so the CLI can keep its local cache synchronized:

```json
{
  "status": "delivered",
  "sent_count": 2,
  "total_count": 2,
  "failed_count": 0,
  "device_keys": [
    {
      "device_id": "dev_123",
      "device_name": "Luca iPhone",
      "paired_at": "2026-04-11T09:30:00Z",
      "age_recipient": "age1...",
      "age_recipient_fingerprint": "7c4f2c9c8d1a4e32"
    }
  ]
}
```

## Request Formats

### JSON (`application/json`)

Used for standard server-trusted notifications.

Example:

```json
{
  "title": "Build failed",
  "body": "Deployment exited with code 1",
  "priority": "high",
  "attachment_url": "https://example.com/image.png"
}
```

Rules:

- `title` is required and must be non-blank
- `title` max length is `64` characters
- `body` is optional and max length is `256` characters
- `priority` may be empty, `normal`, or `high`
- `attachment_url` is optional and must be HTTPS when provided

If `attachment_url` is present, the server downloads the resource with HTTPS-only SSRF protections, enforces the attachment size limit, encrypts it for the paired devices, stores it, and adds an attachment token to the push payload.

### Multipart (`multipart/form-data`)

Used for server-trusted notifications with uploaded file attachments.

Fields:

- `title`
- `body`
- `priority`
- `attachment` file part

Rules:

- the same validation as JSON applies to `title`, `body`, and `priority`
- the uploaded file is MIME-sniffed server-side
- the multipart body is size-limited to `1.5 MB`

The uploaded file is encrypted for the paired devices, stored, and referenced from the push payload by token.

### Octet-stream (`application/octet-stream`)

Used for zero-knowledge E2E notifications.

Rules:

- the body is treated as an opaque encrypted blob
- BeeBuzz does not parse or inspect it
- `X-Priority` may be empty, `normal`, or `high`
- the body must be non-empty
- the body size is limited to the same attachment size cap used elsewhere

This mode is intended for clients that already know the device age public keys and can encrypt locally.

## Attachments

Attachments are always stored as temporary encrypted blobs.

Current attachment properties:

- plaintext size limit: `1 MB`
- retention: `6 hours`
- retrieval: opaque token via `/v1/attachments/{token}`
- storage model: encrypted blob only, fetched on demand by token

The 6-hour retention window is intentional:

- BeeBuzz is optimized for real-time alerts, where a delayed attachment quickly becomes stale
- shorter retention reduces exposure if an attachment token is leaked
- it also keeps storage pressure bounded under sustained notification volume

The current push flow uses immediate delivery rather than long offline queueing, so extending
attachment lifetime beyond a few hours would not materially improve delivery reliability.

In server-trusted mode:

- BeeBuzz reads or downloads the plaintext attachment
- encrypts it for the paired device age recipients
- stores the ciphertext
- includes only token and metadata in the push payload

If no subscribed device has an age recipient, BeeBuzz skips attachment generation and still sends the notification without an attachment payload.

Current attachment payload shape:

```json
{
  "token": "attachment-token",
  "mime": "image/png",
  "filename": "image.png"
}
```

In E2E mode:

- the original request body is already encrypted client-side
- BeeBuzz stores that opaque blob without inspecting it
- the push payload sent to devices is a small JSON envelope with `id`, `token`, and `sent_at`

On the Hive side, the service worker:

- accepts either a plain JSON notification payload or an E2E envelope
- fetches `/v1/attachments/{token}` for E2E envelopes
- decrypts the fetched blob locally
- persists the resolved notification to IndexedDB before showing it

## BeeBuzz CLI

The CLI in `cmd/beebuzz` uses E2E delivery. See [E2E_ENCRYPTION.md](E2E_ENCRYPTION.md) for the full sending flow and plaintext payload shape.

## Webhook Integration

Webhooks are a separate ingestion path that eventually dispatch into the notification service.

Current webhook payload modes:

- `beebuzz`: expects a BeeBuzz-shaped JSON body with `title` and `body`
- `custom`: extracts `title` and `body` using simple dot-separated JSON paths with optional numeric array indexes, for example `event.items[0].title`

Webhook priority is configured on the webhook itself and currently accepts `normal` or `high`, defaulting to `normal`.
Both modes end up producing a standard server-trusted notification send operation.

**Webhook timestamp semantics**: Webhooks do **not** define the notification timestamp. The `sent_at` value is always generated server-side when the notification is accepted and dispatched.

## Failure Handling

During push delivery:

- invalid subscriptions (`404` / `410`) are removed
- when a provider reports `404` / `410`, BeeBuzz also marks the device pairing state as `subscription_gone` so clients can surface reconnect-required UX
- attachment processing failures fail the request with `422 attachment_processing_failed`
- E2E blob storage or envelope encryption failures fail the request
- requests larger than the attachment/body caps fail with `413 Payload Too Large`

## Current Limits and Notes

- topic default: `general`
- priority values: `normal`, `high`
- JSON and multipart title max length: `64`
- JSON and multipart body max length: `256`
- webhook receive body max size: `64 KB`
- attachment plaintext size limit: `1 MB`
- multipart request body limit: `1.5 MB`
- attachment retention: `6 hours`
- push TTL for undelivered notifications: `6 hours`
- attachment retrieval is token-based and does not require a session cookie
- E2E mode depends on paired device age public keys

## Push-Stub Mode (Development Only)

When `BEEBUZZ_PUSH_STUB` is enabled (non-production environments only), the server does not dispatch notifications through real Web Push providers. Instead, it captures the raw push payload in an in-memory broker and exposes it via a long-polling endpoint:

- `GET /_stub/push/next`

This endpoint is restricted to loopback clients as defense-in-depth. It returns `200 OK` with a `PushStubEvent` when a payload is available, or `204 No Content` after a short timeout so clients can retry. Test drivers use this flow to inject pushes directly into the Hive service worker via Chrome DevTools Protocol, bypassing FCM/VAPID entirely.

**Never enable push-stub in production.**

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- change `/v1/push` or `/v1/push/{topic}` request handling, validation, or response shape
- change `/v1/attachments/{token}` behavior or retention policy
- change attachment size limits, retention, or storage model
- change webhook payload modes, dispatch, or priority handling
- change push delivery failure handling or subscription cleanup
- change the Hive service worker notification receive or decrypt flow
- change push-stub capture behavior, broker limits, or the `/_stub/push/next` endpoint
