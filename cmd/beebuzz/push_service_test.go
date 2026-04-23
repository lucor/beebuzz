package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"lucor.dev/beebuzz/internal/notification"
	"lucor.dev/beebuzz/internal/push"
)

func TestResolvePushInputUsesStdinForMissingBody(t *testing.T) {
	originalPipedCheck := pipedInputCheck
	defer func() {
		pipedInputCheck = originalPipedCheck
	}()

	pipedInputCheck = func() bool { return true }

	input, err := resolvePushInput([]string{"title"}, strings.NewReader("stdin body"), push.DefaultTopicName, push.PriorityNormal, "", "")
	if err != nil {
		t.Fatalf("resolvePushInput: %v", err)
	}
	if input.Body != "stdin body" {
		t.Fatalf("Body: got %q, want %q", input.Body, "stdin body")
	}
	if input.Priority != push.PriorityNormal {
		t.Fatalf("Priority: got %q, want %q", input.Priority, push.PriorityNormal)
	}
}

func TestResolvePushInputAllowsMissingBody(t *testing.T) {
	input, err := resolvePushInput([]string{"title"}, strings.NewReader(""), push.DefaultTopicName, push.PriorityNormal, "", "")
	if err != nil {
		t.Fatalf("resolvePushInput: %v", err)
	}
	if input.Body != "" {
		t.Fatalf("Body: got %q, want empty", input.Body)
	}
}

func TestResolvePushInputRejectsInvalidPriority(t *testing.T) {
	_, err := resolvePushInput([]string{"title", "body"}, strings.NewReader(""), push.DefaultTopicName, "urgent", "", "")
	if err == nil {
		t.Fatal("expected invalid priority error")
	}
}

func TestResolvePushInputRejectsTitleAboveMaxLength(t *testing.T) {
	_, err := resolvePushInput([]string{strings.Repeat("a", notification.MaxNotificationTitleLen+1)}, strings.NewReader(""), push.DefaultTopicName, push.PriorityNormal, "", "")
	if err == nil {
		t.Fatal("expected title max length error")
	}
}

func TestResolvePushInputRejectsBodyAboveMaxLength(t *testing.T) {
	_, err := resolvePushInput([]string{"title", strings.Repeat("a", notification.MaxNotificationBodyLen+1)}, strings.NewReader(""), push.DefaultTopicName, push.PriorityNormal, "", "")
	if err == nil {
		t.Fatal("expected body max length error")
	}
}

func TestResolvePushInputRejectsOversizePipedBody(t *testing.T) {
	originalPipedCheck := pipedInputCheck
	defer func() {
		pipedInputCheck = originalPipedCheck
	}()
	pipedInputCheck = func() bool { return true }

	oversizeInput := strings.Repeat("a", maxBodyStdinBytes+1024)
	_, err := resolvePushInput([]string{"title"}, strings.NewReader(oversizeInput), push.DefaultTopicName, push.PriorityNormal, "", "")
	if err == nil {
		t.Fatal("expected body max length error")
	}
	if !strings.Contains(err.Error(), "body must be") {
		t.Fatalf("error: got %q", err)
	}
}

func TestResolvePushInputKeepsAttachmentPathVerbatim(t *testing.T) {
	input, err := resolvePushInput([]string{"title"}, strings.NewReader(""), push.DefaultTopicName, push.PriorityNormal, "@/tmp/attachment.txt", "")
	if err != nil {
		t.Fatalf("resolvePushInput: %v", err)
	}
	if input.AttachmentPath != "@/tmp/attachment.txt" {
		t.Fatalf("AttachmentPath: got %q, want %q", input.AttachmentPath, "@/tmp/attachment.txt")
	}
}

func TestResolvePushInputPassesFlagValues(t *testing.T) {
	input, err := resolvePushInput([]string{"title", "body"}, strings.NewReader(""), "alerts", push.PriorityHigh, "note.txt", "https://api.example.com")
	if err != nil {
		t.Fatalf("resolvePushInput: %v", err)
	}
	if input.Title != "title" {
		t.Fatalf("Title: got %q, want %q", input.Title, "title")
	}
	if input.Body != "body" {
		t.Fatalf("Body: got %q, want %q", input.Body, "body")
	}
	if input.Topic != "alerts" {
		t.Fatalf("Topic: got %q, want %q", input.Topic, "alerts")
	}
	if input.Priority != push.PriorityHigh {
		t.Fatalf("Priority: got %q, want %q", input.Priority, push.PriorityHigh)
	}
	if input.AttachmentPath != "note.txt" {
		t.Fatalf("AttachmentPath: got %q, want %q", input.AttachmentPath, "note.txt")
	}
	if input.APIURL != "https://api.example.com" {
		t.Fatalf("APIURL: got %q, want %q", input.APIURL, "https://api.example.com")
	}
}

func TestResolvePushInputUsesDefaultTopicWhenEmpty(t *testing.T) {
	input, err := resolvePushInput([]string{"title"}, strings.NewReader(""), "   ", push.PriorityNormal, "", "")
	if err != nil {
		t.Fatalf("resolvePushInput: %v", err)
	}
	if input.Topic != push.DefaultTopicName {
		t.Fatalf("Topic: got %q, want %q", input.Topic, push.DefaultTopicName)
	}
}

func TestResolvePushInputRejectsTooManyPositionals(t *testing.T) {
	_, err := resolvePushInput([]string{"title", "body", "extra"}, strings.NewReader(""), push.DefaultTopicName, push.PriorityNormal, "", "")
	if err == nil {
		t.Fatal("expected too many positional arguments error")
	}
}

func TestPushNotificationPostsDecryptableAgeBlob(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity: %v", err)
	}

	var requestBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pushEndpointBasePath+"alerts" {
			t.Fatalf("path: got %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer beebuzz_token" {
			t.Fatalf("authorization: got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Fatalf("content-type: got %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get(push.PriorityHeader) != push.PriorityHigh {
			t.Fatalf("priority: got %q", r.Header.Get(push.PriorityHeader))
		}

		requestBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"sent_count":1,"total_count":1,"failed_count":0,"device_keys":[{"device_id":"dev-a","device_name":"phone","paired_at":"2026-04-11T10:00:00Z","age_recipient":"` + identity.Recipient().String() + `","age_recipient_fingerprint":"fpabc"}]}`))
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
	}))
	defer server.Close()

	config := &Config{
		APIURL:   server.URL,
		APIToken: "beebuzz_token",
		DeviceKeys: []DeviceKey{
			{AgeRecipient: identity.Recipient().String()},
		},
	}

	response, err := pushNotification(context.Background(), server.Client(), config, PushInput{
		Title:    "Build failed",
		Body:     "Disk full",
		Topic:    "alerts",
		Priority: push.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("pushNotification: %v", err)
	}
	if response.SentCount != 1 || response.TotalCount != 1 || response.FailedCount != 0 {
		t.Fatalf("response: got %#v", response)
	}

	reader, err := age.Decrypt(bytes.NewReader(requestBody), identity)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	plaintext, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll plaintext: %v", err)
	}

	var payload EncryptedNotificationPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		t.Fatalf("Unmarshal payload: %v", err)
	}
	if payload.Title != "Build failed" || payload.Body != "Disk full" || payload.Topic != "alerts" {
		t.Fatalf("payload: got %#v", payload)
	}
}

func TestBuildEncryptedPayloadIncludesAttachment(t *testing.T) {
	attachmentPath := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(attachmentPath, []byte("hello attachment"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	payloadBytes, err := buildEncryptedPayload(PushInput{
		Title:          "Build failed",
		Body:           "see file",
		Topic:          "alerts",
		AttachmentPath: attachmentPath,
	})
	if err != nil {
		t.Fatalf("buildEncryptedPayload: %v", err)
	}

	var payload EncryptedNotificationPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if payload.Attachment == nil {
		t.Fatal("expected attachment payload")
	}
	if payload.Attachment.Filename != "note.txt" {
		t.Fatalf("Filename: got %q, want %q", payload.Attachment.Filename, "note.txt")
	}
	if payload.Attachment.MIME != "text/plain; charset=utf-8" {
		t.Fatalf("MIME: got %q", payload.Attachment.MIME)
	}
	if payload.Attachment.Data == "" {
		t.Fatal("expected base64 attachment data")
	}
}

func TestBuildEncryptedPayloadRejectsAttachmentAboveMaxSize(t *testing.T) {
	attachmentPath := filepath.Join(t.TempDir(), "large.bin")
	attachmentData := bytes.Repeat([]byte("a"), maxAttachmentBytes+1)
	if err := os.WriteFile(attachmentPath, attachmentData, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := buildEncryptedPayload(PushInput{
		Title:          "Build failed",
		Body:           "see file",
		Topic:          "alerts",
		AttachmentPath: attachmentPath,
	})
	if err == nil {
		t.Fatal("expected attachment size error")
	}
	if !strings.Contains(err.Error(), "attachment exceeds") {
		t.Fatalf("error: got %q", err)
	}
}

func TestBuildEncryptedPayloadAllowsEmptyBody(t *testing.T) {
	payloadBytes, err := buildEncryptedPayload(PushInput{
		Title: "Build failed",
		Topic: "alerts",
	})
	if err != nil {
		t.Fatalf("buildEncryptedPayload: %v", err)
	}

	var payload EncryptedNotificationPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if payload.Body != "" {
		t.Fatalf("Body: got %q, want empty", payload.Body)
	}
}

func TestPushNotificationReturnsRotatedKeysWithoutMutatingConfig(t *testing.T) {
	identityA, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity A: %v", err)
	}
	identityB, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity B: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, writeErr := w.Write([]byte(`{"sent_count":1,"total_count":1,"failed_count":0,"device_keys":[{"device_id":"dev-b","device_name":"laptop","paired_at":"2026-04-11T10:00:00Z","age_recipient":"` + identityB.Recipient().String() + `","age_recipient_fingerprint":"fpdef"}]}`))
		if writeErr != nil {
			t.Fatalf("Write: %v", writeErr)
		}
	}))
	defer server.Close()

	config := &Config{
		APIURL:   server.URL,
		APIToken: "beebuzz_token",
		DeviceKeys: []DeviceKey{
			{AgeRecipient: identityA.Recipient().String()},
		},
	}

	response, err := pushNotification(context.Background(), server.Client(), config, PushInput{
		Title: "Build failed",
		Body:  "Disk full",
		Topic: "alerts",
	})
	if err != nil {
		t.Fatalf("pushNotification: %v", err)
	}

	if len(response.DeviceKeys) != 1 || response.DeviceKeys[0].AgeRecipient != identityB.Recipient().String() {
		t.Fatalf("response keys: got %#v", response.DeviceKeys)
	}
	if len(config.DeviceKeys) != 1 || config.DeviceKeys[0].AgeRecipient != identityA.Recipient().String() {
		t.Fatalf("config keys mutated: got %#v", config.DeviceKeys)
	}
}

func TestPushNotificationReturnsHTTPError(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()

	config := &Config{
		APIURL:   server.URL,
		APIToken: "beebuzz_token",
		DeviceKeys: []DeviceKey{
			{AgeRecipient: identity.Recipient().String()},
		},
	}

	_, err = pushNotification(context.Background(), server.Client(), config, PushInput{
		Title: "Build failed",
		Body:  "Disk full",
		Topic: "alerts",
	})
	if err == nil {
		t.Fatal("expected push request error")
	}
	if !strings.Contains(err.Error(), "push request failed with status 401") {
		t.Fatalf("error: got %q", err)
	}
}

func TestAppRunSendUsesAPIURLFlagOverrideOverEnvAndConfig(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity: %v", err)
	}

	t.Setenv(envBeeBuzzAPIURL, "https://env.example.com")

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, profilesDirName, fallbackProfileName, configFileName)
	store := newProfileStore(func() (string, error) { return configBasePath() })
	if err := store.saveConfigToPath(configPath, &Config{
		APIURL:   "https://file.example.com",
		APIToken: "beebuzz_token",
		DeviceKeys: []DeviceKey{
			{AgeRecipient: identity.Recipient().String()},
		},
	}); err != nil {
		t.Fatalf("saveConfigToPath: %v", err)
	}

	originalBasePath := configBasePath
	defer func() {
		configBasePath = originalBasePath
	}()
	configBasePath = func() (string, error) { return baseDir, nil }

	requestHit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestHit = true
		if r.URL.Path != pushEndpointBasePath+"alerts" {
			t.Fatalf("path: got %q, want %q", r.URL.Path, pushEndpointBasePath+"alerts")
		}
		if r.Header.Get("Authorization") != "Bearer beebuzz_token" {
			t.Fatalf("authorization: got %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, writeErr := w.Write([]byte(`{"sent_count":1,"total_count":1,"failed_count":0,"device_keys":[{"device_id":"dev-a","device_name":"phone","paired_at":"2026-04-11T10:00:00Z","age_recipient":"` + identity.Recipient().String() + `","age_recipient_fingerprint":"fpabc"}]}`))
		if writeErr != nil {
			t.Fatalf("Write: %v", writeErr)
		}
	}))
	defer server.Close()

	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	app.httpClient = server.Client()

	if err := app.Run([]string{"send", "--topic", "alerts", "--api-url", server.URL, "Build failed"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !requestHit {
		t.Fatal("expected request to api-url flag server")
	}
}

func TestWriteKeyRefreshSummaryWritesDiff(t *testing.T) {
	output := &bytes.Buffer{}

	err := writeKeyRefreshSummary(output,
		[]DeviceKey{
			{DeviceName: "old phone", AgeRecipient: "age1old", AgeRecipientFingerprint: "oldfp"},
			{DeviceName: "keep", AgeRecipient: "age1keep", AgeRecipientFingerprint: "keepfp"},
		},
		[]DeviceKey{
			{DeviceName: "keep", AgeRecipient: "age1keep", AgeRecipientFingerprint: "keepfp"},
			{DeviceName: "new laptop", AgeRecipient: "age1new", AgeRecipientFingerprint: "newfp"},
		},
	)
	if err != nil {
		t.Fatalf("writeKeyRefreshSummary: %v", err)
	}

	if got := output.String(); got != "warning: device keys changed (1 added, 1 removed)\nadded: new laptop [newfp] age1new\nremoved: old phone [oldfp] age1old\n" {
		t.Fatalf("output: got %q", got)
	}
}

func TestWriteKeyRefreshSummarySkipsUnchangedKeys(t *testing.T) {
	output := &bytes.Buffer{}

	err := writeKeyRefreshSummary(output,
		[]DeviceKey{{DeviceName: "same", AgeRecipient: "age1same", AgeRecipientFingerprint: "samefp"}},
		[]DeviceKey{{DeviceName: "same", AgeRecipient: "age1same", AgeRecipientFingerprint: "samefp"}},
	)
	if err != nil {
		t.Fatalf("writeKeyRefreshSummary: %v", err)
	}

	if output.Len() != 0 {
		t.Fatalf("unexpected output: %q", output.String())
	}
}

func TestSummarizeRecipientTruncatesLongKeys(t *testing.T) {
	got := summarizeRecipient("age1abcdefghijklmnop")
	if got != "age1abcd...ijklmnop" {
		t.Fatalf("got %q", got)
	}
}
