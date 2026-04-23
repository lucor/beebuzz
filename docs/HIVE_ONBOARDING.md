# BeeBuzz Hive Onboarding

This document describes the current onboarding flow used by the Hive PWA.

## How To Use This Doc

- Canonical for the Hive onboarding state machine, install policy, pairing sequence, and recovery behavior.
- Read this when changing onboarding UX, browser support gating, install requirements, or pairing flow in the Hive app.
- Update this doc when onboarding states, install rules, platform support policy, or recovery behavior change.

## Goal

Hive needs to bring a device from "fresh browser" to "paired and ready to receive encrypted push notifications" with a flow that works across:

- iOS/iPadOS WebKit
- Safari on macOS
- Chromium browsers
- Firefox desktop

## Current State Model

Hive uses this onboarding state machine:

```text
checking
  -> unsupported
  -> not-installed
  -> ready
  -> permission-prompt
  -> permission-blocked
  -> pairing
  -> paired
  -> error
```

The implementation lives in `web/apps/hive/src/lib/onboarding.svelte.ts`.

## High-Level Flow

Hive separates onboarding into three user-visible steps:

1. install the app when required or strongly preferred
2. grant notification permission
3. submit the pairing code

Key generation, push subscription creation, and backend pairing happen automatically after permission is granted.

## Capability Detection

Hive checks these capabilities during onboarding initialization:

- secure context
- Service Worker support
- X25519 + age support

Hive also checks Push API and Notification API support before allowing a device to reach the pairing step.

Important detail:

- on iOS/iPadOS WebKit in-browser, Push and Notification APIs are not available until the app is launched in standalone mode
- because of that, Hive allows the install screen when base support exists (`secure` + Service Worker + encryption), then re-runs capability checks after standalone launch

Capability detection lives in `web/apps/hive/src/lib/services/capability.ts`.

## Unsupported Browsers

| Browser          | Reason                                         |
| ---------------- | ---------------------------------------------- |
| Safari iOS < 17  | No X25519 Web Crypto (push works from 16.4)    |
| Safari < 17      | No X25519 Web Crypto                           |
| Chrome < 133     | No X25519 Web Crypto                           |
| Edge < 133       | No X25519 Web Crypto                           |
| Firefox < 130    | No X25519 Web Crypto                           |
| Opera Mini       | No Push API                                    |
| IE               | No Push API, no X25519                         |

## Install Policy

Hive is install-first where that improves or is required for push delivery.

| Platform | Current behavior |
|---|---|
| iOS / iPadOS WebKit | install strongly preferred before pairing; a "Can't install? Continue in browser" fallback is available but push APIs may not work in-browser on iOS |
| Safari macOS | install-first flow with Safari-specific Add to Dock guidance, but browser fallback is still available |
| Chromium browsers | install-first flow; uses `beforeinstallprompt` when available and otherwise shows manual install guidance |
| Firefox desktop | browser-mode fallback, no native install support |
| Previously installed sessions | show "already installed" guidance first, with a browser fallback escape hatch |
| Unsupported browsers | blocked at capability check |

Important detail:

- on iOS WebKit, Push and Notification APIs are only available in standalone mode
- Firefox is allowed through a browser-mode fallback because install is not available there
- Safari macOS, Chromium fallback, and previously installed sessions all expose a "Continue in browser" path even though install remains the preferred route

Install lifecycle helpers live in `web/apps/hive/src/lib/services/install.ts`.

## Pairing Flow

Once the device is ready to pair, Hive performs this sequence:

1. request notification permission
2. wait for the existing service worker registration created during onboarding init
3. fetch the VAPID public key
4. create or reuse the push subscription
5. generate and locally store the age keypair
6. send the pairing request to `POST /v1/pairing`
7. persist the returned backend `device_id` and `device_token` locally
8. rename the locally stored pending key to the server-issued device ID

Relevant files:

- `web/apps/hive/src/lib/services/push.ts`
- `web/apps/hive/src/lib/services/encryption.ts`

## Recovery and Degradation Handling

On startup, Hive checks whether an apparently paired device is still healthy:

- if notification permission was revoked, Hive forces re-pair
- if the push subscription disappeared, Hive forces re-pair
- if the local wrapped identity is missing or no longer reloads to the stored recipient, Hive forces re-pair before waiting for the next push to fail
- if BeeBuzz reports the device as `subscription_gone`, `unpaired`, or `pending`, Hive keeps the app shell available but marks the device as reconnect-required with a reason-specific banner
- if the local installation predates stored `device_token` credentials, Hive treats that as reconnect-required so the new pairing-status protection can be enabled

The health check lives in `reconcilePushState()` inside `web/apps/hive/src/lib/onboarding.svelte.ts`.
It is used both during onboarding init and again when the paired app shell boots from `web/apps/hive/src/routes/(app)/+layout.svelte`.

## Testing & Diagnostics

The debug route `/debug` exists to verify browser behavior around:

- X25519 generation
- recipient derivation
- IndexedDB persistence behavior
- wrapped-key persistence and recovery
- service worker state and push subscription diagnostics

The route is only exposed in debug builds when `VITE_BEEBUZZ_DEBUG === true`.
Keep that page available in debug builds so Safari/WebKit behavior can be re-checked after browser updates.

Supporting diagnostic services:

- `web/apps/hive/src/lib/services/debug-diagnostics.ts`
- `web/apps/hive/src/lib/services/encryption-diagnostics.ts`

## Why the Flow Is Split

Hive does not ask for install, permission, and pairing code in one step because that creates unnecessary drop-off and makes failures harder to explain.

The current sequence keeps failure handling clearer:

- install problems stay in the install step
- permission refusal stays in the permission step
- pairing/API problems stay in the pairing step

## Main Implementation Files

- `web/apps/hive/src/lib/onboarding.svelte.ts` — state machine
- `web/apps/hive/src/lib/services/capability.ts` — browser capability detection
- `web/apps/hive/src/lib/services/install.ts` — install lifecycle helpers
- `web/apps/hive/src/lib/services/push.ts` — push subscription and pairing API
- `web/apps/hive/src/lib/services/encryption.ts` — age keypair generation and storage
- `web/apps/hive/src/lib/services/device-keys-repository.ts` — wrapped key material plus persisted device credentials
- `web/apps/hive/src/lib/services/pairing-state.ts` — paired-state persistence
- `web/apps/hive/src/lib/services/startup-recovery.ts` — stale pairing state cleanup
- `web/apps/hive/src/lib/services/startup-error.ts` — startup error formatting
- `web/apps/hive/src/routes/pair/+page.svelte` — onboarding route entrypoint

## Technical Notes

- **Service Workers**: `crypto.subtle` is available in worker contexts.
- Browser support changes quickly. Re-check the support matrix and `/debug` before changing capability gates or install policy.

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- add, remove, or rename onboarding states in `onboarding.svelte.ts`
- change capability checks in `capability.ts`
- change install policy or `skipInstall` behavior in `install.ts` or `onboarding.svelte.ts`
- change the pairing flow in `push.ts` or `encryption.ts`
- add or remove files under `web/apps/hive/src/lib/services/` that affect onboarding, pairing, or diagnostics
