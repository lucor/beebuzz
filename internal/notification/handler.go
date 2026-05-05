package notification

import (
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
	"lucor.dev/beebuzz/internal/push"
	"lucor.dev/beebuzz/internal/validator"
)

const (
	codeMissingToken = "missing_token"
	codeInvalidJSON  = "invalid_json"
	codeUnauthorized = "unauthorized"
	codeInvalidTopic = "invalid_topic"
	// multipartMaxBytes limits multipart body reads to 1.5× the max attachment size.
	multipartMaxBytes = maxAttachmentBytes + maxAttachmentBytes/2
)

// Handler handles notification HTTP requests.
type Handler struct {
	service     Sender
	pushAuth    PushAuthorizer
	keyProvider KeyProvider
	log         *slog.Logger
}

// NewHandler creates a new notification handler.
func NewHandler(svc Sender, pushAuth PushAuthorizer, keyProvider KeyProvider, logger *slog.Logger) *Handler {
	return &Handler{
		service:     svc,
		pushAuth:    pushAuth,
		keyProvider: keyProvider,
		log:         logger,
	}
}

// Send handles POST /v1/push and POST /v1/push/{topic} requests.
// Authentication is performed inline via Bearer API token.
// The request body may be JSON or multipart/form-data (for file attachments).
// Requires the ExtractBearerToken middleware to run first.
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.RequestIDFromContext(r.Context())
	log := h.log.With("request_id", reqID)

	rawToken, ok := middleware.BearerTokenFromContext(r.Context())
	if !ok || rawToken == "" {
		core.WriteUnauthorized(w, codeMissingToken, "Bearer token required")
		return
	}

	// Extract topic from path param; fall back to default.
	topicName := core.GetURLParam(r, "topic")
	if topicName == "" {
		topicName = push.DefaultTopicName
	}

	if err := validator.TopicName("topic", topicName); err != nil {
		core.WriteBadRequest(w, codeInvalidTopic, err.Error())
		return
	}

	userID, topicID, err := h.pushAuth.ValidateAPITokenForTopic(r.Context(), rawToken, topicName)
	if err != nil {
		core.WriteUnauthorized(w, codeUnauthorized, "invalid or unauthorized token")
		return
	}

	var (
		input       SendInput
		parseErr    error
		validErrs   []error
		contentType = r.Header.Get("Content-Type")
	)

	switch {
	case strings.HasPrefix(contentType, "application/octet-stream"):
		input, validErrs, parseErr = h.parseSendOctetStream(r)
	case strings.HasPrefix(contentType, "multipart/form-data"):
		input, validErrs, parseErr = h.parseSendMultipart(r)
	default:
		input, validErrs, parseErr = h.parseSendJSON(r)
	}

	if parseErr != nil {
		if errors.Is(parseErr, core.ErrPayloadTooLarge) {
			core.WritePayloadTooLarge(w)
			return
		}
		core.WriteBadRequest(w, codeInvalidJSON, parseErr.Error())
		return
	}
	if len(validErrs) > 0 {
		core.WriteValidationErrors(w, validErrs)
		return
	}

	input.TopicName = topicName
	if strings.HasPrefix(r.Header.Get("User-Agent"), core.CLIUserAgentPrefix) {
		input.Source = SourceCLI
	} else {
		input.Source = SourceAPI
	}
	if input.DeliveryMode == "" {
		input.DeliveryMode = DeliveryModeServerTrusted
	}

	report, err := h.service.Send(r.Context(), userID, topicID, input, log)
	if err != nil {
		if errors.Is(err, core.ErrPayloadTooLarge) {
			core.WritePayloadTooLarge(w)
			return
		}
		if errors.Is(err, ErrAttachmentProcessingFailed) {
			core.WriteError(w, http.StatusUnprocessableEntity, "attachment_processing_failed", "Failed to process attachment")
			return
		}
		log.Error("failed to send notification", "error", err, "topic", topicName)
		core.WriteInternalError(w, r, err)
		return
	}

	totalCount := len(report.DeviceResults)
	resp := SendResponse{
		Status:      SendStatusDelivered,
		SentCount:   report.TotalSent,
		TotalCount:  totalCount,
		FailedCount: report.TotalFailed,
	}

	if resp.FailedCount > 0 {
		if resp.SentCount == 0 {
			resp.Status = SendStatusFailed
		} else {
			resp.Status = SendStatusPartial
		}
	}

	// Include current device keys for CLI auto-sync.
	if input.DeliveryMode == DeliveryModeE2E && h.keyProvider != nil {
		deviceKeys, keysErr := h.keyProvider.GetDeviceKeys(r.Context(), userID)
		if keysErr != nil {
			h.log.Error("failed to fetch age keys for response", "error", keysErr)
		} else {
			resp.DeviceKeys = deviceKeys
		}
	}

	if resp.Status == SendStatusFailed {
		core.WriteError(w, http.StatusBadGateway, "push_delivery_failed", "Push delivery failed for all devices")
		return
	}

	core.WriteOK(w, resp)
}

// Keys handles GET /v1/push/keys.
// Returns the age public keys for all paired devices of the authenticated user.
func (h *Handler) Keys(w http.ResponseWriter, r *http.Request) {
	rawToken, ok := middleware.BearerTokenFromContext(r.Context())
	if !ok || rawToken == "" {
		core.WriteUnauthorized(w, codeMissingToken, "Bearer token required")
		return
	}

	userID, err := h.pushAuth.ValidateAPIToken(r.Context(), rawToken)
	if err != nil {
		core.WriteUnauthorized(w, codeUnauthorized, "invalid or unauthorized token")
		return
	}
	if h.keyProvider == nil {
		h.log.Error("key provider not configured")
		core.WriteInternalError(w, r, errors.New("key provider not configured"))
		return
	}

	deviceKeys, err := h.keyProvider.GetDeviceKeys(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to fetch age keys", "error", err, "user_id", userID)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, KeysResponse{Data: deviceKeys})
}

// parseSendOctetStream reads a raw octet-stream body into a SendInput.
// The server is zero-knowledge: no validation of the blob content.
// Priority is read from the X-Priority header (defaults to empty = normal).
func (h *Handler) parseSendOctetStream(r *http.Request) (SendInput, []error, error) {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxAttachmentBytes+1))
	if err != nil {
		return SendInput{}, nil, err
	}
	if int64(len(data)) > maxAttachmentBytes {
		return SendInput{}, nil, core.ErrPayloadTooLarge
	}

	priority := r.Header.Get(push.PriorityHeader)
	errs := validator.Validate(
		validator.NotBlank("body", string(data)),
		validator.OneOf("priority", priority, push.ValidPriorities),
	)
	if len(errs) > 0 {
		return SendInput{}, errs, nil
	}

	return SendInput{
		OpaqueBlob:   data,
		Priority:     priority,
		DeliveryMode: DeliveryModeE2E,
	}, nil, nil
}

// parseSendJSON decodes a JSON request body into a SendInput.
// Returns (input, validationErrors, decodeError).
func (h *Handler) parseSendJSON(r *http.Request) (SendInput, []error, error) {
	var req SendRequest
	if err := core.DecodeJSON(r.Body, &req); err != nil {
		return SendInput{}, nil, err
	}

	if errs := req.Validate(); len(errs) > 0 {
		return SendInput{}, errs, nil
	}

	input := SendInput{
		Title:    req.Title,
		Body:     req.Body,
		Priority: req.Priority,
	}

	if req.AttachmentURL != "" {
		u, _ := url.Parse(req.AttachmentURL)
		filename := ""
		if u != nil {
			filename = sanitizeFilename(path.Base(u.Path))
		}
		input.Attachment = &AttachmentInput{
			URL:      req.AttachmentURL,
			Filename: filename,
		}
	}

	return input, nil, nil
}

// parseSendMultipart decodes a multipart/form-data request into a SendInput.
// Returns (input, validationErrors, decodeError).
func (h *Handler) parseSendMultipart(r *http.Request) (SendInput, []error, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, multipartMaxBytes)

	if err := r.ParseMultipartForm(maxAttachmentBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || errors.Is(err, multipart.ErrMessageTooLarge) {
			return SendInput{}, nil, core.ErrPayloadTooLarge
		}
		return SendInput{}, nil, err
	}

	req := SendRequest{
		Title:    r.FormValue("title"),
		Body:     r.FormValue("body"),
		Priority: r.FormValue("priority"),
	}

	if errs := req.Validate(); len(errs) > 0 {
		return SendInput{}, errs, nil
	}

	input := SendInput{
		Title:    req.Title,
		Body:     req.Body,
		Priority: req.Priority,
	}

	file, header, err := r.FormFile("attachment")
	if err == nil {
		// Detect MIME from first 512 bytes.
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		mimeType := http.DetectContentType(buf[:n])

		// Seek back to start — multipart.File supports Seek.
		if seeker, ok := file.(interface {
			Seek(int64, int) (int64, error)
		}); ok {
			_, _ = seeker.Seek(0, 0)
		}

		input.Attachment = &AttachmentInput{
			Data:     file,
			MimeType: mimeType,
			Filename: sanitizeFilename(header.Filename),
		}
	} else if err != http.ErrMissingFile {
		return SendInput{}, nil, err
	}

	return input, nil, nil
}

// sanitizeFilename strips path components and control characters from a filename.
// Returns an empty string if the result is empty or a bare dot.
func sanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7F {
			return -1
		}
		return r
	}, name)
	const maxLen = 255
	if len(name) > maxLen {
		name = name[:maxLen]
	}
	if name == "." || name == "" {
		return ""
	}
	return name
}

// VAPIDPublicKey handles requests for the VAPID public key.
func (h *Handler) VAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	core.WriteJSON(w, http.StatusOK, VAPIDPublicKeyResponse{
		Key: h.service.VAPIDPublicKey(),
	})
}
