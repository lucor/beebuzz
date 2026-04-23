package device

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/middleware"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

func newTestDeviceHandler(t *testing.T) *Handler {
	t.Helper()

	db := testutil.NewDB(t)
	repo := NewRepository(db)
	topicRepo := topic.NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := NewService(repo, topicSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	return NewHandler(svc, "https://hive.example.com", slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestCreateDeviceRejectsEmptyTopics(t *testing.T) {
	handler := newTestDeviceHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewBufferString(`{"name":"phone","description":"desc","topics":[]}`))
	req = req.WithContext(testutil.WithUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	handler.CreateDevice(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateDevice() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestCreateDeviceRejectsForeignTopicSelection(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := NewService(repo, topicSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewHandler(svc, "https://hive.example.com", slog.New(slog.NewTextHandler(io.Discard, nil)))

	owner, _, err := authRepo.GetOrCreateUser(ctx, "device-handler-owner@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser owner: %v", err)
	}
	other, _, err := authRepo.GetOrCreateUser(ctx, "device-handler-other@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser other: %v", err)
	}
	otherTopic, err := topicRepo.Create(ctx, other.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	body, err := json.Marshal(CreateDeviceRequest{
		Name:        "phone",
		Description: "desc",
		Topics:      []string{otherTopic.ID},
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(body))
	req = req.WithContext(testutil.WithUserContext(req.Context(), owner.ID))
	w := httptest.NewRecorder()

	handler.CreateDevice(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateDevice() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestCreateDeviceRejectsDuplicateTopics(t *testing.T) {
	handler := newTestDeviceHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewBufferString(`{"name":"phone","description":"desc","topics":["topic-1","topic-1"]}`))
	req = req.WithContext(testutil.WithUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	handler.CreateDevice(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateDevice() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestHandlerPairRejectsUnsupportedPushEndpoint(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	topicSvc := topic.NewService(topicRepo, logger)
	svc := NewService(repo, topicSvc, logger)
	handler := NewHandler(svc, "https://hive.example.com", logger)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-pair-handler@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}
	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}
	_, otp, _, err := svc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	reqBody := `{"pairing_code":"` + otp + `","endpoint":"https://example.com/push","p256dh":"p256dh","auth":"auth","age_recipient":"age1recipient"}`
	req := httptest.NewRequest(http.MethodPost, "/pairing", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.Pair(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Pair() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestHandlerPairRejectsInvalidAgeRecipient(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	topicSvc := topic.NewService(topicRepo, logger)
	svc := NewService(repo, topicSvc, logger)
	handler := NewHandler(svc, "https://hive.example.com", logger)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-pair-invalid-age@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}
	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}
	_, otp, _, err := svc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	reqBody := `{"pairing_code":"` + otp + `","endpoint":"https://fcm.googleapis.com/fcm/send/test","p256dh":"p256dh","auth":"auth","age_recipient":"age1recipient"}`
	req := httptest.NewRequest(http.MethodPost, "/pairing", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.Pair(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Pair() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}

	var resp struct {
		Code   string   `json:"code"`
		Errors []string `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("json.NewDecoder: %v", err)
	}
	if resp.Code != "validation_error" {
		t.Fatalf("Pair() code = %q, want %q", resp.Code, "validation_error")
	}
	if len(resp.Errors) != 1 || resp.Errors[0] != "age_recipient: must be a valid age X25519 recipient" {
		t.Fatalf("Pair() errors = %#v", resp.Errors)
	}
}

func TestPairingStatusReturnsStatus(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	topicSvc := topic.NewService(topicRepo, logger)
	svc := NewService(repo, topicSvc, logger)
	handler := NewHandler(svc, "https://hive.example.com", logger)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-pairing-health-handler@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}
	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}
	_, otp, _, err := svc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	deviceID, deviceToken, err := svc.Pair(ctx, otp, "https://fcm.googleapis.com/fcm/send/health", "p256dh", "auth", testAgeRecipient)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/pairing/"+deviceID, nil)
	req.Header.Set("Authorization", "Bearer "+deviceToken)
	req = req.WithContext(testutil.WithRouteParams(req.Context(), map[string]string{"deviceID": deviceID}))
	w := httptest.NewRecorder()

	middleware.ExtractBearerToken(http.HandlerFunc(handler.PairingStatus)).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PairingStatus() status = %d, want %d. body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp PairingStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("json.NewDecoder: %v", err)
	}
	if resp.PairingStatus != PairingStatusPaired {
		t.Fatalf("pairing_status = %q, want %q", resp.PairingStatus, PairingStatusPaired)
	}
}

func TestPairingStatusRejectsInvalidDeviceToken(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	topicSvc := topic.NewService(topicRepo, logger)
	svc := NewService(repo, topicSvc, logger)
	handler := NewHandler(svc, "https://hive.example.com", logger)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-pairing-health-invalid-token@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}
	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}
	_, otp, _, err := svc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	deviceID, _, err := svc.Pair(ctx, otp, "https://fcm.googleapis.com/fcm/send/health", "p256dh", "auth", testAgeRecipient)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/pairing/"+deviceID, nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	req = req.WithContext(testutil.WithRouteParams(req.Context(), map[string]string{"deviceID": deviceID}))
	w := httptest.NewRecorder()

	middleware.ExtractBearerToken(http.HandlerFunc(handler.PairingStatus)).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("PairingStatus() status = %d, want %d. body=%s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}
