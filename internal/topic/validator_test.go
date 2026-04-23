package topic

import (
	"strings"
	"testing"

	"lucor.dev/beebuzz/internal/validator"
)

func TestCreateTopicRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateTopicRequest
		wantErr int
	}{
		{
			name: "valid request",
			req: CreateTopicRequest{
				Name:        "alerts",
				Description: "Notifications for alerts",
			},
			wantErr: 0,
		},
		{
			name: "missing name",
			req: CreateTopicRequest{
				Name:        "",
				Description: "desc",
			},
			wantErr: 2, // Required(name) + TopicName(name)
		},
		{
			name: "invalid name characters",
			req: CreateTopicRequest{
				Name:        "alerts!",
				Description: "desc",
			},
			wantErr: 1, // TopicName(name)
		},
		{
			name: "name too long",
			req: CreateTopicRequest{
				Name:        strings.Repeat("a", 33),
				Description: "desc",
			},
			wantErr: 1, // TopicName(name)
		},
		{
			name: "description too long",
			req: CreateTopicRequest{
				Name:        "alerts",
				Description: strings.Repeat("A", validator.MaxDescriptionLen+1),
			},
			wantErr: 1, // MaxLen(description)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			if len(errs) != tt.wantErr {
				t.Errorf("Validate() got %d errors, want %d. Errors: %v", len(errs), tt.wantErr, errs)
			}
		})
	}
}

func TestUpdateTopicRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateTopicRequest
		wantErr int
	}{
		{
			name: "valid request",
			req: UpdateTopicRequest{
				Description: "Updated description",
			},
			wantErr: 0,
		},
		{
			name: "description too long",
			req: UpdateTopicRequest{
				Description: strings.Repeat("A", validator.MaxDescriptionLen+1),
			},
			wantErr: 1, // MaxLen(description)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			if len(errs) != tt.wantErr {
				t.Errorf("Validate() got %d errors, want %d. Errors: %v", len(errs), tt.wantErr, errs)
			}
		})
	}
}
