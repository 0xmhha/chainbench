package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// The lifecycle reroutes are thin wrappers over chainbench.sh, so the unit
// tests use the same fake-script harness as network.stop_all: writeFakeChainbench
// echoes "fake chainbench: $@", and the assertions pin the exact argv the
// handler forwards so a lost flag (e.g. --quiet) surfaces as a deliberate diff.

func TestHandleNetworkInit_Happy(t *testing.T) {
	chainbenchDir := t.TempDir()
	writeFakeChainbench(t, chainbenchDir, 0)
	handler := newHandleNetworkInit(t.TempDir(), chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := handler(nil, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	stdout, _ := data["stdout"].(string)
	if !strings.Contains(stdout, "fake chainbench: init --profile default --quiet") {
		t.Errorf("stdout = %q, want 'init --profile default --quiet'", stdout)
	}
}

func TestHandleNetworkInit_BinaryPathForwarded(t *testing.T) {
	chainbenchDir := t.TempDir()
	writeFakeChainbench(t, chainbenchDir, 0)
	handler := newHandleNetworkInit(t.TempDir(), chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"profile": "minimal", "binary_path": "/opt/build/bin/gstable-pr1234"})
	data, err := handler(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	stdout, _ := data["stdout"].(string)
	want := "init --profile minimal --quiet --binary-path /opt/build/bin/gstable-pr1234"
	if !strings.Contains(stdout, want) {
		t.Errorf("stdout = %q, want substring %q", stdout, want)
	}
}

func TestHandleNetworkInit_InvalidProfile(t *testing.T) {
	chainbenchDir := t.TempDir()
	writeFakeChainbench(t, chainbenchDir, 0)
	handler := newHandleNetworkInit(t.TempDir(), chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"profile": "../etc/passwd"})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNetworkInit_RelativeBinaryPath(t *testing.T) {
	chainbenchDir := t.TempDir()
	writeFakeChainbench(t, chainbenchDir, 0)
	handler := newHandleNetworkInit(t.TempDir(), chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"binary_path": "build/bin/gstable"})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNetworkStartAll_Happy(t *testing.T) {
	chainbenchDir := t.TempDir()
	writeFakeChainbench(t, chainbenchDir, 0)
	handler := newHandleNetworkStartAll(t.TempDir(), chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"binary_path": "/opt/gstable"})
	data, err := handler(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	stdout, _ := data["stdout"].(string)
	if !strings.Contains(stdout, "start --quiet --binary-path /opt/gstable") {
		t.Errorf("stdout = %q, want 'start --quiet --binary-path /opt/gstable'", stdout)
	}
}

func TestHandleNetworkRestart_Happy(t *testing.T) {
	chainbenchDir := t.TempDir()
	writeFakeChainbench(t, chainbenchDir, 0)
	handler := newHandleNetworkRestart(t.TempDir(), chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := handler(nil, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	stdout, _ := data["stdout"].(string)
	if !strings.Contains(stdout, "restart --quiet") {
		t.Errorf("stdout = %q, want 'restart --quiet'", stdout)
	}
}

func TestHandleNetworkClean_Happy(t *testing.T) {
	chainbenchDir := t.TempDir()
	writeFakeChainbench(t, chainbenchDir, 0)
	handler := newHandleNetworkClean(t.TempDir(), chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := handler(nil, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	stdout, _ := data["stdout"].(string)
	if !strings.Contains(stdout, "fake chainbench: clean") {
		t.Errorf("stdout = %q, want 'fake chainbench: clean'", stdout)
	}
}

func TestAllHandlers_RegistersLifecycleReroutes(t *testing.T) {
	handlers := allHandlers("/s", "/c")
	for _, cmd := range []string{"network.init", "network.start_all", "network.restart", "network.clean"} {
		if _, ok := handlers[cmd]; !ok {
			t.Errorf("allHandlers missing %s", cmd)
		}
	}
}

func TestHandleNetworkInit_UpstreamError(t *testing.T) {
	chainbenchDir := t.TempDir()
	writeFakeChainbench(t, chainbenchDir, 2)
	handler := newHandleNetworkInit(t.TempDir(), chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := handler(nil, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
}
