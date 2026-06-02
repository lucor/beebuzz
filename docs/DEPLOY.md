# BeeBuzz Deployment Guide

Reference document for the current production deployment model.

## How To Use This Doc

- Canonical for production topology, required runtime configuration, container responsibilities, and deploy-time assumptions.
- Read this when changing Dockerfiles, release tooling, runtime env vars, or public host routing.
- Update this doc when deployment behavior, required environment variables, or hosting assumptions change.

## Architecture

BeeBuzz production uses two containers built by `beebuzz-release`:

| Service | Image | Role |
|---|---|---|
| `api` | `${REGISTRY_HOST}/${REGISTRY_OWNER}/beebuzz:<release_version>` | Go backend, SQLite, push delivery, auth, webhooks |
| `web` | `${REGISTRY_HOST}/${REGISTRY_OWNER}/beebuzz-web:<release_version>` | Caddy edge, static site, static Hive app, vanity Go import redirects |

## Domains

The server handles these subdomains derived from `BEEBUZZ_DOMAIN`:

| Host | Purpose |
|---|---|
| `api.{domain}` | backend API |
| `push.{domain}` | push API host |
| `hook.{domain}` | webhook host |

The `web` container handles `{domain}` (main site) and `hive.{domain}` (Hive app). `api.{domain}`, `push.{domain}`, and `hook.{domain}` are routed directly to the `api` container by the upstream reverse proxy (e.g. Dokploy/Traefik).

## Release Process

`beebuzz` uses an on-demand release model driven by the `beebuzz-release` tool. There are no GitHub CI or release workflows assumed by this repo.

The release tool guards that the working tree is clean, the current branch is `main`, and local `main` is not behind `origin/main`. It then creates and pushes a `beebuzz@YYYY.MM.DD.N-SHORTSHA` tag, builds and pushes the selected Podman images, and creates the forge release.

Manual verification must happen before invoking the release tool. This is the same gate that would otherwise run in CI:

```bash
mise run setup
mise run tidy
mise run check
mise run test
mise run test-race
mise run lint
mise run vuln
mise run build
```

### How to release

```bash
releaser release beebuzz
```

The tool previews the tag and commits since the last release, then prompts for confirmation before building and pushing both images, creating the forge release, and pushing the tag.

## Image Builds

- build tool: `podman`
- platform: `linux/amd64`
- server Dockerfile: `deploy/server.Dockerfile`
- web Dockerfile: `deploy/web.Dockerfile`
- Caddy config: `deploy/Caddyfile`
- tags:
  - `${REGISTRY_HOST}/${REGISTRY_OWNER}/beebuzz:<release_version>`
  - `${REGISTRY_HOST}/${REGISTRY_OWNER}/beebuzz-web:<release_version>`

Server build args:

| Variable | Purpose |
|---|---|
| `COMMIT_SHA` | Set by `beebuzz-release`; used for health/version metadata |

Web build args:

| Variable | Purpose |
|---|---|
| `VITE_BEEBUZZ_DOMAIN` | Set from `BEEBUZZ_DOMAIN`; public app domain |
| `VITE_BEEBUZZ_DEBUG` | Set to `false` by release config |
| `VITE_BEEBUZZ_DEPLOYMENT_MODE` | Set to `saas` by release config |

The release configuration is embedded in `beebuzz-release` as the `beebuzz` repo config, not stored in this repo as `.release.toml`.

Required release-time env vars:

| Variable | Purpose |
|---|---|
| `FORGE_HOST` | Forgejo/Gitea instance hostname |
| `FORGE_OWNER` | Forge organization or user |
| `FORGE_TOKEN` | Token used to create the release |
| `BEEBUZZ_DOMAIN` | Public BeeBuzz domain used to build and route the web image |
| `REGISTRY_HOST` | Container registry hostname |
| `REGISTRY_OWNER` | Registry image owner |

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

`vapid generate` prints the private key to stdout. Treat the output as a secret and do not paste it into shared logs or terminal recordings.

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
- change a Dockerfile or Caddyfile in `deploy/`
- change the `beebuzzd` subcommands (e.g., `serve`, `healthcheck`, `vapid generate`)
- change persistence paths or health check behavior
