package topic

import (
	"errors"
	"log/slog"
	"net/http"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
)

// Handler handles topic HTTP requests.
type Handler struct {
	topicService *Service
	log          *slog.Logger
}

// NewHandler creates a new topic handler.
func NewHandler(topicSvc *Service, logger *slog.Logger) *Handler {
	return &Handler{
		topicService: topicSvc,
		log:          logger,
	}
}

// GetTopics handles GET requests for topics.
func (h *Handler) GetTopics(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	topics, err := h.topicService.GetTopics(r.Context(), userCtx.ID)
	if err != nil {
		h.log.Error("failed to get topics", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, ToTopicsListResponse(topics))
}

// CreateTopic handles POST requests to create a topic.
func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	var req CreateTopicRequest

	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	topic, err := h.topicService.CreateTopic(r.Context(), userCtx.ID, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, ErrTopicNameReserved) {
			core.WriteBadRequest(w, "invalid_topic_name", err.Error())
			return
		}
		if errors.Is(err, ErrTopicNameConflict) {
			core.WriteConflict(w, "topic_name_conflict", err.Error())
			return
		}
		h.log.Error("failed to create topic", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	resp := ToTopicResponse(topic)
	core.WriteJSON(w, http.StatusCreated, resp)
}

// UpdateTopic handles PATCH requests to update a topic's description.
func (h *Handler) UpdateTopic(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	topicID := core.GetURLParam(r, "topicID")
	if topicID == "" {
		core.WriteBadRequest(w, "missing_param", "topicID is required")
		return
	}

	var req UpdateTopicRequest

	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	if err := h.topicService.UpdateTopic(r.Context(), userCtx.ID, topicID, req.Description); err != nil {
		if errors.Is(err, ErrTopicNotFound) {
			core.WriteNotFound(w, "topic_not_found", "Topic not found")
			return
		}
		h.log.Error("failed to update topic", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}

// DeleteTopic handles DELETE requests to delete a topic.
func (h *Handler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	topicID := core.GetURLParam(r, "topicID")
	if topicID == "" {
		core.WriteBadRequest(w, "missing_param", "topicID is required")
		return
	}

	if err := h.topicService.DeleteTopic(r.Context(), userCtx.ID, topicID); err != nil {
		if errors.Is(err, ErrTopicNotFound) {
			core.WriteNotFound(w, "topic_not_found", "Topic not found")
			return
		}
		if errors.Is(err, ErrTopicProtected) {
			core.WriteConflict(w, "topic_protected", err.Error())
			return
		}
		h.log.Error("failed to delete topic", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}
