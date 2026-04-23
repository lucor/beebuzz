package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MaxJSONBodyBytes caps JSON request bodies globally because BeeBuzz JSON payloads
// are metadata-only; large binary inputs must use multipart or octet-stream paths.
const MaxJSONBodyBytes int64 = 256 * 1024

// DecodeJSON enforces BeeBuzz's shared JSON request policy:
// bounded body size, unknown-field rejection, and trailing-data rejection.
// Oversized bodies are mapped to ErrPayloadTooLarge even when the JSON stream
// is truncated mid-decode by the shared size cap.
func DecodeJSON(body io.Reader, v any) error {
	lr := &io.LimitedReader{
		R: body,
		N: MaxJSONBodyBytes + 1,
	}

	dec := json.NewDecoder(lr)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		if lr.N == 0 {
			return ErrPayloadTooLarge
		}
		return err
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if lr.N == 0 {
			return ErrPayloadTooLarge
		}
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON data")
		}
		return err
	}

	if lr.N == 0 {
		return ErrPayloadTooLarge
	}

	return nil
}

// GetURLParam reads a named chi route parameter through a single shared helper
// so handlers do not depend directly on router-specific access patterns.
func GetURLParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}
