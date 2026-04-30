package webhook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/push"
	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

func newTestService() *Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &Service{log: logger}
}

func newTestServiceWithDeps(repo *Repository, dispatcher Dispatcher, topicSvc *topic.Service) *Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inspectStore := NewInspectStore()
	return NewService(repo, inspectStore, dispatcher, topicSvc, logger)
}

func TestExtractPayload_BeebuzzStandard(t *testing.T) {
	svc := newTestService()
	wh := &Webhook{PayloadType: PayloadTypeBeebuzz}

	tests := []struct {
		name        string
		body        string
		wantTitle   string
		wantMessage string
		wantErr     error
	}{
		{
			name:        "valid payload",
			body:        `{"title":"Alert","body":"Something happened"}`,
			wantTitle:   "Alert",
			wantMessage: "Something happened",
		},
		{
			name:    "missing body field",
			body:    `{"title":"Alert"}`,
			wantErr: ErrPayloadExtraction,
		},
		{
			name:    "missing title field",
			body:    `{"body":"Something happened"}`,
			wantErr: ErrPayloadExtraction,
		},
		{
			name:    "empty title",
			body:    `{"title":"","body":"Something happened"}`,
			wantErr: ErrPayloadExtraction,
		},
		{
			name:    "empty body",
			body:    `{"title":"Alert","body":""}`,
			wantErr: ErrPayloadExtraction,
		},
		{
			name:    "invalid JSON",
			body:    `not json`,
			wantErr: ErrPayloadExtraction,
		},
		{
			name:    "old message field is no longer accepted",
			body:    `{"title":"Alert","message":"Something happened"}`,
			wantErr: ErrPayloadExtraction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, msg, err := svc.extractPayload(wh, []byte(tt.body))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if title != tt.wantTitle {
				t.Errorf("title: want %q, got %q", tt.wantTitle, title)
			}
			if msg != tt.wantMessage {
				t.Errorf("message: want %q, got %q", tt.wantMessage, msg)
			}
		})
	}
}

func TestExtractPayload_Custom(t *testing.T) {
	svc := newTestService()

	tests := []struct {
		name        string
		titlePath   string
		bodyPath    string
		body        string
		wantTitle   string
		wantMessage string
		wantErr     error
	}{
		{
			name:        "flat paths",
			titlePath:   "title",
			bodyPath:    "message",
			body:        `{"title":"Alert","message":"Something happened"}`,
			wantTitle:   "Alert",
			wantMessage: "Something happened",
		},
		{
			name:        "nested paths",
			titlePath:   "data.title",
			bodyPath:    "data.message",
			body:        `{"data":{"title":"Alert","message":"Something happened"}}`,
			wantTitle:   "Alert",
			wantMessage: "Something happened",
		},
		{
			name:        "deeply nested paths",
			titlePath:   "event.notification.title",
			bodyPath:    "event.notification.body",
			body:        `{"event":{"notification":{"title":"Alert","body":"Something happened"}}}`,
			wantTitle:   "Alert",
			wantMessage: "Something happened",
		},
		{
			name:      "leading dot path does not match",
			titlePath: ".data.title",
			bodyPath:  "data.message",
			body:      `{"data":{"title":"Alert","message":"Something happened"}}`,
			wantErr:   ErrPayloadExtraction,
		},
		{
			name:      "title path not found",
			titlePath: "data.title",
			bodyPath:  "data.message",
			body:      `{"data":{"message":"Something happened"}}`,
			wantErr:   ErrPayloadExtraction,
		},
		{
			name:      "message path not found",
			titlePath: "data.title",
			bodyPath:  "data.message",
			body:      `{"data":{"title":"Alert"}}`,
			wantErr:   ErrPayloadExtraction,
		},
		{
			name:      "completely wrong payload",
			titlePath: "data.title",
			bodyPath:  "data.message",
			body:      `{"other":"value"}`,
			wantErr:   ErrPayloadExtraction,
		},
		{
			name:      "operator path rejected",
			titlePath: "data.#.title",
			bodyPath:  "data.message",
			body:      `{"data":{"message":"Something happened"}}`,
			wantErr:   ErrPayloadExtraction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wh := &Webhook{
				PayloadType: PayloadTypeCustom,
				TitlePath:   tt.titlePath,
				BodyPath:    tt.bodyPath,
			}
			title, msg, err := svc.extractPayload(wh, []byte(tt.body))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if title != tt.wantTitle {
				t.Errorf("title: want %q, got %q", tt.wantTitle, title)
			}
			if msg != tt.wantMessage {
				t.Errorf("message: want %q, got %q", tt.wantMessage, msg)
			}
		})
	}
}

func TestExtractPayload_UnsupportedType(t *testing.T) {
	svc := newTestService()
	wh := &Webhook{PayloadType: PayloadType("unknown")}

	_, _, err := svc.extractPayload(wh, []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for unsupported payload type")
	}
}

func TestInspectStoreCreateReturnsRawTokenWithoutPersistingIt(t *testing.T) {
	store := NewInspectStore()

	rawToken, session, err := store.Create("user-1", "inspect", "desc", push.PriorityNormal, []string{"topic-1"})
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if rawToken == "" {
		t.Fatal("Create() rawToken = empty, want token")
	}
	if session == nil {
		t.Fatal("Create() session = nil, want session")
	}
	if session.TokenHash != secure.Hash(rawToken) {
		t.Fatalf("Create() tokenHash = %q, want hash of raw token", session.TokenHash)
	}

	stored := store.GetByUserID("user-1")
	if stored == nil {
		t.Fatal("GetByUserID() = nil, want session")
	}
	if stored.TokenHash != secure.Hash(rawToken) {
		t.Fatalf("GetByUserID() tokenHash = %q, want hash of raw token", stored.TokenHash)
	}
}

func TestCreateInspectSessionReturnsRawTokenAndStoresOnlyHash(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	inspectStore := NewInspectStore()
	svc := NewService(repo, inspectStore, noopDispatcher{}, topicSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	user, _, err := authRepo.GetOrCreateUser(ctx, "inspect-create@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}
	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	response, err := svc.CreateInspectSession(ctx, user.ID, "inspect", "desc", push.PriorityNormal, []string{tp.ID}, "https://hook.example.com")
	if err != nil {
		t.Fatalf("CreateInspectSession() error = %v, want nil", err)
	}
	if response.Token == "" {
		t.Fatal("CreateInspectSession() token = empty, want token")
	}
	if response.URL != "https://hook.example.com/"+response.Token {
		t.Fatalf("CreateInspectSession() url = %q, want token-based url", response.URL)
	}

	session := inspectStore.GetByUserID(user.ID)
	if session == nil {
		t.Fatal("GetByUserID() = nil, want session")
	}
	if session.TokenHash != secure.Hash(response.Token) {
		t.Fatalf("stored tokenHash = %q, want hash of response token", session.TokenHash)
	}
}

// noopDispatcher is a test adapter that accepts all dispatches without sending.
type noopDispatcher struct{}

func (noopDispatcher) Dispatch(_ context.Context, _, _, _, _, _, _ string, _ *slog.Logger) (*DispatchReport, error) {
	return &DispatchReport{}, nil
}

type topicResultDispatcher struct {
	results map[string]dispatchResult
}

type priorityCapturingDispatcher struct {
	got []string
}

type dispatchResult struct {
	report *DispatchReport
	err    error
}

func (d *priorityCapturingDispatcher) Dispatch(_ context.Context, _, _, _, _, _, priority string, _ *slog.Logger) (*DispatchReport, error) {
	d.got = append(d.got, priority)
	return &DispatchReport{TotalSent: 1}, nil
}

func (d topicResultDispatcher) Dispatch(_ context.Context, _, topicID, _, _, _, _ string, _ *slog.Logger) (*DispatchReport, error) {
	result, ok := d.results[topicID]
	if !ok {
		return nil, fmt.Errorf("missing dispatch result for topic %s", topicID)
	}

	return result.report, result.err
}

func TestReceive_SetsLastUsedAt(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	now := time.Now().UTC().UnixMilli()

	// Seed a user.
	userID := "test-user"
	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		userID, "test@example.com", now, now,
	); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	// Seed a topic.
	topicID := "test-topic"
	if _, err := db.ExecContext(ctx,
		`INSERT INTO topics (id, user_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		topicID, userID, "general", now, now,
	); err != nil {
		t.Fatalf("failed to insert topic: %v", err)
	}

	repo := NewRepository(db)
	topicRepo := topic.NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.Default())
	svc := newTestServiceWithDeps(repo, noopDispatcher{}, topicSvc)

	// Create a webhook through the service.
	rawToken, webhookID, err := svc.CreateWebhook(ctx, userID, "test-wh", "", PayloadTypeBeebuzz, "", "", "normal", []string{topicID})
	if err != nil {
		t.Fatalf("failed to create webhook: %v", err)
	}

	// Verify last_used_at is nil before receive.
	wh, err := repo.GetByID(ctx, userID, webhookID)
	if err != nil {
		t.Fatalf("failed to get webhook: %v", err)
	}
	if wh.LastUsedAt != nil {
		t.Fatal("expected LastUsedAt to be nil before Receive")
	}

	// Call Receive with a valid beebuzz payload.
	payload := []byte(`{"title":"Test","body":"Hello"}`)
	if _, err := svc.Receive(ctx, rawToken, payload, slog.Default()); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	// Verify last_used_at is now set.
	wh, err = repo.GetByID(ctx, userID, webhookID)
	if err != nil {
		t.Fatalf("failed to get webhook after Receive: %v", err)
	}
	if wh.LastUsedAt == nil {
		t.Fatal("expected LastUsedAt to be set after Receive")
	}

}

func TestReceiveReturnsDeliveredResponseWhenAllDispatchesSucceed(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	user, _, err := authRepo.GetOrCreateUser(ctx, "webhook-delivered@example.com")
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
			secondTopic.ID: {report: &DispatchReport{TotalSent: 1}},
		},
	}
	svc := newTestServiceWithDeps(repo, dispatcher, topicSvc)

	rawToken, _, err := svc.CreateWebhook(ctx, user.ID, "hook", "", PayloadTypeBeebuzz, "", "", "normal", []string{firstTopic.ID, secondTopic.ID})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	response, err := svc.Receive(ctx, rawToken, []byte(`{"title":"Test","body":"Hello"}`), slog.Default())
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}

	if response.Status != ReceiveStatusDelivered {
		t.Fatalf("Receive() status = %q, want %q", response.Status, ReceiveStatusDelivered)
	}
	if response.TotalCount != 2 {
		t.Fatalf("Receive() total_count = %d, want 2", response.TotalCount)
	}
	if response.FailedCount != 0 {
		t.Fatalf("Receive() failed_count = %d, want 0", response.FailedCount)
	}
	if response.SentCount != 3 {
		t.Fatalf("Receive() sent_count = %d, want 3", response.SentCount)
	}
}

func TestReceiveReturnsPartialResponseWhenSomeDispatchesFail(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	user, _, err := authRepo.GetOrCreateUser(ctx, "webhook-partial@example.com")
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
			firstTopic.ID:  {report: &DispatchReport{TotalSent: 1}},
			secondTopic.ID: {err: errors.New("dispatch failed")},
		},
	}
	svc := newTestServiceWithDeps(repo, dispatcher, topicSvc)

	rawToken, _, err := svc.CreateWebhook(ctx, user.ID, "hook", "", PayloadTypeBeebuzz, "", "", "normal", []string{firstTopic.ID, secondTopic.ID})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	response, err := svc.Receive(ctx, rawToken, []byte(`{"title":"Test","body":"Hello"}`), slog.Default())
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}

	if response.Status != ReceiveStatusPartial {
		t.Fatalf("Receive() status = %q, want %q", response.Status, ReceiveStatusPartial)
	}
	if response.TotalCount != 2 {
		t.Fatalf("Receive() total_count = %d, want 2", response.TotalCount)
	}
	if response.FailedCount != 1 {
		t.Fatalf("Receive() failed_count = %d, want 1", response.FailedCount)
	}
	if response.SentCount != 1 {
		t.Fatalf("Receive() sent_count = %d, want 1", response.SentCount)
	}
}

func TestReceiveReturnsErrorWhenAllDispatchesFail(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	user, _, err := authRepo.GetOrCreateUser(ctx, "webhook-failed@example.com")
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
	svc := newTestServiceWithDeps(repo, dispatcher, topicSvc)

	rawToken, _, err := svc.CreateWebhook(ctx, user.ID, "hook", "", PayloadTypeBeebuzz, "", "", "normal", []string{firstTopic.ID, secondTopic.ID})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	response, err := svc.Receive(ctx, rawToken, []byte(`{"title":"Test","body":"Hello"}`), slog.Default())
	if !errors.Is(err, ErrWebhookDeliveryFailed) {
		t.Fatalf("Receive() error = %v, want %v", err, ErrWebhookDeliveryFailed)
	}

	if response == nil {
		t.Fatal("Receive() response is nil")
	}
	if response.Status != ReceiveStatusFailed {
		t.Fatalf("Receive() status = %q, want %q", response.Status, ReceiveStatusFailed)
	}
	if response.TotalCount != 2 {
		t.Fatalf("Receive() total_count = %d, want 2", response.TotalCount)
	}
	if response.FailedCount != 2 {
		t.Fatalf("Receive() failed_count = %d, want 2", response.FailedCount)
	}
	if response.SentCount != 0 {
		t.Fatalf("Receive() sent_count = %d, want 0", response.SentCount)
	}
}

func TestCreateWebhookRollsBackOnTopicAssociationFailure(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	repo := NewRepository(db)
	topicRepo := topic.NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := newTestServiceWithDeps(repo, noopDispatcher{}, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "webhook-create-rollback@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	_, _, err = svc.CreateWebhook(ctx, user.ID, "test-wh", "", PayloadTypeBeebuzz, "", "", "normal", []string{"missing-topic"})
	if !errors.Is(err, ErrInvalidTopicSelection) {
		t.Fatalf("CreateWebhook() error = %v, want %v", err, ErrInvalidTopicSelection)
	}

	webhooks, err := repo.GetByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByUser: %v", err)
	}
	if len(webhooks) != 0 {
		t.Fatalf("GetByUser() len = %d, want 0", len(webhooks))
	}
}

func TestUpdateWebhookRollsBackOnTopicAssociationFailure(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := newTestServiceWithDeps(repo, noopDispatcher{}, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "webhook-update-rollback@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	originalTopic, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, webhookID, err := svc.CreateWebhook(ctx, user.ID, "test-wh", "desc", PayloadTypeBeebuzz, "", "", "normal", []string{originalTopic.ID})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	err = svc.UpdateWebhook(ctx, user.ID, webhookID, "updated-wh", "updated-desc", PayloadTypeBeebuzz, "", "", "normal", []string{"missing-topic"})
	if !errors.Is(err, ErrInvalidTopicSelection) {
		t.Fatalf("UpdateWebhook() error = %v, want %v", err, ErrInvalidTopicSelection)
	}

	storedWebhook, err := repo.GetByID(ctx, user.ID, webhookID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if storedWebhook.Name != "test-wh" {
		t.Fatalf("webhook name = %q, want %q", storedWebhook.Name, "test-wh")
	}
	if storedWebhook.Description == nil || *storedWebhook.Description != "desc" {
		t.Fatalf("webhook description = %v, want desc", storedWebhook.Description)
	}

	topicIDs, err := repo.GetTopicIDs(ctx, webhookID)
	if err != nil {
		t.Fatalf("GetTopicIDs: %v", err)
	}
	if len(topicIDs) != 1 || topicIDs[0] != originalTopic.ID {
		t.Fatalf("webhook topicIDs = %#v, want [%q]", topicIDs, originalTopic.ID)
	}
}

func TestCreateWebhookRejectsTopicOwnedByAnotherUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := newTestServiceWithDeps(repo, noopDispatcher{}, topicSvc)

	owner, _, err := authRepo.GetOrCreateUser(ctx, "webhook-owner@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser owner: %v", err)
	}
	other, _, err := authRepo.GetOrCreateUser(ctx, "webhook-other@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser other: %v", err)
	}

	otherTopic, err := topicRepo.Create(ctx, other.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, _, err = svc.CreateWebhook(ctx, owner.ID, "test-wh", "", PayloadTypeBeebuzz, "", "", "normal", []string{otherTopic.ID})
	if !errors.Is(err, ErrInvalidTopicSelection) {
		t.Fatalf("CreateWebhook() error = %v, want %v", err, ErrInvalidTopicSelection)
	}
}

func TestReceive_BlockedUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := newTestServiceWithDeps(repo, noopDispatcher{}, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "blocked@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, _, err := svc.CreateWebhook(ctx, user.ID, "test-wh", "", PayloadTypeBeebuzz, "", "", "normal", []string{tp.ID})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'blocked' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set blocked status: %v", err)
	}

	_, err = svc.Receive(ctx, rawToken, []byte(`{"title":"Test","body":"Hello"}`), slog.Default())
	if !errors.Is(err, ErrWebhookNotFound) {
		t.Fatalf("Receive() error = %v, want %v", err, ErrWebhookNotFound)
	}
}

func TestCreateWebhookRequestValidateDefaultsPriority(t *testing.T) {
	req := CreateWebhookRequest{
		Name:        "hook",
		PayloadType: PayloadTypeBeebuzz,
		Topics:      []string{"topic-1"},
	}

	errs := req.Validate()
	if len(errs) != 0 {
		t.Fatalf("Validate() errors = %v, want none", errs)
	}
	if req.Priority != push.PriorityNormal {
		t.Fatalf("Validate() priority = %q, want %q", req.Priority, push.PriorityNormal)
	}
}

func TestReceivePassesWebhookPriorityToDispatcher(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	user, _, err := authRepo.GetOrCreateUser(ctx, "webhook-priority@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	firstTopic, err := topicRepo.Create(ctx, user.ID, "alpha", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}
	secondTopic, err := topicRepo.Create(ctx, user.ID, "beta", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	dispatcher := &priorityCapturingDispatcher{}
	svc := newTestServiceWithDeps(repo, dispatcher, topicSvc)

	rawToken, _, err := svc.CreateWebhook(ctx, user.ID, "hook", "", PayloadTypeBeebuzz, "", "", push.PriorityHigh, []string{firstTopic.ID, secondTopic.ID})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	_, err = svc.Receive(ctx, rawToken, []byte(`{"title":"Test","body":"Hello"}`), slog.Default())
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}

	if len(dispatcher.got) != 2 {
		t.Fatalf("dispatcher got %d priorities, want 2", len(dispatcher.got))
	}
	for _, priority := range dispatcher.got {
		if priority != push.PriorityHigh {
			t.Fatalf("dispatcher priority = %q, want %q", priority, push.PriorityHigh)
		}
	}
}
