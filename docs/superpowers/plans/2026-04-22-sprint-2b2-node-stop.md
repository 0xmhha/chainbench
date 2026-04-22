# Sprint 2b.2 node.stop via LocalDriver Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the `node.stop` wire command: parse envelope, verify node exists via `state.LoadActive`, exec `chainbench.sh node stop <N>` via a new `LocalDriver`, stream subprocess output to slog, emit `node.stopped` event, return `{node_id, stopped:true}`.

**Architecture:** A new `network/internal/drivers/local/` package is a pure process runner — `Driver.Run(ctx, args...)` executes `<chainbenchDir>/chainbench.sh <args...>`, streams stdout/stderr to slog, returns `RunResult{ExitCode, Stdout, Stderr, Duration}`. `StopNode(ctx, nodeNum)` is a thin wrapper. The `node.stop` handler in `cmd/chainbench-net/handlers.go` validates args, invokes the driver, emits events, and returns the result map. E2E validates the full pipeline against a `chainbench-stub.sh` fake.

**Tech Stack:** Go 1.25 stdlib only (`context`, `os/exec`, `bufio`, `io`, `log/slog`, `time`). Builds on Sprint 2a (wire/events) and Sprint 2b.1 (state, errors, handlers, run).

**Spec reference:** `docs/superpowers/specs/2026-04-22-sprint-2b2-node-stop-design.md`.

---

## File Structure

**Created in this plan:**
- `network/internal/drivers/local/doc.go`
- `network/internal/drivers/local/driver.go`
- `network/internal/drivers/local/driver_test.go`
- `network/internal/drivers/local/stop.go`
- `network/internal/drivers/local/stop_test.go`
- `network/cmd/chainbench-net/testdata/chainbench-stub.sh`

**Modified:**
- `network/cmd/chainbench-net/handlers.go` (+ handleNodeStop, allHandlers signature)
- `network/cmd/chainbench-net/handlers_test.go` (+ 5 tests)
- `network/cmd/chainbench-net/run.go` (read CHAINBENCH_DIR env, pass to allHandlers)
- `network/cmd/chainbench-net/run_test.go` (update allHandlers call sites if any)
- `network/cmd/chainbench-net/e2e_test.go` (+ E2E for node.stop)

---

## Task 1: `drivers/local/driver.go` — Driver.Run generic subprocess runner

**Files:**
- Create: `network/internal/drivers/local/doc.go`
- Create: `network/internal/drivers/local/driver.go`
- Create: `network/internal/drivers/local/driver_test.go`

- [ ] **Step 1.1: Write failing test**

Create `network/internal/drivers/local/driver_test.go`:

```go
package local

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// fakeExecFn returns an exec factory that spawns `sh -c "<script>"` to
// reliably control stdout/stderr/exit code in tests without needing
// OS-specific binaries.
func fakeExecFn(script string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Ignore real name/args; run the script via sh -c so tests control
		// stdout/stderr/exit precisely. Capture the real args on the wrapped
		// cmd via Env for assertion if needed.
		full := append([]string{name}, args...)
		c := exec.CommandContext(ctx, "sh", "-c", script)
		c.Env = append(c.Env, "FAKE_ARGS="+strings.Join(full, "|"))
		return c
	}
}

func TestDriver_Run_Success(t *testing.T) {
	d := NewDriverWithExec("/nowhere", fakeExecFn(`echo "hello"; exit 0`))
	res, err := d.Run(context.Background(), "anything")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit: got %d, want 0", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "hello") {
		t.Errorf("stdout: got %q", res.Stdout)
	}
	if res.Duration <= 0 {
		t.Errorf("duration: got %v, want > 0", res.Duration)
	}
}

func TestDriver_Run_NonZeroExit_NoError(t *testing.T) {
	d := NewDriverWithExec("/nowhere", fakeExecFn(`echo "boom" >&2; exit 3`))
	res, err := d.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("run returned error for non-zero exit: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("exit: got %d, want 3", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "boom") {
		t.Errorf("stderr: got %q", res.Stderr)
	}
}

func TestDriver_Run_ContextCanceled_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before exec
	d := NewDriverWithExec("/nowhere", fakeExecFn(`sleep 5; exit 0`))
	_, err := d.Run(ctx, "x")
	if err == nil {
		t.Fatal("expected error for canceled ctx")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("err: got %v, want canceled-related", err)
	}
}

func TestDriver_Run_StreamsLinesToSlog(t *testing.T) {
	// Capture slog output by redirecting the default logger to a buffer.
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	d := NewDriverWithExec("/nowhere", fakeExecFn(`echo "line1"; echo "line2" >&2; exit 0`))
	if _, err := d.Run(context.Background(), "x"); err != nil {
		t.Fatalf("run: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "line1") {
		t.Errorf("slog missing stdout line: %q", out)
	}
	if !strings.Contains(out, "line2") {
		t.Errorf("slog missing stderr line: %q", out)
	}
}

// Ensure NewDriver (prod constructor) compiles and wires exec.CommandContext.
func TestNewDriver_ProdConstructor(t *testing.T) {
	d := NewDriver("/nowhere")
	if d == nil {
		t.Fatal("nil driver")
	}
	// Calling Run with a missing chainbench.sh should return an error with
	// no usable exec, but NOT panic. We use a short-lived ctx so it doesn't
	// hang if somehow things work.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := d.Run(ctx, "help")
	if err == nil {
		t.Fatal("expected error for missing chainbench.sh at /nowhere")
	}
}
```

- [ ] **Step 1.2: Run test — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/drivers/local/... 2>&1 | tail -5
```
Expected: `no Go files` or `undefined: NewDriver` / `undefined: NewDriverWithExec` / `undefined: RunResult`.

- [ ] **Step 1.3: Write `doc.go`**

Create `network/internal/drivers/local/doc.go`:

```go
// Package local implements a subprocess runner that executes the chainbench
// CLI (chainbench.sh) on the local machine. It is a pure process runner —
// stdout/stderr are streamed to the structured logger, and RunResult exposes
// the captured buffers plus exit code for the caller's inspection.
//
// This package does not emit bus events. Higher layers (command handlers)
// translate run outcomes into semantic events like node.stopped.
package local
```

- [ ] **Step 1.4: Write `driver.go`**

Create `network/internal/drivers/local/driver.go`:

```go
package local

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Driver executes chainbench CLI subcommands as subprocesses.
type Driver struct {
	chainbenchDir string
	exec          func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// NewDriver creates a Driver that invokes chainbench.sh under chainbenchDir
// using the stdlib exec.CommandContext.
func NewDriver(chainbenchDir string) *Driver {
	return NewDriverWithExec(chainbenchDir, exec.CommandContext)
}

// NewDriverWithExec is the testable constructor allowing the exec factory
// to be replaced with a fake.
func NewDriverWithExec(chainbenchDir string,
	execFn func(ctx context.Context, name string, args ...string) *exec.Cmd) *Driver {
	return &Driver{chainbenchDir: chainbenchDir, exec: execFn}
}

// RunResult captures a single subprocess invocation outcome.
type RunResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// Run executes `<chainbenchDir>/chainbench.sh <args...>`. Each stdout line
// is logged via slog.Info; each stderr line via slog.Warn. The full text
// is also captured in the returned RunResult.Stdout / Stderr.
//
// A non-zero exit code does NOT produce a returned error — callers inspect
// RunResult.ExitCode. Start / IO / context errors DO return non-nil.
func (d *Driver) Run(ctx context.Context, args ...string) (*RunResult, error) {
	script := filepath.Join(d.chainbenchDir, "chainbench.sh")
	cmd := d.exec(ctx, script, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("local: stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("local: stderr pipe: %w", err)
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("local: start %s: %w", script, err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go streamAndCapture(&wg, stdoutPipe, &stdoutBuf, slog.LevelInfo, "subprocess stdout")
	go streamAndCapture(&wg, stderrPipe, &stderrBuf, slog.LevelWarn, "subprocess stderr")
	wg.Wait()

	waitErr := cmd.Wait()
	duration := time.Since(start)

	exitCode := 0
	if waitErr != nil {
		if ee, ok := waitErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			return nil, fmt.Errorf("local: wait: %w", waitErr)
		}
	}

	return &RunResult{
		ExitCode: exitCode,
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: duration,
	}, nil
}

// streamAndCapture reads r line-by-line, logs each line at level with msg,
// and appends to buf. Signals done on wg.
func streamAndCapture(wg *sync.WaitGroup, r io.Reader, buf *bytes.Buffer, level slog.Level, msg string) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	// Allow larger lines than default (64KB) — node logs can be verbose.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteByte('\n')
		slog.Log(context.Background(), level, msg, "line", line)
	}
}
```

- [ ] **Step 1.5: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/drivers/local/... -v
```
Expected: 5 tests PASS. Note: `TestDriver_Run_ContextCanceled_ReturnsError` relies on `cancel()` before `exec.Start()` returning an error from `cmd.Start()`. If the test fails on darwin because `cancel()` races `Start()`, the pipes created before `Start()` will cause Start to fail with `context canceled`. The assertion tolerates either `context.Canceled` or an error string containing `"canceled"`.

- [ ] **Step 1.6: Race + vet + fmt**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./internal/drivers/local/... && go vet ./... && gofmt -l internal/drivers/local/
```
Expected: all clean.

- [ ] **Step 1.7: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/drivers/local/doc.go network/internal/drivers/local/driver.go network/internal/drivers/local/driver_test.go
git commit -m "network: add local driver subprocess runner"
```

---

## Task 2: `drivers/local/stop.go` — StopNode convenience method

**Files:**
- Create: `network/internal/drivers/local/stop.go`
- Create: `network/internal/drivers/local/stop_test.go`

- [ ] **Step 2.1: Write failing test**

Create `network/internal/drivers/local/stop_test.go`:

```go
package local

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

// recordingExecFn returns a factory that stores the args it received
// for later assertion.
func recordingExecFn(capturedArgs *[]string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Record full argv: name (chainbench.sh path) then args.
		full := append([]string{name}, args...)
		*capturedArgs = append([]string(nil), full...)
		// Still need a runnable command — echo the args to stdout, exit 0.
		return exec.CommandContext(ctx, "sh", "-c", `echo "stopped"`)
	}
}

func TestStopNode_CallsCorrectArgs(t *testing.T) {
	var got []string
	d := NewDriverWithExec("/opt/chainbench", recordingExecFn(&got))
	res, err := d.StopNode(context.Background(), "3")
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit: got %d", res.ExitCode)
	}
	want := []string{"/opt/chainbench/chainbench.sh", "node", "stop", "3"}
	if len(got) != len(want) {
		t.Fatalf("argc: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
	if !strings.Contains(res.Stdout, "stopped") {
		t.Errorf("stdout: got %q", res.Stdout)
	}
}

func TestStopNode_EmptyNodeNum_StillExecs(t *testing.T) {
	// StopNode does not validate input — that's the handler's job. Empty
	// string is passed through to the subprocess.
	var got []string
	d := NewDriverWithExec("/x", recordingExecFn(&got))
	_, err := d.StopNode(context.Background(), "")
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if len(got) != 4 || got[3] != "" {
		t.Errorf("expected empty 4th arg, got %v", got)
	}
}
```

- [ ] **Step 2.2: Run test — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/drivers/local/... -run TestStopNode 2>&1 | tail -5
```
Expected: FAIL — `undefined: StopNode` (the method on Driver).

Wait — `d.StopNode(...)` is a method call. The error will be `d.StopNode undefined (type *Driver has no field or method StopNode)`.

- [ ] **Step 2.3: Write `stop.go`**

Create `network/internal/drivers/local/stop.go`:

```go
package local

import "context"

// StopNode invokes `chainbench.sh node stop <nodeNum>`. nodeNum is the
// numeric pids.json key ("1", "2", ...). This is a thin wrapper over Run
// with a fixed argv shape — it performs no input validation; callers are
// responsible for checking the node exists before invoking.
func (d *Driver) StopNode(ctx context.Context, nodeNum string) (*RunResult, error) {
	return d.Run(ctx, "node", "stop", nodeNum)
}
```

- [ ] **Step 2.4: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/drivers/local/... -v -run TestStopNode
```
Expected: 2 tests PASS.

- [ ] **Step 2.5: Full local package + race + coverage**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race -cover ./internal/drivers/local/...
```
Expected: PASS, coverage ≥ 85%.

- [ ] **Step 2.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/drivers/local/stop.go network/internal/drivers/local/stop_test.go
git commit -m "network: add local driver StopNode wrapper"
```

---

## Task 3: `chainbench-stub.sh` testdata fixture

**Files:**
- Create: `network/cmd/chainbench-net/testdata/chainbench-stub.sh` (executable)

- [ ] **Step 3.1: Write the stub script**

Create `network/cmd/chainbench-net/testdata/chainbench-stub.sh`:

```bash
#!/usr/bin/env bash
# Fake chainbench dispatcher for Sprint 2b.2 tests.
# Supports: chainbench-stub.sh node stop <N>
# Exits 0 on success, 1 if node num is the literal "fail" sentinel,
# 2 on unknown subcommand.

set -u

subcmd="${1:-}"
action="${2:-}"

case "$subcmd $action" in
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
    echo "stub: unknown command: $*" >&2
    exit 2
    ;;
esac
```

- [ ] **Step 3.2: Mark executable**

```bash
chmod +x /Users/wm-it-22-00661/Work/github/tools/chainbench/network/cmd/chainbench-net/testdata/chainbench-stub.sh
ls -l /Users/wm-it-22-00661/Work/github/tools/chainbench/network/cmd/chainbench-net/testdata/chainbench-stub.sh
```
Expected: permissions include `x` (e.g., `-rwxr-xr-x`).

- [ ] **Step 3.3: Smoke the stub**

```bash
bash /Users/wm-it-22-00661/Work/github/tools/chainbench/network/cmd/chainbench-net/testdata/chainbench-stub.sh node stop 1
echo "exit=$?"
bash /Users/wm-it-22-00661/Work/github/tools/chainbench/network/cmd/chainbench-net/testdata/chainbench-stub.sh node stop fail
echo "exit=$?"
```
Expected:
- First: `stub: node 1 stopped`, exit 0
- Second: `stub: forced failure for testing` (stderr), exit 1

- [ ] **Step 3.4: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/testdata/chainbench-stub.sh
git commit -m "network: add chainbench stub script for node.stop tests"
```

---

## Task 4: `handleNodeStop` + update `allHandlers` signature

**Files:**
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`
- Modify: `network/cmd/chainbench-net/run.go` (pass chainbenchDir)
- Modify: `network/cmd/chainbench-net/run_test.go` (update allHandlers call)

- [ ] **Step 4.1: Append failing tests to `handlers_test.go`**

Append to `network/cmd/chainbench-net/handlers_test.go` (after existing tests):

```go
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
```

Also add imports needed (top of file — merge with existing imports):
```go
import (
	// existing:
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	// new:
	"time"
	// existing:
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
)
```

(Only add `"time"` if not already imported.)

- [ ] **Step 4.2: Run — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -run 'TestHandleNodeStop|TestAllHandlers_IncludesNodeStop' 2>&1 | tail -10
```
Expected: FAIL — `undefined: newHandleNodeStop`, `allHandlers` arity mismatch (expects 1 arg).

- [ ] **Step 4.3: Update `handlers.go`**

Read current `network/cmd/chainbench-net/handlers.go` first, then:

1. Change `allHandlers(stateDir string)` → `allHandlers(stateDir, chainbenchDir string)`.
2. Add `"node.stop": newHandleNodeStop(stateDir, chainbenchDir)` to the returned map.
3. Add the `newHandleNodeStop` function at the end of the file.

Append this handler function to `handlers.go`:

```go
// newHandleNodeStop returns a Handler that stops a node by id via LocalDriver.
// Args: { "node_id": "nodeN" } where N is the numeric pids.json key.
// On success: emits a "node.stopped" bus event, returns {node_id, stopped:true}.
func newHandleNodeStop(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		var req struct {
			NodeID string `json:"node_id"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.NodeID == "" {
			return nil, NewInvalidArgs("args.node_id is required")
		}
		if !strings.HasPrefix(req.NodeID, "node") {
			return nil, NewInvalidArgs(fmt.Sprintf(`node_id must start with "node" prefix (got %q)`, req.NodeID))
		}
		nodeNum := strings.TrimPrefix(req.NodeID, "node")
		if nodeNum == "" {
			return nil, NewInvalidArgs("node_id missing numeric suffix")
		}

		// Verify the node exists in pids.json.
		net, err := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: "local"})
		if err != nil {
			return nil, NewUpstream("failed to load active state", err)
		}
		found := false
		for _, n := range net.Nodes {
			if n.Id == req.NodeID {
				found = true
				break
			}
		}
		if !found {
			return nil, NewInvalidArgs(fmt.Sprintf("node_id %q not found in active network", req.NodeID))
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
			Data: map[string]any{"node_id": req.NodeID, "reason": "manual"},
		})
		return map[string]any{"node_id": req.NodeID, "stopped": true}, nil
	}
}
```

Add required imports at top of `handlers.go` (merge with existing):
```go
import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/network/internal/drivers/local"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/state"
	"github.com/0xmhha/chainbench/network/internal/types"
)
```

Update `allHandlers` body:
```go
func allHandlers(stateDir, chainbenchDir string) map[string]Handler {
	return map[string]Handler{
		"network.load": newHandleNetworkLoad(stateDir),
		"node.stop":    newHandleNodeStop(stateDir, chainbenchDir),
	}
}
```

- [ ] **Step 4.4: Update `run.go`**

Read current `run.go` first. Replace the `RunE` body to read `CHAINBENCH_DIR`:

```go
RunE: func(cmd *cobra.Command, _ []string) error {
    stateDir := os.Getenv("CHAINBENCH_STATE_DIR")
    if stateDir == "" {
        stateDir = "state"
    }
    chainbenchDir := os.Getenv("CHAINBENCH_DIR")
    if chainbenchDir == "" {
        chainbenchDir = "."
    }
    return runOnce(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(),
        allHandlers(stateDir, chainbenchDir))
},
```

- [ ] **Step 4.5: Update `run_test.go` to pass chainbenchDir**

Read `run_test.go`. All existing tests that call `allHandlers(...)` with one arg need a second:

Search for `allHandlers(dir)` and replace with `allHandlers(dir, "/nowhere")`. The existing tests don't exercise node.stop, so a bogus chainbench dir is fine.

Similarly, search for `allHandlers("whatever")` in `handlers_test.go` (the existing TestAllHandlers_IncludesNetworkLoad test) and update to `allHandlers("whatever", "also-whatever")`.

- [ ] **Step 4.6: Run all cmd tests — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v 2>&1 | tail -30
```
Expected: all tests pass, including the 6 new node.stop tests.

- [ ] **Step 4.7: Race + vet + fmt**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./cmd/chainbench-net/... && go vet ./... && gofmt -l cmd/chainbench-net/
```
Expected: all clean.

- [ ] **Step 4.8: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go network/cmd/chainbench-net/run.go network/cmd/chainbench-net/run_test.go
git commit -m "network: add node.stop handler using local driver"
```

---

## Task 5: E2E test via root command

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`

- [ ] **Step 5.1: Append failing test**

Append to `network/cmd/chainbench-net/e2e_test.go`:

```go
func TestE2E_NodeStop_ViaRootCommand(t *testing.T) {
	// Lay out state dir + chainbench dir (with our stub renamed to chainbench.sh).
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
	stub, err := os.ReadFile(filepath.Join("testdata", "chainbench-stub.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), stub, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_DIR", chainbenchDir)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"node.stop","args":{"node_id":"node1"}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Collect and classify lines.
	var sawEvent, sawResultOK bool
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		msg, derr := wire.DecodeMessage(line)
		if derr != nil {
			t.Fatalf("decode %q: %v", line, derr)
		}
		switch m := msg.(type) {
		case wire.EventMessage:
			if string(m.Name) == "node.stopped" {
				sawEvent = true
			}
		case wire.ResultMessage:
			if m.Ok {
				sawResultOK = true
			}
		}
		// Also validate each line against the event schema.
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
	}
	if !sawEvent {
		t.Error("expected a node.stopped event")
	}
	if !sawResultOK {
		t.Error("expected a successful result terminator")
	}
}
```

Add imports if missing: `"bufio"`, `"os"`, `"path/filepath"`, `"strings"`.

- [ ] **Step 5.2: Run — expect pass immediately**

(No RED step needed — this E2E test exercises the already-implemented handler from Task 4.)

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v -run TestE2E_NodeStop
```
Expected: PASS.

- [ ] **Step 5.3: Full module verification**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network
go build ./...
go test -race ./...
go vet ./...
gofmt -l .
go build -tags tools ./...
```
Expected: all clean.

- [ ] **Step 5.4: Coverage check**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -cover ./internal/drivers/local/... ./cmd/chainbench-net/...
```
Expected: `drivers/local` ≥ 85%, `cmd/chainbench-net` ≥ 80%.

- [ ] **Step 5.5: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/e2e_test.go
git commit -m "network: add end-to-end test for node.stop command"
```

---

## Final verification

- [ ] **Commit list**

```bash
git log --oneline 9e6a17d..HEAD
```
Expected 5 commits:
1. `network: add local driver subprocess runner`
2. `network: add local driver StopNode wrapper`
3. `network: add chainbench stub script for node.stop tests`
4. `network: add node.stop handler using local driver`
5. `network: add end-to-end test for node.stop command`

- [ ] **Manual smoke with real chainbench (optional)**

If a real local chain is running:
```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
go build -o /tmp/chainbench-net-2b2 ./network/cmd/chainbench-net
CHAINBENCH_DIR=. CHAINBENCH_STATE_DIR=state \
  /tmp/chainbench-net-2b2 run <<< '{"command":"node.stop","args":{"node_id":"node3"}}'
echo "exit=$?"
rm /tmp/chainbench-net-2b2
```

---

## Out of scope (explicit)

- `node.start` / `node.restart` / `node.tail_log` (future sprints)
- `node.rpc` (Sprint 2b.3 or when remote driver lands)
- Remote / SSH drivers
- Signal-based graceful shutdown of in-flight commands
- Signer / keystore (Sprint 4)
- `chainbench.sh` absolute path validation (assume env-provided path is trusted)
