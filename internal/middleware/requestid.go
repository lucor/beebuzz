package middleware

import (
	"context"
	"net/http"
	"strings"

	"lucor.dev/beebuzz/internal/secure"
)

const (
	defaultRequestIDHeader = "x-request-id"
	maxRequestIDLen        = 64
	randomIDBytes          = 16 // 16 bytes → 32 hex chars
)

type requestIDCtxKey struct{}

// RequestIDFromContext returns the request ID stored by the RequestID middleware.
// Returns empty string if not present.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDCtxKey{}).(string)
	return v
}

// RequestID reads or generates a request ID, sets it in the response header
// and request context. The header name is configurable to match the reverse
// proxy convention (e.g. Traefik uses "X-Request-Id" by default).
// Pass an empty string to use the default "X-Request-ID".
type RequestID struct {
	header string
}

// NewRequestID creates a RequestID middleware. header is the HTTP header name
// used to read and write the request ID (e.g. "X-Request-Id").
// Pass "" to use the default "X-Request-ID".
func NewRequestID(header string) *RequestID {
	if header == "" {
		header = defaultRequestIDHeader
	}
	return &RequestID{header: header}
}

// Middleware returns an http middleware that propagates or generates request IDs.
func (rid *RequestID) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(rid.header))
		if len(id) > maxRequestIDLen {
			id = id[:maxRequestIDLen]
		}
		if id == "" || !isValidRequestID(id) {
			id = secure.MustRandomHex(randomIDBytes)
		}

		w.Header().Set(rid.header, id)
		ctx := context.WithValue(r.Context(), requestIDCtxKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isValidRequestID checks that s contains only alphanumeric, dash, or underscore.
func isValidRequestID(s string) bool {
	for _, c := range s {
		if !isAllowed(c) {
			return false
		}
	}
	return true
}

// isAllowed returns true if c is alphanumeric, dash, or underscore.
func isAllowed(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}
