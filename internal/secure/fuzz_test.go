package secure

import (
	"strings"
	"testing"
)

// FuzzHash verifies that Hash never panics on arbitrary input
// and always returns a 64-character hex string.
func FuzzHash(f *testing.F) {
	f.Add("")
	f.Add("beebuzz")
	f.Add("beebuzz_api_abc123")
	f.Add(strings.Repeat("a", 10000))
	f.Add("\x00\x01\x02\xff")
	f.Add("日本語テスト")

	f.Fuzz(func(t *testing.T, input string) {
		h := Hash(input)

		if len(h) != 64 {
			t.Fatalf("Hash output length = %d, want 64", len(h))
		}
	})
}

// FuzzVerify verifies the self-verify invariant and that Verify never panics.
func FuzzVerify(f *testing.F) {
	f.Add("abc", Hash("abc"))
	f.Add("", Hash(""))
	f.Add("123456", Hash("123456"))
	f.Add("beebuzz_api_token", "deadbeef")
	f.Add("\x00\xff", "not-a-hash")
	f.Add(strings.Repeat("x", 10000), strings.Repeat("f", 64))
	f.Add("abc", "")                       // empty hash
	f.Add("abc", strings.Repeat("a", 63))  // off-by-one short
	f.Add("abc", strings.Repeat("a", 65))  // off-by-one long
	f.Add("abc", strings.ToUpper(Hash("abc"))) // uppercase hex (case sensitivity)
	f.Add("abc", Hash("abd"))              // same shape, wrong content

	f.Fuzz(func(t *testing.T, token, hash string) {
		// Must never panic.
		_ = Verify(token, hash)

		// Self-verify invariant: Hash(token) must always verify against itself.
		if !Verify(token, Hash(token)) {
			t.Fatalf("self-verify failed for token %q", token)
		}
	})
}
