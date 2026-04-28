package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xmhha/chainbench/network/internal/abiutil"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/state"
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

// TestHandleNetworkLoad_UnknownRemoteName verifies that requesting a non-local
// network name with no matching state file surfaces as UPSTREAM_ERROR.
// state.LoadActive routes non-local names through loadRemote which returns a
// wrapped ErrStateNotFound; the handler maps that to UPSTREAM_ERROR (the same
// bucket a missing pids.json for "local" uses).
func TestHandleNetworkLoad_UnknownRemoteName(t *testing.T) {
	dir := setupCmdStateDir(t) // contains local state, but no networks/mainnet.json
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"name": "mainnet"})
	_, err := handler(args, bus)
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want APIError UPSTREAM_ERROR, got %v", err)
	}
}

// TestHandleNetworkLoad_BadNamePattern ensures that a structurally invalid
// name is rejected at the handler boundary as INVALID_ARGS, not pushed down
// to the state layer where it would surface as UPSTREAM_ERROR wrapping
// ErrInvalidName.
func TestHandleNetworkLoad_BadNamePattern(t *testing.T) {
	dir := setupCmdStateDir(t)
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	for _, name := range []string{"Has-Upper", "has space", "has/slash", ".hidden"} {
		t.Run(name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]any{"name": name})
			_, err := handler(args, bus)
			if err == nil {
				t.Fatal("expected error")
			}
			var api *APIError
			if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
				t.Errorf("want INVALID_ARGS, got %v", err)
			}
		})
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

// ---- node.start tests ----

func TestHandleNodeStart_HappyPath(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	handler := newHandleNodeStart(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	sub := bus.Subscribe()

	args, _ := json.Marshal(map[string]any{"node_id": "node1"})
	data, err := handler(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["node_id"] != "node1" || data["started"] != true {
		t.Errorf("data: got %+v", data)
	}
	select {
	case ev := <-sub:
		if string(ev.Name) != "node.started" {
			t.Errorf("event name: got %q", ev.Name)
		}
		if ev.Data["node_id"] != "node1" {
			t.Errorf("event data: got %+v", ev.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no node.started event received")
	}
}

func TestHandleNodeStart_MissingNodeID(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	handler := newHandleNodeStart(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	args, _ := json.Marshal(map[string]any{})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeStart_UnknownNodeID(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	handler := newHandleNodeStart(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	args, _ := json.Marshal(map[string]any{"node_id": "node99"})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeStart_SubprocessFails(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	failScript := "#!/usr/bin/env bash\necho 'forced' >&2\nexit 2\n"
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), []byte(failScript), 0o755); err != nil {
		t.Fatal(err)
	}
	handler := newHandleNodeStart(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	args, _ := json.Marshal(map[string]any{"node_id": "node1"})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
}

func TestAllHandlers_IncludesNodeStart(t *testing.T) {
	handlers := allHandlers("/s", "/c")
	if _, ok := handlers["node.start"]; !ok {
		t.Error("allHandlers missing node.start")
	}
}

// ---- node.restart tests ----

func TestHandleNodeRestart_HappyPath_EmitsBothEvents(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	handler := newHandleNodeRestart(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	sub := bus.Subscribe()

	args, _ := json.Marshal(map[string]any{"node_id": "node1"})
	data, err := handler(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["node_id"] != "node1" || data["restarted"] != true {
		t.Errorf("data: got %+v", data)
	}

	// Expect exactly: node.stopped THEN node.started.
	wantOrder := []string{"node.stopped", "node.started"}
	for i, want := range wantOrder {
		select {
		case ev := <-sub:
			if string(ev.Name) != want {
				t.Fatalf("event %d: got %q, want %q", i, ev.Name, want)
			}
		case <-time.After(300 * time.Millisecond):
			t.Fatalf("event %d timeout (want %q)", i, want)
		}
	}
	// No third event.
	select {
	case ev := <-sub:
		t.Errorf("unexpected 3rd event: %q", ev.Name)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHandleNodeRestart_MissingNodeID(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	handler := newHandleNodeRestart(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	args, _ := json.Marshal(map[string]any{})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeRestart_StopFails_NoStartedEvent(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	// Replace chainbench.sh with a script that ALWAYS fails (stop attempt fails first).
	failScript := "#!/usr/bin/env bash\necho 'always fails' >&2\nexit 2\n"
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), []byte(failScript), 0o755); err != nil {
		t.Fatal(err)
	}
	handler := newHandleNodeRestart(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	sub := bus.Subscribe()

	args, _ := json.Marshal(map[string]any{"node_id": "node1"})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
	// No events should have been emitted (stop failed before stopped-event).
	select {
	case ev := <-sub:
		t.Errorf("unexpected event: %q", ev.Name)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestHandleNodeRestart_StartFailsAfterStop(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	// Script that succeeds on stop (arg: "node stop N") but fails on start.
	selectiveScript := `#!/usr/bin/env bash
case "$1 $2" in
  "node stop") echo "stub: stopped"; exit 0 ;;
  "node start") echo "start failed" >&2; exit 1 ;;
  *) echo "unknown $*" >&2; exit 2 ;;
esac
`
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), []byte(selectiveScript), 0o755); err != nil {
		t.Fatal(err)
	}
	handler := newHandleNodeRestart(stateDir, chainbenchDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	sub := bus.Subscribe()

	args, _ := json.Marshal(map[string]any{"node_id": "node1"})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
	// Expect exactly one event: node.stopped.
	select {
	case ev := <-sub:
		if string(ev.Name) != "node.stopped" {
			t.Errorf("first event: got %q, want node.stopped", ev.Name)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no node.stopped event")
	}
	// No second event.
	select {
	case ev := <-sub:
		t.Errorf("unexpected 2nd event: %q", ev.Name)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestAllHandlers_IncludesNodeRestart(t *testing.T) {
	handlers := allHandlers("/s", "/c")
	if _, ok := handlers["node.restart"]; !ok {
		t.Error("allHandlers missing node.restart")
	}
}

// ---- node.tail_log tests ----

// setupCmdStubDirWithLog extends setupCmdStubDir by writing a log file for
// node1 whose path matches the one recorded in testdata/pids.json
// ("/tmp/node-data/logs/node1.log"). Because we cannot write to /tmp paths
// under the test harness reliably, we rewrite pids.json to point at a
// tempdir-local log file instead.
func setupCmdStubDirWithLog(t *testing.T, logBody string) (stateDir, chainbenchDir, logPath string) {
	t.Helper()
	stateDir, chainbenchDir = setupCmdStubDir(t)
	logPath = filepath.Join(stateDir, "node1.log")
	if err := os.WriteFile(logPath, []byte(logBody), 0o644); err != nil {
		t.Fatal(err)
	}
	// Rewrite pids.json in place so node "1"'s log_file points at logPath.
	pidsBytes, err := os.ReadFile(filepath.Join(stateDir, "pids.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(pidsBytes, &raw); err != nil {
		t.Fatal(err)
	}
	nodes := raw["nodes"].(map[string]any)
	n1 := nodes["1"].(map[string]any)
	n1["log_file"] = logPath
	nodes["1"] = n1
	raw["nodes"] = nodes
	patched, _ := json.Marshal(raw)
	if err := os.WriteFile(filepath.Join(stateDir, "pids.json"), patched, 0o644); err != nil {
		t.Fatal(err)
	}
	return stateDir, chainbenchDir, logPath
}

func TestHandleNodeTailLog_HappyPath(t *testing.T) {
	stateDir, _, logPath := setupCmdStubDirWithLog(t, "a\nb\nc\nd\ne\n")
	handler := newHandleNodeTailLog(stateDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"node_id": "node1", "lines": 3})
	data, err := handler(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["node_id"] != "node1" {
		t.Errorf("node_id: got %v", data["node_id"])
	}
	if data["log_file"] != logPath {
		t.Errorf("log_file: got %v", data["log_file"])
	}
	lines, ok := data["lines"].([]string)
	if !ok || len(lines) != 3 {
		t.Fatalf("lines: got %v", data["lines"])
	}
	want := []string{"c", "d", "e"}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d: got %q, want %q", i, lines[i], w)
		}
	}
}

func TestHandleNodeTailLog_DefaultLines(t *testing.T) {
	// Write 100 lines; default should return last 50.
	var b strings.Builder
	for i := 0; i < 100; i++ {
		b.WriteString("line")
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('\n')
	}
	stateDir, _, _ := setupCmdStubDirWithLog(t, b.String())
	handler := newHandleNodeTailLog(stateDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"node_id": "node1"}) // lines omitted
	data, err := handler(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	lines := data["lines"].([]string)
	if len(lines) != 50 {
		t.Errorf("default lines: got %d, want 50", len(lines))
	}
	if lines[0] != "line50" || lines[49] != "line99" {
		t.Errorf("default tail content: first=%q last=%q", lines[0], lines[49])
	}
}

func TestHandleNodeTailLog_MissingNodeID(t *testing.T) {
	stateDir, _, _ := setupCmdStubDirWithLog(t, "a\n")
	handler := newHandleNodeTailLog(stateDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	args, _ := json.Marshal(map[string]any{}) // node_id omitted
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTailLog_UnknownNodeID(t *testing.T) {
	stateDir, _, _ := setupCmdStubDirWithLog(t, "a\n")
	handler := newHandleNodeTailLog(stateDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	args, _ := json.Marshal(map[string]any{"node_id": "node99"})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTailLog_InvalidLines_Zero(t *testing.T) {
	stateDir, _, _ := setupCmdStubDirWithLog(t, "a\n")
	handler := newHandleNodeTailLog(stateDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	args, _ := json.Marshal(map[string]any{"node_id": "node1", "lines": 0})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTailLog_InvalidLines_OverMax(t *testing.T) {
	stateDir, _, _ := setupCmdStubDirWithLog(t, "a\n")
	handler := newHandleNodeTailLog(stateDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	args, _ := json.Marshal(map[string]any{"node_id": "node1", "lines": 1001})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTailLog_LogFileMissing(t *testing.T) {
	stateDir, _, logPath := setupCmdStubDirWithLog(t, "a\n")
	// Remove the log file so open fails.
	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	handler := newHandleNodeTailLog(stateDir)
	bus, _ := newTestBus(t)
	defer bus.Close()
	args, _ := json.Marshal(map[string]any{"node_id": "node1", "lines": 5})
	_, err := handler(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
}

func TestAllHandlers_IncludesNodeTailLog(t *testing.T) {
	handlers := allHandlers("/s", "/c")
	if _, ok := handlers["node.tail_log"]; !ok {
		t.Error("allHandlers missing node.tail_log")
	}
}

// ---- network.probe tests ----

// newStablenetMockRPC returns a mock JSON-RPC server that responds to the
// methods Detect() uses when classifying a stablenet endpoint:
//   - eth_chainId             -> "0x205b"  (8283)
//   - istanbul_getValidators  -> []
//   - everything else         -> -32601 method not found
func newStablenetMockRPC(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			ID     int    `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":"0x205b"}`, req.ID)
		case "istanbul_getValidators":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":[]}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"error":{"code":-32601,"message":"not found"}}`, req.ID)
		}
	}))
}

func TestHandleNetworkProbe_StablenetOK(t *testing.T) {
	srv := newStablenetMockRPC(t)
	defer srv.Close()

	h := newHandleNetworkProbe()
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"rpc_url": srv.URL})
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler err: %v", err)
	}
	if data["chain_type"] != "stablenet" {
		t.Errorf("chain_type = %v, want stablenet", data["chain_type"])
	}
	// JSON numbers decode as float64 via round-trip through map[string]any.
	cid, ok := data["chain_id"].(float64)
	if !ok || int64(cid) != 8283 {
		t.Errorf("chain_id = %v (%T), want 8283", data["chain_id"], data["chain_id"])
	}
	if data["rpc_url"] != srv.URL {
		t.Errorf("rpc_url = %v, want %q", data["rpc_url"], srv.URL)
	}
}

func TestHandleNetworkProbe_MissingURL(t *testing.T) {
	h := newHandleNetworkProbe()
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(json.RawMessage(`{}`), bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNetworkProbe_InvalidURLScheme(t *testing.T) {
	h := newHandleNetworkProbe()
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"rpc_url": "ws://example.com"})
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS (probe sentinel ErrInvalidURL), got %v", err)
	}
}

func TestHandleNetworkProbe_UnknownOverride(t *testing.T) {
	srv := newStablenetMockRPC(t)
	defer srv.Close()

	h := newHandleNetworkProbe()
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{
		"rpc_url":  srv.URL,
		"override": "fakechain",
	})
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS (probe sentinel ErrUnknownOverride), got %v", err)
	}
}

func TestHandleNetworkProbe_UpstreamFailure(t *testing.T) {
	// Server returns HTTP 500 on every request — Detect() produces a
	// non-sentinel error that the handler must classify as UPSTREAM_ERROR.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	h := newHandleNetworkProbe()
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{
		"rpc_url":    srv.URL,
		"timeout_ms": 500,
	})
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
}

func TestHandleNetworkProbe_TimeoutOutOfRange(t *testing.T) {
	h := newHandleNetworkProbe()
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{
		"rpc_url":    "http://127.0.0.1:1",
		"timeout_ms": 10, // < 100
	})
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNetworkProbe_MalformedArgs(t *testing.T) {
	h := newHandleNetworkProbe()
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(json.RawMessage(`{not json`), bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestAllHandlers_IncludesNetworkProbe(t *testing.T) {
	handlers := allHandlers("/s", "/c")
	if _, ok := handlers["network.probe"]; !ok {
		t.Error("allHandlers missing network.probe")
	}
}

// ---- network.attach tests ----

func TestHandleNetworkAttach_HappyPath(t *testing.T) {
	srv := newStablenetMockRPC(t)
	defer srv.Close()

	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	args, _ := json.Marshal(map[string]any{
		"rpc_url": srv.URL,
		"name":    "testnet",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["name"] != "testnet" {
		t.Errorf("name = %v", data["name"])
	}
	if data["chain_type"] != "stablenet" {
		t.Errorf("chain_type = %v", data["chain_type"])
	}
	if data["created"] != true {
		t.Errorf("created = %v, want true", data["created"])
	}
	if data["rpc_url"] != srv.URL {
		t.Errorf("rpc_url = %v, want %q", data["rpc_url"], srv.URL)
	}
	// File must exist.
	if _, err := os.Stat(filepath.Join(stateDir, "networks", "testnet.json")); err != nil {
		t.Errorf("state file missing: %v", err)
	}
}

func TestHandleNetworkAttach_SecondCallOverwrites(t *testing.T) {
	srv := newStablenetMockRPC(t)
	defer srv.Close()

	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	args, _ := json.Marshal(map[string]any{"rpc_url": srv.URL, "name": "testnet"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	if _, err := h(args, bus); err != nil {
		t.Fatal(err)
	}
	data, err := h(args, bus)
	if err != nil {
		t.Fatal(err)
	}
	if data["created"] != false {
		t.Errorf("second call created = %v, want false", data["created"])
	}
}

func TestHandleNetworkAttach_RejectsLocalName(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	args, _ := json.Marshal(map[string]any{"rpc_url": "http://x", "name": "local"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(args, bus)
	if err == nil {
		t.Fatal("expected INVALID_ARGS for reserved name 'local'")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("err = %v, want INVALID_ARGS", err)
	}
}

func TestHandleNetworkAttach_RejectsBadName(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	args, _ := json.Marshal(map[string]any{"rpc_url": "http://x", "name": "Has-Upper"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(args, bus)
	if err == nil {
		t.Fatal("expected INVALID_ARGS for bad name")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("err = %v, want INVALID_ARGS", err)
	}
}

func TestHandleNetworkAttach_MissingURL(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	args, _ := json.Marshal(map[string]any{"name": "testnet"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(args, bus)
	if err == nil {
		t.Fatal("expected INVALID_ARGS for missing rpc_url")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("err = %v, want INVALID_ARGS", err)
	}
}

func TestHandleNetworkAttach_MissingName(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	args, _ := json.Marshal(map[string]any{"rpc_url": "http://x"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(args, bus)
	if err == nil {
		t.Fatal("expected INVALID_ARGS for missing name")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("err = %v, want INVALID_ARGS", err)
	}
}

func TestHandleNetworkAttach_UpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	args, _ := json.Marshal(map[string]any{"rpc_url": srv.URL, "name": "testnet"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("err = %v, want UPSTREAM_ERROR", err)
	}
	// No state file written when probe fails.
	if _, statErr := os.Stat(filepath.Join(stateDir, "networks", "testnet.json")); !os.IsNotExist(statErr) {
		t.Errorf("state file should not exist after upstream failure: %v", statErr)
	}
}

func TestAllHandlers_IncludesNetworkAttach(t *testing.T) {
	handlers := allHandlers("/s", "/c")
	if _, ok := handlers["network.attach"]; !ok {
		t.Error("allHandlers missing network.attach")
	}
}

// ---- node.block_number tests ----

func TestHandleNodeBlockNumber_RemoteHappy(t *testing.T) {
	// Mock a JSON-RPC server that returns eth_blockNumber=0x10.
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_blockNumber" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"0x10"}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"nf"}}`))
	}))
	defer rpcSrv.Close()

	// Pre-populate a remote network state via SaveRemote.
	stateDir := t.TempDir()
	net := &types.Network{
		Name: "testnet", ChainType: "ethereum", ChainId: 1,
		Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: rpcSrv.URL}},
	}
	if err := state.SaveRemote(stateDir, net); err != nil {
		t.Fatal(err)
	}

	h := newHandleNodeBlockNumber(stateDir)
	args, _ := json.Marshal(map[string]any{"network": "testnet", "node_id": "node1"})
	bus, _ := newTestBus(t)
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["network"] != "testnet" {
		t.Errorf("network = %v", data["network"])
	}
	if data["node_id"] != "node1" {
		t.Errorf("node_id = %v", data["node_id"])
	}
	if bn, ok := data["block_number"].(uint64); !ok || bn != 16 {
		if f, fok := data["block_number"].(float64); fok && f == 16 {
			// JSON round-trip may produce float64; accept both.
		} else {
			t.Errorf("block_number = %v (type %T)", data["block_number"], data["block_number"])
		}
	}
}

func TestHandleNodeBlockNumber_MissingNodeID(t *testing.T) {
	h := newHandleNodeBlockNumber(t.TempDir())
	args, _ := json.Marshal(map[string]any{"network": "testnet"})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeBlockNumber_UnknownNetwork(t *testing.T) {
	h := newHandleNodeBlockNumber(t.TempDir())
	args, _ := json.Marshal(map[string]any{"network": "ghost", "node_id": "node1"})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
}

func TestHandleNodeBlockNumber_UnknownNode(t *testing.T) {
	stateDir := t.TempDir()
	net := &types.Network{
		Name: "tn", ChainType: "ethereum", ChainId: 1,
		Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: "http://127.0.0.1:1"}},
	}
	_ = state.SaveRemote(stateDir, net)

	h := newHandleNodeBlockNumber(stateDir)
	args, _ := json.Marshal(map[string]any{"network": "tn", "node_id": "node9"})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

// TestHandleNetworkAttach_WithAuth verifies attach accepts an "auth" arg,
// persists it onto Node.Auth, and the state file records only the env-var
// name — never the env-var value. Test sets MY_KEY=supersecret in the
// environment so an accidental value leak would be visible.
func TestHandleNetworkAttach_WithAuth(t *testing.T) {
	rpcSrv := newStablenetMockRPC(t)
	defer rpcSrv.Close()

	// Set the env var with a distinctive value so any accidental value leak
	// into the state file is detectable via substring search.
	t.Setenv("MY_KEY", "supersecret-value")

	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	args, _ := json.Marshal(map[string]any{
		"rpc_url": rpcSrv.URL,
		"name":    "protected",
		"auth": map[string]any{
			"type":   "api-key",
			"header": "X-Api-Key",
			"env":    "MY_KEY",
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()

	if _, err := h(args, bus); err != nil {
		t.Fatalf("attach: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(stateDir, "networks", "protected.json"))
	if err != nil {
		t.Fatalf("state file: %v", err)
	}
	s := string(raw)
	// Env NAME must be present (either indentation style).
	if !strings.Contains(s, `"env": "MY_KEY"`) && !strings.Contains(s, `"env":"MY_KEY"`) {
		t.Errorf("auth.env not persisted: %s", s)
	}
	// Env VALUE must NOT appear anywhere in the file.
	if strings.Contains(s, "supersecret-value") {
		t.Errorf("state file leaked env-var value: %s", s)
	}
}

// TestHandleNodeBlockNumber_UsesAuth pre-saves a remote network with an
// api-key auth block, sets the env var, and verifies the handler injects
// the configured header on the outbound eth_blockNumber call.
func TestHandleNodeBlockNumber_UsesAuth(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Api-Key")
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_blockNumber" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"0x7"}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"nf"}}`))
	}))
	defer srv.Close()

	stateDir := t.TempDir()
	net := &types.Network{
		Name: "protected", ChainType: "ethereum", ChainId: 1,
		Nodes: []types.Node{{
			Id:       "node1",
			Provider: "remote",
			Http:     srv.URL,
			Auth:     types.Auth{"type": "api-key", "header": "X-Api-Key", "env": "TEST_AUTH_KEY"},
		}},
	}
	if err := state.SaveRemote(stateDir, net); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_AUTH_KEY", "mysecret")

	h := newHandleNodeBlockNumber(stateDir)
	args, _ := json.Marshal(map[string]any{"network": "protected", "node_id": "node1"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if gotKey != "mysecret" {
		t.Errorf("server saw X-Api-Key=%q, want %q", gotKey, "mysecret")
	}
}

// ---- node.chain_id / node.balance / node.gas_price tests ----

// newReadMockRPC returns a mock JSON-RPC server that responds to the methods
// used by the three new read-only handlers: eth_chainId, eth_getBalance,
// eth_gasPrice. Non-matching methods return a -32601 "method not found"
// error so the mock composes cleanly with callers that interleave other RPCs.
func newReadMockRPC(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x2a"}`, req.ID)
		case "eth_getBalance":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x100"}`, req.ID)
		case "eth_gasPrice":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x3b9aca00"}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
}

// saveRemoteFixture pre-seeds a one-node remote network pointing at url.
func saveRemoteFixture(t *testing.T, stateDir, name, url string) {
	t.Helper()
	net := &types.Network{
		Name: name, ChainType: "ethereum", ChainId: 1,
		Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: url}},
	}
	if err := state.SaveRemote(stateDir, net); err != nil {
		t.Fatal(err)
	}
}

func TestHandleNodeChainID_Happy(t *testing.T) {
	srv := newReadMockRPC(t)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeChainID(stateDir)
	args, _ := json.Marshal(map[string]any{"network": "tn", "node_id": "node1"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["network"] != "tn" || data["node_id"] != "node1" {
		t.Errorf("echo fields: %+v", data)
	}
	if cid, ok := data["chain_id"].(uint64); !ok || cid != 42 {
		t.Errorf("chain_id = %v (%T), want 42 (uint64)", data["chain_id"], data["chain_id"])
	}
}

func TestHandleNodeBalance_Happy(t *testing.T) {
	srv := newReadMockRPC(t)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeBalance(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn", "node_id": "node1",
		"address": "0x0000000000000000000000000000000000000001",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["balance"] != "0x100" {
		t.Errorf("balance = %v, want 0x100", data["balance"])
	}
	if data["block"] != "latest" {
		t.Errorf("block = %v, want latest (default)", data["block"])
	}
}

func TestHandleNodeBalance_BadAddress(t *testing.T) {
	srv := newReadMockRPC(t)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeBalance(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn", "node_id": "node1",
		"address": "not-a-hex-address",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeBalance_MissingAddress(t *testing.T) {
	stateDir := t.TempDir()
	h := newHandleNodeBalance(stateDir)
	args, _ := json.Marshal(map[string]any{"node_id": "node1"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

// TestHandleNodeBalance_BlockNumberForms locks in the block_number parsing
// contract: integer, "latest", "earliest", "pending", unknown-label, negative,
// and wrong-shape inputs all produce the expected classification.
func TestHandleNodeBalance_BlockNumberForms(t *testing.T) {
	srv := newReadMockRPC(t)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	h := newHandleNodeBalance(stateDir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	base := map[string]any{
		"network": "tn",
		"node_id": "node1",
		"address": "0x0000000000000000000000000000000000000001",
	}

	cases := []struct {
		name      string
		blockNum  any
		wantCode  string // "" for success
		wantLabel string
	}{
		{"integer", float64(5), "", "5"},
		{"latest string", "latest", "", "latest"},
		{"earliest string", "earliest", "", "earliest"},
		{"pending string", "pending", "", "pending"},
		{"unknown string label", "top", "INVALID_ARGS", ""},
		{"negative integer", float64(-1), "INVALID_ARGS", ""},
		{"wrong shape (object)", map[string]any{"x": 1}, "INVALID_ARGS", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := maps.Clone(base)
			req["block_number"] = tc.blockNum
			args, _ := json.Marshal(req)
			data, err := h(args, bus)
			if tc.wantCode != "" {
				var api *APIError
				if !errors.As(err, &api) || string(api.Code) != tc.wantCode {
					t.Errorf("want %s, got %v", tc.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if data["block"] != tc.wantLabel {
				t.Errorf("block label = %q, want %q", data["block"], tc.wantLabel)
			}
		})
	}
}

// TestHandleNodeChainID_AuthSetupError exercises dialNode's auth-setup error
// branch — Auth references an unset env var, so AuthFromNode returns an error
// before any network round-trip happens. The handler must surface UPSTREAM_ERROR
// with the "auth setup" message fragment (so operators see which layer failed).
func TestHandleNodeChainID_AuthSetupError(t *testing.T) {
	srv := newReadMockRPC(t)
	defer srv.Close()
	stateDir := t.TempDir()
	net := &types.Network{
		Name:      "authfail",
		ChainType: "ethereum",
		ChainId:   1,
		Nodes: []types.Node{{
			Id:       "node1",
			Provider: "remote",
			Http:     srv.URL,
			Auth: types.Auth{
				"type": "api-key",
				"env":  "CHAINBENCH_TEST_UNSET_ENV_VAR_XYZ",
			},
		}},
	}
	if err := state.SaveRemote(stateDir, net); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("CHAINBENCH_TEST_UNSET_ENV_VAR_XYZ")

	h := newHandleNodeChainID(stateDir)
	args, _ := json.Marshal(map[string]any{"network": "authfail", "node_id": "node1"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Fatalf("want UPSTREAM_ERROR, got %v", err)
	}
	if !strings.Contains(api.Message, "auth setup") {
		t.Errorf("message should mention auth setup layer: %v", api.Message)
	}
}

func TestHandleNodeGasPrice_Happy(t *testing.T) {
	srv := newReadMockRPC(t)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeGasPrice(stateDir)
	args, _ := json.Marshal(map[string]any{"network": "tn", "node_id": "node1"})
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["gas_price"] != "0x3b9aca00" {
		t.Errorf("gas_price = %v, want 0x3b9aca00", data["gas_price"])
	}
}

// Attach must reject structurally-invalid auth with INVALID_ARGS before
// touching the state layer. Probe runs first (unauthenticated), then
// ValidateAuth gates persistence.
func TestHandleNetworkAttach_InvalidAuthRejectedAtAttach(t *testing.T) {
	rpcSrv := newStablenetMockRPC(t)
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	args, _ := json.Marshal(map[string]any{
		"rpc_url": rpcSrv.URL, "name": "bad",
		"auth": map[string]any{"type": "totally-unknown-type"},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()

	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
	// State file must NOT have been written — attach rejects before persistence.
	if _, serr := os.Stat(filepath.Join(stateDir, "networks", "bad.json")); !os.IsNotExist(serr) {
		t.Errorf("state file should not exist after invalid-auth attach: %v", serr)
	}
}

func TestAllHandlers_IncludesNewRemoteReadCommands(t *testing.T) {
	h := allHandlers("x", "y")
	for _, name := range []string{"node.chain_id", "node.balance", "node.gas_price"} {
		if _, ok := h[name]; !ok {
			t.Errorf("allHandlers missing %s", name)
		}
	}
}

// Local-only handler guard: node.stop with network:"remote" → NOT_SUPPORTED.
func TestHandleNodeStop_RejectsNonLocalNetwork(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	h := newHandleNodeStop(stateDir, chainbenchDir)
	args, _ := json.Marshal(map[string]any{"network": "remote-foo", "node_id": "node1"})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "NOT_SUPPORTED" {
		t.Errorf("want NOT_SUPPORTED, got %v", err)
	}
}

// TestLocalOnlyHandlers_RejectNonLocalNetwork covers the NOT_SUPPORTED guard
// on every local-only lifecycle handler. Prevents a future refactor from
// breaking one handler silently while the others stay green.
func TestLocalOnlyHandlers_RejectNonLocalNetwork(t *testing.T) {
	stateDir, chainbenchDir := setupCmdStubDir(t)
	bus, _ := newTestBus(t)

	cases := []struct {
		name    string
		handler Handler
	}{
		{"node.start", newHandleNodeStart(stateDir, chainbenchDir)},
		{"node.restart", newHandleNodeRestart(stateDir, chainbenchDir)},
		{"node.tail_log", newHandleNodeTailLog(stateDir)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]any{"network": "remote-foo", "node_id": "node1"})
			_, err := tc.handler(args, bus)
			var api *APIError
			if !errors.As(err, &api) || string(api.Code) != "NOT_SUPPORTED" {
				t.Errorf("want NOT_SUPPORTED, got %v", err)
			}
		})
	}
}

// ---- node.tx_send tests ----

// TestHandleNodeTxSend_Happy exercises the full flow with explicit nonce/gas
// to keep the test hermetic (no EstimateGas / PendingNonceAt round-trips).
// The mock only needs to answer eth_chainId + eth_sendRawTransaction.
func TestHandleNodeTxSend_Happy(t *testing.T) {
	var sawSendRaw bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
			Params []json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			sawSendRaw = true
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1111111111111111111111111111111111111111111111111111111111111111"}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	// Well-known synthetic key (NOT real funds) — test-only.
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	h := newHandleNodeTxSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":   "tn",
		"node_id":   "node1",
		"signer":    "alice",
		"to":        "0x0000000000000000000000000000000000000002",
		"value":     "0x0",
		"gas":       21000,
		"gas_price": "0x1",
		"nonce":     0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !sawSendRaw {
		t.Error("mock did not see eth_sendRawTransaction")
	}
	if tx, _ := data["tx_hash"].(string); !strings.HasPrefix(tx, "0x") || len(tx) != 66 {
		t.Errorf("tx_hash shape wrong: %v", data["tx_hash"])
	}
}

// TestHandleNodeTxSend_AutoFill exercises the auto-fill path where the caller
// omits nonce / gas / gas_price and the handler fetches each from the upstream.
// Covers resolveNonce (eth_getTransactionCount), resolveGasPrice (eth_gasPrice),
// and resolveGas (eth_estimateGas) composition — the integration Sprint 4b
// regression most likely to surface if EIP-1559 changes the CallMsg shape.
func TestHandleNodeTxSend_AutoFill(t *testing.T) {
	var sawSendRaw, sawNonce, sawGasPrice, sawEstimate bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_getTransactionCount":
			sawNonce = true
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x5"}`, req.ID)
		case "eth_gasPrice":
			sawGasPrice = true
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x3b9aca00"}`, req.ID)
		case "eth_estimateGas":
			sawEstimate = true
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x5208"}`, req.ID)
		case "eth_sendRawTransaction":
			sawSendRaw = true
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1111111111111111111111111111111111111111111111111111111111111111"}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	h := newHandleNodeTxSend(stateDir)
	// Omit nonce / gas / gas_price entirely — handler must auto-fill.
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
		"signer":  "alice",
		"to":      "0x0000000000000000000000000000000000000002",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !sawNonce {
		t.Error("mock did not see eth_getTransactionCount (nonce auto-fill missing)")
	}
	if !sawGasPrice {
		t.Error("mock did not see eth_gasPrice (gas_price auto-fill missing)")
	}
	if !sawEstimate {
		t.Error("mock did not see eth_estimateGas (gas auto-fill missing)")
	}
	if !sawSendRaw {
		t.Error("mock did not see eth_sendRawTransaction")
	}
	if tx, _ := data["tx_hash"].(string); !strings.HasPrefix(tx, "0x") || len(tx) != 66 {
		t.Errorf("tx_hash shape wrong: %v", data["tx_hash"])
	}
}

func TestHandleNodeTxSend_MissingSigner(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{"node_id": "node1", "to": "0x0000000000000000000000000000000000000002"})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_MissingTo(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{"node_id": "node1", "signer": "alice"})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_UnknownSigner(t *testing.T) {
	srv := newReadMockRPC(t)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	os.Unsetenv("CHAINBENCH_SIGNER_GHOST_KEY")

	h := newHandleNodeTxSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
		"signer":  "ghost",
		"to":      "0x0000000000000000000000000000000000000002",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_BadToAddress(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
		"signer":  "alice",
		"to":      "not-a-hex-address",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestAllHandlers_IncludesTxSend(t *testing.T) {
	h := allHandlers("x", "y")
	if _, ok := h["node.tx_send"]; !ok {
		t.Error("allHandlers missing node.tx_send")
	}
}

// decodeBroadcastTxType pulls the first byte of the rlp-prefixed raw hex from
// eth_sendRawTransaction params. Type-0 (legacy) does not have a leading
// type byte; types 1+ do. We surface an int (0 for legacy) for assertions.
func decodeBroadcastTxType(t *testing.T, rawHex string) int {
	t.Helper()
	raw := strings.TrimPrefix(rawHex, "0x")
	if len(raw) < 2 {
		t.Fatalf("raw too short: %q", rawHex)
	}
	var b [1]byte
	if _, err := hex.Decode(b[:], []byte(raw[:2])); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Legacy txs are RLP-encoded lists starting with 0xc0..0xff.
	if b[0] >= 0xc0 {
		return 0
	}
	return int(b[0])
}

func TestHandleNodeTxSend_DynamicFee_Happy(t *testing.T) {
	var sentRaw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
			Params []json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			if len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], &sentRaw)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1111111111111111111111111111111111111111111111111111111111111111"}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	h := newHandleNodeTxSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"value":                    "0x0",
		"gas":                      21000,
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"nonce":                    0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if sentRaw == "" {
		t.Fatal("mock did not see eth_sendRawTransaction")
	}
	if got := decodeBroadcastTxType(t, sentRaw); got != 2 {
		t.Errorf("tx type = %d, want 2 (DynamicFee)", got)
	}
}

func TestHandleNodeTxSend_MixedFeeFields(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":         "tn",
		"node_id":         "node1",
		"signer":          "alice",
		"to":              "0x0000000000000000000000000000000000000002",
		"gas_price":       "0x1",
		"max_fee_per_gas": "0x2",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_PartialDynamicFee(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":         "tn",
		"node_id":         "node1",
		"signer":          "alice",
		"to":              "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas": "0x2",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxWait_SuccessImmediate(t *testing.T) {
	txHash := "0x" + strings.Repeat("a", 64)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getTransactionReceipt" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":%q,
                "blockHash":"0x%s",
                "blockNumber":"0x10",
                "cumulativeGasUsed":"0x5208",
                "gasUsed":"0x5208",
                "effectiveGasPrice":"0x3b9aca00",
                "status":"0x1",
                "contractAddress":null,
                "logsBloom":"0x%s",
                "logs":[]}}`, req.ID, txHash, strings.Repeat("b", 64), strings.Repeat("0", 512))
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	h := newHandleNodeTxWait(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
		"tx_hash": txHash,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if got, _ := data["status"].(string); got != "success" {
		t.Errorf("status = %v, want success", data["status"])
	}
	if got, _ := data["block_number"].(uint64); got != 16 {
		t.Errorf("block_number = %v, want 16", data["block_number"])
	}
}

func TestHandleNodeTxWait_FailedReceipt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getTransactionReceipt" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":"0x%s",
                "blockHash":"0x%s",
                "blockNumber":"0x10","cumulativeGasUsed":"0x5208","gasUsed":"0x5208","effectiveGasPrice":"0x1",
                "status":"0x0","contractAddress":null,"logsBloom":"0x%s","logs":[]}}`, req.ID, strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("0", 512))
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	h := newHandleNodeTxWait(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn", "node_id": "node1", "tx_hash": "0x" + strings.Repeat("a", 64),
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if got, _ := data["status"].(string); got != "failed" {
		t.Errorf("status = %v, want failed", data["status"])
	}
}

func TestHandleNodeTxWait_NotFoundThenSuccess(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getTransactionReceipt" {
			calls++
			if calls == 1 {
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":null}`, req.ID)
				return
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":"0x%s","blockHash":"0x%s","blockNumber":"0x1",
                "cumulativeGasUsed":"0x5208","gasUsed":"0x5208","status":"0x1","contractAddress":null,"logsBloom":"0x%s","logs":[]}}`,
				req.ID, strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("0", 512))
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	h := newHandleNodeTxWait(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn", "node_id": "node1", "tx_hash": "0x" + strings.Repeat("a", 64),
		"timeout_ms": 5000,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if got, _ := data["status"].(string); got != "success" {
		t.Errorf("status = %v, want success", data["status"])
	}
	if calls < 2 {
		t.Errorf("expected at least 2 polls, got %d", calls)
	}
}

func TestHandleNodeTxWait_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getTransactionReceipt" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":null}`, req.ID)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	h := newHandleNodeTxWait(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn", "node_id": "node1", "tx_hash": "0x" + strings.Repeat("a", 64),
		"timeout_ms": 1500,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if got, _ := data["status"].(string); got != "pending" {
		t.Errorf("status = %v, want pending", data["status"])
	}
}

func TestHandleNodeTxWait_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "eth_getTransactionReceipt" {
			http.Error(w, "boom", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_chainId" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	h := newHandleNodeTxWait(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn", "node_id": "node1", "tx_hash": "0x" + strings.Repeat("a", 64),
		"timeout_ms": 1500,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err == nil {
		t.Fatalf("expected error, got nil (data=%v)", data)
	}
	if data != nil {
		t.Errorf("data = %v, want nil on error", data)
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
}

func TestHandleNodeTxWait_BadHash(t *testing.T) {
	h := newHandleNodeTxWait(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network": "tn", "node_id": "node1", "tx_hash": "not-a-hash",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxWait_TimeoutOutOfRange(t *testing.T) {
	h := newHandleNodeTxWait(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network": "tn", "node_id": "node1", "tx_hash": "0x" + strings.Repeat("a", 64),
		"timeout_ms": 50,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestAllHandlers_IncludesTxWait(t *testing.T) {
	h := allHandlers("x", "y")
	if _, ok := h["node.tx_wait"]; !ok {
		t.Error("allHandlers missing node.tx_wait")
	}
}

// ---- node.tx_send authorization_list (EIP-7702) tests ----

// keyHexA / keyHexB are deterministic synthetic keys for SetCode tests.
// keyHexA is the existing sender key reused across Sprint 4/4b tests; keyHexB
// is a distinct key so sender + authorizer have different recovered addresses.
// Neither is associated with real funds.
const keyHexA = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
const keyHexB = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
const keyHexC = "5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cde8b0c1"

// newSetCodeMock mirrors newDynamicFee mock shape: replies eth_chainId 0x1,
// captures raw param[0] of eth_sendRawTransaction into *sentRaw, returns a
// canned tx hash. Other methods return -32601 not-found.
func newSetCodeMock(t *testing.T, sentRaw *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
			Params []json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			if sentRaw != nil && len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], sentRaw)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1111111111111111111111111111111111111111111111111111111111111111"}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
}

func TestHandleNodeTxSend_SetCode_Happy_SingleAuth(t *testing.T) {
	var sentRaw string
	srv := newSetCodeMock(t, &sentRaw)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)
	t.Setenv("CHAINBENCH_SIGNER_BOB_KEY", "0x"+keyHexB)

	h := newHandleNodeTxSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"value":                    "0x0",
		"gas":                      100000,
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "nonce": "0x0", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if sentRaw == "" {
		t.Fatal("mock did not see eth_sendRawTransaction")
	}
	if got := decodeBroadcastTxType(t, sentRaw); got != 4 {
		t.Errorf("tx type = %d, want 4 (SetCode)", got)
	}
}

func TestHandleNodeTxSend_SetCode_Happy_MultiAuth(t *testing.T) {
	var sentRaw string
	srv := newSetCodeMock(t, &sentRaw)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)
	t.Setenv("CHAINBENCH_SIGNER_BOB_KEY", "0x"+keyHexB)
	t.Setenv("CHAINBENCH_SIGNER_CAROL_KEY", "0x"+keyHexC)

	h := newHandleNodeTxSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"value":                    "0x0",
		"gas":                      150000,
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "nonce": "0x0", "signer": "bob"},
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000cafe", "nonce": "0x1", "signer": "carol"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if got := decodeBroadcastTxType(t, sentRaw); got != 4 {
		t.Errorf("tx type = %d, want 4 (SetCode)", got)
	}
	// Decode the broadcast bytes and verify both authorizations carry non-zero
	// V/R/S signatures (signed by distinct authorizers).
	rawBytes, err := hex.DecodeString(strings.TrimPrefix(sentRaw, "0x"))
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	var tx ethtypes.Transaction
	if err := tx.UnmarshalBinary(rawBytes); err != nil {
		t.Fatalf("unmarshal binary: %v", err)
	}
	auths := tx.SetCodeAuthorizations()
	if len(auths) != 2 {
		t.Fatalf("auth count = %d, want 2", len(auths))
	}
	for i, a := range auths {
		if a.R.IsZero() || a.S.IsZero() {
			t.Errorf("auth[%d] signature R/S is zero", i)
		}
	}
}

func TestHandleNodeTxSend_AuthorizationListWithLegacy(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":   "tn",
		"node_id":   "node1",
		"signer":    "alice",
		"to":        "0x0000000000000000000000000000000000000002",
		"gas_price": "0x1",
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "nonce": "0x0", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_AuthorizationListWithoutTip(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":         "tn",
		"node_id":         "node1",
		"signer":          "alice",
		"to":              "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas": "0x59682f00",
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "nonce": "0x0", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_AuthorizationListEmptyIsDynamicFee(t *testing.T) {
	var sentRaw string
	srv := newSetCodeMock(t, &sentRaw)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)

	h := newHandleNodeTxSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"value":                    "0x0",
		"gas":                      21000,
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"nonce":                    0,
		"authorization_list":       []map[string]any{},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if got := decodeBroadcastTxType(t, sentRaw); got != 2 {
		t.Errorf("tx type = %d, want 2 (DynamicFee, not SetCode)", got)
	}
}

func TestHandleNodeTxSend_SetCode_AuthSignerUnknown(t *testing.T) {
	srv := newSetCodeMock(t, nil)
	defer srv.Close()
	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)
	os.Unsetenv("CHAINBENCH_SIGNER_GHOST_KEY")
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	h := newHandleNodeTxSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "nonce": "0x0", "signer": "ghost"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_SetCode_BadAuthAddress(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "not-a-hex", "nonce": "0x0", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_SetCode_BadAuthChainID(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"chain_id": "zzz", "address": "0x000000000000000000000000000000000000beef", "nonce": "0x0", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_SetCode_BadAuthNonce(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "nonce": "qqq", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_SetCode_MissingAuthSigner(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "nonce": "0x0"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_SetCode_MissingAuthAddress(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "nonce": "0x0", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_SetCode_MissingAuthChainID(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"address": "0x000000000000000000000000000000000000beef", "nonce": "0x0", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxSend_SetCode_MissingAuthNonce(t *testing.T) {
	h := newHandleNodeTxSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

// keyHexFeePayer is a test-only synthetic ECDSA key used as the fee_payer for
// the go-stablenet 0x16 fee-delegation tests. Distinct from keyHexA / keyHexB /
// keyHexC so cross-test contamination of signer state is unmistakable.
const keyHexFeePayer = "8da4ef21b864d2cc526dbdb2a120bd2874c36c9d0a1fb7f8c63d7f7a8b41de8f"

// saveRemoteFixtureWithChainType pre-seeds a one-node remote network and lets
// the caller choose the chain_type. Wraps state.SaveRemote so the chain-type
// allowlist guard in node.tx_fee_delegation_send can be exercised without
// touching the existing saveRemoteFixture (which hard-codes "ethereum").
func saveRemoteFixtureWithChainType(t *testing.T, stateDir, name, url, chainType string) {
	t.Helper()
	net := &types.Network{
		Name: name, ChainType: types.NetworkChainType(chainType), ChainId: 1,
		Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: url}},
	}
	if err := state.SaveRemote(stateDir, net); err != nil {
		t.Fatal(err)
	}
}

// newFeeDelegateMock mirrors newSetCodeMock: replies eth_chainId 0x1, captures
// the raw param[0] of eth_sendRawTransaction into *sentRaw, returns a canned
// tx hash. Other methods return -32601 not-found.
func newFeeDelegateMock(t *testing.T, sentRaw *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
			Params []json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			if sentRaw != nil && len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], sentRaw)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1111111111111111111111111111111111111111111111111111111111111111"}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
}

func TestHandleNodeTxFeeDelegationSend_Happy_Stablenet(t *testing.T) {
	var sentRaw string
	srv := newFeeDelegateMock(t, &sentRaw)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixtureWithChainType(t, stateDir, "stab", srv.URL, "stablenet")
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)
	t.Setenv("CHAINBENCH_SIGNER_FPAYER_KEY", "0x"+keyHexFeePayer)

	h := newHandleNodeTxFeeDelegationSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "stab",
		"node_id":                  "node1",
		"signer":                   "alice",
		"fee_payer":                "fpayer",
		"to":                       "0x0000000000000000000000000000000000000002",
		"value":                    "0x0",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    7,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()

	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !strings.HasPrefix(sentRaw, "0x16") {
		t.Errorf("raw tx not 0x16-prefixed: %q", sentRaw)
	}
	tx, _ := data["tx_hash"].(string)
	if !strings.HasPrefix(tx, "0x") || len(tx) != 66 {
		t.Errorf("tx_hash shape wrong: %v", data["tx_hash"])
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(tx, "0x")); err != nil {
		t.Errorf("tx_hash is not valid hex: %v", err)
	}
}

func TestHandleNodeTxFeeDelegationSend_ChainTypeNotSupported_Ethereum(t *testing.T) {
	srv := newFeeDelegateMock(t, nil)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixtureWithChainType(t, stateDir, "eth", srv.URL, "ethereum")
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)
	t.Setenv("CHAINBENCH_SIGNER_FPAYER_KEY", "0x"+keyHexFeePayer)

	h := newHandleNodeTxFeeDelegationSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "eth",
		"node_id":                  "node1",
		"signer":                   "alice",
		"fee_payer":                "fpayer",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "NOT_SUPPORTED" {
		t.Errorf("want NOT_SUPPORTED, got %v", err)
	}
}

func TestHandleNodeTxFeeDelegationSend_ChainTypeNotSupported_Wemix(t *testing.T) {
	srv := newFeeDelegateMock(t, nil)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixtureWithChainType(t, stateDir, "wmx", srv.URL, "wemix")
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)
	t.Setenv("CHAINBENCH_SIGNER_FPAYER_KEY", "0x"+keyHexFeePayer)

	h := newHandleNodeTxFeeDelegationSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "wmx",
		"node_id":                  "node1",
		"signer":                   "alice",
		"fee_payer":                "fpayer",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "NOT_SUPPORTED" {
		t.Errorf("want NOT_SUPPORTED, got %v", err)
	}
}

func TestHandleNodeTxFeeDelegationSend_MissingFeePayer(t *testing.T) {
	h := newHandleNodeTxFeeDelegationSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "stab",
		"node_id":                  "node1",
		"signer":                   "alice",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxFeeDelegationSend_FeePayerAliasUnknown(t *testing.T) {
	srv := newFeeDelegateMock(t, nil)
	defer srv.Close()
	stateDir := t.TempDir()
	saveRemoteFixtureWithChainType(t, stateDir, "stab", srv.URL, "stablenet")
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)
	// Intentionally do NOT set CHAINBENCH_SIGNER_FPAYER_KEY — alias is unknown.

	h := newHandleNodeTxFeeDelegationSend(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "stab",
		"node_id":                  "node1",
		"signer":                   "alice",
		"fee_payer":                "fpayer",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxFeeDelegationSend_BadToAddress(t *testing.T) {
	h := newHandleNodeTxFeeDelegationSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "stab",
		"node_id":                  "node1",
		"signer":                   "alice",
		"fee_payer":                "fpayer",
		"to":                       "not-an-address",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
		"nonce":                    0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

// Spec §9.3 BadValueHex: malformed value hex must surface as INVALID_ARGS at
// the boundary, before any signer.Load or RPC round-trip. The handler parses
// `value` after the required-field block but before resolveNode, so this test
// mirrors _BadToAddress (no mock, no fixture, no env-keyed signers required).
func TestHandleNodeTxFeeDelegationSend_BadValueHex(t *testing.T) {
	h := newHandleNodeTxFeeDelegationSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "stab",
		"node_id":                  "node1",
		"signer":                   "alice",
		"fee_payer":                "fpayer",
		"to":                       "0x0000000000000000000000000000000000000002",
		"value":                    "not-hex",
		"max_fee_per_gas":          "0x1",
		"max_priority_fee_per_gas": "0x1",
		"gas":                      21000,
		"nonce":                    0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxFeeDelegationSend_MissingMaxFeeFields(t *testing.T) {
	h := newHandleNodeTxFeeDelegationSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":   "stab",
		"node_id":   "node1",
		"signer":    "alice",
		"fee_payer": "fpayer",
		"to":        "0x0000000000000000000000000000000000000002",
		"gas":       21000,
		"nonce":     0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxFeeDelegationSend_MissingNonce(t *testing.T) {
	h := newHandleNodeTxFeeDelegationSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "stab",
		"node_id":                  "node1",
		"signer":                   "alice",
		"fee_payer":                "fpayer",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"gas":                      21000,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeTxFeeDelegationSend_MissingGas(t *testing.T) {
	h := newHandleNodeTxFeeDelegationSend(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":                  "stab",
		"node_id":                  "node1",
		"signer":                   "alice",
		"fee_payer":                "fpayer",
		"to":                       "0x0000000000000000000000000000000000000002",
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"nonce":                    0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestAllHandlers_IncludesTxFeeDelegationSend(t *testing.T) {
	h := allHandlers("x", "y")
	if _, ok := h["node.tx_fee_delegation_send"]; !ok {
		t.Error("allHandlers missing node.tx_fee_delegation_send")
	}
}

// ---- node.contract_deploy tests ----

// erc20DeployABI is a minimal constructor-only fixture for contract_deploy
// tests. The single uint256 input mirrors the canonical "ERC-20 with initial
// supply" pattern so the encoded args are 32 bytes of big-endian supply value.
const erc20DeployABI = `[{"type":"constructor","inputs":[{"name":"supply","type":"uint256"}],"stateMutability":"nonpayable"}]`

// erc20TwoArgABI is a fixture whose constructor declares two inputs. Used to
// exercise the arg-count mismatch path — the handler must reject before any
// signer load when len(args) != len(inputs).
const erc20TwoArgABI = `[{"type":"constructor","inputs":[{"name":"a","type":"uint256"},{"name":"b","type":"uint256"}],"stateMutability":"nonpayable"}]`

// minimalRuntimeBytecode is a tiny but well-formed deploy bytecode:
//
//	0x6080604052348015600f57600080fd5b50603f80601d6000396000f3fe6080604052600080fdfea2646970667358...
//
// Truncated for the test — the handler treats it as an opaque blob; only the
// "starts with 0x6080" prefix is conventional. We need enough bytes that the
// resulting tx data field is unambiguously non-empty.
const minimalRuntimeBytecode = "0x6080604052348015600f57600080fd5b5060358061001e6000396000f3fe6080604052600080fdfea2646970667358221220abcdef00000000000000000000000000000000000000000000000000000000000064736f6c63430008190033"

// newContractDeployMock mirrors the SetCode mock pattern: replies eth_chainId
// 0x1, captures raw param[0] of eth_sendRawTransaction into *sentRaw, returns
// a canned tx hash. Other methods return -32601 not-found so the test fails
// loudly on any unexpected RPC.
func newContractDeployMock(t *testing.T, sentRaw *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
			Params []json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			if sentRaw != nil && len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], sentRaw)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0",`+
				`"id":%s,`+
				`"result":"0x1111111111111111111111111111111111111111111111111111111111111111"}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
}

// aliceAddrFromKey derives the address belonging to keyHexA so tests can
// independently compute the expected contract_address (= CreateAddress(addr,
// nonce)) without round-tripping through the signer package.
func aliceAddrFromKey(t *testing.T) common.Address {
	t.Helper()
	priv, err := crypto.HexToECDSA(keyHexA)
	if err != nil {
		t.Fatalf("HexToECDSA: %v", err)
	}
	return crypto.PubkeyToAddress(priv.PublicKey)
}

func TestHandleNodeContractDeploy_Happy_Bytecode(t *testing.T) {
	var sentRaw string
	srv := newContractDeployMock(t, &sentRaw)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)

	h := newHandleNodeContractDeploy(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"bytecode":                 minimalRuntimeBytecode,
		"value":                    "0x0",
		"gas":                      3000000,
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"nonce":                    7,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	txHash, _ := data["tx_hash"].(string)
	if !strings.HasPrefix(txHash, "0x") || len(txHash) != 66 {
		t.Errorf("tx_hash shape wrong: %v", data["tx_hash"])
	}
	gotAddr, _ := data["contract_address"].(string)
	expected := crypto.CreateAddress(aliceAddrFromKey(t), 7).Hex()
	if gotAddr != expected {
		t.Errorf("contract_address = %q, want %q", gotAddr, expected)
	}
	// Decode the broadcast tx and verify To==nil + data == bytecode.
	rawBytes, derr := hex.DecodeString(strings.TrimPrefix(sentRaw, "0x"))
	if derr != nil {
		t.Fatalf("decode raw: %v", derr)
	}
	var tx ethtypes.Transaction
	if err := tx.UnmarshalBinary(rawBytes); err != nil {
		t.Fatalf("unmarshal tx: %v", err)
	}
	if tx.To() != nil {
		t.Errorf("tx.To = %v, want nil (contract creation)", tx.To())
	}
	wantBytes, _ := hex.DecodeString(strings.TrimPrefix(minimalRuntimeBytecode, "0x"))
	if !bytes.Equal(tx.Data(), wantBytes) {
		t.Errorf("tx.Data mismatch: got %x want %x", tx.Data(), wantBytes)
	}
}

func TestHandleNodeContractDeploy_Happy_WithABI(t *testing.T) {
	var sentRaw string
	srv := newContractDeployMock(t, &sentRaw)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)

	h := newHandleNodeContractDeploy(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":                  "tn",
		"node_id":                  "node1",
		"signer":                   "alice",
		"bytecode":                 minimalRuntimeBytecode,
		"abi":                      erc20DeployABI,
		"constructor_args":         []any{"1000000000000000000"},
		"gas":                      3000000,
		"max_fee_per_gas":          "0x59682f00",
		"max_priority_fee_per_gas": "0x3b9aca00",
		"nonce":                    0,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}

	// Compute the expected encoded args independently via abiutil so the
	// assertion encodes the same shape the handler does. tx.Data() must equal
	// bytecode || encoded(uint256(1e18)).
	parsed, perr := abiutil.ParseABI(erc20DeployABI)
	if perr != nil {
		t.Fatalf("parse abi: %v", perr)
	}
	encoded, eerr := abiutil.PackConstructor(parsed, []any{"1000000000000000000"})
	if eerr != nil {
		t.Fatalf("pack ctor: %v", eerr)
	}
	bytecodeBytes, _ := hex.DecodeString(strings.TrimPrefix(minimalRuntimeBytecode, "0x"))
	want := append(append([]byte{}, bytecodeBytes...), encoded...)

	rawBytes, derr := hex.DecodeString(strings.TrimPrefix(sentRaw, "0x"))
	if derr != nil {
		t.Fatalf("decode raw: %v", derr)
	}
	var tx ethtypes.Transaction
	if err := tx.UnmarshalBinary(rawBytes); err != nil {
		t.Fatalf("unmarshal tx: %v", err)
	}
	if tx.To() != nil {
		t.Errorf("tx.To = %v, want nil", tx.To())
	}
	if !bytes.Equal(tx.Data(), want) {
		t.Errorf("tx.Data mismatch:\n got %x\nwant %x", tx.Data(), want)
	}
	// Trailing 32 bytes must equal big-endian 1e18 — explicit guard against
	// an off-by-N concat mistake.
	if len(tx.Data()) < 32 {
		t.Fatalf("tx.Data too short: %d bytes", len(tx.Data()))
	}
	tail := tx.Data()[len(tx.Data())-32:]
	wantInt := big.NewInt(0)
	wantInt.SetString("1000000000000000000", 10)
	gotInt := new(big.Int).SetBytes(tail)
	if gotInt.Cmp(wantInt) != 0 {
		t.Errorf("trailing 32 bytes decode = %s, want %s", gotInt.String(), wantInt.String())
	}
}

func TestHandleNodeContractDeploy_MissingBytecode(t *testing.T) {
	h := newHandleNodeContractDeploy(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
		"signer":  "alice",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractDeploy_BadBytecode(t *testing.T) {
	h := newHandleNodeContractDeploy(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":  "tn",
		"node_id":  "node1",
		"signer":   "alice",
		"bytecode": "not-hex",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractDeploy_BadABI(t *testing.T) {
	h := newHandleNodeContractDeploy(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":  "tn",
		"node_id":  "node1",
		"signer":   "alice",
		"bytecode": minimalRuntimeBytecode,
		"abi":      "not-json",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractDeploy_ABIArgsMismatch(t *testing.T) {
	h := newHandleNodeContractDeploy(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"signer":           "alice",
		"bytecode":         minimalRuntimeBytecode,
		"abi":              erc20TwoArgABI,
		"constructor_args": []any{"1"},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

// TestHandleNodeContractDeploy_AuthorizationListRejected verifies that callers
// who try to combine a contract-creation tx with EIP-7702 authorizations are
// rejected at the boundary BEFORE any signer load or RPC. SetCode pairs with a
// `to` address (the authority being upgraded) — contract creation has none, so
// the combination is meaningless and must surface as INVALID_ARGS.
func TestHandleNodeContractDeploy_AuthorizationListRejected(t *testing.T) {
	h := newHandleNodeContractDeploy(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":  "tn",
		"node_id":  "node1",
		"signer":   "alice",
		"bytecode": minimalRuntimeBytecode,
		"authorization_list": []map[string]any{
			{"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "nonce": "0x0", "signer": "bob"},
		},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractDeploy_FeeModeMixed(t *testing.T) {
	h := newHandleNodeContractDeploy(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":         "tn",
		"node_id":         "node1",
		"signer":          "alice",
		"bytecode":        minimalRuntimeBytecode,
		"gas_price":       "0x1",
		"max_fee_per_gas": "0x2",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestAllHandlers_IncludesContractDeploy(t *testing.T) {
	h := allHandlers("x", "y")
	if _, ok := h["node.contract_deploy"]; !ok {
		t.Error("allHandlers missing node.contract_deploy")
	}
}

// erc20BalanceOfABI is a minimal ABI fixture exposing balanceOf(address)
// returns (uint256). Used for contract_call ABI-mode tests.
const erc20BalanceOfABI = `[{"type":"function","name":"balanceOf","stateMutability":"view","inputs":[{"name":"owner","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}]`

// newContractCallMock returns a mock JSON-RPC server that responds to eth_call
// with a 32-byte big-endian uint256 (value = 42 by default). It captures the
// raw `params[0]` (CallMsg as encoded by ethclient) into *capturedCallObj and
// the raw `params[1]` (block tag/number) into *capturedBlock so tests can
// assert on the wire shape. Other methods return -32601 not-found.
func newContractCallMock(t *testing.T, capturedCallObj, capturedBlock *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
			Params []json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_call":
			if len(req.Params) > 0 && capturedCallObj != nil {
				*capturedCallObj = string(req.Params[0])
			}
			if len(req.Params) > 1 && capturedBlock != nil {
				*capturedBlock = string(req.Params[1])
			}
			// 32-byte big-endian uint256 = 42.
			result := "0x" + strings.Repeat("0", 62) + "2a"
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, result)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
}

func TestHandleNodeContractCall_Happy_Calldata(t *testing.T) {
	srv := newContractCallMock(t, nil, nil)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeContractCall(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
		// balanceOf(0xabcd...) selector + arg.
		"calldata": "0x70a08231000000000000000000000000abcdabcdabcdabcdabcdabcdabcdabcdabcdabcd",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	wantRaw := "0x" + strings.Repeat("0", 62) + "2a"
	if got, _ := data["result_raw"].(string); got != wantRaw {
		t.Errorf("result_raw = %q, want %q", got, wantRaw)
	}
	if _, present := data["result_decoded"]; present {
		t.Errorf("result_decoded must be absent in calldata mode, got %v", data["result_decoded"])
	}
}

func TestHandleNodeContractCall_Happy_ABI(t *testing.T) {
	srv := newContractCallMock(t, nil, nil)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeContractCall(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
		"abi":              erc20BalanceOfABI,
		"method":           "balanceOf",
		"args":             []any{"0xabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd"},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	wantRaw := "0x" + strings.Repeat("0", 62) + "2a"
	if got, _ := data["result_raw"].(string); got != wantRaw {
		t.Errorf("result_raw = %q, want %q", got, wantRaw)
	}
	decoded, ok := data["result_decoded"].([]any)
	if !ok {
		t.Fatalf("result_decoded missing or wrong type: %T %v", data["result_decoded"], data["result_decoded"])
	}
	if len(decoded) != 1 {
		t.Fatalf("result_decoded length = %d, want 1", len(decoded))
	}
	gotInt, ok := decoded[0].(*big.Int)
	if !ok {
		t.Fatalf("result_decoded[0] type = %T, want *big.Int", decoded[0])
	}
	if gotInt.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("result_decoded[0] = %s, want 42", gotInt.String())
	}
}

func TestHandleNodeContractCall_BothCalldataAndABI(t *testing.T) {
	h := newHandleNodeContractCall(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
		"calldata":         "0x1234",
		"abi":              erc20BalanceOfABI,
		"method":           "balanceOf",
		"args":             []any{"0xabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd"},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractCall_NeitherCalldataNorABI(t *testing.T) {
	h := newHandleNodeContractCall(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractCall_BadAddress(t *testing.T) {
	h := newHandleNodeContractCall(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "not-hex",
		"calldata":         "0x1234",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractCall_BadCalldataHex(t *testing.T) {
	h := newHandleNodeContractCall(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
		"calldata":         "not-hex",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractCall_BadABI(t *testing.T) {
	h := newHandleNodeContractCall(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
		"abi":              "not-json",
		"method":           "balanceOf",
		"args":             []any{"0xabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd"},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractCall_UnknownMethod(t *testing.T) {
	h := newHandleNodeContractCall(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
		"abi":              erc20BalanceOfABI,
		"method":           "nonExistent",
		"args":             []any{"0xabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd"},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeContractCall_BlockNumberLatest(t *testing.T) {
	var capturedBlock string
	srv := newContractCallMock(t, nil, &capturedBlock)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeContractCall(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
		"calldata":         "0x1234",
		"block_number":     "latest",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !strings.Contains(capturedBlock, "latest") {
		t.Errorf("eth_call block param = %q, want %q", capturedBlock, "latest")
	}
}

func TestHandleNodeContractCall_BlockEcho(t *testing.T) {
	var capturedBlock string
	srv := newContractCallMock(t, nil, &capturedBlock)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeContractCall(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
		"calldata":         "0x1234",
		"block_number":     "pending",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	// "pending" is internally degraded to "latest" for the upstream eth_call,
	// but the result must echo the caller-provided label so the substitution
	// is observable.
	if got := data["block"]; got != "pending" {
		t.Errorf("data[\"block\"] = %v, want %q", got, "pending")
	}
}

func TestHandleNodeContractCall_BlockNumberInteger(t *testing.T) {
	var capturedBlock string
	srv := newContractCallMock(t, nil, &capturedBlock)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeContractCall(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":          "tn",
		"node_id":          "node1",
		"contract_address": "0x000000000000000000000000000000000000beef",
		"calldata":         "0x1234",
		"block_number":     100,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	// ethclient encodes integer block numbers as hex (0x64 = 100).
	if !strings.Contains(capturedBlock, "0x64") {
		t.Errorf("eth_call block param = %q, want it to contain 0x64", capturedBlock)
	}
}

func TestAllHandlers_IncludesContractCall(t *testing.T) {
	h := allHandlers("x", "y")
	if _, ok := h["node.contract_call"]; !ok {
		t.Error("allHandlers missing node.contract_call")
	}
}

// erc20TransferEventABI exposes the canonical ERC20 Transfer event with both
// indexed address args and a non-indexed uint256 value. Used by events_get
// decode-mode tests so we exercise the topics+data split inside DecodeLog.
const erc20TransferEventABI = `[{"type":"event","name":"Transfer","anonymous":false,"inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}]}]`

// newEventsGetMock returns a JSON-RPC server that responds to eth_getLogs by
// echoing the supplied canned logs. When capturedQuery is non-nil it parses
// params[0] (the filter object as encoded by ethclient.toFilterArg) back into
// an ethereum.FilterQuery so tests can assert on the structural shape the
// handler built. Other methods get -32601.
//
// The wire shape ethclient produces is:
//
//	{"address":[<addr>...],"topics":[[h]|null|[h1,h2]...],
//	 "fromBlock":"<hex>"|"latest"|"earliest"|"pending"|"0x0",
//	 "toBlock":"<hex>"|"latest"|"earliest"|"pending"}
//
// Per ethclient.toFilterArg (v1.17.2), address/topics are omitted when nil;
// fromBlock defaults to "0x0" when q.FromBlock == nil; toBlock defaults to
// "latest" when q.ToBlock == nil. Note that "0x0" is the wire encoding of
// BOTH nil-FromBlock and big.NewInt(0)-FromBlock — they're indistinguishable
// downstream of ethclient. Tests that need to assert on the nil-vs-zero
// distinction must inspect the captured raw payload via newEventsGetMockRaw.
func newEventsGetMock(t *testing.T, capturedQuery *ethereum.FilterQuery, logs []ethtypes.Log) *httptest.Server {
	t.Helper()
	return newEventsGetMockRaw(t, capturedQuery, nil, logs)
}

// newEventsGetMockRaw is the underlying mock factory exposing the raw
// params[0] string so tests can assert on the exact wire encoding (e.g.
// distinguishing "0x64" from "100" or confirming "0x0" defaulting). Either
// or both capture pointers may be nil.
func newEventsGetMockRaw(t *testing.T, capturedQuery *ethereum.FilterQuery, capturedRaw *string, logs []ethtypes.Log) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
			Params []json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_getLogs":
			if len(req.Params) > 0 {
				if capturedRaw != nil {
					*capturedRaw = string(req.Params[0])
				}
				if capturedQuery != nil {
					*capturedQuery = decodeFilterArg(t, req.Params[0])
				}
			}
			// Marshal canned logs the same way ethclient deserializes them: each
			// log has hex-encoded fields. Use ethtypes.Log's MarshalJSON which
			// produces the standard JSON-RPC log shape (topics/address/data/
			// blockNumber/transactionHash/etc. as hex strings).
			body, err := json.Marshal(logs)
			if err != nil {
				t.Fatalf("marshal logs: %v", err)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, body)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
}

// decodeFilterArg parses params[0] of an eth_getLogs request back into an
// ethereum.FilterQuery. ethclient's wire form has fromBlock/toBlock as
// strings (hex or label), addresses as a JSON array, and topics as a nested
// array where each position is null (wildcard) or an array of 32-byte hashes.
func decodeFilterArg(t *testing.T, raw json.RawMessage) ethereum.FilterQuery {
	t.Helper()
	var arg struct {
		FromBlock string            `json:"fromBlock"`
		ToBlock   string            `json:"toBlock"`
		Address   []string          `json:"address"`
		Topics    []json.RawMessage `json:"topics"`
	}
	if err := json.Unmarshal(raw, &arg); err != nil {
		t.Fatalf("decodeFilterArg: %v on %s", err, string(raw))
	}
	var q ethereum.FilterQuery
	q.FromBlock = blockTagToBig(arg.FromBlock)
	q.ToBlock = blockTagToBig(arg.ToBlock)
	for _, a := range arg.Address {
		q.Addresses = append(q.Addresses, common.HexToAddress(a))
	}
	if len(arg.Topics) > 0 {
		q.Topics = make([][]common.Hash, len(arg.Topics))
		for i, te := range arg.Topics {
			if len(te) == 0 || string(te) == "null" {
				q.Topics[i] = nil
				continue
			}
			var hashes []string
			if err := json.Unmarshal(te, &hashes); err != nil {
				t.Fatalf("decodeFilterArg topics[%d]: %v on %s", i, err, string(te))
			}
			out := make([]common.Hash, 0, len(hashes))
			for _, h := range hashes {
				out = append(out, common.HexToHash(h))
			}
			q.Topics[i] = out
		}
	}
	return q
}

// blockTagToBig translates a wire tag — hex ("0x..."), the labels
// "latest"/"pending"/"earliest", or empty — into the *big.Int form an
// ethereum.FilterQuery uses. Labels that mean "current head" map to nil so
// tests can assert on the semantic intent rather than the wire encoding.
func blockTagToBig(s string) *big.Int {
	switch s {
	case "", "latest", "pending":
		return nil
	case "earliest":
		return big.NewInt(0)
	}
	if strings.HasPrefix(s, "0x") {
		v, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
		if !ok {
			return nil
		}
		return v
	}
	return nil
}

// makeTransferLog constructs a canned Transfer log: topics = [event_sig, from, to],
// data = uint256(value) big-endian. Used by both happy-decode and decode-failure
// paths (the failure variant truncates data).
func makeTransferLog(t *testing.T, from, to common.Address, value *big.Int, truncate bool) ethtypes.Log {
	t.Helper()
	sig := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	topics := []common.Hash{
		sig,
		common.BytesToHash(from.Bytes()),
		common.BytesToHash(to.Bytes()),
	}
	// uint256 ABI encoding: 32-byte big-endian. abi.Pack would do the same but
	// pulling in the full package for one scalar is overkill — value.FillBytes
	// gives us the exact wire form.
	data := make([]byte, 32)
	value.FillBytes(data)
	if truncate {
		// Drop the last 16 bytes so abi.UnpackIntoMap fails ("output of length
		// 16 is too short for uint256"). INTERNAL, not INVALID_ARGS — the
		// caller's input was valid; the upstream returned malformed data.
		data = data[:16]
	}
	return ethtypes.Log{
		Address:     common.HexToAddress("0x000000000000000000000000000000000000beef"),
		Topics:      topics,
		Data:        data,
		BlockNumber: 123,
		BlockHash:   common.HexToHash("0x" + strings.Repeat("ab", 32)),
		TxHash:      common.HexToHash("0x" + strings.Repeat("cd", 32)),
		TxIndex:     1,
		Index:       0,
		Removed:     false,
	}
}

// TestHandleNodeEventsGet_Happy_NoDecode covers the simplest path: caller
// provides no abi/event, so we return raw log entries with the canonical
// hex-string fields and no `decoded` key.
func TestHandleNodeEventsGet_Happy_NoDecode(t *testing.T) {
	logs := []ethtypes.Log{makeTransferLog(t,
		common.HexToAddress("0x"+strings.Repeat("11", 20)),
		common.HexToAddress("0x"+strings.Repeat("22", 20)),
		big.NewInt(1000), false)}
	srv := newEventsGetMock(t, nil, logs)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeEventsGet(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	out := logsAsMaps(t, data["logs"])
	if len(out) != 1 {
		t.Fatalf("logs length = %d, want 1", len(out))
	}
	entry := out[0]
	for _, k := range []string{"block_number", "block_hash", "tx_hash", "tx_index", "log_index", "address", "topics", "data", "removed"} {
		if _, present := entry[k]; !present {
			t.Errorf("log[0] missing key %q (got keys %v)", k, mapKeys(entry))
		}
	}
	if _, present := entry["decoded"]; present {
		t.Errorf("decoded must be absent without abi+event, got %v", entry["decoded"])
	}
	topics, ok := entry["topics"].([]string)
	if !ok {
		t.Errorf("topics type = %T, want []string", entry["topics"])
	} else if len(topics) != 3 {
		t.Errorf("topics length = %d, want 3", len(topics))
	}
	if got, _ := entry["removed"].(bool); got != false {
		t.Errorf("removed = %v, want false", entry["removed"])
	}
	if got, _ := entry["data"].(string); !strings.HasPrefix(got, "0x") {
		t.Errorf("data = %q, want 0x-prefixed", got)
	}
}

// TestHandleNodeEventsGet_Happy_WithDecode covers the decode path: ABI +
// event name produce a `decoded.args` map keyed by argument name, with
// indexed args (from/to) drawn from topics[1..] and the non-indexed value
// drawn from log.Data.
func TestHandleNodeEventsGet_Happy_WithDecode(t *testing.T) {
	from := common.HexToAddress("0x" + strings.Repeat("11", 20))
	to := common.HexToAddress("0x" + strings.Repeat("22", 20))
	logs := []ethtypes.Log{makeTransferLog(t, from, to, big.NewInt(1000), false)}
	srv := newEventsGetMock(t, nil, logs)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeEventsGet(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
		"abi":     erc20TransferEventABI,
		"event":   "Transfer",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	out := logsAsMaps(t, data["logs"])
	if len(out) != 1 {
		t.Fatalf("logs length = %d, want 1", len(out))
	}
	entry := out[0]
	dec, ok := entry["decoded"].(map[string]any)
	if !ok {
		t.Fatalf("decoded missing or wrong type: %T %v", entry["decoded"], entry["decoded"])
	}
	if dec["event"] != "Transfer" {
		t.Errorf("decoded.event = %v, want Transfer", dec["event"])
	}
	argMap, ok := dec["args"].(map[string]any)
	if !ok {
		t.Fatalf("decoded.args wrong type: %T", dec["args"])
	}
	if got, _ := argMap["from"].(common.Address); got != from {
		t.Errorf("decoded.args.from = %v, want %v", argMap["from"], from)
	}
	if got, _ := argMap["to"].(common.Address); got != to {
		t.Errorf("decoded.args.to = %v, want %v", argMap["to"], to)
	}
	val, ok := argMap["value"].(*big.Int)
	if !ok {
		t.Fatalf("decoded.args.value type = %T, want *big.Int", argMap["value"])
	}
	if val.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("decoded.args.value = %s, want 1000", val.String())
	}
}

func TestHandleNodeEventsGet_BadAddress(t *testing.T) {
	h := newHandleNodeEventsGet(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
		"address": "not-hex",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNodeEventsGet_BadTopicHex(t *testing.T) {
	h := newHandleNodeEventsGet(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
		"topics":  []any{"not-hex"},
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

// TestHandleNodeEventsGet_NoFilter_AllOptional ensures the handler accepts a
// minimal payload (no address / topics / blocks) and returns an empty-but-
// shaped result when the upstream has no matching logs. This is the common
// case for "give me everything from here".
func TestHandleNodeEventsGet_NoFilter_AllOptional(t *testing.T) {
	srv := newEventsGetMock(t, nil, nil) // empty result
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeEventsGet(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	out := logsAsMaps(t, data["logs"])
	if len(out) != 0 {
		t.Errorf("logs length = %d, want 0", len(out))
	}
}

// TestHandleNodeEventsGet_FromBlockInteger asserts integer block numbers are
// hex-encoded on the wire (0x64 = 100), matching ethclient.toFilterArg's
// fromBlock formatting.
func TestHandleNodeEventsGet_FromBlockInteger(t *testing.T) {
	var captured ethereum.FilterQuery
	srv := newEventsGetMock(t, &captured, nil)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeEventsGet(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":    "tn",
		"node_id":    "node1",
		"from_block": 100,
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if captured.FromBlock == nil || captured.FromBlock.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("FromBlock = %v, want 100 (wire 0x64)", captured.FromBlock)
	}
}

// TestHandleNodeEventsGet_FromBlockLatest verifies the "latest" label is
// translated to a nil *big.Int on the FilterQuery — the canonical "no
// specific block" representation. ethclient.toFilterArg (v1.17.2) then
// encodes that nil as fromBlock="0x0" on the wire (its hard-coded default
// for nil FromBlock; toBlock encodes nil as "latest"). Asserting against
// the raw payload is the only way to confirm "latest" did not get
// mis-translated to a positive integer somewhere in the chain — the
// structural FilterQuery decode would conflate nil with big.NewInt(0).
func TestHandleNodeEventsGet_FromBlockLatest(t *testing.T) {
	var captured string
	srv := newEventsGetMockRaw(t, nil, &captured, nil)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeEventsGet(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network":    "tn",
		"node_id":    "node1",
		"from_block": "latest",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(args, bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	// "latest" → nil *big.Int → ethclient encodes as "0x0".
	if !strings.Contains(captured, `"fromBlock":"0x0"`) {
		t.Errorf("fromBlock wire = %q, want it to contain \"fromBlock\":\"0x0\"", captured)
	}
}

// TestHandleNodeEventsGet_TopicsWildcard verifies the middle-position null
// survives all the way to ethereum.FilterQuery as a nil inner slice — the
// "match anything in this position" wire convention.
func TestHandleNodeEventsGet_TopicsWildcard(t *testing.T) {
	var captured ethereum.FilterQuery
	srv := newEventsGetMock(t, &captured, nil)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	t0 := "0x" + strings.Repeat("ab", 32)
	t2 := "0x" + strings.Repeat("ef", 32)

	h := newHandleNodeEventsGet(stateDir)
	body := fmt.Sprintf(`{"network":"tn","node_id":"node1","topics":[%q,null,%q]}`, t0, t2)
	bus, _ := newTestBus(t)
	defer bus.Close()
	if _, err := h(json.RawMessage(body), bus); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(captured.Topics) != 3 {
		t.Fatalf("Topics length = %d, want 3", len(captured.Topics))
	}
	if len(captured.Topics[0]) != 1 || captured.Topics[0][0] != common.HexToHash(t0) {
		t.Errorf("Topics[0] = %v, want [%s]", captured.Topics[0], t0)
	}
	if captured.Topics[1] != nil {
		t.Errorf("Topics[1] = %v, want nil (wildcard)", captured.Topics[1])
	}
	if len(captured.Topics[2]) != 1 || captured.Topics[2][0] != common.HexToHash(t2) {
		t.Errorf("Topics[2] = %v, want [%s]", captured.Topics[2], t2)
	}
}

func TestHandleNodeEventsGet_BadFromBlockHex(t *testing.T) {
	h := newHandleNodeEventsGet(t.TempDir())
	args, _ := json.Marshal(map[string]any{
		"network":    "tn",
		"node_id":    "node1",
		"from_block": "0xnothex",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

// TestHandleNodeEventsGet_DecodeFailure exercises the INTERNAL bucket: the
// caller's args are valid (good ABI, known event), the upstream returns a
// log whose data length is too short for the declared uint256, and
// abi.UnpackIntoMap fails. INVALID_ARGS would mis-attribute the failure to
// the caller.
func TestHandleNodeEventsGet_DecodeFailure(t *testing.T) {
	from := common.HexToAddress("0x" + strings.Repeat("11", 20))
	to := common.HexToAddress("0x" + strings.Repeat("22", 20))
	logs := []ethtypes.Log{makeTransferLog(t, from, to, big.NewInt(1000), true)}
	srv := newEventsGetMock(t, nil, logs)
	defer srv.Close()

	stateDir := t.TempDir()
	saveRemoteFixture(t, stateDir, "tn", srv.URL)

	h := newHandleNodeEventsGet(stateDir)
	args, _ := json.Marshal(map[string]any{
		"network": "tn",
		"node_id": "node1",
		"abi":     erc20TransferEventABI,
		"event":   "Transfer",
	})
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := h(args, bus)
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INTERNAL" {
		t.Errorf("want INTERNAL, got %v", err)
	}
}

func TestAllHandlers_IncludesEventsGet(t *testing.T) {
	h := allHandlers("x", "y")
	if _, ok := h["node.events_get"]; !ok {
		t.Error("allHandlers missing node.events_get")
	}
}

// mapKeys returns the sorted key slice of m for stable error messages. Test-
// only helper; production code paths order keys via the JSON marshaller.
func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// logsAsMaps coerces a handler-produced data["logs"] value into a slice of
// per-entry maps regardless of whether the handler returned []any (post
// JSON-roundtrip shape) or []map[string]any (pre-marshal shape — the actual
// in-memory type the events_get handler produces). Tests assert against
// per-entry keys uniformly without caring which form the dispatcher
// surfaces.
func logsAsMaps(t *testing.T, v any) []map[string]any {
	t.Helper()
	switch typed := v.(type) {
	case []map[string]any:
		return typed
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for i, e := range typed {
			m, ok := e.(map[string]any)
			if !ok {
				t.Fatalf("logs[%d] type = %T, want map", i, e)
			}
			out = append(out, m)
		}
		return out
	case nil:
		return nil
	default:
		t.Fatalf("logs type = %T, want slice of maps", v)
		return nil
	}
}
