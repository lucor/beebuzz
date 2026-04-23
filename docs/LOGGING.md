# Logging Standards

BeeBuzz uses structured logging on both backend and frontend with environment-aware output format.

## How To Use This Doc

- Canonical for logging ownership, log levels, field naming, and sensitive-data rules.
- Read this when adding log statements or reviewing whether a log belongs in a given layer.
- Update this doc when logging policy changes or a new shared logging pattern should become standard.

## Backend Configuration

Logging is configured in `cmd/beebuzz-server/main.go` via `internal/logger` and uses the `BEEBUZZ_ENV` environment variable:

- **dev**: Text format with DEBUG level for human-readable output
- **staging/prod**: JSON format with INFO level for structured log aggregation

```go
log := logger.New(cfg.Env)
slog.SetDefault(log)
```

## Layer Ownership

- **Service layer** owns primary domain logging.
- **Repository layer** does not log.
- **Handler layer** logs only unexpected system errors or lifecycle issues.
- **Frontend services/state** own primary frontend logging; components should log only critical failures.

## Logging Rules

### Do Log These

1. **Authentication and authorization events**
   ```go
   slog.Info("user authenticated", "user_id", userID)
   slog.Warn("authentication failed", "error", err)
   ```

2. **Business and operational errors**
   ```go
   slog.Error("failed to send notification", "error", err)
   slog.Error("failed to create session", "error", err)
   ```

3. **Security events** (failed validation, suspicious activity)
   ```go
   slog.Warn("API key validation failed", "error", err, "topic", req.Topic)
   slog.Warn("magic link not found")
   slog.Warn("OTP verification failed - max attempts exceeded", "attempt_count", count)
   ```

4. **Important business metrics or outcomes**
   ```go
   slog.Info("notifications sent", "topic", topic, "count", count)
   slog.Warn("removing invalid subscription", "endpoint", endpoint, "status", statusCode)
   ```

5. **Server lifecycle events**
   ```go
   slog.Info("beebuzz server starting", "address", addr, "env", cfg.Env)
   slog.Info("gracefully shutting down server")
   slog.Error("server error", "error", err)
   ```

### Don't Log These

1. **Routine operations** (normal business flow)
   - ~~`slog.Info("request handled")`~~
   - ~~`slog.Debug("validation passed")`~~

2. **Sensitive data** (NEVER)
   - ~~Email addresses~~ in log messages
   - ~~Passwords or tokens~~ (only log hashes if needed)
   - ~~Personal information~~ (PII)
   - ~~Raw API keys~~ (use identifiers instead)

3. **Database queries or internal details**
   - ~~SQL statements~~
   - ~~Raw payload contents~~ (log summary instead)

## Log Levels

| Level     | When to use                      | What to log                                                        | What to NEVER log                              |
|-----------|----------------------------------|--------------------------------------------------------------------|------------------------------------------------|
| **DEBUG** | Development / agent troubleshoot | Connection state, internal events, retries, remaining TTL          | E2E message content, tokens, private keys      |
| **INFO**  | Normal production operations     | Service start/stop, device paired/unpaired, notification received  | Encrypted payloads, magic links                |
| **WARN**  | Recoverable situations           | Retry failed, TTL expired, temporary connection loss               | Private messages, attachments                  |
| **ERROR** | Critical errors                  | Send/receive failure, service crash, filesystem errors              | Sensitive user data                            |
| **FATAL** | Irreversible crashes             | Safe stack trace, irreversible error                               | Sensitive payloads or keys                     |

### DEBUG (`slog.Debug`)
Development and troubleshooting only. Never enabled in production.

```go
slog.Debug("auth challenge created", "user_id", userID, "expires_at", expiresAt)
slog.Debug("session token generated", "user_id", userID)
```

### INFO (`slog.Info`)
Important business events and server state changes.

```go
slog.Info("session created", "user_id", userID)
slog.Info("notifications sent", "topic", topic, "count", count)
slog.Info("server stopped cleanly")
```

### WARN (`slog.Warn`)
Security events, validation failures, or unusual conditions that don't stop operation.

```go
slog.Warn("API key validation failed", "error", err, "topic", req.Topic)
slog.Warn("magic link not found")
slog.Warn("removing invalid subscription", "status", statusCode)
```

### ERROR (`slog.Error`)
System or operational failures that require immediate attention.

```go
slog.Error("failed to initialize database", "error", err)
slog.Error("failed to create session", "error", err)
```

### FATAL
Irreversible crashes. Use `slog.Error` followed by `os.Exit(1)` or `log.Fatal`.

```go
slog.Error("fatal: database connection lost", "error", err)
os.Exit(1)
```

## Best Practices

### Use snake_case for Field Names

All slog key names MUST use `snake_case`. This ensures consistency with the JSON API convention and makes log output parseable.

```go
slog.Info("user authenticated", "user_id", userID)          // GOOD
slog.Info("user authenticated", "userID", userID)            // BAD
slog.Debug("challenge created", "expires_at", expiresAt)     // GOOD
slog.Debug("challenge created", "expiresAt", expiresAt)      // BAD
```

### Use Structured Key-Value Pairs

✅ GOOD:
```go
slog.Info("notifications sent", "topic", topic, "count", count)
slog.Warn("subscription failed", "error", err, "endpoint", sub.Endpoint)
```

❌ BAD:
```go
slog.Info(fmt.Sprintf("Sent %d notifications to %s", count, topic))
```

### Be Specific with Context

✅ GOOD:
```go
slog.Warn("OTP verification failed - max attempts exceeded", "attempt_count", 5)
```

❌ BAD:
```go
slog.Warn("OTP verification failed")
```

### Sanitize Identifiers

✅ GOOD:
```go
slog.Warn("API key validation failed", "error", err, "topic", req.Topic)
```

❌ BAD:
```go
slog.Warn("API key validation failed", "api_key", apiKey, "topic", req.Topic)
```

## Output Examples

### Development (Text Format)

```
2026-02-15T10:30:45.123Z	INFO	beebuzz server starting	address=:8899	env=development
2026-02-15T10:30:46.456Z	INFO	user authenticated	user_id=abc-123
2026-02-15T10:30:47.789Z	WARN	API key validation failed	error=invalid key	topic=alerts
2026-02-15T10:30:48.012Z	INFO	notifications sent	topic=alerts	count=5
```

### Production (JSON Format)

```json
{"time":"2026-02-15T10:30:45.123Z","level":"INFO","msg":"beebuzz server starting","address":":8899","env":"prod"}
{"time":"2026-02-15T10:30:46.456Z","level":"INFO","msg":"user authenticated","user_id":"abc-123"}
{"time":"2026-02-15T10:30:47.789Z","level":"WARN","msg":"API key validation failed","error":"invalid key","topic":"alerts"}
{"time":"2026-02-15T10:30:48.012Z","level":"INFO","msg":"notifications sent","topic":"alerts","count":5}
```

## Logging Layer Ownership

Logging responsibilities are split by architectural layer. Never duplicate logs between layers for the same event.

### Backend (Go)

| Layer            | Responsibility                                  | Levels used                |
|------------------|--------------------------------------------------|----------------------------|
| **Service**      | Primary logging: business logic, auth events, recoverable errors | DEBUG, INFO, WARN, ERROR |
| **Handler**      | Only critical/system errors (e.g. JSON decode failure, response write error) | ERROR only |
| **Main**         | Server lifecycle (startup, shutdown, fatal)       | INFO, ERROR, FATAL        |
| **Repository**   | No logging (return errors to service layer)       | None                      |

### Frontend (SvelteKit)

| Layer                  | Responsibility                                  | Levels used                |
|------------------------|--------------------------------------------------|----------------------------|
| **Services / Stores**  | Primary logging: API calls, auth events, business logic errors | DEBUG, INFO, WARN, ERROR |
| **Components**         | Only critical/system errors (e.g. unrecoverable render failure) | ERROR only |
| **Server routes**      | Request lifecycle, SSR errors                    | WARN, ERROR               |

### Why

- **Single source of truth**: each event is logged once, at the layer that owns the logic.
- **No noise**: duplicate logs between handler/component and service make debugging harder, not easier.
- **Clean auditing**: log analysis tools see one entry per event, not two.

---

## Error Capture (Sentry / GlitchTip)

BeeBuzz uses Sentry-compatible error capture for unexpected server failures. Capture is enabled when `BEEBUZZ_SENTRY_DSN` is set (see [DEPLOY.md](DEPLOY.md) for runtime configuration). When the DSN is empty, all capture operations are no-ops.

### When To Capture vs Log Only

| Situation | Action |
|-----------|--------|
| 500 / unexpected internal error | Log **and** capture |
| Panic in request handler | Captured automatically by `sentryhttp` recovery middleware |
| Unexpected push-provider failure (not 404/410) | Log **and** capture |
| Expected failure (404, 410, 401, validation) | Log only |
| Business-logic warning (expired TTL, retry) | Log only |

**Rule:** log once at the owning layer; capture once at the boundary where the failure becomes unexpected or system-level. Never duplicate a capture between service and handler for the same event.

### Current Capture Points

| Location | Trigger |
|----------|---------|
| `cmd/beebuzz-server/serve.go` — `sentryhttp` middleware | Panic recovery |
| `internal/core/response.go` — `WriteInternalError` | Any 500 response |
| `internal/notification/service.go` | Unexpected push delivery failure (non-404/410) |

### Standard Tags

The `SentryTags` middleware (`internal/middleware/sentrytags.go`) sets these tags automatically on the request-scoped hub:

- `request_id` — from `X-Request-ID` header
- `route` — matched chi route pattern
- `user_id` — authenticated user ID (when present)

These tags are available only for **request-scoped captures** (i.e., captures that use the hub from `r.Context()`). Captures outside request scope (e.g., background goroutines using `sentry.WithScope`) must set their own tags explicitly.

### Forbidden Data

Never send the following to Sentry — not in tags, not in breadcrumbs, not in extra context:

- Email addresses, PII
- Passwords, API keys, session tokens, magic links, OTPs
- Raw request/response bodies
- Push endpoint URLs (use host-only: `push_host`)
- Attachment data, attachment tokens/URLs
- E2E ciphertext, encrypted payloads, private keys

This extends the same sensitive-data rules defined in the logging section above.

## Frontend Configuration

The frontend uses a lightweight logger utility at `web/packages/shared/src/logger/index.ts`.

### Setup

Debug output is controlled by the `VITE_BEEBUZZ_DEBUG` environment variable:

- **`VITE_BEEBUZZ_DEBUG=true`**: Enables `logger.debug()` output (development)
- **Not set / any other value**: `logger.debug()` is silenced (production)

```ts
import { logger } from '@beebuzz/shared/logger';

logger.debug("subscription created", { user_id: userId, topic: topic });
logger.info("device paired", { device_id: deviceId });
logger.warn("API call failed, retrying", { endpoint: url, status: res.status });
logger.error("failed to send notification", { error: err.message });
```

### Frontend Log Levels

| Level     | When to use                      | What to log                                                        | What to NEVER log                              |
|-----------|----------------------------------|--------------------------------------------------------------------|------------------------------------------------|
| **DEBUG** | Development only                 | API request/response details, state transitions, internal events   | Tokens, passwords, private keys, PII           |
| **INFO**  | Normal production operations     | Device paired, subscription created, auth success                  | Email addresses, session IDs                   |
| **WARN**  | Recoverable situations           | API retry, expired session, validation failure                     | Raw API keys, personal data                    |
| **ERROR** | Critical failures                | Unrecoverable API failure, service crash, render failure           | Sensitive user data                            |

### Frontend Best Practices

#### Use snake_case for Data Keys

All log data keys MUST use `snake_case`, consistent with the backend convention.

```ts
logger.info("device paired", { device_id: deviceId });       // GOOD
logger.info("device paired", { deviceId: deviceId });         // BAD
logger.warn("request failed", { status_code: res.status });   // GOOD
logger.warn("request failed", { statusCode: res.status });    // BAD
```

#### Log in Services/Stores, Not in Components

```ts
// ✅ GOOD: logging in a service (src/lib/services/auth.ts)
export const requestAuth = async (email: string) => {
    const res = await fetch(`${base}/api/auth/request`, { ... });
    if (!res.ok) {
        logger.warn("auth request failed", { status: res.status });
        throw new Error("Auth request failed");
    }
    logger.debug("auth request sent");
};

// ❌ BAD: logging in a component (+page.svelte)
// const handleLogin = async () => {
//     logger.info("user clicked login");  // noise, not a business event
// };
```

#### Never Log Sensitive Data

```ts
logger.info("user authenticated", { user_id: userId });           // GOOD
logger.info("user authenticated", { email: email });               // BAD - PII
logger.debug("session created", { expires_in: ttl });              // GOOD
logger.debug("session created", { session_token: token });         // BAD - secret
```

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- add or change a Sentry capture point (update the "Current Capture Points" table)
- add or change Sentry tags in middleware or capture calls
- change logging layer ownership rules or log level policy
- change the frontend logger utility at `web/packages/shared/src/logger/`
- change `BEEBUZZ_ENV` or `VITE_BEEBUZZ_DEBUG` logging behavior
