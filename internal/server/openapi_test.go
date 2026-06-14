package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPIYAML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	openAPIYAML(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != openAPIYAMLContentType {
		t.Fatalf("content-type = %q, want %q", got, openAPIYAMLContentType)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Fatalf("cache-control = %q, want public max-age", got)
	}

	body := rec.Body.String()
	if strings.TrimSpace(body) == "" {
		t.Fatal("body is empty")
	}
	for _, want := range []string{"openapi:", "paths:"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body does not contain %q", want)
		}
	}
}
