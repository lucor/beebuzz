package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"lucor.dev/beebuzz/internal/attachment"
	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/device"
	"lucor.dev/beebuzz/internal/middleware"
	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/token"
	"lucor.dev/beebuzz/internal/topic"
)

const testAgeRecipient = "age1seaxfh0rsf6z4y0j2rc0h6j8x0nafpdkttkjxc3a3ka7r3y49g7s6sn0ur"

var errTestSendFailure = errors.New("send failed")

// testPushAuthorizer wraps token.Service to implement PushAuthorizer for tests.
type testPushAuthorizer struct {
	svc *token.Service
}

// ValidateAPIToken delegates to the real token service.
func (a *testPushAuthorizer) ValidateAPIToken(ctx context.Context, rawToken string) (string, error) {
	return a.svc.ValidateAPIToken(ctx, rawToken)
}

// ValidateAPITokenForTopic delegates to the real token service.
func (a *testPushAuthorizer) ValidateAPITokenForTopic(ctx context.Context, rawToken, topicName string) (string, string, error) {
	return a.svc.ValidateAPITokenForTopic(ctx, rawToken, topicName)
}

// testDeviceProvider wraps device.Service to implement DeviceProvider for tests.
type testDeviceProvider struct {
	svc *device.Service
}

// GetSubscribedDevices delegates to the real device service.
func (p *testDeviceProvider) GetSubscribedDevices(ctx context.Context, userID, topicName string) ([]PushSub, error) {
	subs, err := p.svc.GetSubscribedDevices(ctx, userID, topicName)
	if err != nil {
		return nil, err
	}
	result := make([]PushSub, len(subs))
	for i, s := range subs {
		result[i] = PushSub{
			DeviceID:     s.DeviceID,
			Endpoint:     s.Endpoint,
			P256dh:       s.P256dh,
			Auth:         s.Auth,
			AgeRecipient: s.AgeRecipient,
		}
	}
	return result, nil
}

// MarkSubscriptionGone delegates to the real device service.
func (p *testDeviceProvider) MarkSubscriptionGone(ctx context.Context, deviceID string) error {
	return p.svc.MarkSubscriptionGone(ctx, deviceID)
}

// testKeyProvider wraps device.Service to implement KeyProvider for tests.
type testKeyProvider struct {
	svc *device.Service
}

// GetDeviceKeys delegates to the real device service.
func (p *testKeyProvider) GetDeviceKeys(ctx context.Context, userID string) ([]device.DeviceKeyDescriptor, error) {
	return p.svc.GetDeviceKeysByUser(ctx, userID)
}

// testAttachmentStorer wraps attachment.Service to implement AttachmentStorer for tests.
type testAttachmentStorer struct {
	svc *attachment.Service
}

// Store delegates to the real attachment service.
func (a *testAttachmentStorer) Store(ctx context.Context, topicID, mimeType string, originalSize int, data []byte) (string, error) {
	return a.svc.Store(ctx, topicID, mimeType, originalSize, data)
}

type stubSender struct {
	report *SendReport
	err    error
}

func (s *stubSender) Send(_ context.Context, _, _ string, _ SendInput, _ *slog.Logger) (*SendReport, error) {
	return s.report, s.err
}

func (s *stubSender) VAPIDPublicKey() string {
	return ""
}

// buildHandler creates a fully wired Handler backed by an in-memory DB.
// Returns the handler, a valid raw API token, and the topic name it is authorized for.
func buildHandler(t *testing.T) (*Handler, string, string) {
	t.Helper()

	db := testutil.NewDB(t)
	ctx := context.Background()
	log := slog.Default()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := token.NewRepository(db)
	deviceRepo := device.NewRepository(db)
	attachmentRepo := attachment.NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "handler-test@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("NewAPIToken: %v", err)
	}

	tokenID, err := tokenRepo.CreateAPIToken(ctx, user.ID, "handler-test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if err := tokenRepo.AddTopicToAPIToken(ctx, tokenID, tp.ID); err != nil {
		t.Fatalf("AddTopicToAPIToken: %v", err)
	}

	topicSvc := topic.NewService(topicRepo, log)
	tokenSvc := token.NewService(tokenRepo, topicSvc)
	deviceSvc := device.NewService(deviceRepo, topicSvc, log)
	attachmentSvc := attachment.NewService(attachmentRepo, t.TempDir(), log)

	vapidKeys := &VAPIDKeys{
		PublicKey:  "BCEHjfHghF_wV2jjrwKaRCgvIJfN0Fzb-gDFfuXKYzE",
		PrivateKey: "test-private-key",
	}

	svc := NewService(
		&testDeviceProvider{svc: deviceSvc},
		&testAttachmentStorer{svc: attachmentSvc},
		nil, // no tracker in tests
		vapidKeys,
		"mailto:test@example.com",
		log,
	)
	handler := NewHandler(svc, &testPushAuthorizer{svc: tokenSvc}, &testKeyProvider{svc: deviceSvc}, log)

	return handler, rawToken, "alerts"
}

func buildHandlerWithPairedDevice(t *testing.T) (*Handler, string, string) {
	t.Helper()

	db := testutil.NewDB(t)
	ctx := context.Background()
	log := slog.Default()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := token.NewRepository(db)
	deviceRepo := device.NewRepository(db)
	attachmentRepo := attachment.NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "handler-paired-test@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("NewAPIToken: %v", err)
	}

	tokenID, err := tokenRepo.CreateAPIToken(ctx, user.ID, "handler-test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if err := tokenRepo.AddTopicToAPIToken(ctx, tokenID, tp.ID); err != nil {
		t.Fatalf("AddTopicToAPIToken: %v", err)
	}

	seedPairedDevice(t, ctx, db, user.ID, tp.ID, testAgeRecipient)

	topicSvc := topic.NewService(topicRepo, log)
	tokenSvc := token.NewService(tokenRepo, topicSvc)
	deviceSvc := device.NewService(deviceRepo, topicSvc, log)
	attachmentSvc := attachment.NewService(attachmentRepo, t.TempDir(), log)

	vapidKeys := &VAPIDKeys{
		PublicKey:  "BCEHjfHghF_wV2jjrwKaRCgvIJfN0Fzb-gDFfuXKYzE",
		PrivateKey: "test-private-key",
	}

	svc := NewService(
		&testDeviceProvider{svc: deviceSvc},
		&testAttachmentStorer{svc: attachmentSvc},
		nil,
		vapidKeys,
		"mailto:test@example.com",
		log,
	)
	handler := NewHandler(svc, &testPushAuthorizer{svc: tokenSvc}, &testKeyProvider{svc: deviceSvc}, log)

	return handler, rawToken, "alerts"
}

func buildHandlerWithSender(t *testing.T, sender Sender) (*Handler, string, string) {
	t.Helper()

	db := testutil.NewDB(t)
	ctx := context.Background()
	log := slog.Default()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := token.NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "handler-sender-test@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("NewAPIToken: %v", err)
	}

	tokenID, err := tokenRepo.CreateAPIToken(ctx, user.ID, "handler-sender-test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if err := tokenRepo.AddTopicToAPIToken(ctx, tokenID, tp.ID); err != nil {
		t.Fatalf("AddTopicToAPIToken: %v", err)
	}

	topicSvc := topic.NewService(topicRepo, log)
	tokenSvc := token.NewService(tokenRepo, topicSvc)
	handler := NewHandler(sender, &testPushAuthorizer{svc: tokenSvc}, nil, log)

	return handler, rawToken, "alerts"
}

// withTopic sets the chi URL param "topic" on the request.
func withTopic(r *http.Request, topic string) *http.Request {
	return r.WithContext(testutil.WithRouteParams(r.Context(), map[string]string{"topic": topic}))
}

// withBearer runs the request through ExtractBearerToken middleware so
// the raw token is available in the request context for the handler.
func withBearer(r *http.Request) *http.Request {
	var out *http.Request
	middleware.ExtractBearerToken(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		out = r
	})).ServeHTTP(httptest.NewRecorder(), r)
	return out
}

// seedPairedDevice inserts a paired device and push subscription for the given user/topic.
func seedPairedDevice(t *testing.T, ctx context.Context, db *sqlx.DB, userID, topicID string, ageRecipient string) {
	t.Helper()

	deviceID := "device-test-id"
	deviceName := "test-device"
	description := "paired device"
	endpoint := "https://example.com/push"
	p256dh := "test-p256dh"
	authKey := "test-auth"
	now := time.Now().UnixMilli()

	if _, err := db.ExecContext(ctx,
		`INSERT INTO devices (id, user_id, name, description, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 1, ?, ?)`,
		deviceID, userID, deviceName, description, now, now,
	); err != nil {
		t.Fatalf("insert device: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`INSERT INTO device_topics (device_id, topic_id, created_at) VALUES (?, ?, ?)`,
		deviceID, topicID, now,
	); err != nil {
		t.Fatalf("insert device topic: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`INSERT INTO push_subscriptions (device_id, endpoint, p256dh, auth, age_recipient, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		deviceID, endpoint, p256dh, authKey, ageRecipient, now, now,
	); err != nil {
		t.Fatalf("insert push subscription: %v", err)
	}
}

func TestSendHandler_MissingAuthHeader(t *testing.T) {
	handler, _, _ := buildHandler(t)

	body := `{"title":"Test","body":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTopic(req, "alerts")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp["code"] != codeMissingToken {
		t.Errorf("code: got %q, want %q", resp["code"], codeMissingToken)
	}
}

func TestSendHandler_NoBearerPrefix(t *testing.T) {
	handler, rawToken, _ := buildHandler(t)

	body := `{"title":"Test","body":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", rawToken) // missing "Bearer " prefix
	req = withTopic(req, "alerts")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestSendHandler_EmptyBearerToken(t *testing.T) {
	handler, _, _ := buildHandler(t)

	body := `{"title":"Test","body":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ") // prefix present but token empty
	req = withBearer(req)
	req = withTopic(req, "alerts")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp["code"] != codeMissingToken {
		t.Errorf("code: got %q, want %q", resp["code"], codeMissingToken)
	}
}

func TestSendHandler_InvalidJSON(t *testing.T) {
	handler, rawToken, _ := buildHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/push/alerts", bytes.NewBufferString("{bad json}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, "alerts")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp["code"] != codeInvalidJSON {
		t.Errorf("code: got %q, want %q", resp["code"], codeInvalidJSON)
	}
}

func TestSendHandler_JSONPayloadTooLarge(t *testing.T) {
	handler, rawToken, topicName := buildHandler(t)

	body := `{"title":"` + strings.Repeat("a", int(core.MaxJSONBodyBytes)) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] != "payload_too_large" {
		t.Fatalf("code: got %q, want %q", resp["code"], "payload_too_large")
	}
}

func TestSendHandler_MissingTitle(t *testing.T) {
	handler, rawToken, _ := buildHandler(t)

	body := `{"body":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, "alerts")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestSendHandler_InvalidPriority(t *testing.T) {
	handler, rawToken, _ := buildHandler(t)

	body := `{"title":"Test","body":"Hello","priority":"ultra"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, "alerts")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestSendHandler_TitleTooLong(t *testing.T) {
	handler, rawToken, _ := buildHandler(t)

	body := `{"title":"` + strings.Repeat("a", MaxNotificationTitleLen+1) + `","body":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, "alerts")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}

	var resp struct {
		Code   string   `json:"code"`
		Errors []string `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != "validation_error" {
		t.Fatalf("code: got %q, want %q", resp.Code, "validation_error")
	}
	if len(resp.Errors) != 1 || resp.Errors[0] != "title: must be 64 characters or less" {
		t.Fatalf("errors: got %#v", resp.Errors)
	}
}

func TestSendHandler_BodyTooLong(t *testing.T) {
	handler, rawToken, _ := buildHandler(t)

	body := `{"title":"Test","body":"` + strings.Repeat("a", MaxNotificationBodyLen+1) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, "alerts")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}

	var resp struct {
		Code   string   `json:"code"`
		Errors []string `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != "validation_error" {
		t.Fatalf("code: got %q, want %q", resp.Code, "validation_error")
	}
	if len(resp.Errors) != 1 || resp.Errors[0] != "body: must be 256 characters or less" {
		t.Fatalf("errors: got %#v", resp.Errors)
	}
}

func TestSendHandler_InvalidToken(t *testing.T) {
	handler, _, _ := buildHandler(t)

	body := `{"title":"Test","body":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer beebuzz_api_invalid000000000000000000000000")
	req = withBearer(req)
	req = withTopic(req, "alerts")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp["code"] != codeUnauthorized {
		t.Errorf("code: got %q, want %q", resp["code"], codeUnauthorized)
	}
}

func TestSendHandler_TokenNotAuthorizedForTopic(t *testing.T) {
	handler, rawToken, _ := buildHandler(t)

	// Token is authorized for "alerts" — use "general" to trigger auth failure.
	body := `{"title":"Test","body":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/push/general", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, "general")
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestSendHandler_ValidToken_NoSubscriptions(t *testing.T) {
	handler, rawToken, topicName := buildHandler(t)

	body, _ := json.Marshal(map[string]string{
		"title": "Test",
		"body":  "Hello",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d — body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp SendResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// No subscriptions in DB, so all counts are 0.
	if resp.TotalCount != 0 || resp.SentCount != 0 || resp.FailedCount != 0 {
		t.Errorf("counts: got sent=%d total=%d failed=%d, want 0/0/0", resp.SentCount, resp.TotalCount, resp.FailedCount)
	}
	if resp.Status != SendStatusDelivered {
		t.Fatalf("status: got %q, want %q", resp.Status, SendStatusDelivered)
	}
}

func TestSendHandler_ReturnsPartialStatusWhenSomeDeliveriesFail(t *testing.T) {
	handler, rawToken, topicName := buildHandlerWithSender(t, &stubSender{
		report: &SendReport{
			DeviceResults: []DeviceResult{{DeviceID: "device-1"}, {DeviceID: "device-2", Err: errTestSendFailure}},
			TotalSent:     1,
			TotalFailed:   1,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, bytes.NewBufferString(`{"title":"Test","body":"Hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp SendResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != SendStatusPartial {
		t.Fatalf("status: got %q, want %q", resp.Status, SendStatusPartial)
	}
	if resp.SentCount != 1 || resp.TotalCount != 2 || resp.FailedCount != 1 {
		t.Fatalf("counts: got sent=%d total=%d failed=%d, want 1/2/1", resp.SentCount, resp.TotalCount, resp.FailedCount)
	}
}

func TestSendHandler_ReturnsBadGatewayWhenAllDeliveriesFail(t *testing.T) {
	handler, rawToken, topicName := buildHandlerWithSender(t, &stubSender{
		report: &SendReport{
			DeviceResults: []DeviceResult{
				{DeviceID: "device-1", Err: errTestSendFailure},
				{DeviceID: "device-2", Err: errTestSendFailure},
			},
			TotalSent:   0,
			TotalFailed: 2,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, bytes.NewBufferString(`{"title":"Test","body":"Hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusBadGateway, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] != "push_delivery_failed" {
		t.Fatalf("code: got %q, want %q", resp["code"], "push_delivery_failed")
	}
}

func TestSendHandler_ReturnsUnprocessableEntityWhenAttachmentProcessingFails(t *testing.T) {
	handler, rawToken, topicName := buildHandlerWithSender(t, &stubSender{
		err: ErrAttachmentProcessingFailed,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, bytes.NewBufferString(`{"title":"Test","body":"Hello","attachment_url":"https://example.com/file.png"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] != "attachment_processing_failed" {
		t.Fatalf("code: got %q, want %q", resp["code"], "attachment_processing_failed")
	}
}

func TestKeysHandler_ReturnsPairedDeviceKeys(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	log := slog.Default()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := token.NewRepository(db)
	deviceRepo := device.NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "keys-test@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("NewAPIToken: %v", err)
	}

	tokenID, err := tokenRepo.CreateAPIToken(ctx, user.ID, "keys-test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if err := tokenRepo.AddTopicToAPIToken(ctx, tokenID, tp.ID); err != nil {
		t.Fatalf("AddTopicToAPIToken: %v", err)
	}

	topicSvc := topic.NewService(topicRepo, log)
	tokenSvc := token.NewService(tokenRepo, topicSvc)
	deviceSvc := device.NewService(deviceRepo, topicSvc, log)

	seedPairedDevice(t, ctx, db, user.ID, tp.ID, testAgeRecipient)

	handler := NewHandler(
		nil,
		&testPushAuthorizer{svc: tokenSvc},
		&testKeyProvider{svc: deviceSvc},
		log,
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/push/keys", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	w := httptest.NewRecorder()

	handler.Keys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp KeysResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("keys len: got %d, want 1", len(resp.Data))
	}
	if resp.Data[0].AgeRecipient != testAgeRecipient {
		t.Fatalf("key: got %q, want %q", resp.Data[0].AgeRecipient, testAgeRecipient)
	}
	if resp.Data[0].AgeRecipientFingerprint == "" {
		t.Fatal("fingerprint: got empty value")
	}
}

func TestSendHandler_OctetStreamEmptyBody(t *testing.T) {
	handler, rawToken, topicName := buildHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, http.NoBody)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}

	var resp struct {
		Code   string   `json:"code"`
		Errors []string `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != "validation_error" {
		t.Fatalf("code: got %q, want %q", resp.Code, "validation_error")
	}
	if len(resp.Errors) != 1 || resp.Errors[0] != "body: is required" {
		t.Fatalf("errors: got %#v, want [\"body: is required\"]", resp.Errors)
	}
}

func TestSendHandler_OctetStreamPayloadTooLarge(t *testing.T) {
	handler, rawToken, topicName := buildHandler(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/push/"+topicName,
		bytes.NewReader(bytes.Repeat([]byte("a"), maxAttachmentBytes+1)),
	)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] != "payload_too_large" {
		t.Fatalf("code: got %q, want %q", resp["code"], "payload_too_large")
	}
}

func TestSendHandler_MultipartPayloadTooLarge(t *testing.T) {
	handler, rawToken, topicName := buildHandlerWithPairedDevice(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "Test"); err != nil {
		t.Fatalf("WriteField title: %v", err)
	}
	part, err := writer.CreateFormFile("attachment", "large.bin")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(bytes.Repeat([]byte("a"), maxAttachmentBytes+1)); err != nil {
		t.Fatalf("part.Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] != "payload_too_large" {
		t.Fatalf("code: got %q, want %q", resp["code"], "payload_too_large")
	}
}

func TestSendHandler_JSONAllowsMissingBody(t *testing.T) {
	handler, rawToken, topicName := buildHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, bytes.NewBufferString(`{"title":"Test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// sourceCaptureSender is a test double that records the SendInput it receives.
type sourceCaptureSender struct {
	captured SendInput
	report   *SendReport
	err      error
}

func (s *sourceCaptureSender) Send(_ context.Context, _, _ string, input SendInput, _ *slog.Logger) (*SendReport, error) {
	s.captured = input
	return s.report, s.err
}

func (s *sourceCaptureSender) VAPIDPublicKey() string {
	return ""
}

func TestSendHandler_SourceCLI(t *testing.T) {
	sender := &sourceCaptureSender{report: &SendReport{}}
	handler, rawToken, topicName := buildHandlerWithSender(t, sender)

	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, bytes.NewBufferString(`{"title":"Test","body":"Hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req.Header.Set("User-Agent", core.CLIUserAgentPrefix+"/1.0.0")
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if sender.captured.Source != "cli" {
		t.Fatalf("source: got %q, want %q", sender.captured.Source, "cli")
	}
}

func TestSendHandler_SourceAPI(t *testing.T) {
	sender := &sourceCaptureSender{report: &SendReport{}}
	handler, rawToken, topicName := buildHandlerWithSender(t, sender)

	req := httptest.NewRequest(http.MethodPost, "/v1/push/"+topicName, bytes.NewBufferString(`{"title":"Test","body":"Hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req = withBearer(req)
	req = withTopic(req, topicName)
	w := httptest.NewRecorder()

	handler.Send(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if sender.captured.Source != "api" {
		t.Fatalf("source: got %q, want %q", sender.captured.Source, "api")
	}
}
