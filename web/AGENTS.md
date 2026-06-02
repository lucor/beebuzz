# BeeBuzz Web Agent Guide

This repo owns the BeeBuzz frontend workspace:

- `apps/site`: public site, account UI, admin UI, and docs
- `apps/hive`: Hive PWA
- `packages/shared`: shared frontend API clients, stores, services, components, types, and assets

## Local Rules

- Treat this repo as isolated. Run commands from this repo root, not from the parent workspace.
- Use `mise` for tooling and task execution.
- Use `pnpm` for frontend work. Do not use `npm` or `bun`.
- Keep `pnpm-lock.yaml` as the only package lock.
- Do not edit the legacy reference repo at `~/Developer/beebuzz`; it is read-only migration input.

## Commands

- Setup: `mise run setup`
- Type/Svelte checks: `mise run check`
- Lint/format check: `mise run lint`
- Unit tests: `mise run test`
- Build: `mise run build`
- Site dev server: `mise run dev-site`
- Hive dev server: `mise run dev-hive`

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
