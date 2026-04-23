# Local Dev

This guide is for running BeeBuzz locally with the shortest practical setup.

It is intentionally focused on development, not production self-hosting.

## Prerequisites

- [git](https://git-scm.com/install/)
- [mise](https://mise.jdx.dev/installing-mise.html)
- a [supported browser](/docs/browser-support) on the same machine or LAN device you want to pair

## What This Uses

- BeeBuzz server, site, and Hive — the core services
- [Caddy](https://caddyserver.com/) — reverse proxy
- [lancert.dev](https://lancert.dev) — local HTTPS certificates and subdomains
- [Mailpit](https://mailpit.axllent.org/) — email capture
- [mise](https://mise.jdx.dev/) — task runner and tool manager

## 1. Clone The Repo

```bash
git clone https://github.com/lucor/beebuzz.git
cd beebuzz
```

## 2. Prepare The Environment

Copy the dev template:

```bash
cp .env.example .env
```

Set a bootstrap admin email in `.env` so you can sign in immediately during private beta:

```bash
BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL=you@example.com
```

Generate VAPID keys once:

```bash
go run ./cmd/beebuzz-server vapid generate
```

Paste the generated public and private keys into `.env`.

## 3. Install Tools And Dependencies

```bash
mise run setup
```

This installs the required tools, frontend dependencies, Go modules, and local Caddy trust.

## 4. Start The Full Dev Stack

```bash
mise run dev
```

On first run, BeeBuzz:

- detects your LAN IP
- derives a `*.lancert.dev` domain (e.g. `192-168-1-42.lancert.dev`)
- offers to update `BEEBUZZ_DOMAIN` in `.env`
- fetches TLS certificates if needed

The stack includes:

- site
- Hive
- API
- Caddy
- Mailpit

## 5. Open The Local Apps

Once the stack is up, use these URLs:

- site: `https://$BEEBUZZ_DOMAIN`
- Hive: `https://hive.$BEEBUZZ_DOMAIN`
- API: `https://api.$BEEBUZZ_DOMAIN`
- Mailpit: `http://localhost:8025`

## 6. Sign In

Open the site and request access with the bootstrap email you configured.

The OTP email lands in Mailpit, so you can complete sign-in locally without external email infrastructure.

## 7. Pair A Device

Open **Account** -> **Devices** and click **Add Device**.

Then:

1. open Hive from the generated link or QR code
2. install the PWA if prompted
3. grant notification permission
4. enter the pairing code

For the smoothest results, pair on the same machine first, then test on additional devices if needed.

## 8. Create A Token

Open **Account** -> **API Tokens** and create a token for the topic you want to use.

Copy the token immediately.

## 9. Connect The CLI

```bash
beebuzz connect --api-url "https://api.$BEEBUZZ_DOMAIN"
```

Paste the token when prompted.

## 10. Send A Test Message

```bash
beebuzz send "Local dev works"
```

If the device is paired and permissions are still valid, the notification should arrive right away.

## Notes

- This flow is for local development only.
- It is the recommended way to validate the full BeeBuzz loop quickly during beta.
- Detailed production deployment docs will come later, once the self-host path is ready to support properly.
