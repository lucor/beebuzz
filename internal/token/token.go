// Package token manages API tokens used for authenticating external clients.
package token

import (
	"context"
	"errors"
	"time"

	"lucor.dev/beebuzz/internal/validator"
)

// APIToken is the DB struct — db tags only.
type APIToken struct {
	ID          string  `db:"id"`
	UserID      string  `db:"user_id"`
	TokenHash   string  `db:"token_hash"`
	Name        string  `db:"name"`
	Description *string `db:"description"`
	ExpiresAt   *int64  `db:"expires_at"`
	RevokedAt   *int64  `db:"revoked_at"`
	CreatedAt   int64   `db:"created_at"`
	LastUsedAt  *int64  `db:"last_used_at"`
	IsActive    bool    `db:"is_active"`
}

// APITokenResponse is the HTTP response struct — json tags only.
// LastFour shows the last 4 chars of the token hash (never the full hash).
type APITokenResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	LastFour    string     `json:"last_four"`
	TopicIDs    []string   `json:"topic_ids,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	IsActive    bool       `json:"is_active"`
}

// APITokensListResponse wraps a collection.
type APITokensListResponse struct {
	Data []APITokenResponse `json:"data"`
}

// CreatedAPITokenResponse is returned on POST /tokens (one-time token reveal).
type CreatedAPITokenResponse struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

// unixMilliPtr converts an optional int64 Unix-millis value to *time.Time.
func unixMilliPtr(v *int64) *time.Time {
	if v == nil {
		return nil
	}
	t := time.UnixMilli(*v).UTC()
	return &t
}

// ToAPITokenResponse converts an APIToken and its topic IDs to its HTTP response representation.
func ToAPITokenResponse(t *APIToken, topicIDs []string) APITokenResponse {
	hashLen := len(t.TokenHash)
	lastFour := t.TokenHash[max(0, hashLen-4):]

	return APITokenResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		LastFour:    lastFour,
		TopicIDs:    topicIDs,
		CreatedAt:   time.UnixMilli(t.CreatedAt).UTC(),
		ExpiresAt:   unixMilliPtr(t.ExpiresAt),
		LastUsedAt:  unixMilliPtr(t.LastUsedAt),
		IsActive:    t.IsActive,
	}
}

// CreateAPITokenRequest is the request body for POST /tokens.
type CreateAPITokenRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Topics      []string `json:"topics"`
}

// Validate validates the create API token request fields.
func (r *CreateAPITokenRequest) Validate() []error {
	return validator.Validate(
		validator.NotBlank("name", r.Name),
		validator.RequiredSlice("topics", r.Topics),
		validator.UniqueStrings("topics", r.Topics),
		validator.MaxLen("name", r.Name, validator.MaxDisplayNameLen),
		validator.MaxLen("description", r.Description, validator.MaxDisplayNameLen),
	)
}

// UpdateAPITokenRequest is the request body for PATCH /tokens/{tokenID}.
type UpdateAPITokenRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Topics      []string `json:"topics"`
}

// Validate validates the update API token request fields.
func (r *UpdateAPITokenRequest) Validate() []error {
	return validator.Validate(
		validator.NotBlank("name", r.Name),
		validator.RequiredSlice("topics", r.Topics),
		validator.UniqueStrings("topics", r.Topics),
		validator.MaxLen("name", r.Name, validator.MaxDisplayNameLen),
		validator.MaxLen("description", r.Description, validator.MaxDisplayNameLen),
	)
}

// ErrTokenNotFound is returned when an API token does not exist or does not belong to the user.
var ErrTokenNotFound = errors.New("api token not found")

// ErrAtLeastOneTopic is returned when an API token request contains no topics.
var ErrAtLeastOneTopic = errors.New("at least one topic is required")

// ErrInvalidTopicSelection is returned when one or more topics are invalid for the user.
var ErrInvalidTopicSelection = errors.New("invalid topic selection")

// TopicValidator verifies that topic IDs belong to the given user.
type TopicValidator interface {
	ValidateTopicIDs(ctx context.Context, userID string, topicIDs []string) error
}
