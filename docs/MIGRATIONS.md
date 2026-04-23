# Database Migrations

BeeBuzz uses embedded SQL migrations executed through `golang-migrate`.

## How To Use This Doc

- Canonical for migration workflow, naming, forward-only policy, and recorded schema history.
- Read this when changing schema, adding a migration, or debugging migration state in an environment.
- Update this doc when a new migration is added or the migration process changes.

## Overview

- migration files live in `internal/migrations/`
- SQL files are embedded into the server binary
- the runtime driver is SQLite via `modernc.org/sqlite`
- applied versions are tracked in `schema_migrations`

The migration runner is implemented in `internal/migrations/migrations.go`.

## How Migrations Run

On server startup:

1. BeeBuzz opens the SQLite database
2. BeeBuzz applies SQLite connection pragmas such as WAL mode and `foreign_keys=ON`
3. BeeBuzz initializes the migration source from embedded files
4. `golang-migrate` applies all pending `.up.sql` files in version order
5. applied versions are tracked in `schema_migrations`

If there are no pending migrations, startup continues normally.

Tests also run the same embedded migrations against an in-memory SQLite database via `internal/testutil.NewDB(t)`.

## File Naming

Migration files follow this pattern:

```text
000001_init_schema.up.sql
000001_init_schema.down.sql
```

Rules:

- the numeric prefix is the migration version
- `.up.sql` is the forward migration
- `.down.sql` is kept for reference, but runtime migration is forward-only

## Creating a New Migration

Create the next numbered pair in `internal/migrations/`:

```text
000002_some_change.up.sql
000002_some_change.down.sql
```

Then rebuild and run the server. Migrations execute automatically on startup, and tests that use `internal/testutil.NewDB(t)` will also apply the full migration set.

Change checklist:

- add the next numbered `.up.sql` and `.down.sql` pair
- keep runtime behavior forward-only; use a new migration instead of rewriting applied history
- update this document's migration history table
- update `docs/openapi.yaml` too if the schema change affects an HTTP contract

## Checking Status

You can inspect the migration table directly with SQLite:

```bash
sqlite3 /path/to/beebuzz.db "SELECT version, dirty FROM schema_migrations;"
```

Example:

```text
1|0
```

`dirty = 1` means a migration failed partway through and needs manual attention.

## Current Migration History

| Version | Name | Description |
|---|---|---|
| 1 | `000001_init_schema` | Full initial schema including analytics, account status fields, and webhook priority |
| 2 | `000002_device_pairing_status` | Add canonical device pairing state to the `devices` table and backfill existing rows |
| 3 | `000003_device_token` | Add hashed device authentication token storage for Hive pairing health checks |

## Current Schema Notes

General conventions:

- timestamps are stored as `INTEGER` Unix milliseconds
- booleans are stored as SQLite integers
- foreign keys are enforced via SQLite pragmas at connection setup

Main tables today include:

- `users`
- `auth_challenges`
- `sessions`
- `api_tokens`
- `api_token_topics`
- `topics`
- `devices`
- `device_pairing_codes`
- `push_subscriptions`
- `device_topics`
- `webhooks`
- `webhook_topics`
- `attachments`
- `notification_events`
- `daily_usage_summary`

## Operational Notes

- migrations are embedded into the server binary
- startup migration is automatic
- test databases created through `internal/testutil.NewDB(t)` run the same embedded migrations
- failed migrations abort startup
- BeeBuzz treats schema migration as forward-only
- if you need to undo a change, create a new forward migration that reverses the previous one

## References

- [internal/migrations/migrations.go](../internal/migrations/migrations.go)
- [internal/database/database.go](../internal/database/database.go)
- [internal/migrations/](../internal/migrations/)

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- add a new migration file under `internal/migrations/` (update the "Current Migration History" table)
- add or remove a table (update the "Current Schema Notes" table list)
- change the migration runner or startup behavior in `internal/migrations/migrations.go`
