// Package topic manages notification topics owned by users.
package topic

import (
	"errors"
	"time"

	"lucor.dev/beebuzz/internal/validator"
)

// Topic is the DB struct — db tags only, no json tags.
type Topic struct {
	ID          string  `db:"id"`
	UserID      string  `db:"user_id"`
	Name        string  `db:"name"`
	Description *string `db:"description"`
	CreatedAt   int64   `db:"created_at"`
	UpdatedAt   int64   `db:"updated_at"`
}

// TopicResponse is the HTTP response struct — json tags only.
type TopicResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TopicsListResponse wraps a collection for future pagination.
type TopicsListResponse struct {
	Data []TopicResponse `json:"data"`
}

// ToTopicResponse converts a Topic DB struct to a TopicResponse.
func ToTopicResponse(t *Topic) TopicResponse {
	return TopicResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		CreatedAt:   time.UnixMilli(t.CreatedAt).UTC(),
		UpdatedAt:   time.UnixMilli(t.UpdatedAt).UTC(),
	}
}

// ToTopicsListResponse converts a slice of Topics to TopicsListResponse.
func ToTopicsListResponse(topics []Topic) TopicsListResponse {
	responses := make([]TopicResponse, len(topics))
	for i, t := range topics {
		responses[i] = ToTopicResponse(&t)
	}
	return TopicsListResponse{Data: responses}
}

// CreateTopicRequest is the request body for POST /topics.
type CreateTopicRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Validate validates the create topic request fields.
func (r *CreateTopicRequest) Validate() []error {
	return validator.Validate(
		validator.NotBlank("name", r.Name),
		validator.TopicName("name", r.Name),
		validator.MaxLen("description", r.Description, validator.MaxDescriptionLen),
	)
}

// UpdateTopicRequest updates a topic's description.
type UpdateTopicRequest struct {
	Description string `json:"description"`
}

// Validate validates the update topic request fields.
func (r *UpdateTopicRequest) Validate() []error {
	return validator.Validate(
		validator.MaxLen("description", r.Description, validator.MaxDescriptionLen),
	)
}

// Sentinel errors — used by service, matched by handler via errors.Is.
var (
	ErrTopicNotFound     = errors.New("topic not found")
	ErrTopicProtected    = errors.New("cannot delete the protected 'general' topic")
	ErrTopicNameReserved = errors.New("topic name 'general' is reserved")
	ErrTopicNameConflict = errors.New("a topic with this name already exists")
	ErrDuplicateTopicIDs = errors.New("duplicate topic IDs are not allowed")
)
