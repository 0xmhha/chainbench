package main

import (
	"bytes"
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
