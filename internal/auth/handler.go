package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
)

// Handler handles auth HTTP requests.
type Handler struct {
	authService  *Service
	cookieDomain string
	log          *slog.Logger
}

// NewHandler creates a new auth handler.
func NewHandler(authSvc *Service, cookieDomain string, logger *slog.Logger) *Handler {
	return &Handler{
		authService:  authSvc,
		cookieDomain: cookieDomain,
		log:          logger,
	}
}

// Login handles login requests.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	_, err := h.authService.RequestAuth(r.Context(), req.Email, req.State, req.Reason)
	if err != nil {
		if errors.Is(err, ErrGlobalRateLimit) {
			// Retry-After makes the global safety valve explicit without leaking account state.
			w.Header().Set("Retry-After", "60")
			core.WriteTooManyRequests(w, "too_many_requests_auth", "too many requests, please try again later")
			return
		}
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}

// VerifyOTP handles OTP verification requests.
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest

	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	userID, err := h.authService.VerifyOTP(r.Context(), req.OTP, req.State)
	if err != nil {
		if errors.Is(err, core.ErrUnauthorized) {
			if errors.Is(err, ErrOTPMaxAttempts) {
				core.WriteTooManyRequests(w, "too_many_requests_otp", "too many attempts, please request a new code")
				return
			}
			core.WriteUnauthorized(w, "invalid_otp", "invalid or expired OTP")
			return
		}
		core.WriteInternalError(w, r, err)
		return
	}

	session, err := h.authService.CreateSession(r.Context(), userID)
	if err != nil {
		core.WriteInternalError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieSessionName,
		Value:    session.Token,
		Path:     "/",
		Domain:   h.cookieDomain,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieLoggedInName,
		Value:    "1",
		Path:     "/",
		Domain:   h.cookieDomain,
		Expires:  session.ExpiresAt,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	core.WriteJSON(w, http.StatusOK, MessageResponse{
		Message: "Authentication successful",
	})
}

// Logout handles logout requests.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteInternalError(w, r, errors.New("failed to fetch user from context"))
		return
	}

	sessionID, err := r.Cookie(middleware.CookieSessionName)
	if err != nil {
		core.WriteUnauthorized(w, "invalid_session", "No session found")
		return
	}

	if err := h.authService.RevokeSession(r.Context(), user.ID, sessionID.Value); err != nil {
		h.log.Error("failed to revoke session", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieSessionName,
		Value:    "",
		Path:     "/",
		Domain:   h.cookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieLoggedInName,
		Value:    "",
		Path:     "/",
		Domain:   h.cookieDomain,
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	core.WriteNoContent(w)
}
