package middleware

import (
	"log/slog"
	"net/http"
	"strings"
)

// HostRewrite returns middleware that rewrites the request path based on the Host header.
// Requests to pushHost get /v1/push prepended; requests to hookHost get /v1/webhooks prepended.
// When pushHost or hookHost is empty, that rewrite is inactive.
// The host values must be bare host (with optional port), not full URLs.
func HostRewrite(pushHost, hookHost string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host

			switch {
			case pushHost != "" && matchHost(host, pushHost):
				original := r.URL.Path
				if original == "/" || original == "" {
					r.URL.Path = "/v1/push"
				} else {
					r.URL.Path = "/v1/push" + original
				}
				slog.Debug("host rewrite: push", "from", original, "to", r.URL.Path, "host", host)

			case hookHost != "" && matchHost(host, hookHost):
				original := r.URL.Path
				r.URL.Path = "/v1/webhooks" + original
				slog.Debug("host rewrite: hook", "from", original, "to", r.URL.Path, "host", host)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// matchHost returns true if the request host matches the configured host.
// Comparison is case-insensitive and ignores a trailing slash.
func matchHost(requestHost, configuredHost string) bool {
	return strings.EqualFold(requestHost, configuredHost)
}
