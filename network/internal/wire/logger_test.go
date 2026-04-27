package wire

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
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

func TestParseLevel_AllBranches(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"warn":    slog.LevelWarn,
		"WARN":    slog.LevelWarn,
		"error":   slog.LevelError,
		"unknown": slog.LevelInfo, // default fallback
		"":        slog.LevelInfo, // empty fallback
	}
	for input, want := range cases {
		if got := parseLevel(input); got != want {
			t.Errorf("parseLevel(%q): got %v, want %v", input, got, want)
		}
	}
}

func TestSetupLogger_EnvDefaultsToStderr(t *testing.T) {
	t.Setenv(envLogLevel, "")
	t.Setenv(envLogFile, "")
	logger := SetupLogger()
	if logger == nil {
		t.Fatal("logger is nil")
	}
	// Smoke: default logger should accept Info without panic. Output goes to
	// stderr in this mode; we cannot easily capture it but we verified no panic.
	logger.Info("smoke")
}

func TestSetupLogger_EnvPathWritesToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "net.log")
	t.Setenv(envLogFile, path)
	t.Setenv(envLogLevel, "debug")

	logger := SetupLogger()
	logger.Debug("env-debug")
	logger.Info("env-info")

	// Read back the file and verify both lines landed.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "env-debug") {
		t.Errorf("file missing env-debug: %q", out)
	}
	if !strings.Contains(out, "env-info") {
		t.Errorf("file missing env-info: %q", out)
	}
}

func TestSetupLogger_EnvPathBadFallsBackToStderr(t *testing.T) {
	// A directory that cannot be opened as a file should trigger silent fallback.
	t.Setenv(envLogFile, "/nonexistent/dir/that/does/not/exist/net.log")
	t.Setenv(envLogLevel, "info")
	logger := SetupLogger()
	if logger == nil {
		t.Fatal("logger is nil despite fallback path")
	}
	// No panic, no assertion on output (it went to stderr); just proving the
	// fallback branch is reached.
}

func TestSetupLoggerWithFallback_NoEnv_UsesFallback(t *testing.T) {
	t.Setenv(envLogFile, "")
	t.Setenv(envLogLevel, "")
	var buf bytes.Buffer
	logger := SetupLoggerWithFallback(&buf)
	logger.Info("from-fallback")
	if !strings.Contains(buf.String(), "from-fallback") {
		t.Errorf("fallback writer should receive log: %q", buf.String())
	}
}

func TestSetupLoggerWithFallback_EnvPath_BypassesFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "net.log")
	t.Setenv(envLogFile, path)
	t.Setenv(envLogLevel, "info")

	var fallback bytes.Buffer
	logger := SetupLoggerWithFallback(&fallback)
	logger.Info("env-routed")

	if fallback.Len() != 0 {
		t.Errorf("fallback should be empty when env redirects to file: %q", fallback.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "env-routed") {
		t.Errorf("log file missing env-routed: %q", string(data))
	}
}

func TestSetupLoggerWithFallback_RespectsLogLevel(t *testing.T) {
	t.Setenv(envLogFile, "")
	t.Setenv(envLogLevel, "warn")
	var buf bytes.Buffer
	logger := SetupLoggerWithFallback(&buf)
	logger.Info("info-hidden")
	logger.Warn("warn-shown")
	out := buf.String()
	if strings.Contains(out, "info-hidden") {
		t.Errorf("info should be hidden at warn level: %q", out)
	}
	if !strings.Contains(out, "warn-shown") {
		t.Errorf("warn should be visible at warn level: %q", out)
	}
}

func TestSetupLoggerWithFallback_EnvPathBadFallsBack(t *testing.T) {
	t.Setenv(envLogFile, "/nonexistent/dir/that/does/not/exist/net.log")
	t.Setenv(envLogLevel, "info")
	var buf bytes.Buffer
	logger := SetupLoggerWithFallback(&buf)
	logger.Info("after-bad-env")
	if !strings.Contains(buf.String(), "after-bad-env") {
		t.Errorf("fallback should receive log when env path is unreachable: %q", buf.String())
	}
}
