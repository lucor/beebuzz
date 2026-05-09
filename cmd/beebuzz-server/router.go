package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"lucor.dev/beebuzz/internal/admin"
	"lucor.dev/beebuzz/internal/attachment"
	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/config"
	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/device"
	"lucor.dev/beebuzz/internal/event"
	"lucor.dev/beebuzz/internal/health"
	"lucor.dev/beebuzz/internal/middleware"
	"lucor.dev/beebuzz/internal/notification"
	systemnotifications "lucor.dev/beebuzz/internal/system/notifications"
	"lucor.dev/beebuzz/internal/token"
	"lucor.dev/beebuzz/internal/topic"
	"lucor.dev/beebuzz/internal/user"
	"lucor.dev/beebuzz/internal/webhook"
)

// rateLimitHandler is the shared JSON 429 response for all rate limiters.
func rateLimitHandler(w http.ResponseWriter, _ *http.Request) {
	core.WriteTooManyRequests(w, "rate_limit_exceeded", "rate limit exceeded, try again later")
}

// rateLimit creates an httprate middleware with a custom KeyFunc.
func rateLimit(requests int, window time.Duration, keyFn httprate.KeyFunc) func(http.Handler) http.Handler {
	return httprate.Limit(requests, window,
		httprate.WithKeyFuncs(keyFn),
		httprate.WithLimitHandler(rateLimitHandler),
	)
}

var (
	rateLimitLogin   = rateLimit(5, time.Hour, middleware.RateKeyByHashedIP)
	rateLimitVerify  = rateLimit(10, time.Hour, middleware.RateKeyByHashedIP)
	rateLimitPairing = rateLimit(10, time.Minute, middleware.RateKeyByHashedIP)

	// Per-token limits on token-authenticated endpoints.
	rateLimitPushToken       = rateLimit(30, time.Minute, middleware.RateKeyByBearerToken)
	rateLimitWebhookToken    = rateLimit(10, time.Minute, middleware.RateKeyByURLParam("token"))
	rateLimitAttachmentToken = rateLimit(30, time.Minute, middleware.RateKeyByURLParam("token"))
)

// NewRouter creates the main HTTP router with all routes.
func NewRouter(
	sessionValidator middleware.SessionValidator,
	authHandler *auth.Handler,
	healthHandler *health.Handler,
	userHandler *user.Handler,
	topicHandler *topic.Handler,
	adminHandler *admin.Handler,
	systemNotificationsHandler *systemnotifications.Handler,
	eventHandler *event.Handler,
	notificationHandler *notification.Handler,
	deviceHandler *device.Handler,
	webhookHandler *webhook.Handler,
	attachmentHandler *attachment.Handler,
	tokenHandler *token.Handler,
	pushStubHandler http.HandlerFunc,
	cfg *config.Config,
	log *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.HostRewrite(extractHost(cfg.PushURL), extractHost(cfg.HookURL)))
	r.Use(middleware.BaseSecurity)
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.Get("/health", healthHandler.Health)

	r.Route("/v1", func(v1 chi.Router) {
		v1.Use(middleware.APISecurity)

		v1.With(rateLimitWebhookToken).Post("/webhooks/{token}", webhookHandler.Receive)

		// Public
		v1.Get("/health", healthHandler.Health)
		v1.Get("/vapid-public-key", notificationHandler.VAPIDPublicKey)
		v1.With(rateLimitPairing).Post("/pairing", deviceHandler.Pair)
		v1.With(middleware.ExtractBearerToken, rateLimitPairing).Get("/pairing/{deviceID}", deviceHandler.PairingStatus)
		v1.With(middleware.ExtractBearerToken, rateLimitPushToken).Get("/push/keys", notificationHandler.Keys)
		v1.With(middleware.ExtractBearerToken, rateLimitPushToken).Post("/push", notificationHandler.Send)
		v1.With(middleware.ExtractBearerToken, rateLimitPushToken).Post("/push/{topic}", notificationHandler.Send)

		// Attachments (public, token-based access)
		v1.With(rateLimitAttachmentToken).Get("/attachments/{token}", attachmentHandler.Get)

		// Auth routes (public)
		v1.Route("/auth", func(authR chi.Router) {
			// Public
			authR.With(middleware.RequireOrigin(cfg.SiteURL), rateLimitLogin).Post("/login", authHandler.Login)
			authR.With(middleware.RequireOrigin(cfg.SiteURL), rateLimitVerify).Post("/otp/verify", authHandler.VerifyOTP)

			// Authenticated
			authR.Group(func(authAuthenticated chi.Router) {
				authAuthenticated.Use(middleware.RequireSession(sessionValidator))
				authAuthenticated.Post("/logout", authHandler.Logout)
			})
		})

		// Authenticated routes
		v1.Group(func(authenticated chi.Router) {
			authenticated.Use(middleware.RequireSession(sessionValidator))

			// Me
			authenticated.Get("/me", userHandler.Me)
			authenticated.Get("/me/usage", eventHandler.AccountUsage)

			// Topics
			authenticated.Get("/topics", topicHandler.GetTopics)
			authenticated.Post("/topics", topicHandler.CreateTopic)
			authenticated.Patch("/topics/{topicID}", topicHandler.UpdateTopic)
			authenticated.Delete("/topics/{topicID}", topicHandler.DeleteTopic)

			// Devices
			authenticated.Get("/devices", deviceHandler.GetDevices)
			authenticated.Post("/devices", deviceHandler.CreateDevice)
			authenticated.Patch("/devices/{deviceID}", deviceHandler.UpdateDevice)
			authenticated.Delete("/devices/{deviceID}", deviceHandler.DeleteDevice)
			authenticated.Post("/devices/{deviceID}/pairing-code", deviceHandler.RegeneratePairingOTP)
			authenticated.Post("/devices/{deviceID}/unpair", deviceHandler.UnpairDevice)

			// API Tokens
			authenticated.Get("/tokens", tokenHandler.ListAPITokens)
			authenticated.Post("/tokens", tokenHandler.CreateAPIToken)
			authenticated.Patch("/tokens/{tokenID}", tokenHandler.UpdateAPIToken)
			authenticated.Delete("/tokens/{tokenID}", tokenHandler.RevokeAPIToken)

			// Webhooks
			authenticated.Get("/webhooks", webhookHandler.ListWebhooks)
			authenticated.Post("/webhooks", webhookHandler.CreateWebhook)
			authenticated.Patch("/webhooks/{webhookID}", webhookHandler.UpdateWebhook)
			authenticated.Delete("/webhooks/{webhookID}", webhookHandler.DeleteWebhook)
			authenticated.Post("/webhooks/{webhookID}/token", webhookHandler.RegenerateToken)

			// Webhook Inspect
			authenticated.Post("/webhooks/inspect", webhookHandler.CreateInspectSession)
			authenticated.Get("/webhooks/inspect", webhookHandler.GetInspectSession)
			authenticated.Post("/webhooks/inspect/finalize", webhookHandler.FinalizeInspect)

			// Admin routes
			authenticated.Group(func(adminR chi.Router) {
				adminR.Use(middleware.RequireAdmin())
				adminR.Get("/admin/users", adminHandler.ListUsers)
				adminR.Patch("/admin/users/{userID}", adminHandler.UpdateUserStatus)
				adminR.Get("/admin/dashboard", eventHandler.Dashboard)
				adminR.Get("/admin/system/notifications", systemNotificationsHandler.GetSettings)
				adminR.Patch("/admin/system/notifications", systemNotificationsHandler.UpdateSettings)
			})
		})
	})

	if pushStubHandler != nil {
		r.Get("/_stub/push/next", pushStubHandler)
	}

	return r
}

// extractHost parses a raw URL and returns the host (with port if present).
// Returns empty string if rawURL is empty or cannot be parsed.
func extractHost(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}
