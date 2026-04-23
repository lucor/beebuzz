# BeeBuzz Engineering Style

Shared engineering rules for contributors and coding agents.

## How To Use This Doc

- Canonical for repo-wide defaults. More specific docs can add detail, but they should not silently contradict this file.
- Read this before making changes anywhere in the repository.
- Update this doc when a rule should apply broadly across backend, frontend, tests, or review workflow.

## General

- Prefer clarity over cleverness.
- Use early returns to keep nesting shallow.
- Avoid redundant code.
- Keep responsibilities separated.
- Use constants instead of magic primitives in conditionals when that improves readability.
- Add brief comments to functions you create.
- Never use inline HTML styles.
- Never silently swallow errors.
- Never log sensitive data such as emails, tokens, passwords, or personal information.

## Testing

- Do not use mocks. Use real repositories and services.
- DB-backed tests must use `internal/testutil.NewDB(t)` or another helper from `internal/testutil/` when the helper is truly generic.
- Keep domain-specific test fixtures and wiring inside the package tests.
- Prefer one test per main behavior and use `t.Run(...)` for sub-cases instead of one large kitchen-sink test.
- Use `strings.Repeat` for max-length test strings instead of hardcoded long literals.
- Before running tests or checks, verify no other heavy test or check process is already running.

## Backend

- Follow the domain-driven structure described in [CODE_PATTERNS.md](CODE_PATTERNS.md).

### API Conventions

- Use `snake_case` for JSON fields.
- Use named response types. Do not return `map[string]any` or `map[string]string`.
- Collection responses use `{ "data": [...] }`.
- Use `core.WriteOK`, `core.WriteJSON`, and `core.WriteNoContent`.
- Keep validation split clean: `internal/validator` for input format checks, service layer for business invariants.
- Required path parameters can be checked directly in handlers when the rule is simple presence/non-empty.
- Do not manually parse bearer auth headers in handlers. Use shared middleware to extract transport credentials and read them from request context.
- Handlers map typed service errors with `errors.Is`.
- If an endpoint path, request shape, response shape, or HTTP status behavior changes, update `docs/openapi.yaml` in the same task.
- Annotate every OpenAPI operation in `docs/openapi.yaml` with `x-audience` to identify its intended consumer. Allowed values are `public`, `hive`, `site`, and `admin`. Use an array so operations can belong to multiple audiences when needed.
- Annotate every OpenAPI operation in `docs/openapi.yaml` with `x-stability`. Allowed values are `stable`, `experimental`, and `deprecated`.

### Logging

- Follow [LOGGING.md](LOGGING.md).
- Scope logs as soon as a stable identifier is available, for example `user_id`.
- Do not duplicate the same log event across handler and service.

## Frontend

- Use `pnpm` for frontend work. Do not use `npm` or `bun`.
- For framework reference, use the official LLM-friendly docs when needed:
  - Svelte / SvelteKit: `https://svelte.dev/docs/llms`
  - daisyUI: `https://daisyui.com/llms.txt`
- When using daisyUI components or patterns, prefer the official daisyUI markup and CSS structure first. Do not invent custom variants if the documented component already covers the use case. Add only minimal Tailwind customization when needed for app integration or theme alignment.
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

### Deprecation

- Do not introduce deprecated symbols in new code. Migrate existing usages when they are flagged.
- Frontend: `@typescript-eslint/no-deprecated` is enabled as a warning in the shared ESLint config and will surface deprecated APIs at lint time.
- Backend: `staticcheck` runs in CI and catches `SA1019` (using deprecated stdlib symbols). Project-internal deprecations must use a `// Deprecated: <reason> <alternative>` comment; removal is enforced by code review.

## Frontend Verification

- After frontend changes, run `pnpm run check` from `web/`.
- After frontend changes, run `pnpm run lint` from `web/`.
- Lint must pass before closing the task.

## Naming

### Go

- Use clear, explicit names.
- Prefer full words over abbreviations unless the abbreviation is standard and obvious.

### TypeScript

- `SCREAMING_SNAKE_CASE` for immutable primitive constants.
- `camelCase` for variables, functions, and objects.
- `PascalCase` for types, interfaces, classes, and enums.
- Use `snake_case` only at the JSON API boundary when mirroring backend fields.
- Frontend app, store, service, and component code must convert boundary `snake_case` fields to `camelCase` before they become shared UI or domain models.
- Repeated frontend domain state and reason strings must be defined once as named constants and reused across services, stores, and components instead of being duplicated inline.

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- add or change a repo-wide engineering convention (naming, testing, error handling)
- add or change a frontend or backend convention that applies across multiple domains
- change the frontend verification commands or toolchain (e.g., `pnpm` usage, lint/check commands)
