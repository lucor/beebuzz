package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"lucor.dev/beebuzz/internal/push"
)

var pipedInputCheck = isPipedInput

// readAttachment reads and encodes a file attachment for the encrypted payload.
func readAttachment(attachmentPath string) (*AttachmentPayload, error) {
	file, err := os.Open(attachmentPath)
	if err != nil {
		return nil, fmt.Errorf("read attachment: %w", err)
	}
	defer file.Close() //nolint:errcheck

	data, err := io.ReadAll(io.LimitReader(file, maxAttachmentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read attachment: %w", err)
	}
	if len(data) > maxAttachmentBytes {
		return nil, fmt.Errorf("attachment exceeds %d bytes", maxAttachmentBytes)
	}

	sniffSize := 512
	if len(data) < sniffSize {
		sniffSize = len(data)
	}
	mimeType := http.DetectContentType(data[:sniffSize])

	return &AttachmentPayload{
		Data:     base64.StdEncoding.EncodeToString(data),
		MIME:     mimeType,
		Filename: filepath.Base(attachmentPath),
	}, nil
}

// postEncryptedPush sends the age ciphertext to the BeeBuzz push endpoint.
func postEncryptedPush(ctx context.Context, client *http.Client, config *Config, input PushInput, ciphertext []byte) (*PushResponse, error) {
	requestURL := config.APIURL + pushEndpointBasePath + url.PathEscape(input.Topic)
	req, err := buildAuthorizedRequest(ctx, http.MethodPost, requestURL, config.APIToken, bytes.NewReader(ciphertext))
	if err != nil {
		return nil, fmt.Errorf("build push request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	if input.Priority != "" {
		req.Header.Set(push.PriorityHeader, input.Priority)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send notification request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, FormatHTTPError("push request", resp)
	}

	var pushResponse PushResponse
	if err := decodeJSONResponse(resp, &pushResponse); err != nil {
		return nil, fmt.Errorf("decode push response: %w", err)
	}

	return &pushResponse, nil
}

// isPipedInput reports whether stdin is connected to a pipe or redirected input.
func isPipedInput() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&stdinCharDevice == 0
}
