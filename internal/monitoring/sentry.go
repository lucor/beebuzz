package monitoring

import (
	"log/slog"
	"time"

	sentrysdk "github.com/getsentry/sentry-go"
)

// Init initializes the Sentry SDK with the provided configuration.
// If dsn is empty, Sentry is disabled and all operations become no-ops.
func Init(dsn, environment, release string) error {
	if dsn == "" {
		slog.Debug("sentry disabled: no DSN provided")
		return nil
	}

	err := sentrysdk.Init(sentrysdk.ClientOptions{
		Dsn:         dsn,
		Environment: environment,
		Release:     release,
	})
	if err != nil {
		return err
	}

	slog.Info("sentry initialized", "environment", environment, "release", release)
	return nil
}

// Flush waits for queued events to be sent to Sentry.
// Should be called on shutdown with a reasonable timeout (e.g., 2 seconds).
func Flush(timeout time.Duration) bool {
	return sentrysdk.Flush(timeout)
}
