package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestIPHasher_Hash(t *testing.T) {
	h := NewIPHasher("test-secret")

	a := h.Hash("95.12.34.56")
	b := h.Hash("95.12.34.56")
	if a != b {
		t.Error("expected same hash for same input")
	}
	if len(a) != 32 {
		t.Errorf("len(a) = %d, want 32", len(a))
	}

	c := h.Hash("10.0.0.1")
	if a == c {
		t.Error("expected different hashes for different IPs")
	}

	h2 := NewIPHasher("other-secret")
	d := h2.Hash("95.12.34.56")
	if a == d {
		t.Error("expected different hashes for different secrets")
	}
}

func TestIPHasher_Hash_KnownAnswer(t *testing.T) {
	h := NewIPHasher("test-secret")
	got := h.Hash("95.12.34.56")
	if got != "6c4d87fbe83916df59517fa983340507" {
		t.Errorf("Hash() = %q, want %q", got, "6c4d87fbe83916df59517fa983340507")
	}
}

func TestIPHasher_Hash_Concurrent(t *testing.T) {
	h := NewIPHasher("test-secret")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := h.Hash("95.12.34.56")
			if len(got) != 32 {
				t.Errorf("len(got) = %d, want 32", len(got))
			}
		}()
	}
	wg.Wait()
}

func TestIPHasher_Middleware(t *testing.T) {
	h := NewIPHasher("test-secret")

	t.Run("hashes IP from context", func(t *testing.T) {
		var got string
		var ok bool
		inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got, ok = HashedIPFromContext(r.Context())
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(withClientIP(req.Context(), "95.12.34.56"))

		h.Middleware(inner).ServeHTTP(httptest.NewRecorder(), req)

		if !ok {
			t.Error("expected ok to be true")
		}
		if got != h.Hash("95.12.34.56") {
			t.Errorf("got = %q, want %q", got, h.Hash("95.12.34.56"))
		}
	})

	t.Run("returns 500 without leaking internals when IP missing", func(t *testing.T) {
		inner := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			t.Fatal("handler should not be called")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.Middleware(inner).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("rec.Code = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		if strings.Contains(rec.Body.String(), "missing") {
			t.Error("response should not contain 'missing'")
		}
	})
}

func TestHashedIPFromContext(t *testing.T) {
	ctx := withHashedIP(t.Context(), "abc123")
	ip, ok := HashedIPFromContext(ctx)
	if !ok {
		t.Error("expected ok to be true")
	}
	if ip != "abc123" {
		t.Errorf("ip = %q, want %q", ip, "abc123")
	}

	_, ok = HashedIPFromContext(t.Context())
	if ok {
		t.Error("expected ok to be false")
	}
}
