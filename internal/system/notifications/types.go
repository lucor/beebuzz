// Package notifications manages system-generated notification policy.
package notifications

import (
	"context"
	"errors"
	"time"

	"lucor.dev/beebuzz/internal/core"
)

var (
	ErrInvalidTopicSelection = errors.New("invalid topic selection")
	ErrTopicRequired         = errors.New("topic_id is required when system notifications are enabled")
)

// Topic is the minimal topic view needed by system notifications.
type Topic struct {
	ID     string
	UserID string
	Name   string
}

// TopicProvider validates and resolves user-owned topics.
type TopicProvider interface {
	GetTopicByID(ctx context.Context, userID, topicID string) (*Topic, error)
}

// Delivery sends a system notification through the delivery engine.
type Delivery interface {
	SendSystemNotification(ctx context.Context, input DeliveryInput) error
}

// DeliveryInput carries a resolved system notification delivery request.
type DeliveryInput struct {
	RecipientUserID string
	TopicID         string
	TopicName       string
	Title           string
	Body            string
}

// Settings is the persisted system notifications configuration.
type Settings struct {
	Enabled              bool   `db:"enabled"`
	RecipientUserID      string `db:"recipient_user_id"`
	TopicID              string `db:"topic_id"`
	SignupCreatedEnabled bool   `db:"signup_created_enabled"`
	CreatedAt            int64  `db:"created_at"`
	UpdatedAt            int64  `db:"updated_at"`
}

// SettingsResponse is the admin API response for system notification settings.
type SettingsResponse struct {
	Enabled              bool      `json:"enabled"`
	RecipientUserID      string    `json:"recipient_user_id,omitempty"`
	TopicID              string    `json:"topic_id,omitempty"`
	SignupCreatedEnabled bool      `json:"signup_created_enabled"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
	UpdatedAt            time.Time `json:"updated_at,omitempty"`
}

// UpdateSettingsRequest is the admin API request for system notification settings.
type UpdateSettingsRequest struct {
	Enabled              bool   `json:"enabled"`
	TopicID              string `json:"topic_id"`
	SignupCreatedEnabled bool   `json:"signup_created_enabled"`
}

// ToSettingsResponse converts persisted settings to the admin API shape.
func ToSettingsResponse(settings *Settings) SettingsResponse {
	if settings == nil {
		return SettingsResponse{}
	}

	return SettingsResponse{
		Enabled:              settings.Enabled,
		RecipientUserID:      settings.RecipientUserID,
		TopicID:              settings.TopicID,
		SignupCreatedEnabled: settings.SignupCreatedEnabled,
		CreatedAt:            time.UnixMilli(settings.CreatedAt).UTC(),
		UpdatedAt:            time.UnixMilli(settings.UpdatedAt).UTC(),
	}
}

// SignupEvent carries the facts needed for a signup-created notification.
type SignupEvent struct {
	CreatedUserID string
	AccountStatus core.AccountStatus
}
