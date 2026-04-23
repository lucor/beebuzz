package topic

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"lucor.dev/beebuzz/internal/middleware"
	"lucor.dev/beebuzz/internal/testutil"
)

func newTestHandler(t *testing.T) *Handler {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	svc := NewService(repo, logger)
	return NewHandler(svc, logger)
}

func withUserContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, middleware.CtxKeyUser, &middleware.CtxUser{
		ID:      userID,
		IsAdmin: false,
	})
}

func TestCreateTopicHandler(t *testing.T) {
	h := newTestHandler(t)

	tests := []struct {
		name       string
		body       CreateTopicRequest
		wantStatus int
	}{
		{
			name: "success",
			body: CreateTopicRequest{
				Name:        "alerts",
				Description: "Alert notifications",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "validation error",
			body: CreateTopicRequest{
				Name:        "",
				Description: "desc",
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "reserved name",
			body: CreateTopicRequest{
				Name:        "general",
				Description: "desc",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader(body))
			req = req.WithContext(withUserContext(req.Context(), "user-1"))
			w := httptest.NewRecorder()

			h.CreateTopic(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("CreateTopic() status = %v, want %v. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestCreateTopicHandlerRejectsUnknownFields(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader([]byte(`{"name":"alerts","description":"desc","unexpected":true}`)))
	req = req.WithContext(withUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	h.CreateTopic(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateTopic() status = %d, want %d. body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateTopicHandlerRejectsTrailingJSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader([]byte(`{"name":"alerts","description":"desc"}{"extra":true}`)))
	req = req.WithContext(withUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	h.CreateTopic(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateTopic() status = %d, want %d. body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestGetTopicsHandler(t *testing.T) {
	t.Run("returns existing topics", func(t *testing.T) {
		h := newTestHandler(t)
		svc := h.topicService

		if err := svc.CreateDefaultTopic(context.Background(), "user-1"); err != nil {
			t.Fatalf("CreateDefaultTopic() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/topics", nil)
		req = req.WithContext(withUserContext(req.Context(), "user-1"))
		w := httptest.NewRecorder()

		h.GetTopics(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GetTopics() status = %v, want %v", w.Code, http.StatusOK)
		}

		var resp TopicsListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Data) != 1 {
			t.Fatalf("GetTopics() len = %v, want 1", len(resp.Data))
		}
		if resp.Data[0].Name != "general" {
			t.Errorf("GetTopics() first topic = %v, want general", resp.Data[0].Name)
		}
	})

	t.Run("does not create default topic on read", func(t *testing.T) {
		h := newTestHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/topics", nil)
		req = req.WithContext(withUserContext(req.Context(), "test-user-2"))
		w := httptest.NewRecorder()

		h.GetTopics(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GetTopics() status = %v, want %v", w.Code, http.StatusOK)
		}

		var resp TopicsListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Data) != 0 {
			t.Fatalf("GetTopics() len = %v, want 0", len(resp.Data))
		}
	})
}

func TestUpdateTopicHandler(t *testing.T) {
	h := newTestHandler(t)

	// Create topic through handler
	req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader([]byte(`{"name":"alerts","description":"old desc"}`)))
	req = req.WithContext(withUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.CreateTopic(w, req)

	// Extract created topic ID from response
	var created TopicResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	tests := []struct {
		name       string
		topicID    string
		body       UpdateTopicRequest
		wantStatus int
	}{
		{
			name:    "success",
			topicID: created.ID,
			body: UpdateTopicRequest{
				Description: "new desc",
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:    "not found",
			topicID: "non-existent",
			body: UpdateTopicRequest{
				Description: "desc",
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPatch, "/topics/"+tt.topicID, bytes.NewReader(body))
			req = req.WithContext(withUserContext(req.Context(), "user-1"))
			req = req.WithContext(testutil.WithRouteParams(req.Context(), map[string]string{"topicID": tt.topicID}))
			w := httptest.NewRecorder()

			h.UpdateTopic(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("UpdateTopic() status = %v, want %v. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestDeleteTopicHandler(t *testing.T) {
	h := newTestHandler(t)

	// Create topic through handler
	req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader([]byte(`{"name":"alerts","description":"desc"}`)))
	req = req.WithContext(withUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.CreateTopic(w, req)

	var created TopicResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	tests := []struct {
		name       string
		topicID    string
		wantStatus int
	}{
		{
			name:       "success",
			topicID:    created.ID,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "not found",
			topicID:    "non-existent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "protected topic",
			topicID:    created.ID, // Will be different topic
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/topics/"+tt.topicID, nil)
			req = req.WithContext(withUserContext(req.Context(), "user-1"))
			req = req.WithContext(testutil.WithRouteParams(req.Context(), map[string]string{"topicID": tt.topicID}))
			w := httptest.NewRecorder()

			h.DeleteTopic(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("DeleteTopic() status = %v, want %v. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestDeleteTopicHandlerProtected(t *testing.T) {
	h := newTestHandler(t)
	svc := h.topicService

	if err := svc.CreateDefaultTopic(context.Background(), "user-1"); err != nil {
		t.Fatalf("CreateDefaultTopic() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/topics", nil)
	req = req.WithContext(withUserContext(req.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.GetTopics(w, req)

	var resp TopicsListResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	// Find the general topic
	var generalID string
	for _, t := range resp.Data {
		if t.Name == "general" {
			generalID = t.ID
			break
		}
	}

	if generalID == "" {
		t.Fatal("general topic not found")
	}

	req = httptest.NewRequest(http.MethodDelete, "/topics/"+generalID, nil)
	req = req.WithContext(withUserContext(req.Context(), "user-1"))
	req = req.WithContext(testutil.WithRouteParams(req.Context(), map[string]string{"topicID": generalID}))
	w = httptest.NewRecorder()

	h.DeleteTopic(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("DeleteTopic() status = %v, want %v. Body: %s", w.Code, http.StatusConflict, w.Body.String())
	}

	var errResp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp["code"] != "topic_protected" {
		t.Errorf("DeleteTopic() code = %v, want topic_protected", errResp["code"])
	}
}

func TestTopicHandlerUnauthorized(t *testing.T) {
	h := newTestHandler(t)

	tests := []struct {
		name string
		fn   func(http.ResponseWriter, *http.Request)
	}{
		{"GetTopics", h.GetTopics},
		{"CreateTopic", h.CreateTopic},
		{"UpdateTopic", h.UpdateTopic},
		{"DeleteTopic", h.DeleteTopic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/topics", nil)
			w := httptest.NewRecorder()

			tt.fn(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s() status = %v, want %v", tt.name, w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestTopicHandlerMissingTopicID(t *testing.T) {
	h := newTestHandler(t)

	tests := []struct {
		name string
		fn   func(http.ResponseWriter, *http.Request)
	}{
		{"UpdateTopic", h.UpdateTopic},
		{"DeleteTopic", h.DeleteTopic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/topics/", nil)
			req = req.WithContext(withUserContext(req.Context(), "user-1"))
			w := httptest.NewRecorder()

			tt.fn(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("%s() status = %v, want %v", tt.name, w.Code, http.StatusBadRequest)
			}
		})
	}
}
