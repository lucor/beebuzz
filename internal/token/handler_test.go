package token

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
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

func newTestTokenHandler(t *testing.T) *Handler {
	t.Helper()

	db := testutil.NewDB(t)
	repo := NewRepository(db)
	topicRepo := topic.NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := NewService(repo, topicSvc)

	return NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestCreateAPITokenRejectsEmptyTopics(t *testing.T) {
	handler := newTestTokenHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewBufferString(`{"name":"token-name","description":"desc","topics":[]}`))
	req = req.WithContext(testutil.WithUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	handler.CreateAPIToken(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateAPIToken() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestCreateAPITokenRejectsForeignTopicSelection(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := NewService(repo, topicSvc)
	handler := NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	owner, _, err := authRepo.GetOrCreateUser(ctx, "token-handler-owner@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser owner: %v", err)
	}
	other, _, err := authRepo.GetOrCreateUser(ctx, "token-handler-other@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser other: %v", err)
	}
	otherTopic, err := topicRepo.Create(ctx, other.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	body, err := json.Marshal(CreateAPITokenRequest{
		Name:        "token-name",
		Description: "desc",
		Topics:      []string{otherTopic.ID},
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewReader(body))
	req = req.WithContext(testutil.WithUserContext(req.Context(), owner.ID))
	w := httptest.NewRecorder()

	handler.CreateAPIToken(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateAPIToken() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestCreateAPITokenRejectsDuplicateTopics(t *testing.T) {
	handler := newTestTokenHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewBufferString(`{"name":"token-name","description":"desc","topics":["topic-1","topic-1"]}`))
	req = req.WithContext(testutil.WithUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	handler.CreateAPIToken(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("CreateAPIToken() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}
