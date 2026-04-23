package middleware

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"lucor.dev/beebuzz/internal/secure"
)

// RateKeyByHashedIP is an httprate.KeyFunc that uses the hashed client IP
// from the request context as the rate-limit key. Requires RealIP and
// IPHasher middleware to run first.
func RateKeyByHashedIP(r *http.Request) (string, error) {
	ip, ok := HashedIPFromContext(r.Context())
	if !ok || ip == "" {
		return "", errors.New("missing hashed client IP in context")
	}
	return ip, nil
}

// RateKeyByBearerToken is an httprate.KeyFunc that reads the raw Bearer
// token from context (set by ExtractBearerToken middleware) and uses its
// SHA-256 hash as key. Falls back to hashed IP when the token is absent
// so the request still reaches the handler for a proper 401.
func RateKeyByBearerToken(r *http.Request) (string, error) {
	token, ok := BearerTokenFromContext(r.Context())
	if ok && token != "" {
		return secure.Hash(token), nil
	}

	// No valid Bearer token — fall back to IP so the handler can return 401.
	ip, ok := HashedIPFromContext(r.Context())
	if !ok || ip == "" {
		return "", errors.New("missing hashed client IP in context")
	}
	return "push-notoken:" + ip, nil
}

// RateKeyByURLParam returns an httprate.KeyFunc that reads the named URL
// parameter and uses its SHA-256 hash as rate-limit key. Falls back to
// hashed IP when the parameter is empty.
func RateKeyByURLParam(param string) func(*http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		token := chi.URLParam(r, param)
		if token != "" {
			return secure.Hash(token), nil
		}

		// No URL param — fall back to IP so the handler can return the appropriate error.
		ip, ok := HashedIPFromContext(r.Context())
		if !ok || ip == "" {
			return "", errors.New("missing hashed client IP in context")
		}
		return param + "-nokey:" + ip, nil
	}
}
