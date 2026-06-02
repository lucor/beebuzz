# BeeBuzz App Agent Guide

This repo contains the merged BeeBuzz application:

- **Server** (Go) — at the repo root: HTTP API, SQLite, web-push delivery, auth, mailer, admin backend. Module path `beebuzz.app/beebuzzd`, binary `beebuzzd`.
- **Frontend** (SvelteKit pnpm workspace) — under `web/`: `apps/site`, `apps/hive`, `packages/shared`.
- **Local-dev orchestration** — under `.mise/`: Procfile, Caddyfile, setup-dev.sh, .air.toml.

The two halves share the OpenAPI contract in `docs/openapi.yaml` and ship together. They are intentionally NOT split across repos — see `PLAN.md` Step 4 in the workspace root for the reasoning.

## Local Rules

- Use `mise` for tooling and task execution. The unified [mise.toml](./mise.toml) at the repo root covers Go, Node, pnpm, air, caddy, goreman, mailpit, and all dev/test/lint/build tasks.
- Do not edit the legacy reference repo at `~/Developer/beebuzz`; it is the read-only archive of the pre-split monorepo.
- Module path for the server is the immutable vanity import `beebuzz.app/beebuzzd`. The GitHub repo URL may change; the module path may not.

## Commands

Run all commands from the repo root.

| Task | Description |
|---|---|
| `mise run setup` | Install Go modules and frontend dependencies |
| `mise run dev` | Boot the full local stack (server + site + hive + Caddy TLS + Mailpit) |
| `mise run dev-api` | Hot-reload Go server only |
| `mise run dev-site` | Vite dev server for `apps/site` only |
| `mise run dev-hive` | Vite dev server for `apps/hive` only |
| `mise run dev-caddy` | Caddy reverse proxy with lancert.dev TLS |
| `mise run dev-mailpit` | Local Mailpit SMTP capture |
| `mise run test` | Run server + frontend unit tests |
| `mise run test-api` / `test-app` / `test-race` / `test-e2e` | Targeted suites |
| `mise run lint` | go vet + staticcheck + frontend ESLint/format check |
| `mise run check` | Svelte/TypeScript checks |
| `mise run vuln` | govulncheck |
| `mise run build` | Compile every Go package and build the frontend |
| `mise run binary` | Build the `beebuzzd` binary into `./bin` |
| `mise run quickstart-demo` | Run the end-to-end quickstart demo against the local stack |

## Package Layout

- `main.go` — thin entry point dispatching subcommands (`serve`, `healthcheck`, `vapid generate`)
- `healthcheck.go`, `vapid.go` — top-level subcommand implementations
- `internal/server/` — service wiring, routing, cross-domain adapters
- `internal/<domain>/` — business domains (auth, device, notification, etc.)
- `web/apps/site` — public site, account UI, admin UI, docs
- `web/apps/hive` — Hive PWA receiver
- `web/packages/shared` — shared frontend API clients, stores, services, components, types, assets
- `docs/` — server docs (`openapi.yaml`, `DEPLOY.md`, `STYLE.md`, etc.)
- `deploy/server.Dockerfile` — server container image
- `.release.toml` — release configuration read by `beebuzz-release`
- `scripts/quickstart-demo.sh` — full-stack demo recorder
- `.mise/` — local-dev orchestration (Procfile, Caddyfile, setup-dev.sh, .air.toml)

## Code Conventions

- Server: follow `docs/STYLE.md`. Prefer clarity over cleverness, early returns, shallow nesting, brief comments on exported and significant unexported functions, never silently swallow errors, never log sensitive data. One test per main behavior; use `t.Run(...)` for sub-cases.
- Frontend: see [web/AGENTS.md](./web/AGENTS.md) for Svelte 5 / TypeScript / accessibility conventions.

## Dependency Rules

- **SDK is the contract.** The server may only consume public symbols from `beebuzz.app/beebuzz-go`. Do not duplicate SDK types locally.
- **No reach-throughs into other repos.** Never import from the `beebuzz-cli` repo or from any other repo's `internal/` packages.
- Server-specific packages (`internal/push`, `internal/notification`, etc.) remain local; do not try to replace them with SDK packages.
- Frontend ↔ server contract: `docs/openapi.yaml` is the source of truth. The frontend must conform.

## Release Workflow

The merged app is tagged **after** `beebuzz-go` SDK is tagged. Order:

1. Tag `beebuzz-go vX.Y.Z` (in the SDK repo).
2. In this repo: bump `go.mod` to the new SDK tag, verify, run `releaser release` to push a `beebuzzd@<short_sha>` tag.
3. Deploy via Docker (SHA tags) — the server does **not** use GoReleaser. The frontend ships in a separate container alongside (see `deploy/`).

Never the reverse.
