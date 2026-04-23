package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"filippo.io/age"
	"lucor.dev/beebuzz/internal/notification"
	"lucor.dev/beebuzz/internal/push"
)

const (
	pushEndpointBasePath = "/v1/push/"
	maxAttachmentBytes   = 1024 * 1024
	stdinCharDevice      = os.ModeCharDevice
)

const maxBodyStdinBytes = notification.MaxNotificationBodyLen*4 + 1

// AttachmentPayload is the inline attachment stored inside the encrypted CLI payload.
type AttachmentPayload struct {
	Data     string `json:"data"`
	MIME     string `json:"mime"`
	Filename string `json:"filename"`
}

// EncryptedNotificationPayload is the plaintext JSON encrypted by the CLI.
type EncryptedNotificationPayload struct {
	Title      string             `json:"title"`
	Body       string             `json:"body"`
	Topic      string             `json:"topic"`
	Attachment *AttachmentPayload `json:"attachment,omitempty"`
}

// PushResponse is the CLI view of the push response.
type PushResponse struct {
	SentCount   int         `json:"sent_count"`
	TotalCount  int         `json:"total_count"`
	FailedCount int         `json:"failed_count"`
	DeviceKeys  []DeviceKey `json:"device_keys"`
}

// PushInput holds resolved CLI arguments for a notification push operation.
type PushInput struct {
	Title          string
	Body           string
	Topic          string
	Priority       string
	AttachmentPath string
	APIURL         string
	Profile        string
}

// resolvePushInput validates positional arguments and builds a PushInput.
func resolvePushInput(positionals []string, stdin io.Reader, topic, priority, attachmentPath, apiURL string) (PushInput, error) {
	if len(positionals) > 2 {
		return PushInput{}, fmt.Errorf("%w: expected: beebuzz send [flags] <title> [body]", ErrUsage)
	}

	var resolvedTitle string
	if len(positionals) > 0 {
		resolvedTitle = strings.TrimSpace(positionals[0])
	}
	if resolvedTitle == "" {
		return PushInput{}, fmt.Errorf("%w: title is required", ErrUsage)
	}
	if len([]rune(resolvedTitle)) > notification.MaxNotificationTitleLen {
		return PushInput{}, fmt.Errorf("%w: title must be %d characters or less", ErrUsage, notification.MaxNotificationTitleLen)
	}

	var resolvedBody string
	if len(positionals) > 1 {
		resolvedBody = strings.TrimSpace(positionals[1])
	}
	if resolvedBody == "" && stdin != nil && pipedInputCheck() {
		stdinData, err := io.ReadAll(io.LimitReader(stdin, maxBodyStdinBytes))
		if err != nil {
			return PushInput{}, fmt.Errorf("read stdin: %w", err)
		}
		resolvedBody = strings.TrimSpace(string(stdinData))
	}
	if len([]rune(resolvedBody)) > notification.MaxNotificationBodyLen {
		return PushInput{}, fmt.Errorf("%w: body must be %d characters or less", ErrUsage, notification.MaxNotificationBodyLen)
	}

	if priority != push.PriorityNormal && priority != push.PriorityHigh {
		return PushInput{}, fmt.Errorf("%w: priority must be one of: %s, %s", ErrUsage, push.PriorityNormal, push.PriorityHigh)
	}

	resolvedTopic := strings.TrimSpace(topic)
	if resolvedTopic == "" {
		resolvedTopic = push.DefaultTopicName
	}

	return PushInput{
		Title:          resolvedTitle,
		Body:           resolvedBody,
		Topic:          resolvedTopic,
		Priority:       priority,
		AttachmentPath: strings.TrimSpace(attachmentPath),
		APIURL:         strings.TrimSpace(apiURL),
	}, nil
}

// pushNotification encrypts and pushes a notification using the cached device age keys.
func pushNotification(ctx context.Context, client *http.Client, config *Config, input PushInput) (*PushResponse, error) {
	payload, err := buildEncryptedPayload(input)
	if err != nil {
		return nil, err
	}

	ciphertext, err := encryptPayload(payload, config.AgeRecipients())
	if err != nil {
		return nil, err
	}

	response, err := postEncryptedPush(ctx, client, config, input, ciphertext)
	if err != nil {
		return nil, err
	}

	if response.DeviceKeys == nil {
		response.DeviceKeys = []DeviceKey{}
	}

	return response, nil
}

// buildEncryptedPayload builds the plaintext JSON payload before age encryption.
func buildEncryptedPayload(input PushInput) ([]byte, error) {
	payload := EncryptedNotificationPayload{
		Title: input.Title,
		Body:  input.Body,
		Topic: input.Topic,
	}

	if input.AttachmentPath != "" {
		attachment, err := readAttachment(input.AttachmentPath)
		if err != nil {
			return nil, err
		}
		payload.Attachment = attachment
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}

	return data, nil
}

// encryptPayload encrypts a plaintext payload for all configured age recipients.
func encryptPayload(plaintext []byte, keys []string) ([]byte, error) {
	recipients := make([]age.Recipient, 0, len(keys))
	for _, key := range keys {
		recipient, err := age.ParseX25519Recipient(key)
		if err != nil {
			return nil, fmt.Errorf("parse age recipient %q: %w", key, err)
		}
		recipients = append(recipients, recipient)
	}

	var buffer bytes.Buffer
	writer, err := age.Encrypt(&buffer, recipients...)
	if err != nil {
		return nil, fmt.Errorf("age.Encrypt: %w", err)
	}

	if _, err := writer.Write(plaintext); err != nil {
		return nil, fmt.Errorf("age write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("age close: %w", err)
	}

	return buffer.Bytes(), nil
}
