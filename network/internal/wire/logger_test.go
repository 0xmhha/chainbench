package wire

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestSetupLoggerTo_LevelDefault_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := SetupLoggerTo(&buf, slog.LevelInfo)
	logger.Debug("hidden")
	logger.Info("shown")
	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Errorf("debug should not appear at info level: %q", out)
	}
	if !strings.Contains(out, "shown") {
		t.Errorf("info should appear: %q", out)
	}
}

func TestSetupLoggerTo_WritesValidJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := SetupLoggerTo(&buf, slog.LevelInfo)
	logger.Info("hello", "key", "value")
	out := strings.TrimSpace(buf.String())
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("log line not valid JSON: %v (%q)", err, out)
	}
	for _, field := range []string{"time", "level", "msg"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("missing %q field: %v", field, parsed)
		}
	}
	if parsed["msg"] != "hello" {
		t.Errorf("msg: got %v, want hello", parsed["msg"])
	}
}

func TestSetupLoggerTo_DebugLevel_ShowsAll(t *testing.T) {
	var buf bytes.Buffer
	logger := SetupLoggerTo(&buf, slog.LevelDebug)
	logger.Debug("debug-msg")
	logger.Info("info-msg")
	if !strings.Contains(buf.String(), "debug-msg") {
		t.Errorf("debug should appear at debug level")
	}
	if !strings.Contains(buf.String(), "info-msg") {
		t.Errorf("info should appear at debug level")
	}
}
