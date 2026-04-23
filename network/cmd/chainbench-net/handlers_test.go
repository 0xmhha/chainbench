package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

// setupCmdStateDir builds a temp state dir populated from cmd testdata.
func setupCmdStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func newTestBus(t *testing.T) (*events.Bus, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	return events.NewBus(wire.NewEmitter(&buf)), &buf
}

func TestHandleNetworkLoad_HappyPath(t *testing.T) {
	dir := setupCmdStateDir(t)
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"name": "local"})
	data, err := handler(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["name"] != "local" {
		t.Errorf("name: got %v", data["name"])
	}
	nodes, ok := data["nodes"].([]any)
	if !ok || len(nodes) != 2 {
		t.Errorf("nodes: got %v", data["nodes"])
	}
}

func TestHandleNetworkLoad_WrongName(t *testing.T) {
	dir := setupCmdStateDir(t)
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"name": "mainnet"})
	_, err := handler(args, bus)
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want APIError INVALID_ARGS, got %v", err)
	}
}

func TestHandleNetworkLoad_MissingArgs(t *testing.T) {
	dir := setupCmdStateDir(t)
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{}) // name omitted
	_, err := handler(args, bus)
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNetworkLoad_StateMissing(t *testing.T) {
	dir := t.TempDir() // empty
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"name": "local"})
	_, err := handler(args, bus)
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
}

func TestAllHandlers_IncludesNetworkLoad(t *testing.T) {
	handlers := allHandlers("whatever", "also-whatever")
	if _, ok := handlers["network.load"]; !ok {
		t.Error("allHandlers missing network.load")
	}
}

// Keep the types import referenced for future table-expansion tests.
var _ = types.ResultErrorCode("NOT_SUPPORTED")

// ---- node.stop tests ----

// setupCmdStubDir lays out pids.json + profile + a symlink-style structure
// so chainbenchDir points to the testdata/ folder where chainbench-stub.sh
// lives. The handler is expected to call <chainbenchDir>/chainbench.sh,
// but the driver appends ".sh", so we rename-symlink to make testdata/chainbench.sh
// point at chainbench-stub.sh inside the temp directory.
func setupCmdStubDir(t *testing.T) (stateDir, chainbenchDir string) {
	t.Helper()
	stateDir = t.TempDir()
	// Copy state fixtures into stateDir.
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stateDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Build chainbenchDir with a chainbench.sh that delegates to the stub.
	chainbenchDir = t.TempDir()
	stub, err := os.ReadFile(filepath.Join("testdata", "chainbench-stub.sh"))
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(chainbenchDir, "chainbench.sh")
	if err := os.WriteFile(dst, stub, 0o755); err != nil {
		t.Fatal(err)
	}
	return stateDir, chainbenchDir
}

func TestHandleNodeStop_HappyPath(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	handler := newHandleNodeStop(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	sub := bus.Subscribe()

	args, _ := json.Marshal(map[string]any{"node_id": "node1"})
	data, err := handler(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["node_id"] != "node1" || data["stopped"] != true {
		t.Errorf("data: got %+v", data)
	}
	// Expect a node.stopped event.
	select {
	case ev := <-sub:
		if string(ev.Name) != "node.stopped" {
			t.Errorf("event name: got %q, want node.stopped", ev.Name)
		}
		if ev.Data["node_id"] != "node1" {
			t.Errorf("event data: got %+v", ev.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no node.stopped event received")
	}
}

func TestHandleNodeStop_MissingNodeID(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	handler := newHandleNodeStop(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{}) // node_id omitted
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeStop_BadPrefix(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	handler := newHandleNodeStop(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"node_id": "validator1"}) // wrong prefix
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeStop_UnknownNodeID(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	handler := newHandleNodeStop(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"node_id": "node99"})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeStop_SubprocessFails(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	// Replace chainbench.sh with a script that always exits 2.
	failScript := "#!/usr/bin/env bash\necho 'forced' >&2\nexit 2\n"
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), []byte(failScript), 0o755); err != nil {
		t.Fatal(err)
	}
	handler := newHandleNodeStop(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"node_id": "node1"})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
}

func TestAllHandlers_IncludesNodeStop(t *testing.T) {
	handlers := allHandlers("/state", "/chainbench")
	if _, ok := handlers["node.stop"]; !ok {
		t.Error("allHandlers missing node.stop")
	}
}
