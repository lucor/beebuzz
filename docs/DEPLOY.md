# BeeBuzz Deployment Guide

Reference document for the current production deployment model.

## How To Use This Doc

- Canonical for production topology, required runtime configuration, container responsibilities, and deploy-time assumptions.
- Read this when changing Dockerfiles, CI deploy flow, runtime env vars, or public host routing.
- Update this doc when deployment behavior, required environment variables, or hosting assumptions change.

## Architecture

BeeBuzz runs as two containers:

| Service | Image | Role |
|---|---|---|
| `api` | `ghcr.io/lucor/beebuzz:server` | Go backend, SQLite, push delivery, auth, webhooks |
| `web` | `ghcr.io/lucor/beebuzz:web` | Caddy + static `site` and `hive` builds |

## Domains

All public subdomains are derived from `BEEBUZZ_DOMAIN`.

| Host | Purpose |
|---|---|
| `{domain}` | main site |
| `hive.{domain}` | Hive app |
| `api.{domain}` | backend API |
| `push.{domain}` | push API host |
| `hook.{domain}` | webhook host |

At runtime:

- an edge proxy terminates TLS before traffic reaches the BeeBuzz containers
- Caddy listens on HTTP only inside the `web` container
- BeeBuzz cookies use `.{domain}` for cross-subdomain session sharing

## Release Process

BeeBuzz uses an on-demand, tag-driven release model. Merging to `main` runs CI tests only; deployment happens when a `beebuzz@<short_sha>` tag is pushed.

### How to release

```bash
./scripts/release.sh
```

The script previews the tag and commits since the last release, then prompts for confirmation before creating and pushing the tag.

This triggers three workflows:

1. **`docker.yml`** — runs tests, builds both Docker images, pushes to GHCR, and triggers Dokploy deploy.
2. **`release.yml`** — creates a GitHub Release with auto-generated notes from merged PRs since the previous tag.
3. **`cli-release.yml`** — only triggered by `v*` tags (see below).

A `workflow_dispatch` trigger is available on `docker.yml` as an escape hatch for manual rebuilds without a tag.

### CI on Pull Requests

`.github/workflows/ci.yml` runs server and web tests on every PR targeting `main`. Tests are path-filtered: server tests run only when backend files change, web tests only when frontend files change.

### CLI releases

The `beebuzz` CLI is released independently via `v*` semver tags (e.g., `v0.9.0`). GoReleaser builds cross-platform binaries and publishes them as GitHub Releases. CLI releases are decoupled from server/web deploys.

## Image Builds

Images are built by `.github/workflows/docker.yml` on `beebuzz@*` tag push or `workflow_dispatch`.

### Server image

- Dockerfile: `deploy/server.Dockerfile`
- tags:
  - `ghcr.io/lucor/beebuzz:server`
  - `ghcr.io/lucor/beebuzz:server-<short_sha>`

### Web image

- Dockerfile: `deploy/web.Dockerfile`
- tags:
  - `ghcr.io/lucor/beebuzz:web`
  - `ghcr.io/lucor/beebuzz:web-<short_sha>`

## Build-Time Web Configuration

The web image bakes these values at build time:

| Variable | Source |
|---|---|
| `VITE_BEEBUZZ_DOMAIN` | GitHub Actions repo variable |
| `VITE_BEEBUZZ_DEBUG` | GitHub Actions repo variable |
| `VITE_BEEBUZZ_DEPLOYMENT_MODE` | GitHub Actions repo variable |

These values are compiled into the static frontend build. Changing them requires rebuilding the web image.

`VITE_BEEBUZZ_DEPLOYMENT_MODE` is the public-site deployment switch for hosted-only frontend features:

- `self_hosted` (default): hides hosted public routes such as `/docs` and `/legal`
- `saas`: enables the hosted public docs and legal hub

This variable replaces the older docs-only gating flag. New hosted-only public site features should depend on the same deployment mode instead of introducing parallel build flags.

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
beebuzz-server vapid generate
```

`vapid generate` prints the private key to stdout. Treat the output as a secret and do not paste it into shared CI logs or terminal recordings.

## First Admin Bootstrap

To bootstrap the first admin:

1. set `BEEBUZZ_BOOTSTRAP_ADMIN_EMAIL` to the intended admin email
2. log in with that email and complete OTP verification
3. BeeBuzz promotes that user to admin during session creation

The bootstrap email also bypasses private-beta waitlist gating.

## Server Commands

`beebuzz-server` requires an explicit subcommand:

```bash
beebuzz-server serve
beebuzz-server healthcheck
beebuzz-server vapid generate
```

Containers and orchestrators should invoke `serve` explicitly and use `healthcheck` for container health probes.

## Persistence

Server persistence points:

| Path | Purpose |
|---|---|
| `/var/lib/beebuzz/db` | SQLite database storage |
| `/var/lib/beebuzz/attachments` | attachment file storage |

These are the only persistent application data paths that need regular backup.

## Caddy Responsibilities

The `web` container serves:

- static `site` files for `{domain}`
- static `hive` files for `hive.{domain}`
- reverse proxying for `api.{domain}`, `push.{domain}`, and `hook.{domain}` to `api:8899`

Caddy also owns:

- security headers
- host-based routing
- service worker delivery for the web apps

## Health Checks

The server image uses `beebuzz-server healthcheck`, which checks the backend `/health` endpoint.

The `api` service should expose a healthcheck, and any dependent `web` service should wait for the API to become healthy before serving traffic.

## Deployment Split

The server and web images should be deployable as separate applications or services.

Recommended split:

- server app handles `api.{domain}`, `push.{domain}`, `hook.{domain}`
- web app handles `{domain}` and `hive.{domain}`

This keeps deploys independent and matches the image and routing model described above.

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- add, remove, or rename a server environment variable in `internal/config/config.go`
- add or change a build-time `VITE_` variable
- change Dockerfiles in `deploy/`
- change the Caddyfile routing or domain structure
- change the `beebuzz-server` subcommands (e.g., `serve`, `healthcheck`, `vapid generate`)
- change persistence paths or health check behavior
