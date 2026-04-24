# Code Patterns — BeeBuzz Backend

Quick reference for the Domain-Driven Architecture (DDA) used in BeeBuzz.

## How To Use This Doc

- Canonical for backend package layout, layer boundaries, and cross-domain wiring patterns.
- Read this when changing `internal/` domains, adding adapters, or moving responsibilities between handler, service, and repository layers.
- Update this doc when the backend architecture pattern changes or a repeated package-level convention becomes standard.

## Architecture

Each business domain lives in its own package under `internal/<domain>/`. **Domains never import each other directly.** Cross-domain dependencies are resolved via interfaces and adapters.

```
internal/
├── auth/          # Authentication: OTP, sessions, beta signup
├── user/          # User management
├── topic/         # Topic CRUD
├── device/        # Device registration, push subscriptions
├── webhook/       # Webhook CRUD
├── attachment/    # Attachment storage
├── notification/  # Push notification dispatch (VAPID)
├── admin/         # Admin panel domain
├── event/         # Notification event tracking and analytics
├── system/        # Platform-generated operational policies, grouped by area
├── token/         # API token management
├── health/        # Health check endpoint
├── push/          # Web Push delivery internals
│
├── core/          # Shared: response helpers, request decoding, sentinel errors, account status
├── middleware/    # Auth middleware, request context helpers
├── database/      # SQLite connection (sqlx)
├── migrations/    # DB schema migrations
├── secure/        # Token generation, hashing, OTP
├── validator/     # Input validation
├── logger/        # Structured logger setup
├── config/        # Configuration loading
├── mailer/        # Email sending abstraction
├── httpfetch/     # HTTP fetch utility for remote resources
├── monitoring/    # Error monitoring (Sentry) setup
└── testutil/      # Test helpers (NewDB, etc.)
```

Adapters that bridge domains live in `cmd/beebuzz-server/adapter.go`. Service wiring and server lifecycle are in `cmd/beebuzz-server/serve.go`. Route registration is in `cmd/beebuzz-server/router.go`. The `main.go` entrypoint dispatches subcommands.

---

## Domain Package Structure

Preferred domain layout:

| File | Responsibility |
|------|----------------|
| `types.go` | Domain structs, DB row types, HTTP request/response types, domain-level interfaces |
| `repository.go` | sqlx DB queries only — no business logic, no logging |
| `service.go` | Business logic — calls repository, owns all domain logging |
| `handler.go` | HTTP: parse request, call service, write response — no business logic |

Some existing domains still use `<domain>.go` instead of `types.go`. When touching those packages, prefer moving toward the structure above instead of introducing new variations.

---

## Patterns

### 1. Types

Keep domain types in the domain root file (`types.go` preferred, `<domain>.go` tolerated in older packages). DB models, HTTP request/response types, and domain interfaces live there.

```go
// internal/topic/types.go
package topic

type Topic struct {
 ID          string  `db:"id"           json:"id"`
 UserID      string  `db:"user_id"      json:"user_id"`
 Name        string  `db:"name"         json:"name"`
 DisplayName string  `db:"display_name" json:"display_name"`
 Description *string `db:"description"  json:"description,omitempty"`
 CreatedAt   int64   `db:"created_at"   json:"created_at"`
 UpdatedAt   int64   `db:"updated_at"   json:"updated_at"`
}
```

HTTP-only types (request bodies, response shapes) that don't map to DB rows can also live in `types.go` or inline in the handler if used only once.

### 2. Repository

DB access only. No logging. Wrap errors with context. Return `nil, nil` for "not found" — never propagate `sql.ErrNoRows` to the caller.

```go
// internal/topic/repository.go
package topic

// Repository provides data access for the topic domain.
type Repository struct {
 db *sqlx.DB
}

// NewRepository creates a new topic repository.
func NewRepository(db *sqlx.DB) *Repository {
 return &Repository{db: db}
}

// GetByID retrieves a topic by ID and user. Returns nil, nil if not found.
func (r *Repository) GetByID(ctx context.Context, userID, id string) (*Topic, error) {
 var t Topic
 err := r.db.GetContext(ctx, &t,
  `SELECT id, user_id, name, display_name, description, created_at, updated_at
   FROM topics WHERE id = ? AND user_id = ?`,
  id, userID,
 )
 if errors.Is(err, sql.ErrNoRows) {
  return nil, nil
 }
 if err != nil {
  return nil, fmt.Errorf("failed to get topic: %w", err)
 }
 return &t, nil
}
```

### 3. Service

Business logic. Owns all logging for the domain. Calls repository and local interfaces only — never touches HTTP directly.

Follow [LOGGING.md](LOGGING.md) for logging policy, field naming, levels, and layer ownership. At the architecture level, the key rule is simple: repository returns errors, service is the first place to log them.

```go
// internal/auth/service.go
package auth

// Service provides authentication business logic.
type Service struct {
 repo             *Repository
 mailer           mailer.Mailer
 topicInitializer TopicInitializer // interface — never a concrete domain type
}

// RequestAuth initiates authentication for the given email.
func (s *Service) RequestAuth(ctx context.Context, email, state string, reason *string) (bool, error) {
 user, created, err := s.repo.GetOrCreateUser(ctx, email)
 if err != nil {
  slog.Error("failed to get or create user", "error", err)
  return false, err
 }

 logger := slog.With("user_id", user.ID)

 if created {
  logger.Info("new user created", "is_admin", user.IsAdmin)
 }

 // Pattern: define msg once, use in both log and error wrap
 otp, err := secure.NewOTP()
 if err != nil {
  msg := "failed to generate OTP"
  logger.Error(msg, "error", err)
  return false, fmt.Errorf("%s: %w", msg, err)
 }

 logger.Debug("auth challenge created", "expires_at", expiresAt) // DEBUG: dev/troubleshoot only
 logger.Info("auth email sent")
 return true, nil
}

// VerifyOTP verifies an OTP code against the stored challenge.
func (s *Service) VerifyOTP(ctx context.Context, otp, state string) (string, error) {
 challenge, err := s.repo.GetAuthChallengeByState(ctx, state)
 if err != nil {
  msg := "failed to retrieve auth challenge"
  slog.Error(msg, "error", err)
  return "", fmt.Errorf("%s: %w", msg, err)
 }

 if challenge == nil {
  slog.Warn("challenge not found for OTP verification")
  return "", core.ErrOTPNotFound
 }

 logger := slog.With("user_id", challenge.UserID)

 if challenge.AttemptCount >= maxOTPAttempts {
  logger.Warn("max OTP attempts exceeded", "attempts", challenge.AttemptCount)
  return "", core.ErrOTPMaxAttempts
 }

 if !secure.Verify(otp, challenge.OTPHash) {
  logger.Warn("invalid OTP", "attempts", challenge.AttemptCount+1)
  return "", core.ErrOTPInvalid
 }

 slog.Info("OTP verified")
 return challenge.UserID, nil
}
```

**`slog.With` — attach context once, use throughout:**

```go
// Good: user_id attached once, inherited by all calls on logger
logger := slog.With("user_id", user.ID)
logger.Info("new user created", "is_admin", user.IsAdmin)
logger.Debug("auth challenge created", "expires_at", expiresAt)
logger.Info("auth email sent")

// Bad: key repeated on every call
slog.Info("new user created", "user_id", user.ID, "is_admin", user.IsAdmin)
slog.Debug("auth challenge created", "user_id", user.ID, "expires_at", expiresAt)
slog.Info("auth email sent", "user_id", user.ID)
```

Use `slog.With` whenever a scoped identifier is available (`user_id`, `topic_id`, `device_id`, etc.). Create it as soon as that identifier is known within the function.

### 4. Handler

Parse request → call service → write response. No business logic.

Handlers follow [LOGGING.md](LOGGING.md): log only system-level failures or request lifecycle problems, and do not duplicate service-level events.

```go
// internal/topic/handler.go
package topic

// Handler handles topic HTTP requests.
type Handler struct {
 topicService *Service
 log          *slog.Logger
}

// NewHandler creates a new topic handler.
func NewHandler(svc *Service, logger *slog.Logger) *Handler {
 return &Handler{topicService: svc, log: logger}
}

// GetTopics handles GET requests for topics.
func (h *Handler) GetTopics(w http.ResponseWriter, r *http.Request) {
 userCtx, ok := middleware.UserFromContext(r.Context())
 if !ok {
  // System error: middleware should always set this
  h.log.Error("user not found in context")
  core.WriteInternalError(w, r, errors.New("user not found in context"))
  return
 }

 topics, err := h.topicService.GetTopics(r.Context(), userCtx.ID)
 if err != nil {
  // Service already logged — handler just responds
  core.WriteInternalError(w, r, err)
  return
 }

 core.WriteJSON(w, http.StatusOK, topics)
}

// CreateTopic handles POST requests to create a topic.
func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
 userCtx, ok := middleware.UserFromContext(r.Context())
 if !ok {
  h.log.Error("user not found in context")
  core.WriteInternalError(w, r, errors.New("user not found in context"))
  return
 }

 var req struct {
  Name        string `json:"name"`
  DisplayName string `json:"display_name"`
  Description string `json:"description"`
 }
 if err := core.DecodeJSON(r.Body, &req); err != nil {
 		core.WriteDecodeError(w, err)
 		return
 	}

 topic, err := h.topicService.CreateTopic(r.Context(), userCtx.ID, req.Name, req.DisplayName, req.Description)
 if err != nil {
  core.WriteInternalError(w, r, err)
  return
 }

 core.WriteJSON(w, http.StatusCreated, topic)
}
```

**`core` response helpers:**

```go
core.WriteOK(w, data)
core.WriteNoContent(w)
core.WriteJSON(w, statusCode, data)
core.WriteBadRequest(w, "error_code", "human message")
core.WriteUnauthorized(w, "error_code", "human message")
core.WriteNotFound(w, "error_code", "human message")
core.WriteForbidden(w, "error_code", "human message")
core.WriteConflict(w, "error_code", "human message")
core.WriteTooManyRequests(w, "error_code", "human message")
core.WritePayloadTooLarge(w)
core.WriteValidationErrors(w, errs)
core.WriteDecodeError(w, err)
core.WriteInternalError(w, r, err)
```

**`core` sentinel errors (shared across domains):**

```go
core.ErrUnauthorized
core.ErrNotFound
core.ErrPayloadTooLarge
```

**`auth` sentinel errors (auth domain, defined in `internal/auth/auth.go`):**

```go
auth.ErrOTPNotFound     // wraps core.ErrUnauthorized
auth.ErrOTPExpired      // wraps core.ErrUnauthorized
auth.ErrOTPUsed         // wraps core.ErrUnauthorized
auth.ErrOTPInvalid      // wraps core.ErrUnauthorized
auth.ErrOTPMaxAttempts  // wraps core.ErrUnauthorized
auth.ErrGlobalRateLimit // standalone, does NOT wrap ErrUnauthorized
```

Handlers check sentinel errors with `errors.Is`:

```go
if errors.Is(err, core.ErrUnauthorized) {
 core.WriteUnauthorized(w, "invalid_otp", "invalid or expired OTP")
 return
}
```

If you add a new domain-specific sentinel error, define it in the domain package (not in `core`) and wrap `core.ErrUnauthorized` or `core.ErrNotFound` when the error maps to that HTTP status.

---

### 5. Cross-Domain Dependencies (Adapters)

Domains must not import each other. When domain A needs behavior from domain B:

1. **Define an interface** in domain A (in `types.go` or `service.go`)
2. **Implement an adapter** in `cmd/beebuzz-server/adapter.go` using the concrete domain B service
3. **Wire it** in `cmd/beebuzz-server/serve.go`

```go
// Step 1 — internal/auth/service.go: define what auth needs
type TopicInitializer interface {
 CreateDefaultTopic(ctx context.Context, userID string) error
}

// Step 2 — cmd/beebuzz-server/adapter.go: implement using topic.Service
type authTopicInitializerAdapter struct {
 topicSvc *topic.Service
}

func (a *authTopicInitializerAdapter) CreateDefaultTopic(ctx context.Context, userID string) error {
 return a.topicSvc.CreateDefaultTopic(ctx, userID)
}

// Step 3 — cmd/beebuzz-server/serve.go: wire it
topicRepo := topic.NewRepository(db)
topicSvc := topic.NewService(topicRepo, log)

authTopicInit := &authTopicInitializerAdapter{topicSvc: topicSvc}
authSvc := auth.NewService(authRepo, m, cfg.URL, authTopicInit, log)
```

---

## Adding a New Domain

1. Create `internal/<domain>/` with `types.go`, `repository.go`, `service.go`, `handler.go`
2. Wire in `cmd/beebuzz-server/serve.go`: `NewRepository(db)` → `NewService(repo, log)` → `NewHandler(svc, log)`
3. Register routes in `cmd/beebuzz-server/router.go`
4. For cross-domain deps: define interface in the consuming domain, implement adapter in `cmd/beebuzz-server/adapter.go`

`internal/system/<area>` is reserved for platform-generated operational policy, such as deciding whether an internal BeeBuzz event should produce a user-facing notification. It must not become a generic utility namespace. Delivery mechanics still belong to the owning delivery domain, for example `internal/notification`.

---

## Token / Security Utilities

```go
// internal/secure package
secure.NewOTP()           // 6-digit OTP string
secure.NewSessionToken()  // cryptographically random session token
secure.Hash(value)        // SHA256 hex — always store hashes, never raw tokens
secure.Verify(raw, hash)  // constant-time comparison
```

---

## Agent Maintenance Rule

If you change any of the following, update the relevant section of this document in the same task:

- add, remove, or rename a domain package under `internal/`
- add, remove, or rename a system area under `internal/system/`
- add or change a response helper or sentinel error in `internal/core/`
- add or change a sentinel error in any domain package
- add or change an adapter in `cmd/beebuzz-server/adapter.go`
- change service wiring in `cmd/beebuzz-server/serve.go`
- change route registration in `cmd/beebuzz-server/router.go`
