package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

// setupRunStateDir reuses cmd testdata via an in-process temp dir.
func setupRunStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, _ := os.ReadFile(filepath.Join("testdata", name))
		_ = os.WriteFile(filepath.Join(dir, name), data, 0o644)
	}
	return dir
}

func TestRunOnce_HappyPath_NetworkLoad(t *testing.T) {
	dir := setupRunStateDir(t)
	stdin := strings.NewReader(`{"command":"network.load","args":{"name":"local"}}`)
	var stdout, stderr bytes.Buffer
	handlers := allHandlers(dir, "/nowhere")

	err := runOnce(stdin, &stdout, &stderr, handlers)
	if err != nil {
		t.Fatalf("runOnce: %v", err)
	}

	// Last line must be a successful result; decode all lines and verify.
	var terminator *wire.ResultMessage
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		msg, derr := wire.DecodeMessage(scanner.Bytes())
		if derr != nil {
			t.Fatalf("decode line %q: %v", scanner.Bytes(), derr)
		}
		if rm, ok := msg.(wire.ResultMessage); ok {
			rm := rm
			terminator = &rm
		}
	}
	if terminator == nil {
		t.Fatal("no result line in output")
	}
	if terminator.Ok != true {
		t.Fatalf("ok: got %v", terminator.Ok)
	}
	if terminator.Data == nil {
		t.Fatal("result.data is nil")
	}
}

func TestRunOnce_MalformedStdin_ProtocolError(t *testing.T) {
	stdin := strings.NewReader(`not json`)
	var stdout, stderr bytes.Buffer
	err := runOnce(stdin, &stdout, &stderr, map[string]Handler{})
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "PROTOCOL_ERROR" {
		t.Errorf("want PROTOCOL_ERROR, got %v", err)
	}
	// Result terminator must still have been emitted.
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	last := lines[len(lines)-1]
	var envelope struct {
		Type string `json:"type"`
		OK   bool   `json:"ok"`
	}
	_ = json.Unmarshal([]byte(last), &envelope)
	if envelope.Type != "result" || envelope.OK != false {
		t.Errorf("last line not an error result: %q", last)
	}
}

func TestRunOnce_UnknownCommand_NotSupported(t *testing.T) {
	// command.load is valid enum but no handler registered — simulate
	// by passing an empty handler table.
	stdin := strings.NewReader(`{"command":"network.load","args":{}}`)
	var stdout, stderr bytes.Buffer
	err := runOnce(stdin, &stdout, &stderr, map[string]Handler{})
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "NOT_SUPPORTED" {
		t.Errorf("want NOT_SUPPORTED, got %v", err)
	}
}

func TestRunOnce_HandlerPanic_Internal(t *testing.T) {
	stdin := strings.NewReader(`{"command":"network.load","args":{"name":"local"}}`)
	var stdout, stderr bytes.Buffer
	panicking := map[string]Handler{
		"network.load": func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
			panic("boom")
		},
	}
	err := runOnce(stdin, &stdout, &stderr, panicking)
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INTERNAL" {
		t.Errorf("want INTERNAL, got %v", err)
	}
}
