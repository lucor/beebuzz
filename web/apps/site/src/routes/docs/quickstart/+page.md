# Quickstart

This guide is for approved BeeBuzz beta users.

Goal: go from account access to your first delivered notification with as little friction as possible.

## What You'll Do

1. sign in to BeeBuzz
2. pair one device
3. create one API token
4. connect the CLI
5. send your first message

## 1. Get Started

Open [beebuzz.app](https://beebuzz.app) and click **Get Started**.

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

1. open Hive on the device you want to receive notifications on
2. install the PWA if prompted
3. grant notification permission
4. enter the pairing code

If you are using the hosted beta, Hive is available at [hive.beebuzz.app](https://hive.beebuzz.app).

Once pairing completes, the device appears in **Devices**.

## 3. Create An API Token

Open **Account** -> **API Tokens** and create a token.

Copy it immediately. BeeBuzz shows the raw token only once.

If you have only the default topic, the token can stay scoped to that.

## 4. Choose How To Send

At this point you have:

- one paired device
- one API token

Now choose the sending mode you want.

### Trusted Mode

Fastest path. Good for testing BeeBuzz immediately.

```bash
curl https://push.beebuzz.app \
  -H "Authorization: Bearer $TOKEN" \
  -F title="Hello from BeeBuzz" \
  -F body="Trusted mode test"
```

In trusted mode, BeeBuzz can inspect the plaintext notification payload before delivery.

### End-To-End Mode

Recommended when you want encrypted delivery to your paired devices.

## 5. Install The CLI

```bash
go install lucor.dev/beebuzz/cmd/beebuzz@latest
```

## 6. Connect The CLI

Run:

```bash
beebuzz connect
```

Paste your API token when prompted.

For the hosted beta, `connect` already defaults to the BeeBuzz API. You do not need to enter a custom URL unless you are targeting another instance.

## 7. Send Your First Encrypted Message

```bash
beebuzz send "Hello from BeeBuzz"
```

The CLI encrypts the payload before sending it.

If everything is set up correctly, the notification should arrive on your paired device immediately.

## If Nothing Arrives

Check these first:

1. the device is listed as paired in **Account** -> **Devices**
2. the Hive app is installed where BeeBuzz recommends it
3. notification permission is still granted
4. you connected the CLI with the same account that owns the paired device

If the CLI reports paired device key changes, review them carefully before sending again.

## Why We Recommend Installing The PWA

Even where install is not strictly required, BeeBuzz recommends it.

Why:

- it gives the client a more stable app-like runtime
- it makes notification behavior more predictable
- it reduces confusion around tabs, background state, and permissions
- on iPhone and iPad, installation is required for Web Push

See [Browser Support](/docs/browser-support) for the platform details.
