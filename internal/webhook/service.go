package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tidwall/gjson"

	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/validator"
)

// beebuzzPayload is the expected JSON structure for the standard payload type.
type beebuzzPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Service provides webhook business logic.
type Service struct {
	repo           *Repository
	inspectStore   *InspectStore
	dispatcher     Dispatcher
	topicValidator TopicValidator
	log            *slog.Logger
}

// NewService creates a new webhook service.
func NewService(repo *Repository, inspectStore *InspectStore, dispatcher Dispatcher, topicValidator TopicValidator, log *slog.Logger) *Service {
	return &Service{repo: repo, inspectStore: inspectStore, dispatcher: dispatcher, topicValidator: topicValidator, log: log}
}

// CreateWebhook creates a new webhook and returns the raw token (one-time reveal) and webhook ID.
func (s *Service) CreateWebhook(ctx context.Context, userID, name, description string, payloadType PayloadType, titlePath, bodyPath, priority string, topics []string) (string, string, error) {
	if len(topics) == 0 {
		return "", "", ErrAtLeastOneTopic
	}
	if err := s.topicValidator.ValidateTopicIDs(ctx, userID, topics); err != nil {
		if errors.Is(err, ErrInvalidTopicSelection) {
			return "", "", err
		}
		return "", "", ErrInvalidTopicSelection
	}

	rawToken, err := secure.NewWebhookToken()
	if err != nil {
		return "", "", err
	}

	tokenHash := secure.Hash(rawToken)
	webhookID, err := s.repo.CreateWithTopics(ctx, userID, name, description, payloadType, tokenHash, titlePath, bodyPath, priority, topics)
	if err != nil {
		return "", "", err
	}

	return rawToken, webhookID, nil
}

// ListWebhooks lists all active webhooks for the user.
func (s *Service) ListWebhooks(ctx context.Context, userID string) ([]WebhookResponse, error) {
	webhooks, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]WebhookResponse, len(webhooks))
	for i, wh := range webhooks {
		topicIDs, err := s.repo.GetTopicIDs(ctx, wh.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load webhook topics: %w", err)
		}
		result[i] = toWebhookResponse(wh, topicIDs)
	}

	return result, nil
}

// UpdateWebhook updates a webhook's settings and topic associations.
func (s *Service) UpdateWebhook(ctx context.Context, userID, webhookID, name, description string, payloadType PayloadType, titlePath, bodyPath, priority string, topicIDs []string) error {
	if len(topicIDs) == 0 {
		return ErrAtLeastOneTopic
	}
	if err := s.topicValidator.ValidateTopicIDs(ctx, userID, topicIDs); err != nil {
		if errors.Is(err, ErrInvalidTopicSelection) {
			return err
		}
		return ErrInvalidTopicSelection
	}

	return s.repo.UpdateWithTopics(ctx, userID, webhookID, name, description, payloadType, titlePath, bodyPath, priority, topicIDs)
}

// RevokeWebhook revokes a webhook, returning ErrWebhookNotFound if it does not exist.
func (s *Service) RevokeWebhook(ctx context.Context, userID, webhookID string) error {
	wh, err := s.repo.GetByID(ctx, userID, webhookID)
	if err != nil {
		return err
	}
	if wh == nil {
		return ErrWebhookNotFound
	}
	return s.repo.Revoke(ctx, userID, webhookID)
}

// Receive validates the token, extracts title/body from the payload, and dispatches notifications.
func (s *Service) Receive(ctx context.Context, tokenStr string, body []byte, log *slog.Logger) (*ReceiveResponse, error) {
	captured, err := s.CaptureInspectPayload(tokenStr, body)
	if err != nil {
		return nil, err
	}
	if captured {
		return &ReceiveResponse{Status: ReceiveStatusDelivered}, nil
	}

	tokenHash := secure.Hash(tokenStr)

	wh, err := s.repo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if wh == nil {
		return nil, ErrWebhookNotFound
	}
	if !wh.IsActive {
		return nil, ErrWebhookInactive
	}

	title, message, err := s.extractPayload(wh, body)
	if err != nil {
		return nil, err
	}

	topics, err := s.repo.GetTopicsWithIDs(ctx, wh.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topics: %w", err)
	}

	response := &ReceiveResponse{
		TotalCount: len(topics),
	}

	for _, topic := range topics {
		report, err := s.dispatcher.Dispatch(ctx, wh.UserID, topic.ID, topic.Name, title, message, wh.Priority, log)
		if err != nil {
			response.FailedCount++
			log.Error("dispatch failed", "webhook_id", wh.ID, "topic_id", topic.ID, "error", err)
			continue
		}
		response.SentCount += report.TotalSent
		log.Info("webhook dispatched", "webhook_id", wh.ID, "topic", topic.Name, "sent", report.TotalSent)
	}

	if err := s.repo.TouchLastUsedAt(ctx, wh.ID); err != nil {
		return nil, fmt.Errorf("failed to update webhook usage: %w", err)
	}

	if response.FailedCount == response.TotalCount {
		response.Status = ReceiveStatusFailed
		return response, ErrWebhookDeliveryFailed
	}

	if response.FailedCount > 0 {
		response.Status = ReceiveStatusPartial
		return response, nil
	}

	response.Status = ReceiveStatusDelivered
	return response, nil
}

// extractPayload parses title and message from the body based on the webhook's payload type.
func (s *Service) extractPayload(wh *Webhook, body []byte) (title, bodyText string, err error) {
	switch wh.PayloadType {
	case PayloadTypeBeebuzz:
		var p beebuzzPayload
		if err := json.Unmarshal(body, &p); err != nil {
			return "", "", fmt.Errorf("%w: invalid JSON", ErrPayloadExtraction)
		}
		if p.Title == "" || p.Body == "" {
			return "", "", fmt.Errorf("%w: title and body are required", ErrPayloadExtraction)
		}
		return p.Title, p.Body, nil

	case PayloadTypeCustom:
		if err := validator.JSONPath("title_path", wh.TitlePath); err != nil {
			return "", "", fmt.Errorf("%w: invalid title_path", ErrPayloadExtraction)
		}
		if err := validator.JSONPath("body_path", wh.BodyPath); err != nil {
			return "", "", fmt.Errorf("%w: invalid body_path", ErrPayloadExtraction)
		}
		titleResult := gjson.GetBytes(body, wh.TitlePath)
		bodyResult := gjson.GetBytes(body, wh.BodyPath)
		if !titleResult.Exists() || !bodyResult.Exists() {
			return "", "", fmt.Errorf("%w: path not found in payload", ErrPayloadExtraction)
		}
		return titleResult.String(), bodyResult.String(), nil

	default:
		return "", "", fmt.Errorf("unsupported payload_type: %s", string(wh.PayloadType))
	}
}

// RegenerateToken generates a new token for the given webhook and returns the raw token.
func (s *Service) RegenerateToken(ctx context.Context, userID, webhookID string) (string, error) {
	wh, err := s.repo.GetByID(ctx, userID, webhookID)
	if err != nil {
		return "", err
	}
	if wh == nil {
		return "", ErrWebhookNotFound
	}

	rawToken, err := secure.NewWebhookToken()
	if err != nil {
		return "", err
	}

	newHash := secure.Hash(rawToken)
	if err := s.repo.UpdateTokenHash(ctx, userID, webhookID, newHash); err != nil {
		return "", fmt.Errorf("failed to update token hash: %w", err)
	}

	s.log.Info("webhook token regenerated", "webhook_id", webhookID, "user_id", userID)
	return rawToken, nil
}

// CreateInspectSession creates a new inspect session and returns the raw token and webhook URL.
func (s *Service) CreateInspectSession(ctx context.Context, userID, name, description, priority string, topics []string, hookURL string) (*InspectSessionResponse, error) {
	if len(topics) == 0 {
		return nil, ErrAtLeastOneTopic
	}
	if err := s.topicValidator.ValidateTopicIDs(ctx, userID, topics); err != nil {
		if errors.Is(err, ErrInvalidTopicSelection) {
			return nil, err
		}
		return nil, ErrInvalidTopicSelection
	}

	rawToken, session, err := s.inspectStore.Create(userID, name, description, priority, topics)
	if err != nil {
		return nil, err
	}

	return &InspectSessionResponse{
		Token:     rawToken,
		URL:       hookURL + "/" + rawToken,
		Status:    session.Status,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

// GetInspectSession returns the current inspect session status for the user.
func (s *Service) GetInspectSession(ctx context.Context, userID string) *InspectSessionStatusResponse {
	session := s.inspectStore.GetByUserID(userID)
	if session == nil {
		return nil
	}

	return &InspectSessionStatusResponse{
		Status:     session.Status,
		Payload:    session.Payload,
		CapturedAt: session.CapturedAt,
		ExpiresAt:  session.ExpiresAt,
	}
}

// FinalizeInspect creates the actual webhook from a completed inspect session.
func (s *Service) FinalizeInspect(ctx context.Context, userID, titlePath, bodyPath string) (string, string, error) {
	session := s.inspectStore.GetByUserID(userID)
	if session == nil {
		return "", "", ErrInspectSessionNotFound
	}
	if session.Status != InspectStatusCaptured {
		return "", "", ErrInspectNotCaptured
	}

	if !gjson.GetBytes(session.Payload, titlePath).Exists() {
		return "", "", fmt.Errorf("%w: title_path not found in payload", ErrPayloadExtraction)
	}
	if !gjson.GetBytes(session.Payload, bodyPath).Exists() {
		return "", "", fmt.Errorf("%w: body_path not found in payload", ErrPayloadExtraction)
	}

	rawToken, err := secure.NewWebhookToken()
	if err != nil {
		return "", "", err
	}
	tokenHash := secure.Hash(rawToken)

	webhookID, err := s.repo.CreateWithTopics(ctx, userID, session.Name, session.Description, PayloadTypeCustom, tokenHash, titlePath, bodyPath, session.Priority, session.Topics)
	if err != nil {
		return "", "", err
	}

	s.inspectStore.Delete(userID)

	s.log.Info("webhook created from inspect session", "webhook_id", webhookID, "user_id", userID)
	return rawToken, webhookID, nil
}

// CaptureInspectPayload attempts to capture a payload for an active inspect session.
func (s *Service) CaptureInspectPayload(tokenStr string, body []byte) (bool, error) {
	if !strings.HasPrefix(tokenStr, "beebuzz_whi_") {
		return false, nil
	}

	tokenHash := secure.Hash(tokenStr)
	session := s.inspectStore.GetByTokenHash(tokenHash)
	if session == nil {
		return false, nil
	}
	if session.Status != InspectStatusWaiting {
		return false, nil
	}

	if !json.Valid(body) {
		return false, fmt.Errorf("%w: invalid JSON", ErrPayloadExtraction)
	}

	if err := s.inspectStore.Capture(tokenHash, body); err != nil {
		return false, err
	}

	return true, nil
}
