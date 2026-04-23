// Package attachment handles temporary attachment storage and secure token-based retrieval.
package attachment

import (
	"fmt"

	"lucor.dev/beebuzz/internal/core"
)

type DBAttachment struct {
	ID            string `db:"id"`
	Token         string `db:"token"`
	TopicID       string `db:"topic_id"`
	MimeType      string `db:"mime_type"`
	FileSizeBytes int    `db:"file_size_bytes"`
	CreatedAt     int64  `db:"created_at"`
	ExpiresAt     int64  `db:"expires_at"`
}

var ErrAttachmentExpired = fmt.Errorf("attachment expired: %w", core.ErrNotFound)
