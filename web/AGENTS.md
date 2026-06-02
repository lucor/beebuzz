# BeeBuzz Web Agent Guide

This is the frontend subdirectory of the `beebuzz` app repo. It is a pnpm workspace:

- `apps/site`: public site, account UI, admin UI, and docs
- `apps/hive`: Hive PWA
- `packages/shared`: shared frontend API clients, stores, services, components, types, and assets

## Local Rules

- This is **not** a standalone repo. All tasks (setup, dev, test, lint, build) are defined in the unified `mise.toml` at the parent repo root and must be invoked from there. See [../AGENTS.md](../AGENTS.md) for the full task list.
- Use `pnpm` for frontend work. Do not use `npm` or `bun`.
- Keep `pnpm-lock.yaml` here as the only package lock.

## Commands (run from the parent repo root)

- Setup: `mise run setup` (installs Go modules + frontend dependencies)
- Type/Svelte checks: `mise run check`
- Lint/format check: `mise run lint-app`
- Unit tests: `mise run test-app`
- E2E tests: `mise run test-e2e`
- Build: `mise run build`
- Site dev server (standalone): `mise run dev-site`
- Hive dev server (standalone): `mise run dev-hive`
- Full local stack (server + site + hive + Caddy + Mailpit): `mise run dev`

## Frontend Conventions

- Use Svelte 5 runes syntax.
- Prefer aliases such as `$lib`, `$components`, and `$config`.
- Do not use wildcard imports.
- Do not use `any`; use a real type or `unknown`.
- Use buttons for clickable elements.
- Program for accessibility.
- In server-side SvelteKit code, prefer `error()` from `@sveltejs/kit` for error responses.
- For client env vars, use `VITE_` and `import.meta.env`.
- For server env vars, use `$env/static/private`.
- Do not use `Set` or `Map` inside `$state`.
- Normalize backend or raw payload fields to frontend naming immediately after the boundary. Keep `snake_case` only in DTOs or raw payload shapes that intentionally mirror backend or API data.

## Documentation

- `apps/site/src/routes/docs` contains user-facing docs.
- When docs mention the CLI, server, SDK, or deployment, keep references aligned with the split repos and target vanity imports.
- If a doc depends on a later migration step, prefer a clear temporary note or defer the rewrite rather than inventing final behavior.
