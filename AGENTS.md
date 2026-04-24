# BeeBuzz Agent Guide

Start here when working in this repository.

## How To Use This Guide

1. Read this file first to identify the right source documents for the task.
2. If `AGENTS.local.md` exists in the repository root, read it and follow its rules. Local rules take precedence over this file.
3. Read [docs/STYLE.md](docs/STYLE.md) on every task, then read only the area-specific docs that match the code you are changing.
4. When behavior, contracts, or operational policy change, update the owning document in the same task instead of leaving the new rule implicit in code.
5. Follow repo-wide conventions from [docs/STYLE.md](docs/STYLE.md): simple required path params can be checked in handlers, bearer/header credential extraction must go through shared middleware rather than custom parsing in handlers, and repeated frontend state/reason strings must be centralized as named constants instead of duplicated inline.

## Source Of Truth Order

- Code and tests define the implemented behavior. If docs disagree with code, fix the docs in the same task.
- [docs/openapi.yaml](docs/openapi.yaml) is canonical for exact HTTP endpoints, schemas, status codes, and endpoint metadata such as `x-audience` and `x-stability`.
- The most specific document wins over a broader one: for example `E2E_ENCRYPTION.md` over `MESSAGING.md`, or `LOGGING.md` over the logging summary in `STYLE.md`.

## What is BeeBuzz

BeeBuzz is a push delivery system supporting two modes: end-to-end encrypted delivery, where the server only stores and forwards opaque ciphertext, and server-trusted notifications, where the server can read content.

- **Backend**: Go + SQLite — domain-driven, no ORM
- **Frontend**: SvelteKit + TypeScript + Tailwind CSS + daisyUI
- **Client**: Hive PWA with Service Worker, Web Push, and age-based X25519 E2E encryption

The server can deliver either server-trusted notification payloads or opaque E2E-encrypted blobs to paired devices. See [E2E_ENCRYPTION.md](docs/E2E_ENCRYPTION.md) for the cryptography model.

## Repo Map

```
cmd/beebuzz-server/ server entrypoint and subcommand dispatch (main.go),
                    service wiring and lifecycle (serve.go),
                    route registration (router.go),
                    cross-domain adapters (adapter.go)
cmd/beebuzz/        CLI (E2E encrypted push)
internal/           backend domains and shared packages
web/apps/site/      main site + public user docs
web/apps/hive/      Hive PWA (device pairing, push receive, decrypt)
web/packages/shared shared frontend code (logger, services, types)
deploy/             Dockerfiles, Caddyfile
docs/               engineering docs (this index)
```

## Project References

Read [docs/STYLE.md](docs/STYLE.md) first for shared engineering rules.  

| Document | Canonical for | Read when | Update when |
|----------|---------------|-----------|-------------|
| [docs/STYLE.md](docs/STYLE.md) | repo-wide engineering rules, naming, testing, frontend/backend conventions | always | a rule should apply across multiple areas of the repo |
| [docs/CODE_PATTERNS.md](docs/CODE_PATTERNS.md) | backend domain architecture, package layout, adapters, layer responsibilities | adding/changing backend domains or wiring | backend structure or layer conventions change |
| [docs/LOGGING.md](docs/LOGGING.md) | logging standards, levels, field naming, layer ownership, error capture / Sentry policy | adding or reviewing log statements, adding or changing error capture | logging policy, ownership, sensitive-data rules, or capture policy change |
| [docs/MIGRATIONS.md](docs/MIGRATIONS.md) | migration workflow, schema history, forward-only policy | changing DB schema | a migration is added or the migration process changes |
| [docs/MESSAGING.md](docs/MESSAGING.md) | messaging behavior, delivery modes, payload semantics, attachment policy | changing `/v1/push`, CLI send, Hive receive, webhooks | message flow, payload semantics, retention, or limits change |
| [docs/E2E_ENCRYPTION.md](docs/E2E_ENCRYPTION.md) | E2E key model, trust boundaries, envelope rules, client key storage | changing encryption, pairing, key storage | key handling, encryption flow, or E2E guarantees change |
| [docs/HIVE_ONBOARDING.md](docs/HIVE_ONBOARDING.md) | Hive onboarding state machine, install policy, pairing UX, browser gating | changing onboarding UX or pairing | onboarding states, install requirements, or recovery flow change |
| [docs/DEPLOY.md](docs/DEPLOY.md) | deployment topology, env vars, containers, runtime assumptions | changing deploy config, env vars, Docker, hosting | production deployment behavior or required config changes |
| [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) | security goals, attacker classes, trust boundaries, positioning guidance | security review, E2E claims, threat analysis | security claims, attacker assumptions, or trust boundaries change |
| [docs/openapi.yaml](docs/openapi.yaml) | canonical API contract (endpoints, schemas, status codes, audience/stability metadata) | changing any HTTP endpoint, request/response shape, or intended API consumer | HTTP contract or endpoint audience/stability changes |

## Doc Maintenance Rules

- Prefer narrow, canonical docs over duplicating the same explanation in multiple places.
- Use prose docs for intent, invariants, and operational rules; use `docs/openapi.yaml` for exact API schema.
- If a fact is unstable and can easily go stale, either remove it or clearly label how it should be re-validated.
- Every doc in `docs/` has an **Agent Maintenance Rule** section at the bottom listing the specific code changes that require updating that doc. Follow those rules when making changes in the listed areas.
