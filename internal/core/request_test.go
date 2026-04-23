package core

import (
	"strings"
	"testing"
)

func TestDecodeJSONRejectsPayloadTooLarge(t *testing.T) {
	var payload struct {
		Name string `json:"name"`
	}

	body := `{"name":"` + strings.Repeat("a", int(MaxJSONBodyBytes)) + `"}`
	err := DecodeJSON(strings.NewReader(body), &payload)
	if err != ErrPayloadTooLarge {
		t.Fatalf("DecodeJSON: got %v, want %v", err, ErrPayloadTooLarge)
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	var payload struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(strings.NewReader(`{"name":"ok","extra":"nope"}`), &payload)
	if err == nil {
		t.Fatal("DecodeJSON: got nil, want error")
	}
}

func TestDecodeJSONRejectsTrailingData(t *testing.T) {
	var payload struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(strings.NewReader(`{"name":"ok"}{"name":"still here"}`), &payload)
	if err == nil {
		t.Fatal("DecodeJSON: got nil, want error")
	}
}

func TestDecodeJSONAcceptsValidPayload(t *testing.T) {
	var payload struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(strings.NewReader(`{"name":"ok"}`), &payload)
	if err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if payload.Name != "ok" {
		t.Fatalf("payload.Name: got %q, want %q", payload.Name, "ok")
	}
}
