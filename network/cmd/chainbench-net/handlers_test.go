package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

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
