package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/jmoiron/sqlx"

	"lucor.dev/beebuzz/internal/admin"
	"lucor.dev/beebuzz/internal/attachment"
	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/config"
	"lucor.dev/beebuzz/internal/database"
	"lucor.dev/beebuzz/internal/device"
	"lucor.dev/beebuzz/internal/event"
	"lucor.dev/beebuzz/internal/health"
	"lucor.dev/beebuzz/internal/logger"
	"lucor.dev/beebuzz/internal/mailer"
	"lucor.dev/beebuzz/internal/middleware"
	"lucor.dev/beebuzz/internal/migrations"
	"lucor.dev/beebuzz/internal/monitoring"
	"lucor.dev/beebuzz/internal/notification"
	systemnotifications "lucor.dev/beebuzz/internal/system/notifications"
	"lucor.dev/beebuzz/internal/token"
	"lucor.dev/beebuzz/internal/topic"
	"lucor.dev/beebuzz/internal/user"
	"lucor.dev/beebuzz/internal/webhook"
)

const (
	httpReadTimeout           = 15 * time.Second
	httpWriteTimeout          = 15 * time.Second
	httpIdleTimeout           = 60 * time.Second
	httpReadHeaderTimeout     = 5 * time.Second
	httpShutdownTimeout       = 30 * time.Second
	housekeepingInterval      = 1 * time.Hour
	eventCompactionInterval   = 24 * time.Hour
	authGlobalThrottleLimit   = 20
	authGlobalThrottleWindow  = 1 * time.Minute
	authEmailThrottleLimit    = 3
	authEmailThrottleWindow   = 15 * time.Minute
	authEmailThrottleCooldown = 1 * time.Minute
)

// appServices collects the domain services used by handlers and background jobs.
type appServices struct {
	db             health.DBPinger
	adminSvc       *admin.Service
	attachmentSvc  *attachment.Service
	authSvc        *auth.Service
	deviceSvc      *device.Service
	eventSvc       *event.Service
	notifSvc       *notification.Service
	systemNotifSvc *systemnotifications.Service
	tokenSvc       *token.Service
	topicSvc       *topic.Service
	userSvc        *user.Service
	webhookSvc     *webhook.Service
	pushStubBroker *notification.PushStubBroker
}

// runServe bootstraps and runs the HTTP server lifecycle.
func runServe() error {
	version := "beebuzz@" + commitHash[:min(7, len(commitHash))]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Env)
	slog.SetDefault(log)

	// Initialize monitoring/Sentry (no-op if DSN is empty)
	if err := monitoring.Init(cfg.SentryDSN, cfg.Env, version); err != nil {
		log.Error("monitoring initialization failed", "error", err)
	}
	defer monitoring.Flush(2 * time.Second)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.New(cfg.DBDir)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close() //nolint:errcheck

	if err := migrations.Run(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	m, err := mailer.New(cfg.Mailer)
	if err != nil {
		return fmt.Errorf("initialize mailer: %w", err)
	}

	services, err := buildServices(db, cfg, log, m)
	if err != nil {
		return err
	}

	handler := buildHTTPHandler(services, cfg, log, version)
	httpServer := newHTTPServer(cfg.Port, handler)
	startBackgroundWorkers(rootCtx, services, log)

	log.Info("beebuzz server starting", "address", httpServer.Addr, "env", cfg.Env, "version", version)

	if err := runHTTPServer(rootCtx, httpServer); err != nil {
		return err
	}

	log.Info("server stopped cleanly")
	return nil
}

// buildServices wires all domain services and adapters.
func buildServices(db *sqlx.DB, cfg *config.Config, log *slog.Logger, m mailer.Mailer) (*appServices, error) {
	topicRepo := topic.NewRepository(db)
	topicSvc := topic.NewService(topicRepo, log)

	authRepo := auth.NewRepository(db)
	authTopicInit := &authTopicInitializerAdapter{topicSvc: topicSvc}
	authSvc := auth.NewService(authRepo, m, cfg.URL, authTopicInit, log)
	authSvc.UsePrivateBeta(cfg.PrivateBeta)
	authSvc.SetBootstrapAdminEmail(cfg.BootstrapAdminEmail)
	authSvc.SetGlobalThrottle(auth.NewGlobalAuthThrottle(
		authGlobalThrottleLimit,
		authGlobalThrottleWindow,
	))
	authSvc.SetEmailThrottle(auth.NewEmailThrottle(
		authEmailThrottleLimit,
		authEmailThrottleWindow,
		authEmailThrottleCooldown,
	))

	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)

	adminRepo := admin.NewRepository(db)
	adminSessionRevoker := &adminSessionRevokerAdapter{svc: authSvc}
	adminSvc := admin.NewService(adminRepo, adminSessionRevoker, m, log)

	eventRepo := event.NewRepository(db)
	eventSvc := event.NewService(eventRepo, log)

	deviceRepo := device.NewRepository(db)
	deviceTopicValidator := &deviceTopicValidatorAdapter{topicSvc: topicSvc}
	deviceSvc := device.NewService(deviceRepo, deviceTopicValidator, log)

	tokenRepo := token.NewRepository(db)
	tokenTopicValidator := &tokenTopicValidatorAdapter{topicSvc: topicSvc}
	tokenSvc := token.NewService(tokenRepo, tokenTopicValidator)

	attachmentRepo := attachment.NewRepository(db)
	attachmentSvc := attachment.NewService(attachmentRepo, cfg.AttachmentsDir, log)

	vapidKeys, err := loadVAPIDKeys(cfg)
	if err != nil {
		return nil, fmt.Errorf("load VAPID keys: %w", err)
	}

	notifDeviceAdapter := &notificationDeviceAdapter{deviceSvc: deviceSvc}
	notifAttachmentAdapter := &notificationAttachmentAdapter{attachmentSvc: attachmentSvc}
	notifEventTracker := &notificationEventTrackerAdapter{eventSvc: eventSvc}
	notifSvc := notification.NewService(notifDeviceAdapter, notifAttachmentAdapter, notifEventTracker, vapidKeys, cfg.VAPIDSubject, log)

	var pushStubBroker *notification.PushStubBroker
	if cfg.PushStub && cfg.Env != config.EnvProduction {
		pushStubBroker = notification.NewPushStubBroker(log)
		notifSvc.SetPushStubBroker(pushStubBroker)
	}

	systemNotifRepo := systemnotifications.NewRepository(db)
	systemNotifTopics := &systemNotificationTopicProviderAdapter{topicSvc: topicSvc}
	systemNotifDelivery := &systemNotificationDeliveryAdapter{notifSvc: notifSvc, log: log}
	systemNotifSubscriptions := &systemNotificationDeviceSubscriptionAdapter{deviceSvc: deviceSvc}
	systemNotifSvc := systemnotifications.NewService(systemNotifRepo, systemNotifTopics, systemNotifDelivery, systemNotifSubscriptions, log)
	authSvc.SetSignupNotifier(systemNotifSvc)

	webhookRepo := webhook.NewRepository(db)
	webhookInspectStore := webhook.NewInspectStore()
	webhookDispatcher := &webhookDispatcherAdapter{notifSvc: notifSvc}
	webhookTopicValidator := &webhookTopicValidatorAdapter{topicSvc: topicSvc}
	webhookSvc := webhook.NewService(webhookRepo, webhookInspectStore, webhookDispatcher, webhookTopicValidator, log)

	return &appServices{
		db:             db,
		adminSvc:       adminSvc,
		attachmentSvc:  attachmentSvc,
		authSvc:        authSvc,
		deviceSvc:      deviceSvc,
		eventSvc:       eventSvc,
		notifSvc:       notifSvc,
		systemNotifSvc: systemNotifSvc,
		tokenSvc:       tokenSvc,
		topicSvc:       topicSvc,
		userSvc:        userSvc,
		webhookSvc:     webhookSvc,
		pushStubBroker: pushStubBroker,
	}, nil
}

// loadVAPIDKeys requires explicit env-backed keys so the server never treats
// SQLite as a secret store for push signing material.
func loadVAPIDKeys(cfg *config.Config) (*notification.VAPIDKeys, error) {
	if cfg.VAPIDPublicKey == "" || cfg.VAPIDPrivateKey == "" {
		return nil, fmt.Errorf("BEEBUZZ_VAPID_PUBLIC_KEY and BEEBUZZ_VAPID_PRIVATE_KEY are required")
	}

	return &notification.VAPIDKeys{
		PublicKey:  cfg.VAPIDPublicKey,
		PrivateKey: cfg.VAPIDPrivateKey,
	}, nil
}

// buildHTTPHandler creates the router and wraps it with global middleware.
func buildHTTPHandler(services *appServices, cfg *config.Config, log *slog.Logger, version string) http.Handler {
	authHandler := auth.NewHandler(services.authSvc, cfg.CookieDomain, log)
	healthHandler := health.NewHandler(version, services.db)
	userHandler := user.NewHandler(services.userSvc, log)
	topicHandler := topic.NewHandler(services.topicSvc, log)
	adminHandler := admin.NewHandler(services.adminSvc, log)
	systemNotificationsHandler := systemnotifications.NewHandler(services.systemNotifSvc, log)
	eventHandler := event.NewHandler(services.eventSvc, log)
	pushAuth := &pushAuthorizerAdapter{tokenSvc: services.tokenSvc}
	keyProvider := &keyProviderAdapter{deviceSvc: services.deviceSvc}
	notificationHandler := notification.NewHandler(services.notifSvc, pushAuth, keyProvider, log)
	deviceHandler := device.NewHandler(services.deviceSvc, cfg.HiveURL, log)
	webhookHandler := webhook.NewHandler(services.webhookSvc, cfg.HookURL, log)
	attachmentHandler := attachment.NewHandler(services.attachmentSvc, log)
	tokenHandler := token.NewHandler(services.tokenSvc, log)

	var pushStubHandler http.HandlerFunc
	if services.pushStubBroker != nil {
		pushStubHandler = notification.NewPushStubHandler(services.pushStubBroker, log)
	}

	realIP := middleware.NewRealIP(cfg.ProxySubnet)
	ipHasher := middleware.NewIPHasher(cfg.IPHashSalt)
	requestID := middleware.NewRequestID(cfg.RequestIDHeader)
	sessionAdapter := &sessionValidatorAdapter{authSvc: services.authSvc}
	router := NewRouter(
		sessionAdapter,
		authHandler,
		healthHandler,
		userHandler,
		topicHandler,
		adminHandler,
		systemNotificationsHandler,
		eventHandler,
		notificationHandler,
		deviceHandler,
		webhookHandler,
		attachmentHandler,
		tokenHandler,
		pushStubHandler,
		cfg,
		log,
	)

	// Sentry handler for panic recovery and scope management
	sentryHandler := sentryhttp.New(sentryhttp.Options{
		Repanic:         false,
		WaitForDelivery: false,
		Timeout:         2 * time.Second,
	})

	// Middleware chain: Sentry (recovery) -> RealIP -> IPHasher -> RequestID -> SentryTags -> Router
	return sentryHandler.Handle(realIP.Middleware(
		ipHasher.Middleware(
			requestID.Middleware(
				middleware.SentryTags(router),
			),
		),
	))
}

// newHTTPServer creates the configured HTTP server instance.
func newHTTPServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadTimeout:       httpReadTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}
}

// startBackgroundWorkers launches long-running maintenance jobs tied to the root context.
func startBackgroundWorkers(ctx context.Context, services *appServices, log *slog.Logger) {
	go runHousekeeping(ctx, services.attachmentSvc, services.authSvc, log)
	go runEventCompaction(ctx, services.eventSvc, log)
}

// runHousekeeping periodically removes stale low-priority data until shutdown.
func runHousekeeping(ctx context.Context, attachmentSvc *attachment.Service, authSvc *auth.Service, log *slog.Logger) {
	ticker := time.NewTicker(housekeepingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := attachmentSvc.CleanupExpired(ctx); err != nil {
				log.Error("attachment cleanup failed", "error", err)
			}
			if err := authSvc.CleanupExpired(ctx); err != nil {
				log.Error("auth cleanup failed", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// runEventCompaction periodically removes old events until shutdown.
func runEventCompaction(ctx context.Context, eventSvc *event.Service, log *slog.Logger) {
	ticker := time.NewTicker(eventCompactionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := eventSvc.CompactOldEvents(ctx); err != nil {
				log.Error("event compaction failed", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// runHTTPServer serves requests until startup failure or root context cancellation.
func runHTTPServer(ctx context.Context, httpServer *http.Server) error {
	serverErrCh := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- nil
			return
		}
		serverErrCh <- err
	}()

	select {
	case err := <-serverErrCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := <-serverErrCh; err != nil {
		return fmt.Errorf("serve server: %w", err)
	}

	return nil
}
