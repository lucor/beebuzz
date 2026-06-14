# BeeBuzz App Agent Guide

This repo contains the BeeBuzz application:

- **Server** (Go) — at the repo root: HTTP API, SQLite, web-push delivery, auth, mailer, admin backend. Module path `go.beebuzz.app/beebuzz`, binary `beebuzz`.
- **Frontend** (SvelteKit pnpm workspace) — under `web/`: `apps/dashboard`, `apps/hive`, `packages/shared`.
- **Local-dev orchestration** — under `.mise/`: Procfile, Caddyfile, setup-dev.sh, .air.toml.

The backend and frontend share the OpenAPI contract in `docs/openapi.yaml` and ship together as a single deployable unit.

## Local Rules

- Use `mise` for tooling and task execution. The unified [mise.toml](./mise.toml) at the repo root covers Go, Node, pnpm, air, caddy, goreman, mailpit, and all dev/test/lint/build tasks.
- Module path for the server is the immutable vanity import `go.beebuzz.app/beebuzz`. The repository URL may change; the module path may not.

## Commands

Run all commands from the repo root.

| Task | Description |
|---|---|
| `mise run setup` | Install Go modules and frontend dependencies |
| `mise run dev` | Boot the full local stack (server + dashboard + hive + Caddy TLS + Mailpit) |
| `mise run dev-api` | Hot-reload Go server only |
| `mise run dev-dashboard` | Vite dev server for `apps/dashboard` only |
| `mise run dev-hive` | Vite dev server for `apps/hive` only |
| `mise run dev-caddy` | Caddy reverse proxy with lancert.dev TLS |
| `mise run dev-mailpit` | Local Mailpit SMTP capture |
| `mise run test` | Run server + frontend unit tests |
| `mise run test-api` / `test-app` / `test-race` / `test-e2e` | Targeted suites |
| `mise run lint` | go vet + staticcheck + frontend ESLint/format check |
| `mise run check` | Svelte/TypeScript checks |
| `mise run vuln` | govulncheck |
| `mise run build` | Compile every Go package and build the frontend |
| `mise run binary` | Build the `beebuzz` binary into `./bin` |
| `mise run quickstart-demo` | Run the end-to-end quickstart demo against the local stack |
| `mise run ci` | Full pre-release verification |

## Package Layout

- `main.go` — thin entry point dispatching subcommands (`serve`, `healthcheck`, `vapid generate`)
- `healthcheck.go`, `vapid.go` — top-level subcommand implementations
- `internal/server/` — service wiring, routing, cross-domain adapters
- `internal/<domain>/` — business domains (auth, device, notification, etc.)
- `web/apps/dashboard` — account UI and admin UI (dashboard.beebuzz.app)
- `web/apps/hive` — Hive PWA receiver
- `web/packages/shared` — shared frontend API clients, stores, services, components, types, assets
- `docs/` — server docs (`openapi.yaml`, `STYLE.md`, etc.)
- `deploy/server.Dockerfile`, `deploy/web.Dockerfile`, `deploy/Caddyfile` — production container images
- `scripts/quickstart-demo.sh` — full-stack demo recorder
- `.mise/` — local-dev orchestration (Procfile, Caddyfile, setup-dev.sh, .air.toml)

## Code Conventions

- Server: follow `docs/STYLE.md`. Prefer clarity over cleverness, early returns, shallow nesting, brief comments on exported and significant unexported functions, never silently swallow errors, never log sensitive data. One test per main behavior; use `t.Run(...)` for sub-cases.
- Frontend: see [web/AGENTS.md](./web/AGENTS.md) for Svelte 5 / TypeScript / accessibility conventions.
- **Conventions:** "age" (cryptographic library) is always lowercase in identifiers, comments, and docs — never "AGE" or "Age".

## Dependency Rules

- **Server is the API provider; it does not depend on the client SDK.** The `go.beebuzz.app/beebuzz-go` client is a consumer of the public HTTP contract documented in `docs/openapi.yaml`. The server defines its own types internally and must not import the SDK.
- **No reach-throughs into other repos.** Never import from the `cli` repo or from any other repo's `internal/` packages.
- Server-specific packages (`internal/push`, `internal/notification`, etc.) remain local; do not try to replace them with SDK packages.
- Frontend ↔ server contract: `docs/openapi.yaml` is the source of truth. The frontend must conform.


