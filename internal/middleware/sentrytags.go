package middleware

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
)

// SentryTags sets Sentry scope tags for the request.
// Must be placed AFTER sentryhttp handler and RequestID middleware.
// Sets: request_id, route, user_id (if authenticated).
func SentryTags(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			next.ServeHTTP(w, r)
			return
		}

		hub.Scope().SetTag("request_id", RequestIDFromContext(r.Context()))

		if route := chi.RouteContext(r.Context()); route != nil {
			if pattern := route.RoutePattern(); pattern != "" {
				hub.Scope().SetTag("route", pattern)
			}
		}

		if user, ok := UserFromContext(r.Context()); ok {
			hub.Scope().SetUser(sentry.User{ID: user.ID})
			hub.Scope().SetTag("user_id", user.ID)
		}

		next.ServeHTTP(w, r)
	})
}
