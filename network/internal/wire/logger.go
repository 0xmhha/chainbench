package wire

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	envLogLevel = "CHAINBENCH_NET_LOG_LEVEL"
	envLogFile  = "CHAINBENCH_NET_LOG"
)

// SetupLogger configures a slog JSON handler. Writer defaults to os.Stderr,
// or to the file at CHAINBENCH_NET_LOG if that env var is a non-empty path.
// Level from CHAINBENCH_NET_LOG_LEVEL: debug|info|warn|error (default info).
func SetupLogger() *slog.Logger {
	var w io.Writer = os.Stderr
	if path := os.Getenv(envLogFile); path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			w = f
		}
		// on error, silently fall back to stderr — logging must not block startup
	}
	return SetupLoggerTo(w, parseLevel(os.Getenv(envLogLevel)))
}

// SetupLoggerWithFallback configures a slog JSON handler with the same env
// behaviour as SetupLogger (CHAINBENCH_NET_LOG / CHAINBENCH_NET_LOG_LEVEL),
// but routes to the provided fallback writer when CHAINBENCH_NET_LOG is unset.
// Callers like runOnce pass their caller-injected stderr writer here so that
// production honours env redirection while tests still capture log output via
// the writer they injected.
func SetupLoggerWithFallback(fallback io.Writer) *slog.Logger {
	w := fallback
	if path := os.Getenv(envLogFile); path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			w = f
		}
		// on error, silently fall back — logging must not block startup
	}
	return SetupLoggerTo(w, parseLevel(os.Getenv(envLogLevel)))
}

// SetupLoggerTo is the testable variant with explicit writer and level.
// It also installs the returned logger as slog.Default for convenience.
func SetupLoggerTo(w io.Writer, level slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	logger := slog.New(h)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
