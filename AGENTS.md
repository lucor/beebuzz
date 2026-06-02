# BeeBuzz Server Agent Guide

This repo contains `beebuzzd` — the BeeBuzz HTTP API server, web-push delivery engine, and admin backend.

## Local Rules

- Treat this repo as isolated. Run commands from this repo root, not from the parent workspace.
- Use `mise` for tooling and task execution.
- Do not edit the legacy reference repo at `~/Developer/beebuzz`; it is read-only migration input.
- Module path is the vanity import `beebuzz.app/beebuzzd`.

## Commands

- Setup: `mise run setup`
- Tidy modules: `mise run tidy`
- Test: `mise run test`
- Test with race detector: `mise run test-race`
- Lint: `mise run lint`
- Vulnerability scan: `mise run vuln`
- Build: `mise run build`

## Code Conventions

- Follow the conventions from `docs/STYLE.md` that apply to the server:
  - Prefer clarity over cleverness.
  - Early returns; shallow nesting.
  - Brief comments on exported and significant unexported functions.
  - Never silently swallow errors; never log sensitive data.
- One test per main behavior; use `t.Run(...)` for sub-cases.

## Package Layout

- `main.go` — thin entry point that dispatches subcommands (`serve`, `healthcheck`, `vapid generate`)
- `internal/server/` — service wiring, routing, and cross-domain adapters
- `internal/<domain>/` — business domains (auth, device, notification, etc.)

## Dependency Rules

- **SDK is the contract.** `beebuzzd` may only consume public symbols from `beebuzz.app/beebuzz-go`.
- **No reach-throughs.** Never import from `beebuzz/` (CLI) or from private packages of any other repo.
- Server-specific packages (`internal/push`, `internal/notification`, etc.) remain local; do not try to replace them with SDK packages.

## Release Workflow

The server is tagged **after** `beebuzz-go` SDK is tagged. Order:

1. Tag `beebuzz-go vX.Y.Z`.
2. In `beebuzzd/`: bump `go.mod` to the new tag, verify, tag the server.
3. Deploy via Docker (SHA tags) — the server does **not** use GoReleaser.

Never the reverse.
