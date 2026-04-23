package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

func newTestWebhookHandler(t *testing.T) *Handler {
	t.Helper()

	db := testutil.NewDB(t)
	repo := NewRepository(db)
	topicRepo := topic.NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	inspectStore := NewInspectStore()
	svc := NewService(repo, inspectStore, noopDispatcher{}, topicSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	return NewHandler(svc, "https://hook.example.com", slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestCreateWebhookRejectsEmptyTopics(t *testing.T) {
	handler := newTestWebhookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewBufferString(`{"name":"hook","description":"desc","payload_type":"beebuzz","topics":[]}`))
	req = req.WithContext(testutil.WithUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	handler.CreateWebhook(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateWebhook() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestCreateWebhookRejectsForeignTopicSelection(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	inspectStore := NewInspectStore()
	svc := NewService(repo, inspectStore, noopDispatcher{}, topicSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewHandler(svc, "https://hook.example.com", slog.New(slog.NewTextHandler(io.Discard, nil)))

	owner, _, err := authRepo.GetOrCreateUser(ctx, "webhook-handler-owner@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser owner: %v", err)
	}
	other, _, err := authRepo.GetOrCreateUser(ctx, "webhook-handler-other@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser other: %v", err)
	}
	otherTopic, err := topicRepo.Create(ctx, other.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	body, err := json.Marshal(CreateWebhookRequest{
		Name:        "hook",
		Description: "desc",
		PayloadType: "beebuzz",
		Topics:      []string{otherTopic.ID},
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(body))
	req = req.WithContext(testutil.WithUserContext(req.Context(), owner.ID))
	w := httptest.NewRecorder()

	handler.CreateWebhook(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateWebhook() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestCreateWebhookRejectsBeebuzzPaths(t *testing.T) {
	handler := newTestWebhookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewBufferString(`{"name":"hook","description":"desc","payload_type":"beebuzz","title_path":"data.title","body_path":"data.body","topics":["topic-1"]}`))
	req = req.WithContext(testutil.WithUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	handler.CreateWebhook(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateWebhook() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestCreateWebhookRejectsDuplicateTopics(t *testing.T) {
	handler := newTestWebhookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewBufferString(`{"name":"hook","description":"desc","payload_type":"beebuzz","topics":["topic-1","topic-1"]}`))
	req = req.WithContext(testutil.WithUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	handler.CreateWebhook(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateWebhook() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestCreateWebhookRejectsCustomPathsWithLeadingDot(t *testing.T) {
	handler := newTestWebhookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewBufferString(`{"name":"hook","description":"desc","payload_type":"custom","title_path":".data.title","body_path":"data.body","topics":["topic-1"]}`))
	req = req.WithContext(testutil.WithUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	handler.CreateWebhook(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateWebhook() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestCreateWebhookRejectsCustomPathsWithGJSONOperators(t *testing.T) {
	handler := newTestWebhookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewBufferString(`{"name":"hook","description":"desc","payload_type":"custom","title_path":"data.#.title","body_path":"@this","topics":["topic-1"]}`))
	req = req.WithContext(testutil.WithUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	handler.CreateWebhook(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateWebhook() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestUpdateWebhookReturnsNotFoundForMissingWebhook(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	inspectStore := NewInspectStore()
	svc := NewService(repo, inspectStore, noopDispatcher{}, topicSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewHandler(svc, "https://hook.example.com", slog.New(slog.NewTextHandler(io.Discard, nil)))

	user, _, err := authRepo.GetOrCreateUser(ctx, "webhook-update-missing@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}
	topicRecord, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	body, err := json.Marshal(UpdateWebhookRequest{
		Name:        "hook",
		Description: "desc",
		PayloadType: PayloadTypeBeebuzz,
		Topics:      []string{topicRecord.ID},
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/webhooks/missing-webhook", bytes.NewReader(body))
	req = req.WithContext(testutil.WithUserContext(req.Context(), user.ID))
	req = req.WithContext(testutil.WithRouteParams(req.Context(), map[string]string{"webhookID": "missing-webhook"}))
	w := httptest.NewRecorder()

	handler.UpdateWebhook(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("UpdateWebhook() status = %d, want %d. body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if resp["code"] != "webhook_not_found" {
		t.Fatalf("UpdateWebhook() code = %q, want %q", resp["code"], "webhook_not_found")
	}
}

func TestReceiveReturnsPayloadTooLarge(t *testing.T) {
	handler := newTestWebhookHandler(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/webhooks/token",
		strings.NewReader(strings.Repeat("a", maxWebhookBodyBytes+1)),
	)
	req = req.WithContext(testutil.WithRouteParams(req.Context(), map[string]string{"token": "token"}))
	w := httptest.NewRecorder()

	handler.Receive(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("Receive() status = %d, want %d. body=%s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if resp["code"] != "payload_too_large" {
		t.Fatalf("Receive() code = %q, want %q", resp["code"], "payload_too_large")
	}
}

func TestReceiveReturnsPartialStatusWhenSomeDispatchesFail(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	user, _, err := authRepo.GetOrCreateUser(ctx, "webhook-handler-partial@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	firstTopic, err := topicRepo.Create(ctx, user.ID, "alpha", "")
	if err != nil {
		t.Fatalf("topic.Create firstTopic: %v", err)
	}
	secondTopic, err := topicRepo.Create(ctx, user.ID, "beta", "")
	if err != nil {
		t.Fatalf("topic.Create secondTopic: %v", err)
	}

	dispatcher := topicResultDispatcher{
		results: map[string]dispatchResult{
			firstTopic.ID:  {report: &DispatchReport{TotalSent: 2}},
			secondTopic.ID: {err: errors.New("dispatch failed")},
		},
	}
	inspectStore := NewInspectStore()
	svc := NewService(repo, inspectStore, dispatcher, topicSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewHandler(svc, "https://hook.example.com", slog.New(slog.NewTextHandler(io.Discard, nil)))

	rawToken, _, err := svc.CreateWebhook(ctx, user.ID, "hook", "", PayloadTypeBeebuzz, "", "", "normal", []string{firstTopic.ID, secondTopic.ID})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+rawToken, bytes.NewBufferString(`{"title":"Alert","body":"Hello"}`))
	req = req.WithContext(testutil.WithRouteParams(req.Context(), map[string]string{"token": rawToken}))
	w := httptest.NewRecorder()

	handler.Receive(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Receive() status = %d, want %d. body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp ReceiveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if resp.Status != ReceiveStatusPartial {
		t.Fatalf("Receive() status body = %q, want %q", resp.Status, ReceiveStatusPartial)
	}
	if resp.TotalCount != 2 || resp.FailedCount != 1 || resp.SentCount != 2 {
		t.Fatalf("Receive() body = %+v, want total_count=2 failed_count=1 sent_count=2", resp)
	}
}

func TestReceiveReturnsBadGatewayWhenAllDispatchesFail(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	user, _, err := authRepo.GetOrCreateUser(ctx, "webhook-handler-failed@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	firstTopic, err := topicRepo.Create(ctx, user.ID, "alpha", "")
	if err != nil {
		t.Fatalf("topic.Create firstTopic: %v", err)
	}
	secondTopic, err := topicRepo.Create(ctx, user.ID, "beta", "")
	if err != nil {
		t.Fatalf("topic.Create secondTopic: %v", err)
	}

	dispatcher := topicResultDispatcher{
		results: map[string]dispatchResult{
			firstTopic.ID:  {err: errors.New("dispatch failed")},
			secondTopic.ID: {err: errors.New("dispatch failed")},
		},
	}
	inspectStore := NewInspectStore()
	svc := NewService(repo, inspectStore, dispatcher, topicSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewHandler(svc, "https://hook.example.com", slog.New(slog.NewTextHandler(io.Discard, nil)))

	rawToken, _, err := svc.CreateWebhook(ctx, user.ID, "hook", "", PayloadTypeBeebuzz, "", "", "normal", []string{firstTopic.ID, secondTopic.ID})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+rawToken, bytes.NewBufferString(`{"title":"Alert","body":"Hello"}`))
	req = req.WithContext(testutil.WithRouteParams(req.Context(), map[string]string{"token": rawToken}))
	w := httptest.NewRecorder()

	handler.Receive(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("Receive() status = %d, want %d. body=%s", w.Code, http.StatusBadGateway, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if resp["code"] != "webhook_delivery_failed" {
		t.Fatalf("Receive() code = %q, want %q", resp["code"], "webhook_delivery_failed")
	}
}
