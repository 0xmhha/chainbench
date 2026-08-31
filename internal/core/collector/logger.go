package collector

import (
	"context"
	"io"
	"log/slog"
)

// NewLogger returns a structured slog.Logger writing JSON to w at the given
// level. JSON keeps chain (node) logs machine-parseable for the log-analysis
// surface; human-facing CLI output is formatted separately by the cmd layer.
func NewLogger(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}

// LogEvent records an Event through a slog.Logger at a level derived from the
// event kind (errors at ERROR, everything else at INFO). It is the bridge for
// sinks that want events persisted to the structured log as well as the bus.
func LogEvent(l *slog.Logger, e Event) {
	attrs := []any{
		slog.String("phase", string(e.Phase)),
		slog.String("kind", string(e.Kind)),
	}
	if e.Network != "" {
		attrs = append(attrs, slog.String("network", e.Network))
	}
	if e.Node != 0 {
		attrs = append(attrs, slog.Int("node", e.Node))
	}
	for k, v := range e.Fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	level := slog.LevelInfo
	if e.Kind == KindError {
		level = slog.LevelError
	}
	l.Log(context.Background(), level, e.Message, attrs...)
}
