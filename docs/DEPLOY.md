# BeeBuzz Deployment Guide

Reference document for the current production deployment model.

## How To Use This Doc

- Canonical for production topology, required runtime configuration, container responsibilities, and deploy-time assumptions.
- Read this when changing Dockerfiles, CI deploy flow, runtime env vars, or public host routing.
- Update this doc when deployment behavior, required environment variables, or hosting assumptions change.

## Architecture

`beebuzzd` runs as a single container:

| Service | Image | Role |
|---|---|---|
| `api` | `${REGISTRY_HOST}/${REGISTRY_OWNER}/beebuzzd:server` | Go backend, SQLite, push delivery, auth, webhooks |

## Domains

The server handles these subdomains derived from `BEEBUZZ_DOMAIN`:

| Host | Purpose |
|---|---|
| `api.{domain}` | backend API |
| `push.{domain}` | push API host |
| `hook.{domain}` | webhook host |

The frontend (`web/` repo) handles `{domain}` (main site) and `hive.{domain}` (Hive app).

## Release Process

`beebuzzd` uses an on-demand, tag-driven release model. Deployment happens when a `beebuzzd@<short_sha>` tag is pushed.

### How to release

```bash
releaser release
```

The tool previews the tag and commits since the last release, then prompts for confirmation before building the image, creating the forge release, and pushing the tag.

## Image Build

- Dockerfile: `deploy/server.Dockerfile`
- tags:
  - `${REGISTRY_HOST}/${REGISTRY_OWNER}/beebuzzd:server`
  - `${REGISTRY_HOST}/${REGISTRY_OWNER}/beebuzzd:server-<short_sha>`

## Runtime Server Configuration

Important server env vars:

| Variable | Purpose |
|---|---|
| `BEEBUZZ_ENV` | environment mode |
| `BEEBUZZ_DOMAIN` | base domain used to derive all public URLs |
| `BEEBUZZ_DB_DIR` | SQLite database directory |
| `BEEBUZZ_ATTACHMENTS_DIR` | attachment storage directory |
| `BEEBUZZ_PORT` | internal HTTP port |
| `BEEBUZZ_PRIVATE_BETA` | toggles private-beta waitlist gating |
| `BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL` | optional email promoted to admin after successful OTP verification |
| `BEEBUZZ_PROXY_SUBNET` | trusted reverse-proxy CIDR used to accept `X-Forwarded-For` |
| `BEEBUZZ_IP_HASH_SALT` | secret salt used to hash client IPs; required in production |
| `BEEBUZZ_VAPID_PUBLIC_KEY` | VAPID public key for Web Push |
| `BEEBUZZ_VAPID_PRIVATE_KEY` | VAPID private key for Web Push |
| `BEEBUZZ_REQUEST_ID_HEADER` | HTTP header for request ID propagation (default: `X-Request-ID`) |
| `BEEBUZZ_MAILER_SENDER` | sender email |
| `BEEBUZZ_MAILER_REPLY_TO` | reply-to email |
| `BEEBUZZ_MAILER_RESEND_API_KEY` or SMTP settings | mail transport |
| `BEEBUZZ_SENTRY_DSN` | Sentry/GlitchTip DSN for error monitoring (empty = disabled) |

Derived values such as API URL, Hive URL, Push URL, Hook URL, cookie domain, and allowed origins are computed from `BEEBUZZ_DOMAIN`.

`BEEBUZZ_VAPID_PUBLIC_KEY` and `BEEBUZZ_VAPID_PRIVATE_KEY` are required in every environment. This keeps VAPID signing material out of SQLite and makes local/dev behavior match production.

Generate a fresh keypair with:

```bash
beebuzzd vapid generate
```

`vapid generate` prints the private key to stdout. Treat the output as a secret and do not paste it into shared CI logs or terminal recordings.

## First Admin Bootstrap

To bootstrap the first admin:

1. set `BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL` to the intended admin email
2. log in with that email and complete OTP verification
3. BeeBuzz promotes that user to admin during session creation

The bootstrap email also bypasses private-beta waitlist gating.

## Server Commands

`beebuzzd` requires an explicit subcommand:

```bash
beebuzzd serve
beebuzzd healthcheck
beebuzzd vapid generate
```

Containers and orchestrators should invoke `serve` explicitly and use `healthcheck` for container health probes.

## Persistence

Server persistence points:

| Path | Purpose |
|---|---|
| `/var/lib/beebuzz/db` | SQLite database storage |
| `/var/lib/beebuzz/attachments` | attachment file storage |

These are the only persistent application data paths that need regular backup.

## Health Checks

The server image uses `beebuzzd healthcheck`, which checks the backend `/health` endpoint.

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- add, remove, or rename a server environment variable in `internal/config/config.go`
- change the server Dockerfile in `deploy/`
- change the `beebuzzd` subcommands (e.g., `serve`, `healthcheck`, `vapid generate`)
- change persistence paths or health check behavior
