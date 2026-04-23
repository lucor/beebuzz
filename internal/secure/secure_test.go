package secure

import (
	"encoding/hex"
	"strings"
	"testing"
	"unicode"
)

func TestNewToken(t *testing.T) {
	t.Run("short token has correct length", func(t *testing.T) {
		tok, err := newToken(16)
		if err != nil {
			t.Fatalf("newToken(16): %v", err)
		}
		if len(tok) != 22 {
			t.Errorf("len(tok) = %d, want 22", len(tok))
		}
	})

	t.Run("long token has correct length", func(t *testing.T) {
		tok, err := newToken(tokenSizeLong)
		if err != nil {
			t.Fatalf("newToken(tokenSizeLong): %v", err)
		}
		if len(tok) != 43 {
			t.Errorf("len(tok) = %d, want 43", len(tok))
		}
	})

	t.Run("two tokens are unique", func(t *testing.T) {
		tok1, err := newToken(tokenSizeLong)
		if err != nil {
			t.Fatalf("newToken(tokenSizeLong): %v", err)
		}
		tok2, err := newToken(tokenSizeLong)
		if err != nil {
			t.Fatalf("newToken(tokenSizeLong): %v", err)
		}
		if tok1 == tok2 {
			t.Error("expected two different tokens")
		}
	})
}

func TestNewOTP(t *testing.T) {
	t.Run("otp has correct length", func(t *testing.T) {
		otp, err := NewOTP()
		if err != nil {
			t.Fatalf("NewOTP(): %v", err)
		}
		if len(otp) != 6 {
			t.Errorf("len(otp) = %d, want 6", len(otp))
		}
	})

	t.Run("otp contains only digits", func(t *testing.T) {
		otp, err := NewOTP()
		if err != nil {
			t.Fatalf("NewOTP(): %v", err)
		}
		for _, c := range otp {
			if !unicode.IsDigit(c) {
				t.Errorf("expected only digits, got %q", c)
			}
		}
	})

	t.Run("otp is zero padded", func(t *testing.T) {
		for range 1000 {
			otp, err := NewOTP()
			if err != nil {
				t.Fatalf("NewOTP(): %v", err)
			}
			if len(otp) != 6 {
				t.Errorf("expected padded length 6, got %s", otp)
			}
		}
	})
}

func TestHash(t *testing.T) {
	t.Run("same input produces same hash", func(t *testing.T) {
		h1 := Hash("beebuzz")
		h2 := Hash("beebuzz")
		if h1 != h2 {
			t.Error("expected same hash for same input")
		}
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		if Hash("beebuzz") == Hash("beebuzz2") {
			t.Error("expected different hashes for different inputs")
		}
	})

	t.Run("output is 64 hex characters", func(t *testing.T) {
		h := Hash("beebuzz")
		if len(h) != 64 {
			t.Errorf("len(h) = %d, want 64", len(h))
		}
		_, err := hex.DecodeString(h)
		if err != nil {
			t.Fatalf("hex.DecodeString: %v", err)
		}
	})
}

func TestVerify(t *testing.T) {
	t.Run("correct token verifies successfully", func(t *testing.T) {
		tok, err := newToken(tokenSizeLong)
		if err != nil {
			t.Fatalf("newToken(tokenSizeLong): %v", err)
		}
		if !Verify(tok, Hash(tok)) {
			t.Error("expected Verify to return true")
		}
	})

	t.Run("wrong token does not verify", func(t *testing.T) {
		tok, err := newToken(tokenSizeLong)
		if err != nil {
			t.Fatalf("newToken(tokenSizeLong): %v", err)
		}
		if Verify("wrongtoken", Hash(tok)) {
			t.Error("expected Verify to return false")
		}
	})

	t.Run("hash of different token does not verify", func(t *testing.T) {
		tok1, err := newToken(tokenSizeLong)
		if err != nil {
			t.Fatalf("newToken(tokenSizeLong): %v", err)
		}
		tok2, err := newToken(tokenSizeLong)
		if err != nil {
			t.Fatalf("newToken(tokenSizeLong): %v", err)
		}
		if Verify(tok1, Hash(tok2)) {
			t.Error("expected Verify to return false")
		}
	})
}

func TestPrefixedTokens(t *testing.T) {
	tests := []struct {
		name   string
		fn     func() (string, error)
		prefix string
	}{
		{"API token", NewAPIToken, "beebuzz_api_"},
		{"webhook token", NewWebhookToken, "beebuzz_wh_"},
	}

	for _, tt := range tests {
		t.Run(tt.name+" has correct prefix", func(t *testing.T) {
			tok, err := tt.fn()
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if !strings.HasPrefix(tok, tt.prefix) {
				t.Errorf("expected prefix %q, got %q", tt.prefix, tok)
			}
		})

		t.Run(tt.name+" is unique", func(t *testing.T) {
			tok1, err := tt.fn()
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			tok2, err := tt.fn()
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if tok1 == tok2 {
				t.Errorf("expected two different %s tokens", tt.name)
			}
		})
	}
}

func TestMustRandomHex(t *testing.T) {
	t.Run("returns correct hex length", func(t *testing.T) {
		h := MustRandomHex(16)
		if len(h) != 32 {
			t.Errorf("len(h) = %d, want 32", len(h))
		}
		_, err := hex.DecodeString(h)
		if err != nil {
			t.Fatalf("hex.DecodeString: %v", err)
		}
	})

	t.Run("two calls produce different values", func(t *testing.T) {
		h1 := MustRandomHex(16)
		h2 := MustRandomHex(16)
		if h1 == h2 {
			t.Error("expected two different hex strings")
		}
	})

	t.Run("empty length returns empty string", func(t *testing.T) {
		h := MustRandomHex(0)
		if h != "" {
			t.Errorf("expected empty string, got %q", h)
		}
	})
}
