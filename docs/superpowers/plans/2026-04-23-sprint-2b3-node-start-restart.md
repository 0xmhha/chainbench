# Sprint 2b.3 node.start + node.restart Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `node.start` (single subprocess call) and `node.restart` (stop + start composition with both events) wire commands, reusing the LocalDriver established in Sprint 2b.2.

**Architecture:** Extend `drivers/local` with `StartNode` (thin wrapper mirroring `StopNode`). In `cmd/chainbench-net`, extract a shared `resolveNodeID` helper (refactor of Sprint 2b.2 `handleNodeStop`), then add `handleNodeStart` and `handleNodeRestart` (composed from Stop+Start with explicit event ordering). Extend the stub script to cover `node start`. E2E tests cross-validate event sequence against the schema.

**Tech Stack:** Go 1.25 stdlib only for driver; cmd layer uses Sprint 2a wire/events + Sprint 2b.1 state + Sprint 2b.2 LocalDriver.

**Spec reference:** `docs/superpowers/specs/2026-04-23-sprint-2b3-node-start-restart-design.md`.

---

## File Structure

**Created in this plan:**
- `network/internal/drivers/local/start.go`
- `network/internal/drivers/local/start_test.go`

**Modified:**
- `network/cmd/chainbench-net/testdata/chainbench-stub.sh` (add `node start` case)
- `network/cmd/chainbench-net/handlers.go` (extract `resolveNodeID`, add 2 new handlers, register)
- `network/cmd/chainbench-net/handlers_test.go` (append 9 tests)
- `network/cmd/chainbench-net/e2e_test.go` (append 2 E2E tests)

---

## Task 1: `drivers/local/start.go` — StartNode wrapper

**Files:**
- Create: `network/internal/drivers/local/start.go`
- Create: `network/internal/drivers/local/start_test.go`

- [ ] **Step 1.1: Write failing test**

Create `network/internal/drivers/local/start_test.go`:

```go
package local

import (
	"context"
	"strings"
	"testing"
)

func TestStartNode_CallsCorrectArgs(t *testing.T) {
	var got []string
	d := NewDriverWithExec("/opt/chainbench", recordingExecFn(&got))
	res, err := d.StartNode(context.Background(), "3")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit: got %d", res.ExitCode)
	}
	want := []string{"/opt/chainbench/chainbench.sh", "node", "start", "3"}
	if len(got) != len(want) {
		t.Fatalf("argc: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
	if !strings.Contains(res.Stdout, "stopped") {
		// recordingExecFn in stop_test.go echoes "stopped" — fine for this test
		// since we only care about args.
		t.Logf("stdout (informational): %q", res.Stdout)
	}
}

func TestStartNode_EmptyNodeNum_StillExecs(t *testing.T) {
	var got []string
	d := NewDriverWithExec("/x", recordingExecFn(&got))
	_, err := d.StartNode(context.Background(), "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(got) != 4 || got[3] != "" {
		t.Errorf("expected empty 4th arg, got %v", got)
	}
}
```

Note: `recordingExecFn` already exists in `stop_test.go` — Go allows in-package test helper reuse.

- [ ] **Step 1.2: Run test — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/drivers/local/... -run TestStartNode 2>&1 | tail -5
```
Expected: FAIL — `d.StartNode undefined (type *Driver has no field or method StartNode)`.

- [ ] **Step 1.3: Write `start.go`**

Create `network/internal/drivers/local/start.go`:

```go
package local

import "context"

// StartNode invokes `chainbench.sh node start <nodeNum>`. nodeNum is the
// numeric pids.json key ("1", "2", ...). Thin wrapper over Run with a
// fixed argv shape — it performs no input validation; callers are
// responsible for checking the node exists and is currently stopped.
func (d *Driver) StartNode(ctx context.Context, nodeNum string) (*RunResult, error) {
	return d.Run(ctx, "node", "start", nodeNum)
}
```

- [ ] **Step 1.4: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/drivers/local/... -v -run TestStartNode
```
Expected: 2 tests PASS.

- [ ] **Step 1.5: Full driver package + race + coverage**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race -cover ./internal/drivers/local/...
```
Expected: PASS, coverage ≥ 85%.

- [ ] **Step 1.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/drivers/local/start.go network/internal/drivers/local/start_test.go
git commit -m "network: add local driver StartNode wrapper"
```

---

## Task 2: Stub script — add `node start` case

**Files:**
- Modify: `network/cmd/chainbench-net/testdata/chainbench-stub.sh`

- [ ] **Step 2.1: Read current stub**

```bash
cat /Users/wm-it-22-00661/Work/github/tools/chainbench/network/cmd/chainbench-net/testdata/chainbench-stub.sh
```
(Confirm current case structure.)

- [ ] **Step 2.2: Add `node start` case**

Use Edit tool on `network/cmd/chainbench-net/testdata/chainbench-stub.sh`.

Find:
```
  "node stop")
    node="${3:-}"
    if [[ -z "$node" ]]; then
      echo "missing node num" >&2
      exit 1
    fi
    if [[ "$node" == "fail" ]]; then
      echo "stub: forced failure for testing" >&2
      exit 1
    fi
    echo "stub: node $node stopped"
    exit 0
    ;;
  *)
```

Replace with (inserts new `"node start"` case before `*`):
```
  "node stop")
    node="${3:-}"
    if [[ -z "$node" ]]; then
      echo "missing node num" >&2
      exit 1
    fi
    if [[ "$node" == "fail" ]]; then
      echo "stub: forced failure for testing" >&2
      exit 1
    fi
    echo "stub: node $node stopped"
    exit 0
    ;;
  "node start")
    node="${3:-}"
    if [[ -z "$node" ]]; then
      echo "missing node num" >&2
      exit 1
    fi
    if [[ "$node" == "fail" ]]; then
      echo "stub: forced start failure for testing" >&2
      exit 1
    fi
    echo "stub: node $node started"
    exit 0
    ;;
  *)
```

- [ ] **Step 2.3: Smoke test the stub**

```bash
bash /Users/wm-it-22-00661/Work/github/tools/chainbench/network/cmd/chainbench-net/testdata/chainbench-stub.sh node start 1
echo "exit=$?"
bash /Users/wm-it-22-00661/Work/github/tools/chainbench/network/cmd/chainbench-net/testdata/chainbench-stub.sh node start fail
echo "exit=$?"
```
Expected:
- First: stdout `stub: node 1 started`, exit 0
- Second: stderr `stub: forced start failure for testing`, exit 1

- [ ] **Step 2.4: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/testdata/chainbench-stub.sh
git commit -m "network: extend stub script with node start case"
```

---

## Task 3: Refactor — extract `resolveNodeID` helper in `handlers.go`

**Files:**
- Modify: `network/cmd/chainbench-net/handlers.go`

This is a pure refactor — no behavior change. Existing `handleNodeStop` changes to use the helper. Existing tests must continue to pass.

- [ ] **Step 3.1: Read current `handlers.go`**

```bash
cat /Users/wm-it-22-00661/Work/github/tools/chainbench/network/cmd/chainbench-net/handlers.go
```
Identify the block inside `newHandleNodeStop` that does: args parse, prefix check, suffix extraction, state.LoadActive check, Nodes lookup.

- [ ] **Step 3.2: Add `resolveNodeID` helper**

Insert this function in `handlers.go` (placement: after `newHandleNetworkLoad`, before `newHandleNodeStop`):

```go
// resolveNodeID parses the command envelope args into a (nodeID, nodeNum) pair,
// validates the "node" prefix, and confirms the id exists in the active
// network's pids.json. Returns APIError sentinels:
//   INVALID_ARGS  — malformed args, missing/bad node_id, node not found
//   UPSTREAM_ERROR — state.LoadActive failure (pids.json missing, parse error)
func resolveNodeID(stateDir string, args json.RawMessage) (nodeID, nodeNum string, err error) {
	var req struct {
		NodeID string `json:"node_id"`
	}
	if len(args) > 0 {
		if uerr := json.Unmarshal(args, &req); uerr != nil {
			return "", "", NewInvalidArgs(fmt.Sprintf("args: %v", uerr))
		}
	}
	if req.NodeID == "" {
		return "", "", NewInvalidArgs("args.node_id is required")
	}
	if !strings.HasPrefix(req.NodeID, "node") {
		return "", "", NewInvalidArgs(fmt.Sprintf(`node_id must start with "node" prefix (got %q)`, req.NodeID))
	}
	num := strings.TrimPrefix(req.NodeID, "node")
	if num == "" {
		return "", "", NewInvalidArgs("node_id missing numeric suffix")
	}
	net, lerr := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: "local"})
	if lerr != nil {
		return "", "", NewUpstream("failed to load active state", lerr)
	}
	for _, n := range net.Nodes {
		if n.Id == req.NodeID {
			return req.NodeID, num, nil
		}
	}
	return "", "", NewInvalidArgs(fmt.Sprintf("node_id %q not found in active network", req.NodeID))
}
```

- [ ] **Step 3.3: Refactor `newHandleNodeStop` to use the helper**

Find the existing `newHandleNodeStop` body (the block after its opening `return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {` up through the end of the `local.NewDriver(...).StopNode(...)` block).

Replace the initial validation block (everything from `var req struct ...` through the node existence check) with a single call:

```go
return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
    nodeID, nodeNum, err := resolveNodeID(stateDir, args)
    if err != nil {
        return nil, err
    }

    driver := local.NewDriver(chainbenchDir)
    result, err := driver.StopNode(context.Background(), nodeNum)
    if err != nil {
        return nil, NewUpstream("subprocess exec failed", err)
    }
    if result.ExitCode != 0 {
        tail := strings.TrimSpace(result.Stderr)
        if len(tail) > 512 {
            tail = tail[:512]
        }
        return nil, NewUpstream(
            fmt.Sprintf("chainbench.sh node stop %s exited %d: %s", nodeNum, result.ExitCode, tail),
            nil,
        )
    }

    _ = bus.Publish(events.Event{
        Name: types.EventName("node.stopped"),
        Data: map[string]any{"node_id": nodeID, "reason": "manual"},
    })
    return map[string]any{"node_id": nodeID, "stopped": true}, nil
}
```

- [ ] **Step 3.4: Run — all existing tests still pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/...
```
Expected: all tests PASS (no behavior change). 25+ tests.

- [ ] **Step 3.5: Build + vet + fmt**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./... && go vet ./... && gofmt -l cmd/chainbench-net/
```
Expected: clean.

- [ ] **Step 3.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/handlers.go
git commit -m "network: extract resolveNodeID helper for node handlers"
```

---

## Task 4: `handleNodeStart` + register + tests

**Files:**
- Modify: `network/cmd/chainbench-net/handlers.go` (add function + register)
- Modify: `network/cmd/chainbench-net/handlers_test.go` (append tests)

- [ ] **Step 4.1: Write failing tests**

Append to `network/cmd/chainbench-net/handlers_test.go`:

```go
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
```

- [ ] **Step 4.2: Run — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -run 'TestHandleNodeStart|TestAllHandlers_IncludesNodeStart' 2>&1 | tail -5
```
Expected: FAIL — `undefined: newHandleNodeStart`.

- [ ] **Step 4.3: Add `newHandleNodeStart` to `handlers.go`**

Append to `handlers.go` (after `newHandleNodeStop`):

```go
// newHandleNodeStart returns a Handler that starts a previously-stopped node
// via LocalDriver. Args: { "node_id": "nodeN" }.
// On success: emits "node.started" event, returns {node_id, started:true}.
func newHandleNodeStart(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		nodeID, nodeNum, err := resolveNodeID(stateDir, args)
		if err != nil {
			return nil, err
		}

		driver := local.NewDriver(chainbenchDir)
		result, err := driver.StartNode(context.Background(), nodeNum)
		if err != nil {
			return nil, NewUpstream("subprocess exec failed", err)
		}
		if result.ExitCode != 0 {
			tail := strings.TrimSpace(result.Stderr)
			if len(tail) > 512 {
				tail = tail[:512]
			}
			return nil, NewUpstream(
				fmt.Sprintf("chainbench.sh node start %s exited %d: %s", nodeNum, result.ExitCode, tail),
				nil,
			)
		}

		_ = bus.Publish(events.Event{
			Name: types.EventName("node.started"),
			Data: map[string]any{"node_id": nodeID},
		})
		return map[string]any{"node_id": nodeID, "started": true}, nil
	}
}
```

- [ ] **Step 4.4: Register in `allHandlers`**

Update the `allHandlers` return map:

```go
func allHandlers(stateDir, chainbenchDir string) map[string]Handler {
	return map[string]Handler{
		"network.load": newHandleNetworkLoad(stateDir),
		"node.stop":    newHandleNodeStop(stateDir, chainbenchDir),
		"node.start":   newHandleNodeStart(stateDir, chainbenchDir),
	}
}
```

- [ ] **Step 4.5: Run — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v -run 'TestHandleNodeStart|TestAllHandlers_IncludesNodeStart'
```
Expected: 5 tests PASS.

- [ ] **Step 4.6: Full cmd test + race**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./cmd/chainbench-net/...
```
Expected: all tests PASS.

- [ ] **Step 4.7: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go
git commit -m "network: add node.start handler using local driver"
```

---

## Task 5: `handleNodeRestart` — composed handler + tests

**Files:**
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`

- [ ] **Step 5.1: Write failing tests**

Append to `handlers_test.go`:

```go
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
```

- [ ] **Step 5.2: Run — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -run 'TestHandleNodeRestart|TestAllHandlers_IncludesNodeRestart' 2>&1 | tail -5
```
Expected: FAIL — `undefined: newHandleNodeRestart`.

- [ ] **Step 5.3: Add `newHandleNodeRestart` to `handlers.go`**

Append to `handlers.go`:

```go
// newHandleNodeRestart returns a Handler that composes node.stop then
// node.start via LocalDriver. Args: { "node_id": "nodeN" }.
//
// Event ordering invariant:
//   1. If stop fails: no events emitted, return UPSTREAM_ERROR.
//   2. If stop succeeds + start fails: emit "node.stopped", return UPSTREAM_ERROR.
//   3. If both succeed: emit "node.stopped" then "node.started".
//
// Returns {node_id, restarted:true} on full success.
func newHandleNodeRestart(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		nodeID, nodeNum, err := resolveNodeID(stateDir, args)
		if err != nil {
			return nil, err
		}

		driver := local.NewDriver(chainbenchDir)

		// --- stop phase ---
		stopRes, err := driver.StopNode(context.Background(), nodeNum)
		if err != nil {
			return nil, NewUpstream("restart aborted: stop exec failed", err)
		}
		if stopRes.ExitCode != 0 {
			tail := strings.TrimSpace(stopRes.Stderr)
			if len(tail) > 512 {
				tail = tail[:512]
			}
			return nil, NewUpstream(
				fmt.Sprintf("restart aborted: stop exited %d: %s", stopRes.ExitCode, tail),
				nil,
			)
		}
		_ = bus.Publish(events.Event{
			Name: types.EventName("node.stopped"),
			Data: map[string]any{"node_id": nodeID, "reason": "restart"},
		})

		// --- start phase ---
		startRes, err := driver.StartNode(context.Background(), nodeNum)
		if err != nil {
			return nil, NewUpstream("restart incomplete: stop ok, start exec failed", err)
		}
		if startRes.ExitCode != 0 {
			tail := strings.TrimSpace(startRes.Stderr)
			if len(tail) > 512 {
				tail = tail[:512]
			}
			return nil, NewUpstream(
				fmt.Sprintf("restart incomplete: stop ok, start exited %d: %s", startRes.ExitCode, tail),
				nil,
			)
		}
		_ = bus.Publish(events.Event{
			Name: types.EventName("node.started"),
			Data: map[string]any{"node_id": nodeID},
		})

		return map[string]any{"node_id": nodeID, "restarted": true}, nil
	}
}
```

Update `allHandlers`:

```go
func allHandlers(stateDir, chainbenchDir string) map[string]Handler {
	return map[string]Handler{
		"network.load": newHandleNetworkLoad(stateDir),
		"node.stop":    newHandleNodeStop(stateDir, chainbenchDir),
		"node.start":   newHandleNodeStart(stateDir, chainbenchDir),
		"node.restart": newHandleNodeRestart(stateDir, chainbenchDir),
	}
}
```

- [ ] **Step 5.4: Run — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v -run 'TestHandleNodeRestart|TestAllHandlers_IncludesNodeRestart'
```
Expected: 5 tests PASS.

- [ ] **Step 5.5: Full cmd test + race**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./cmd/chainbench-net/...
```
Expected: all PASS.

- [ ] **Step 5.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go
git commit -m "network: add node.restart composed handler"
```

---

## Task 6: E2E tests — start + restart via root command

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`

- [ ] **Step 6.1: Append failing tests**

Append to `e2e_test.go`:

```go
func TestE2E_NodeStart_ViaRootCommand(t *testing.T) {
	stateDir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stateDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	chainbenchDir := t.TempDir()
	stub, _ := os.ReadFile(filepath.Join("testdata", "chainbench-stub.sh"))
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), stub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_DIR", chainbenchDir)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"node.start","args":{"node_id":"node1"}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	var sawEvent, sawResultOK bool
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
		msg, _ := wire.DecodeMessage(line)
		switch m := msg.(type) {
		case wire.EventMessage:
			if string(m.Name) == "node.started" {
				sawEvent = true
			}
		case wire.ResultMessage:
			if m.Ok {
				sawResultOK = true
			}
		}
	}
	if !sawEvent {
		t.Error("expected node.started event")
	}
	if !sawResultOK {
		t.Error("expected successful result")
	}
}

func TestE2E_NodeRestart_ViaRootCommand_EventOrder(t *testing.T) {
	stateDir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stateDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	chainbenchDir := t.TempDir()
	stub, _ := os.ReadFile(filepath.Join("testdata", "chainbench-stub.sh"))
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), stub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_DIR", chainbenchDir)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"node.restart","args":{"node_id":"node1"}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Parse all lines, collect event names in order.
	var eventNames []string
	var sawResultOK bool
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
		msg, _ := wire.DecodeMessage(line)
		switch m := msg.(type) {
		case wire.EventMessage:
			eventNames = append(eventNames, string(m.Name))
		case wire.ResultMessage:
			if m.Ok {
				sawResultOK = true
			}
		}
	}
	wantOrder := []string{"node.stopped", "node.started"}
	if len(eventNames) != 2 {
		t.Fatalf("event count: got %d (%v), want 2", len(eventNames), eventNames)
	}
	for i := range wantOrder {
		if eventNames[i] != wantOrder[i] {
			t.Errorf("event[%d]: got %q, want %q", i, eventNames[i], wantOrder[i])
		}
	}
	if !sawResultOK {
		t.Error("expected successful result")
	}
}
```

- [ ] **Step 6.2: Run — expect pass (handlers already exist)**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v -run 'TestE2E_NodeStart|TestE2E_NodeRestart'
```
Expected: 2 tests PASS.

- [ ] **Step 6.3: Full module verify**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network
go build ./...
go test -race ./...
go vet ./...
gofmt -l .
go build -tags tools ./...
```
Expected: all clean.

- [ ] **Step 6.4: Coverage**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -cover ./internal/drivers/local/... ./cmd/chainbench-net/...
```
Expected: `drivers/local` ≥ 85%, `cmd/chainbench-net` ≥ 80%.

- [ ] **Step 6.5: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/e2e_test.go
git commit -m "network: add end-to-end tests for node.start and node.restart"
```

---

## Final verification

- [ ] **Commit list**

```bash
git log --oneline 684b91e..HEAD
```
Expected 6 commits:
1. `network: add local driver StartNode wrapper`
2. `network: extend stub script with node start case`
3. `network: extract resolveNodeID helper for node handlers`
4. `network: add node.start handler using local driver`
5. `network: add node.restart composed handler`
6. `network: add end-to-end tests for node.start and node.restart`

---

## Out of scope (explicit)

- `node.tail_log` (streaming — separate concurrency model)
- `node.rpc` (HTTP — not subprocess)
- Remote drivers
- Restart retry/rollback (start failure leaves node in stopped state, operator re-runs)
