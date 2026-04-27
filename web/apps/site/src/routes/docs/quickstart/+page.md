# Quickstart

This guide is for approved BeeBuzz beta users.

Goal: go from account access to your first delivered notification with as little friction as possible.

## What You'll Do

1. sign in to BeeBuzz
2. pair one device
3. create one API token
4. choose **one** sending mode:
   - **Trusted mode**: send over HTTP directly, or use a webhook URL for external services (no CLI required)
   - **End-to-end mode**: install the BeeBuzz CLI and send encrypted notifications

## 1. Sign In

Open [beebuzz.app](https://beebuzz.app) and click **Request beta access** to sign in or request access.

If access is enabled for your email:

1. enter your email
2. submit the form
3. check your inbox for the verification code
4. complete sign-in

After verification, open your account dashboard.

## 2. Pair Your First Device

Open **Account** -> **Devices** and click **Add Device**.

BeeBuzz gives you:

- a Hive link / QR code
- a pairing code

Then:

1. open [Hive](https://hive.beebuzz.app) on the device you want to receive notifications on
2. install the PWA if prompted
3. grant notification permission
4. enter the pairing code

Once pairing completes, the device appears in **Devices**.

> **Recommended:** install Hive as a PWA. It gives the client a more stable app-like runtime, makes notification behavior more predictable, and reduces confusion around tabs, background state, and permissions.
> On iPhone and iPad, installation is required for Web Push.
> See [Browser Support](/docs/browser-support) for the platform details.

## 3. Create An API Token

Open **Account** -> **API Tokens** and create a token.

Copy it immediately. BeeBuzz shows the raw token only once.

If you have only the default topic, the token can stay scoped to that.

This same API token is used by both:

- trusted-mode HTTP requests
- the BeeBuzz CLI when you connect it for end-to-end sending

## Choose Your Sending Mode

At this point you have:

- one paired device
- one API token

BeeBuzz has **two** sending modes. The main choice is whether BeeBuzz can read the notification content.

| Mode | Best for | How you send | What BeeBuzz can see |
|---|---|---|---|
| Trusted mode (server-trusted) | Fastest first test, apps/scripts, and external-service integrations | Direct Push API with a bearer token, or a [webhook URL](/docs/webhooks) for external services | Plaintext notification content |
| End-to-end mode | Encrypted delivery from sender to device | BeeBuzz CLI (`application/octet-stream`) | Ciphertext only; BeeBuzz only sees routing/storage metadata |

> Need a stable URL for Home Assistant, CI, or another service that can POST to HTTPS but does not fit direct API auth well? Use [Webhooks](/docs/webhooks). Webhooks are a **trusted-mode integration**, not a separate privacy mode.

## Option A — Trusted Mode

Fastest path. Good for testing BeeBuzz immediately from any HTTP client.

> **No CLI required for trusted mode.**

```bash
curl https://push.beebuzz.app \
  -H "Authorization: Bearer $TOKEN" \
  -F title="Hello from BeeBuzz" \
  -F body="Trusted mode test"
```

In trusted mode, BeeBuzz can read the notification content and sends it via standard Web Push transport encryption to the devices subscribed to that topic.

### Trusted Mode Variants

You can use trusted mode in two ways:

- **Direct Push API** — best for `curl`, scripts, and apps that can send a bearer token
- **[Webhooks](/docs/webhooks)** — best for external services that need a stable URL with preconfigured topic, priority, or payload mapping

Webhooks use the same trust model and delivery semantics as trusted mode: BeeBuzz receives plaintext notification content and dispatches a standard server-trusted notification.

## Option B — End-To-End Mode

Recommended when you want encrypted delivery to your devices.

In E2E mode, the BeeBuzz CLI fetches your paired-device public keys, encrypts the payload locally, and uploads ciphertext as `application/octet-stream`. BeeBuzz can route and store it, but does not read the plaintext message body.

### Install The CLI

The easiest install path for most beta users is the published CLI release:

1. open [BeeBuzz releases](https://github.com/lucor/beebuzz/releases)
2. download the archive for your OS and CPU architecture
3. unpack it and put the `beebuzz` binary somewhere in your `PATH`
4. run `beebuzz version`

BeeBuzz currently publishes CLI binaries for macOS and Linux on `amd64` and `arm64`.

If you already have Go installed, you can also install the CLI from source:

```bash
go install lucor.dev/beebuzz/cmd/beebuzz@latest
```

### Connect The CLI

Run:

```bash
beebuzz connect
```

Paste your API token when prompted.

For the hosted beta, `connect` already defaults to the BeeBuzz API. You do not need to enter a custom URL unless you are targeting another instance.

### Send Your First Encrypted Message

```bash
beebuzz send "Hello from BeeBuzz"
```

The CLI encrypts the payload before sending it.

If everything is set up correctly, the notification should arrive on your paired device immediately.

## If Nothing Arrives

### Check These First

These checks apply to both modes:

1. the device is listed as paired in **Account** -> **Devices**
2. the Hive app is installed where BeeBuzz recommends it
3. notification permission is still granted
4. the API token is allowed to send to the topic you are using
5. the device is subscribed to that topic

### Trusted Mode Checks

If your trusted-mode HTTP request returns success but no notification arrives, send a minimal trusted-mode test to confirm that the API token, topic, and device routing work:

```bash
curl https://push.beebuzz.app \
  -H "Authorization: Bearer $TOKEN" \
  -F title="BeeBuzz test" \
  -F body="Trusted mode delivery check"
```

### End-To-End Mode Checks

If you are sending with the CLI:

- confirm the CLI is connected with the same account that owns the paired device
- if the CLI reports paired device key changes, review them carefully before sending again

If trusted mode works but CLI E2E does not, refresh the CLI device key cache:

```bash
beebuzz keys
beebuzz send "BeeBuzz encrypted test"
```

### Check Hive

Open Hive on the receiving device and check whether it asks you to reconnect.

Reconnect the device if:

- notification permission was revoked
- the browser removed the push subscription
- the local device key is missing
- BeeBuzz shows the device as no longer paired

On iPhone and iPad, open Hive from the installed app icon. Web Push is not available from a normal browser tab.

## Next Steps

- [Webhooks](/docs/webhooks) — trusted-mode integrations that need a stable URL
- [Browser Support](/docs/browser-support) — platform and PWA details
