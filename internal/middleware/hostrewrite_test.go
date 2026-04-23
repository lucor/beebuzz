package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"lucor.dev/beebuzz/internal/middleware"
)

func TestHostRewrite(t *testing.T) {
	const pushHost = "push.example.com"
	const hookHost = "hook.example.com"

	tests := []struct {
		name         string
		pushHost     string
		hookHost     string
		requestHost  string
		requestPath  string
		requestQuery string
		wantPath     string
	}{
		{
			name:        "push host root rewrites to /v1/push",
			pushHost:    pushHost,
			hookHost:    hookHost,
			requestHost: pushHost,
			requestPath: "/",
			wantPath:    "/v1/push",
		},
		{
			name:        "push host with topic rewrites to /v1/push/{topic}",
			pushHost:    pushHost,
			hookHost:    hookHost,
			requestHost: pushHost,
			requestPath: "/alerts",
			wantPath:    "/v1/push/alerts",
		},
		{
			name:         "push host preserves query params",
			pushHost:     pushHost,
			hookHost:     hookHost,
			requestHost:  pushHost,
			requestPath:  "/alerts",
			requestQuery: "?foo=bar",
			wantPath:     "/v1/push/alerts",
		},
		{
			name:        "hook host with token rewrites to /v1/webhooks/{token}",
			pushHost:    pushHost,
			hookHost:    hookHost,
			requestHost: hookHost,
			requestPath: "/abc123",
			wantPath:    "/v1/webhooks/abc123",
		},
		{
			name:        "non-matching host passes through unchanged",
			pushHost:    pushHost,
			hookHost:    hookHost,
			requestHost: "api.example.com",
			requestPath: "/v1/health",
			wantPath:    "/v1/health",
		},
		{
			name:        "empty pushHost skips push rewrite",
			pushHost:    "",
			hookHost:    hookHost,
			requestHost: pushHost,
			requestPath: "/alerts",
			wantPath:    "/alerts",
		},
		{
			name:        "empty hookHost skips hook rewrite",
			pushHost:    pushHost,
			hookHost:    "",
			requestHost: hookHost,
			requestPath: "/abc123",
			wantPath:    "/abc123",
		},
		{
			name:        "host with port matches correctly",
			pushHost:    "push.example.com:9999",
			hookHost:    hookHost,
			requestHost: "push.example.com:9999",
			requestPath: "/",
			wantPath:    "/v1/push",
		},
		{
			name:        "host without port matches correctly",
			pushHost:    "push.example.com",
			hookHost:    "hook.example.com",
			requestHost: "push.example.com",
			requestPath: "/",
			wantPath:    "/v1/push",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			handler := middleware.HostRewrite(tt.pushHost, tt.hookHost)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
			}))

			req := httptest.NewRequest(http.MethodPost, tt.requestPath+tt.requestQuery, nil)
			req.Host = tt.requestHost
			rw := httptest.NewRecorder()
			handler.ServeHTTP(rw, req)

			if gotPath != tt.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}
