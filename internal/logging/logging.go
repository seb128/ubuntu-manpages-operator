package logging

import (
	"log/slog"
	"os"
	"strings"
)

// BuildLogger creates a structured logger writing to stderr at the given level.
func BuildLogger(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})
	return slog.New(handler)
}
