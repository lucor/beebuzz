// Package logger provides a logger for structured logging.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"go.beebuzz.app/beebuzz/internal/config"
)

// New creates and returns a configured slog.Logger based on environment.
// Text format for dev, JSON format for production.
func New(env string) *slog.Logger {
	output := io.Writer(os.Stderr)
	if path := os.Getenv("BEEBUZZ_LOG_FILE"); path != "" {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "beebuzz: open log file %q: %v\n", path, err)
		} else {
			output = io.MultiWriter(os.Stderr, file)
		}
	}

	var handler slog.Handler

	if env == config.EnvDevelopment {
		// Text format with nice defaults for development
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		return slog.New(handler)
	}
	// JSON format for production for structured logging
	handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler)
}
