# Sprint 2b.4 node.tail_log Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the `node.tail_log` wire command (finite tail). Handler reads `log_file` path from pids.json (via `state.LoadActive`'s ProviderMeta), tails the last N lines with a pure-Go ring buffer, returns `{node_id, log_file, lines:[...]}`. No subprocess. No streaming. No schema change.

**Architecture:** Two `internal/state` additions: `TailFile(path, n)` helper + one-line extension of `buildNodes` to surface `log_file` in `ProviderMeta`. A new handler `newHandleNodeTailLog` in `cmd/chainbench-net/handlers.go` composes these via the existing `resolveNodeID` helper (extended slightly to expose the resolved Node). E2E exercises the full wire pipeline with a tempdir log fixture.

**Tech Stack:** Go 1.25 stdlib only (`bufio`, `os`, `strconv`, `encoding/json`). Builds on Sprint 2a wire/events, 2b.1 state + errors + handlers, 2b.3 `resolveNodeID`.

**Spec reference:** `docs/superpowers/specs/2026-04-23-sprint-2b4-node-tail-log-design.md`.

---

## File Structure

**Created:**
- `network/internal/state/tail.go`
- `network/internal/state/tail_test.go`

**Modified:**
- `network/internal/state/network.go` (add `log_file` to `ProviderMeta`)
- `network/internal/state/network_test.go` (assert on new key)
- `network/cmd/chainbench-net/handlers.go` (add handler + tweak `resolveNodeID` return)
- `network/cmd/chainbench-net/handlers_test.go` (append 8 tests)
- `network/cmd/chainbench-net/e2e_test.go` (append 1 E2E test)

---

## Task 1: `state/tail.go` — TailFile helper

**Files:**
- Create: `network/internal/state/tail.go`
- Create: `network/internal/state/tail_test.go`

- [ ] **Step 1.1: Write failing test**

Create `network/internal/state/tail_test.go`:

```go
package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempLog(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "node.log")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTailFile_ReturnsLastNLines(t *testing.T) {
	path := writeTempLog(t, "a\nb\nc\nd\ne\n")
	got, err := TailFile(path, 3)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	want := []string{"c", "d", "e"}
	if len(got) != 3 {
		t.Fatalf("len: got %d, want 3 (%v)", len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("line %d: got %q, want %q", i, got[i], w)
		}
	}
}

func TestTailFile_FewerLinesThanN(t *testing.T) {
	path := writeTempLog(t, "a\nb\nc\n")
	got, err := TailFile(path, 10)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	want := []string{"a", "b", "c"}
	if len(got) != 3 {
		t.Fatalf("len: got %d", len(got))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("line %d: got %q, want %q", i, got[i], w)
		}
	}
}

func TestTailFile_EmptyFile(t *testing.T) {
	path := writeTempLog(t, "")
	got, err := TailFile(path, 5)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty: got %v, want []", got)
	}
}

func TestTailFile_SingleLineNoTrailingNewline(t *testing.T) {
	path := writeTempLog(t, "only-one")
	got, err := TailFile(path, 3)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(got) != 1 || got[0] != "only-one" {
		t.Errorf("got %v", got)
	}
}

func TestTailFile_NEqualOne(t *testing.T) {
	path := writeTempLog(t, "a\nb\nc\n")
	got, err := TailFile(path, 1)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(got) != 1 || got[0] != "c" {
		t.Errorf("got %v, want [c]", got)
	}
}

func TestTailFile_MissingFile(t *testing.T) {
	_, err := TailFile("/no/such/file.log", 10)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestTailFile_LargeLine(t *testing.T) {
	// A single line larger than default bufio.Scanner buffer (64 KiB).
	big := strings.Repeat("x", 200*1024)
	path := writeTempLog(t, big+"\n")
	got, err := TailFile(path, 1)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(got) != 1 || len(got[0]) != 200*1024 {
		t.Errorf("large line not preserved: got len=%d", len(got[0]))
	}
}
```

- [ ] **Step 1.2: Run — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... -run TestTailFile 2>&1 | tail -5
```
Expected: FAIL — `undefined: TailFile`.

- [ ] **Step 1.3: Write `tail.go`**

Create `network/internal/state/tail.go`:

```go
package state

import (
	"bufio"
	"fmt"
	"os"
)

// TailFile returns the last n lines of the file at path (n must be >= 1).
// Lines are returned without their trailing newline. If the file has fewer
// than n lines, all lines are returned.
//
// Uses a ring buffer — O(file-size) read time, O(n * avg_line_len) memory.
// Acceptable for log files up to hundreds of MB with modest n.
func TailFile(path string, n int) ([]string, error) {
	if n < 1 {
		return nil, fmt.Errorf("state: TailFile: n must be >= 1, got %d", n)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("state: TailFile open: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Allow large lines — some node logs may exceed the default 64 KiB.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	ring := make([]string, 0, n)
	for scanner.Scan() {
		if len(ring) == n {
			// Drop oldest (first) — shift in place.
			copy(ring, ring[1:])
			ring = ring[:n-1]
		}
		ring = append(ring, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("state: TailFile read: %w", err)
	}
	return ring, nil
}
```

- [ ] **Step 1.4: Run — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... -v -run TestTailFile
```
Expected: 7 subtests PASS.

- [ ] **Step 1.5: Race + vet + fmt + coverage**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race -cover ./internal/state/... && go vet ./... && gofmt -l internal/state/
```
Expected: clean, coverage ≥ 85% (existing state coverage should not drop).

- [ ] **Step 1.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/state/tail.go network/internal/state/tail_test.go
git commit -m "network: add state TailFile ring-buffer helper"
```

---

## Task 2: Expose `log_file` in `ProviderMeta`

**Files:**
- Modify: `network/internal/state/network.go`
- Modify: `network/internal/state/network_test.go`

- [ ] **Step 2.1: Update test assertion (RED)**

Read `network/internal/state/network_test.go` to locate `TestLoadActive_HappyPath`. After the existing node[0] field assertions, append an assertion block. Use Edit with exact match on a nearby line to insert.

Locate (near the end of `TestLoadActive_HappyPath`):
```go
	if net.Nodes[1].Role == nil || string(*net.Nodes[1].Role) != "endpoint" {
		t.Errorf("role: got %v", net.Nodes[1].Role)
	}
}
```

Replace with:
```go
	if net.Nodes[1].Role == nil || string(*net.Nodes[1].Role) != "endpoint" {
		t.Errorf("role: got %v", net.Nodes[1].Role)
	}
	// ProviderMeta must expose log_file for node.tail_log handler.
	if meta := net.Nodes[0].ProviderMeta; meta["log_file"] != "/tmp/node-data/logs/node1.log" {
		t.Errorf("node1 log_file: got %v", meta["log_file"])
	}
	if meta := net.Nodes[1].ProviderMeta; meta["log_file"] != "/tmp/node-data/logs/node5.log" {
		t.Errorf("node5 log_file: got %v", meta["log_file"])
	}
}
```

- [ ] **Step 2.2: Run — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... -run TestLoadActive_HappyPath 2>&1 | tail -5
```
Expected: FAIL — assertions report `log_file: got <nil>` because `ProviderMeta` doesn't yet include the key.

- [ ] **Step 2.3: Update `network.go`**

Read `network/internal/state/network.go`. Find the `buildNodes` function, specifically the `node := types.Node{...}` literal with `ProviderMeta: types.NodeProviderMeta{"pid_key": id}`.

Replace:
```go
			ProviderMeta: types.NodeProviderMeta{"pid_key": id},
```
with:
```go
			ProviderMeta: types.NodeProviderMeta{
				"pid_key":  id,
				"log_file": info.LogFile,
			},
```

- [ ] **Step 2.4: Run — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... -v -run TestLoadActive_HappyPath
```
Expected: PASS. Other state tests unaffected.

- [ ] **Step 2.5: Full state test + schema cross-check**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./internal/state/...
```
Expected: all PASS. The existing `TestLoadActive_OutputValidatesAgainstSchema` must still pass — `log_file` lives inside `ProviderMeta` which is a free-form object, so no schema change.

- [ ] **Step 2.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/state/network.go network/internal/state/network_test.go
git commit -m "network: expose log_file in ProviderMeta for tail handler"
```

---

## Task 3: `handleNodeTailLog` handler + register + tests

**Files:**
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`

- [ ] **Step 3.1: Write failing tests**

Append to `network/cmd/chainbench-net/handlers_test.go`:

```go
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
```

Also add `"strconv"` to the test file's imports if not already present.

- [ ] **Step 3.2: Run — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -run 'TestHandleNodeTailLog|TestAllHandlers_IncludesNodeTailLog' 2>&1 | tail -5
```
Expected: FAIL — `undefined: newHandleNodeTailLog`.

- [ ] **Step 3.3: Add `newHandleNodeTailLog` to `handlers.go`**

Append to `network/cmd/chainbench-net/handlers.go`:

```go
const (
	defaultTailLines = 50
	maxTailLines     = 1000
)

// newHandleNodeTailLog returns a Handler that reads the tail of a node's log
// file. Args: { "node_id": "nodeN", "lines": 50 (optional) }.
// No subprocess, no events — pure file read. Returns {node_id, log_file, lines}.
func newHandleNodeTailLog(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			NodeID string `json:"node_id"`
			Lines  *int   `json:"lines"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		// Delegate the common node_id resolution to the helper by re-marshaling
		// just the node_id field. This keeps resolveNodeID's surface unchanged.
		nidPayload, _ := json.Marshal(map[string]any{"node_id": req.NodeID})
		nodeID, _, rerr := resolveNodeID(stateDir, nidPayload)
		if rerr != nil {
			return nil, rerr
		}
		lines := defaultTailLines
		if req.Lines != nil {
			lines = *req.Lines
		}
		if lines < 1 {
			return nil, NewInvalidArgs(fmt.Sprintf("args.lines must be >= 1, got %d", lines))
		}
		if lines > maxTailLines {
			return nil, NewInvalidArgs(fmt.Sprintf("args.lines must be <= %d, got %d", maxTailLines, lines))
		}

		net, err := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: "local"})
		if err != nil {
			return nil, NewUpstream("failed to load active state", err)
		}
		var logPath string
		for _, n := range net.Nodes {
			if n.Id == nodeID {
				if v, ok := n.ProviderMeta["log_file"].(string); ok {
					logPath = v
				}
				break
			}
		}
		if logPath == "" {
			return nil, NewUpstream(fmt.Sprintf("log_file unknown for node %q", nodeID), nil)
		}

		tailed, err := state.TailFile(logPath, lines)
		if err != nil {
			return nil, NewUpstream(fmt.Sprintf("tail log %s", logPath), err)
		}
		return map[string]any{
			"node_id":  nodeID,
			"log_file": logPath,
			"lines":    tailed,
		}, nil
	}
}
```

Register in `allHandlers`:
```go
func allHandlers(stateDir, chainbenchDir string) map[string]Handler {
	return map[string]Handler{
		"network.load":  newHandleNetworkLoad(stateDir),
		"node.stop":     newHandleNodeStop(stateDir, chainbenchDir),
		"node.start":    newHandleNodeStart(stateDir, chainbenchDir),
		"node.restart":  newHandleNodeRestart(stateDir, chainbenchDir),
		"node.tail_log": newHandleNodeTailLog(stateDir),
	}
}
```

- [ ] **Step 3.4: Run — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v -run 'TestHandleNodeTailLog|TestAllHandlers_IncludesNodeTailLog'
```
Expected: 8 subtests PASS.

- [ ] **Step 3.5: Full cmd test + race + vet + fmt**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./cmd/chainbench-net/... && go vet ./... && gofmt -l cmd/chainbench-net/
```
Expected: all clean.

- [ ] **Step 3.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go
git commit -m "network: add node.tail_log handler for finite log tailing"
```

---

## Task 4: E2E test via root command

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`

- [ ] **Step 4.1: Append failing test**

Append to `network/cmd/chainbench-net/e2e_test.go`:

```go
func TestE2E_NodeTailLog_ViaRootCommand(t *testing.T) {
	stateDir := t.TempDir()
	// Load pids.json and rewrite log_file to a tempdir path, then write the log.
	pidsBytes, err := os.ReadFile(filepath.Join("testdata", "pids.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(pidsBytes, &raw); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(stateDir, "node1.log")
	raw["nodes"].(map[string]any)["1"].(map[string]any)["log_file"] = logPath
	patched, _ := json.Marshal(raw)
	if err := os.WriteFile(filepath.Join(stateDir, "pids.json"), patched, 0o644); err != nil {
		t.Fatal(err)
	}
	profBytes, _ := os.ReadFile(filepath.Join("testdata", "current-profile.yaml"))
	if err := os.WriteFile(filepath.Join(stateDir, "current-profile.yaml"), profBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("line1\nline2\nline3\nline4\nline5\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	// Handler doesn't need CHAINBENCH_DIR, but set something harmless.
	t.Setenv("CHAINBENCH_DIR", t.TempDir())

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"node.tail_log","args":{"node_id":"node1","lines":3}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Parse all lines — expect zero events and one OK result with data.lines.
	var gotLines []any
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
		msg, _ := wire.DecodeMessage(line)
		if rm, ok := msg.(wire.ResultMessage); ok {
			if !rm.Ok {
				t.Fatalf("result not ok: %s", line)
			}
			if ls, ok := rm.Data["lines"].([]any); ok {
				gotLines = ls
			}
		}
	}
	if len(gotLines) != 3 {
		t.Fatalf("lines: got %v (len %d)", gotLines, len(gotLines))
	}
	want := []string{"line3", "line4", "line5"}
	for i, w := range want {
		if gotLines[i] != w {
			t.Errorf("line %d: got %v, want %q", i, gotLines[i], w)
		}
	}
}
```

Imports already available from prior E2E tests (`bufio`, `bytes`, `encoding/json`, `os`, `path/filepath`, `strings`, `wire`, `schema`).

- [ ] **Step 4.2: Run — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v -run TestE2E_NodeTailLog
```
Expected: PASS.

- [ ] **Step 4.3: Full module verification**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network
go build ./...
go test -race ./...
go vet ./...
gofmt -l .
go build -tags tools ./...
```
Expected: all clean.

- [ ] **Step 4.4: Coverage**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -cover ./internal/state/... ./cmd/chainbench-net/...
```
Expected: `state` ≥ 85% (should stay near 92%), `cmd/chainbench-net` ≥ 80%.

- [ ] **Step 4.5: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/e2e_test.go
git commit -m "network: add end-to-end test for node.tail_log command"
```

---

## Final verification

- [ ] **Commit list**

```bash
git log --oneline 21735c5..HEAD
```
Expected 4 commits:
1. `network: add state TailFile ring-buffer helper`
2. `network: expose log_file in ProviderMeta for tail handler`
3. `network: add node.tail_log handler for finite log tailing`
4. `network: add end-to-end test for node.tail_log command`

---

## Out of scope (explicit)

- Streaming / follow mode (separate sprint + schema extension for `node.log.line` event)
- Log level filtering (stdout/stderr split, JSON log parsing)
- Remote tail (remote driver sprint)
- `chainbench.sh node log` bash reuse (path bug — orthogonal fix, not here)
