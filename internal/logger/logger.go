// Package logger provides a logger for structured logging.
package logger

import (
	"log/slog"
	"os"

	"lucor.dev/beebuzz/internal/config"
)

// New creates and returns a configured slog.Logger based on environment.
// Text format for dev, JSON format for production.
func New(env string) *slog.Logger {
	var handler slog.Handler

	if env == config.EnvDevelopment {
		// Text format with nice defaults for development
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		return slog.New(handler)
	}
	// JSON format for production for structured logging
	handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler)
}
