package middleware

import (
	"context"
	"net/http"
	"strings"
)

const bearerPrefix = "Bearer "

type bearerTokenCtxKey struct{}

// BearerTokenFromContext returns the raw Bearer token extracted by the
// ExtractBearerToken middleware. Returns ("", false) when the Authorization
// header was missing or malformed.
func BearerTokenFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(bearerTokenCtxKey{}).(string)
	return v, ok
}

// ExtractBearerToken reads the Authorization header, strips the "Bearer "
// prefix, and stores the raw token in the request context. If the header
// is absent or malformed the request proceeds without a context value,
// letting the handler return 401.
func ExtractBearerToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, bearerPrefix) && len(authHeader) > len(bearerPrefix) {
			ctx := context.WithValue(r.Context(), bearerTokenCtxKey{}, authHeader[len(bearerPrefix):])
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
