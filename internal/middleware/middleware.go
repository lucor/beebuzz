package middleware

import (
	"context"
	"net/http"
	"slices"

	"lucor.dev/beebuzz/internal/core"
)

const (
	CookieSessionName  = "beebuzz_session"
	CookieLoggedInName = "beebuzz_logged_in"

	messageInvalidSession = "Invalid or expired session"
	messageForbidden      = "Forbidden"
)

type contextKey string

const CtxKeyUser contextKey = "user"

// CtxUser represents the authenticated user in the request context.
type CtxUser struct {
	ID      string
	IsAdmin bool
}

// SessionUser is the result of session validation returned by the auth provider.
type SessionUser struct {
	ID      string
	IsAdmin bool
}

// SessionValidator validates a session token and returns the user.
type SessionValidator interface {
	ValidateSession(ctx context.Context, sessionToken string) (*SessionUser, error)
}

// UserFromContext retrieves the authenticated user from the request context.
func UserFromContext(ctx context.Context) (*CtxUser, bool) {
	user, ok := ctx.Value(CtxKeyUser).(*CtxUser)
	return user, ok
}

// RequireSession creates a middleware that requires a valid session.
func RequireSession(validator SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieSessionName)
			if err != nil {
				core.WriteUnauthorized(w, "invalid_session", messageInvalidSession)
				return
			}

			u, err := validator.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				core.WriteUnauthorized(w, "invalid_session", messageInvalidSession)
				return
			}

			user := &CtxUser{ID: u.ID, IsAdmin: u.IsAdmin}
			ctx := context.WithValue(r.Context(), CtxKeyUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin creates a middleware that requires admin privileges.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || !user.IsAdmin {
				core.WriteForbidden(w, "forbidden", messageForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BaseSecurity adds base security headers to all responses.
// CSP is enforced at Caddy layer, not here.
func BaseSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// APISecurity adds API-specific security headers to all /v1 responses.
// Cache-Control: no-store prevents sensitive data from being cached
// by browsers, proxies, or CDNs.
func APISecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// CORS adds Cross-Origin Resource Sharing headers. It checks the request Origin
// against allowedOrigins and, on match, reflects the origin with credentials support.
// OPTIONS preflight requests are answered immediately with 204.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			originAllowed := origin != "" && slices.Contains(allowedOrigins, origin)
			if originAllowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Add("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrigin silently rejects requests whose Origin header does not match
// the expected browser origin for site-only endpoints.
func RequireOrigin(expectedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Origin") != expectedOrigin {
				core.WriteNoContent(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
