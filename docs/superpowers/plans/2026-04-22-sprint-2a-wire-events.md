# Sprint 2a Wire + Events Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `network/internal/wire/` and `network/internal/events/` Go packages as a pure library (no CLI entry changes). Each package fully unit-tested. All wire output cross-validated against `network/schema/event.json` via an integration test.

**Architecture:** Two packages with one-way dependency: `events → wire → types/schema`. `wire` contains IO primitives (stdin envelope decode, stdout NDJSON emit, stderr slog, stream decoder). `events` contains channel-based pub/sub that delegates stdout writes to `wire.Emitter`.

**Tech Stack:** Go 1.25, standard library (`encoding/json`, `io`, `log/slog`, `sync`, `time`), `github.com/santhosh-tekuri/jsonschema/v5` (already pinned for cross-validation test), `github.com/0xmhha/chainbench/network/internal/types` (generated).

**Spec reference:** `docs/superpowers/specs/2026-04-22-sprint-2a-wire-events-design.md`.

---

## File Structure

**Created in this plan:**
- `network/internal/wire/doc.go`
- `network/internal/wire/command.go`
- `network/internal/wire/command_test.go`
- `network/internal/wire/emitter.go`
- `network/internal/wire/emitter_test.go`
- `network/internal/wire/decoder.go`
- `network/internal/wire/decoder_test.go`
- `network/internal/wire/logger.go`
- `network/internal/wire/logger_test.go`
- `network/internal/wire/wire_schema_test.go`
- `network/internal/events/doc.go`
- `network/internal/events/bus.go`
- `network/internal/events/bus_test.go`

**Modified:** none.

---

## Task 1: `wire/command.go` — Command envelope decoder

**Files:**
- Create: `network/internal/wire/doc.go`
- Create: `network/internal/wire/command.go`
- Create: `network/internal/wire/command_test.go`

- [ ] **Step 1.1: Write failing test**

Create `network/internal/wire/command_test.go`:

```go
package wire

import (
	"bytes"
	"strings"
	"testing"
)

func TestDecodeCommand_ValidEnvelope(t *testing.T) {
	input := []byte(`{"command":"network.load","args":{"name":"my-local"}}`)
	cmd, err := DecodeCommand(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cmd == nil {
		t.Fatal("cmd is nil")
	}
	if string(cmd.Command) != "network.load" {
		t.Errorf("command: got %q, want %q", cmd.Command, "network.load")
	}
	if cmd.Args == nil {
		t.Fatal("args is nil")
	}
}

func TestDecodeCommand_MalformedJSON(t *testing.T) {
	input := []byte(`{"command":"network.load"`) // truncated
	if _, err := DecodeCommand(bytes.NewReader(input)); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDecodeCommand_UnknownField(t *testing.T) {
	input := []byte(`{"command":"network.load","args":{},"bogus":1}`)
	_, err := DecodeCommand(bytes.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unknown top-level field")
	}
	if !strings.Contains(err.Error(), "unknown field") && !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should mention unknown field: %v", err)
	}
}
```

- [ ] **Step 1.2: Run test — expect compile failure**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/wire/... 2>&1 | tail -5
```
Expected: FAIL — `package github.com/0xmhha/chainbench/network/internal/wire: no Go files` or `undefined: DecodeCommand`.

- [ ] **Step 1.3: Write package doc**

Create `network/internal/wire/doc.go`:

```go
// Package wire provides transport primitives for chainbench-net:
// command envelope decoding from stdin, NDJSON event/progress/result
// emission to stdout, structured logging to stderr, and a stream
// message decoder for subprocess consumers.
//
// wire is a pure library — it never manages process lifecycle or
// dispatches to drivers. Those concerns live in higher layers.
package wire
```

- [ ] **Step 1.4: Implement `command.go`**

Create `network/internal/wire/command.go`:

```go
package wire

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// DecodeCommand reads one JSON command envelope from r and returns
// a validated Command. Unknown top-level fields are rejected.
// Schema-level validation (enum membership, required fields) is
// already enforced by the generated types.Command.UnmarshalJSON.
func DecodeCommand(r io.Reader) (*types.Command, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var cmd types.Command
	if err := dec.Decode(&cmd); err != nil {
		return nil, fmt.Errorf("wire: command envelope: %w", err)
	}
	return &cmd, nil
}
```

- [ ] **Step 1.5: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/wire/... -v -run TestDecodeCommand
```
Expected: PASS — 3 subtests green.

- [ ] **Step 1.6: Build/vet/fmt check**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./... && go vet ./... && gofmt -l internal/wire/
```
Expected: all exit 0, `gofmt -l` empty.

- [ ] **Step 1.7: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/wire/doc.go network/internal/wire/command.go network/internal/wire/command_test.go
git commit -m "network: add wire command envelope decoder"
```

---

## Task 2: `wire/logger.go` — slog setup

**Files:**
- Create: `network/internal/wire/logger.go`
- Create: `network/internal/wire/logger_test.go`

- [ ] **Step 2.1: Write failing test**

Create `network/internal/wire/logger_test.go`:

```go
package wire

import (
	"bytes"
	"encoding/json"
	"log/slog"
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
```

- [ ] **Step 2.2: Run test — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/wire/... -run TestSetupLogger 2>&1 | tail -5
```
Expected: FAIL — `undefined: SetupLoggerTo`.

- [ ] **Step 2.3: Implement `logger.go`**

Create `network/internal/wire/logger.go`:

```go
package wire

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	envLogLevel = "CHAINBENCH_NET_LOG_LEVEL"
	envLogFile  = "CHAINBENCH_NET_LOG"
)

// SetupLogger configures a slog JSON handler. Writer defaults to os.Stderr,
// or to the file at CHAINBENCH_NET_LOG if that env var is a non-empty path.
// Level from CHAINBENCH_NET_LOG_LEVEL: debug|info|warn|error (default info).
func SetupLogger() *slog.Logger {
	var w io.Writer = os.Stderr
	if path := os.Getenv(envLogFile); path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			w = f
		}
		// on error, silently fall back to stderr — logging must not block startup
	}
	return SetupLoggerTo(w, parseLevel(os.Getenv(envLogLevel)))
}

// SetupLoggerTo is the testable variant with explicit writer and level.
// It also installs the returned logger as slog.Default for convenience.
func SetupLoggerTo(w io.Writer, level slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	logger := slog.New(h)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

- [ ] **Step 2.4: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/wire/... -v -run TestSetupLogger
```
Expected: PASS — 3 subtests green.

- [ ] **Step 2.5: Build/vet/fmt + full wire test**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./... && go vet ./... && gofmt -l internal/wire/ && go test ./internal/wire/...
```
Expected: all exit 0.

- [ ] **Step 2.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/wire/logger.go network/internal/wire/logger_test.go
git commit -m "network: add wire structured logger setup"
```

---

## Task 3: `wire/emitter.go` — Stdout NDJSON emitter (largest task)

**Files:**
- Create: `network/internal/wire/emitter.go`
- Create: `network/internal/wire/emitter_test.go`

- [ ] **Step 3.1: Write failing test**

Create `network/internal/wire/emitter_test.go`:

```go
package wire

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/network/internal/types"
)

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func decodeLine(t *testing.T, line []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		t.Fatalf("line not valid JSON: %v (%q)", err, line)
	}
	return m
}

func TestEmitter_EmitEvent_WritesValidNDJSON(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	e.clock = fixedClock(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))
	if err := e.EmitEvent(types.EventName("chain.block"), map[string]any{"height": 42}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	m := decodeLine(t, bytes.TrimSpace(buf.Bytes()))
	if m["type"] != "event" {
		t.Errorf("type: got %v, want event", m["type"])
	}
	if m["name"] != "chain.block" {
		t.Errorf("name: got %v", m["name"])
	}
	if m["ts"] != "2026-04-22T10:00:00Z" {
		t.Errorf("ts: got %v", m["ts"])
	}
	data, ok := m["data"].(map[string]any)
	if !ok || data["height"].(float64) != 42 {
		t.Errorf("data: got %v", m["data"])
	}
}

func TestEmitter_EmitProgress(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	if err := e.EmitProgress("init", 2, 4); err != nil {
		t.Fatalf("emit: %v", err)
	}
	m := decodeLine(t, bytes.TrimSpace(buf.Bytes()))
	if m["type"] != "progress" || m["step"] != "init" {
		t.Errorf("progress mismatch: %v", m)
	}
	if int(m["done"].(float64)) != 2 || int(m["total"].(float64)) != 4 {
		t.Errorf("done/total mismatch: %v", m)
	}
}

func TestEmitter_EmitResult_OK(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	if err := e.EmitResult(true, map[string]any{"blockNumber": "0x2a"}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	m := decodeLine(t, bytes.TrimSpace(buf.Bytes()))
	if m["type"] != "result" || m["ok"] != true {
		t.Errorf("result mismatch: %v", m)
	}
	if _, has := m["error"]; has {
		t.Errorf("ok:true result must not have error field: %v", m)
	}
}

func TestEmitter_EmitResultError(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	if err := e.EmitResultError(types.ResultErrorCode("NOT_SUPPORTED"), "no process cap"); err != nil {
		t.Fatalf("emit: %v", err)
	}
	m := decodeLine(t, bytes.TrimSpace(buf.Bytes()))
	if m["type"] != "result" || m["ok"] != false {
		t.Errorf("error result: %v", m)
	}
	errObj, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("error obj missing: %v", m)
	}
	if errObj["code"] != "NOT_SUPPORTED" || errObj["message"] != "no process cap" {
		t.Errorf("error fields: %v", errObj)
	}
	if _, has := m["data"]; has {
		t.Errorf("ok:false result must not have data field: %v", m)
	}
}

func TestEmitter_ResultTerminator(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	if err := e.EmitResult(true, nil); err != nil {
		t.Fatalf("first result: %v", err)
	}
	if err := e.EmitEvent(types.EventName("chain.block"), nil); !errors.Is(err, ErrStreamClosed) {
		t.Errorf("event after result: got %v, want ErrStreamClosed", err)
	}
	if err := e.EmitProgress("x", 0, 1); !errors.Is(err, ErrStreamClosed) {
		t.Errorf("progress after result: got %v, want ErrStreamClosed", err)
	}
	if err := e.EmitResult(true, nil); !errors.Is(err, ErrStreamClosed) {
		t.Errorf("second result: got %v, want ErrStreamClosed", err)
	}
}

func TestEmitter_ConcurrentEmits_AllLinesValidJSON(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	const goroutines = 100
	const perGoroutine = 10
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				_ = e.EmitEvent(types.EventName("chain.block"), map[string]any{"g": n, "j": j})
			}
		}(i)
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if got, want := len(lines), goroutines*perGoroutine; got != want {
		t.Fatalf("line count: got %d, want %d", got, want)
	}
	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %d not valid JSON: %v (%q)", i, err, line)
		}
	}
}
```

- [ ] **Step 3.2: Run test — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/wire/... -run TestEmitter 2>&1 | tail -10
```
Expected: FAIL — `undefined: NewEmitter`, `undefined: ErrStreamClosed`, etc.

- [ ] **Step 3.3: Implement `emitter.go`**

Create `network/internal/wire/emitter.go`:

```go
package wire

import (
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// ErrStreamClosed is returned when Emit* is called after the stream has been
// terminated by EmitResult or EmitResultError.
var ErrStreamClosed = errors.New("wire: stream already closed by result")

// Emitter writes NDJSON stream messages to an underlying io.Writer.
// Concurrent use by multiple goroutines is safe; each emitted line is
// serialized under an internal mutex.
//
// After EmitResult/EmitResultError, further Emit* calls return ErrStreamClosed.
type Emitter struct {
	mu     sync.Mutex
	enc    *json.Encoder
	clock  func() time.Time
	closed bool
}

// NewEmitter creates an Emitter that writes to w.
func NewEmitter(w io.Writer) *Emitter {
	return &Emitter{
		enc:   json.NewEncoder(w),
		clock: time.Now,
	}
}

func (e *Emitter) EmitEvent(name types.EventName, data map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ErrStreamClosed
	}
	msg := map[string]any{
		"type": "event",
		"name": string(name),
		"ts":   e.clock().UTC().Format(time.RFC3339),
	}
	if data != nil {
		msg["data"] = data
	} else {
		msg["data"] = map[string]any{}
	}
	return e.enc.Encode(msg)
}

func (e *Emitter) EmitProgress(step string, done, total int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ErrStreamClosed
	}
	msg := map[string]any{
		"type":  "progress",
		"step":  step,
		"done":  done,
		"total": total,
	}
	return e.enc.Encode(msg)
}

func (e *Emitter) EmitResult(ok bool, data map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ErrStreamClosed
	}
	if !ok {
		// Guard: callers must use EmitResultError for ok=false.
		return errors.New("wire: EmitResult requires ok=true; use EmitResultError for failures")
	}
	msg := map[string]any{
		"type": "result",
		"ok":   true,
	}
	if data != nil {
		msg["data"] = data
	}
	e.closed = true
	return e.enc.Encode(msg)
}

func (e *Emitter) EmitResultError(code types.ResultErrorCode, message string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ErrStreamClosed
	}
	msg := map[string]any{
		"type": "result",
		"ok":   false,
		"error": map[string]any{
			"code":    string(code),
			"message": message,
		},
	}
	e.closed = true
	return e.enc.Encode(msg)
}
```

- [ ] **Step 3.4: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/wire/... -v -run TestEmitter
```
Expected: PASS — 6 subtests green, including `TestEmitter_ConcurrentEmits_AllLinesValidJSON`.

- [ ] **Step 3.5: Race detector pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./internal/wire/...
```
Expected: exit 0, no `DATA RACE` warnings.

- [ ] **Step 3.6: Build/vet/fmt**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./... && go vet ./... && gofmt -l internal/wire/
```
Expected: exit 0, gofmt empty.

- [ ] **Step 3.7: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/wire/emitter.go network/internal/wire/emitter_test.go
git commit -m "network: add wire NDJSON emitter with terminator guard"
```

---

## Task 4: `wire/decoder.go` — StreamMessage tagged-union decoder

**Files:**
- Create: `network/internal/wire/decoder.go`
- Create: `network/internal/wire/decoder_test.go`

- [ ] **Step 4.1: Write failing test**

Create `network/internal/wire/decoder_test.go`:

```go
package wire

import (
	"strings"
	"testing"
)

func TestDecodeMessage_Event(t *testing.T) {
	line := []byte(`{"type":"event","name":"node.started","data":{"node_id":"node1"},"ts":"2026-04-22T10:00:00Z"}`)
	msg, err := DecodeMessage(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	em, ok := msg.(EventMessage)
	if !ok {
		t.Fatalf("type: got %T, want EventMessage", msg)
	}
	if string(em.Name) != "node.started" {
		t.Errorf("name: got %q", em.Name)
	}
}

func TestDecodeMessage_Progress(t *testing.T) {
	line := []byte(`{"type":"progress","step":"init","done":2,"total":4}`)
	msg, err := DecodeMessage(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	pm, ok := msg.(ProgressMessage)
	if !ok {
		t.Fatalf("type: got %T, want ProgressMessage", msg)
	}
	if pm.Step != "init" {
		t.Errorf("step: got %q", pm.Step)
	}
}

func TestDecodeMessage_ResultOK(t *testing.T) {
	line := []byte(`{"type":"result","ok":true,"data":{"blockNumber":"0x2a"}}`)
	msg, err := DecodeMessage(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	rm, ok := msg.(ResultMessage)
	if !ok {
		t.Fatalf("type: got %T, want ResultMessage", msg)
	}
	if rm.Ok != true {
		t.Errorf("ok: got %v", rm.Ok)
	}
}

func TestDecodeMessage_ResultError(t *testing.T) {
	line := []byte(`{"type":"result","ok":false,"error":{"code":"NOT_SUPPORTED","message":"no"}}`)
	msg, err := DecodeMessage(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	rm, ok := msg.(ResultMessage)
	if !ok {
		t.Fatalf("type: got %T, want ResultMessage", msg)
	}
	if rm.Ok != false {
		t.Errorf("ok: got %v", rm.Ok)
	}
	if rm.Error == nil || string(rm.Error.Code) != "NOT_SUPPORTED" {
		t.Errorf("error: got %+v", rm.Error)
	}
}

func TestDecodeMessage_UnknownType(t *testing.T) {
	line := []byte(`{"type":"bogus"}`)
	_, err := DecodeMessage(line)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), "bogus") && !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error should mention unknown type: %v", err)
	}
}

func TestDecodeMessage_MalformedJSON(t *testing.T) {
	line := []byte(`{"type":"event"`) // truncated
	if _, err := DecodeMessage(line); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDecodeMessage_RoundtripViaEmitter(t *testing.T) {
	// Emit via Emitter → decode via DecodeMessage → verify fields survive.
	var buf = new(stringBuf)
	e := NewEmitter(buf)
	if err := e.EmitEvent("chain.block", map[string]any{"height": 42}); err != nil {
		t.Fatal(err)
	}
	if err := e.EmitProgress("step1", 1, 3); err != nil {
		t.Fatal(err)
	}
	if err := e.EmitResult(true, map[string]any{"done": true}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(lines))
	}
	for i, line := range lines {
		msg, err := DecodeMessage([]byte(line))
		if err != nil {
			t.Fatalf("line %d decode: %v (%q)", i, err, line)
		}
		switch i {
		case 0:
			if _, ok := msg.(EventMessage); !ok {
				t.Errorf("line 0: got %T, want EventMessage", msg)
			}
		case 1:
			if _, ok := msg.(ProgressMessage); !ok {
				t.Errorf("line 1: got %T, want ProgressMessage", msg)
			}
		case 2:
			if _, ok := msg.(ResultMessage); !ok {
				t.Errorf("line 2: got %T, want ResultMessage", msg)
			}
		}
	}
}

// stringBuf is an io.Writer that captures output as a string,
// kept local to this test to avoid dependency on bytes package choices.
type stringBuf struct{ b strings.Builder }

func (s *stringBuf) Write(p []byte) (int, error) { return s.b.Write(p) }
func (s *stringBuf) String() string               { return s.b.String() }
```

- [ ] **Step 4.2: Run test — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/wire/... -run TestDecodeMessage 2>&1 | tail -10
```
Expected: FAIL — `undefined: DecodeMessage`, `undefined: EventMessage`, etc.

- [ ] **Step 4.3: Implement `decoder.go`**

Create `network/internal/wire/decoder.go`:

```go
package wire

import (
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// Message is a sealed interface implemented by EventMessage, ProgressMessage,
// and ResultMessage. Consumers use a type switch to handle each case.
type Message interface {
	isMessage()
}

// EventMessage wraps a decoded non-terminator event line.
type EventMessage struct{ types.Event }

// ProgressMessage wraps a decoded progress line.
type ProgressMessage struct{ types.Progress }

// ResultMessage wraps a decoded result (terminator) line.
type ResultMessage struct{ types.Result }

func (EventMessage) isMessage()    {}
func (ProgressMessage) isMessage() {}
func (ResultMessage) isMessage()   {}

// DecodeMessage parses one NDJSON line into a Message. Dispatches on the
// "type" discriminator field. Returns an error for unknown type or malformed JSON.
func DecodeMessage(line []byte) (Message, error) {
	var hdr struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &hdr); err != nil {
		return nil, fmt.Errorf("wire: decode header: %w", err)
	}
	switch hdr.Type {
	case "event":
		var ev types.Event
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, fmt.Errorf("wire: decode event: %w", err)
		}
		return EventMessage{ev}, nil
	case "progress":
		var p types.Progress
		if err := json.Unmarshal(line, &p); err != nil {
			return nil, fmt.Errorf("wire: decode progress: %w", err)
		}
		return ProgressMessage{p}, nil
	case "result":
		var r types.Result
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("wire: decode result: %w", err)
		}
		return ResultMessage{r}, nil
	default:
		return nil, fmt.Errorf("wire: unknown message type %q", hdr.Type)
	}
}
```

- [ ] **Step 4.4: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/wire/... -v -run TestDecodeMessage
```
Expected: PASS — 7 subtests green.

- [ ] **Step 4.5: Full wire package test + race**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./internal/wire/... && go vet ./... && gofmt -l internal/wire/
```
Expected: exit 0.

- [ ] **Step 4.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/wire/decoder.go network/internal/wire/decoder_test.go
git commit -m "network: add wire stream message decoder"
```

---

## Task 5: `wire/wire_schema_test.go` — Integration: output ↔ schema

**Files:**
- Create: `network/internal/wire/wire_schema_test.go`

- [ ] **Step 5.1: Write test (no prod code change)**

Create `network/internal/wire/wire_schema_test.go`:

```go
package wire_test

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
	"github.com/0xmhha/chainbench/network/schema"
)

// TestEmitter_OutputValidatesAgainstSchema ensures every line produced by
// wire.Emitter conforms to network/schema/event.json. This cross-check makes
// any drift between schema and wire emission a compile-time / test-time
// failure.
func TestEmitter_OutputValidatesAgainstSchema(t *testing.T) {
	var buf bytes.Buffer
	e := wire.NewEmitter(&buf)

	// Event (non-terminator)
	if err := e.EmitEvent(types.EventName("chain.block"), map[string]any{
		"height": 42,
		"hash":   "0xabc",
	}); err != nil {
		t.Fatalf("emit event: %v", err)
	}
	// Progress
	if err := e.EmitProgress("init", 2, 4); err != nil {
		t.Fatalf("emit progress: %v", err)
	}
	// Terminator (must be last; Emitter enforces).
	if err := e.EmitResult(true, map[string]any{"n": 1}); err != nil {
		t.Fatalf("emit result: %v", err)
	}

	scanner := bufio.NewScanner(&buf)
	var count int
	for scanner.Scan() {
		line := scanner.Bytes()
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("line %d fails schema: %v\nline: %s", count, err, line)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 lines, got %d", count)
	}

	_ = time.Second // keep import referenced if future needs; no-op
}

// TestEmitter_ResultErrorValidatesAgainstSchema covers the ok=false branch.
func TestEmitter_ResultErrorValidatesAgainstSchema(t *testing.T) {
	var buf bytes.Buffer
	e := wire.NewEmitter(&buf)
	if err := e.EmitResultError(types.ResultErrorCode("NOT_SUPPORTED"), "no process cap"); err != nil {
		t.Fatalf("emit: %v", err)
	}
	line := bytes.TrimSpace(buf.Bytes())
	if err := schema.ValidateBytes("event", line); err != nil {
		t.Fatalf("schema validation: %v\nline: %s", err, line)
	}
}
```

> Note: `time` import placeholder removed if Go complains; if lint flags the unused import, remove the `time.Second` no-op line and the import together.

- [ ] **Step 5.2: Run integration test**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/wire/... -v -run TestEmitter_.*ValidatesAgainstSchema
```
Expected: both tests PASS — every emitted line validates against `event.json`.

If validation fails, the mismatch between Emitter output and schema is a real bug — fix Emitter or schema (not the test).

- [ ] **Step 5.3: Full wire package test + race + coverage check**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race -cover ./internal/wire/...
```
Expected: PASS, coverage printed (aim ≥ 85%). If < 85%, add tests before committing.

- [ ] **Step 5.4: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/wire/wire_schema_test.go
git commit -m "network: cross-validate wire output against event schema"
```

---

## Task 6: `events/bus.go` — Pub/sub layer

**Files:**
- Create: `network/internal/events/doc.go`
- Create: `network/internal/events/bus.go`
- Create: `network/internal/events/bus_test.go`

- [ ] **Step 6.1: Write failing test**

Create `network/internal/events/bus_test.go`:

```go
package events

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

func newTestBus() (*Bus, *bytes.Buffer) {
	var buf bytes.Buffer
	emitter := wire.NewEmitter(&buf)
	bus := NewBus(emitter)
	return bus, &buf
}

func TestBus_PublishDeliversToSubscribers(t *testing.T) {
	bus, _ := newTestBus()
	defer bus.Close()
	sub1 := bus.Subscribe()
	sub2 := bus.Subscribe()
	ev := Event{Name: types.EventName("chain.block"), Data: map[string]any{"height": 42}}
	if err := bus.Publish(ev); err != nil {
		t.Fatalf("publish: %v", err)
	}
	for i, ch := range []<-chan Event{sub1, sub2} {
		select {
		case got := <-ch:
			if got.Name != ev.Name {
				t.Errorf("sub %d name: got %q, want %q", i, got.Name, ev.Name)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("sub %d timeout", i)
		}
	}
}

func TestBus_PublishAlsoEmitsToWire(t *testing.T) {
	bus, buf := newTestBus()
	defer bus.Close()
	ev := Event{Name: types.EventName("chain.block"), Data: map[string]any{"h": 1}}
	if err := bus.Publish(ev); err != nil {
		t.Fatalf("publish: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"type":"event"`) {
		t.Errorf("wire output missing type:event: %q", out)
	}
	if !strings.Contains(out, `"name":"chain.block"`) {
		t.Errorf("wire output missing name: %q", out)
	}
}

func TestBus_NonBlockingDropWhenSubscriberFull(t *testing.T) {
	bus, _ := newTestBus()
	defer bus.Close()
	sub := bus.Subscribe() // buffer = DefaultSubscriberBuffer (16)
	// Publish 100 events without draining; should not block.
	start := time.Now()
	for i := 0; i < 100; i++ {
		if err := bus.Publish(Event{Name: types.EventName("chain.block")}); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Errorf("publish blocked unexpectedly (%.2fms)", float64(elapsed.Microseconds())/1000.0)
	}
	// Consume what fits in the buffer to confirm drops occurred (received < published).
	received := 0
drain:
	for {
		select {
		case <-sub:
			received++
		case <-time.After(50 * time.Millisecond):
			break drain
		}
	}
	if received >= 100 {
		t.Errorf("expected drops, got all %d events", received)
	}
}

func TestBus_CloseRejectsSubsequentPublish(t *testing.T) {
	bus, _ := newTestBus()
	if err := bus.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	err := bus.Publish(Event{Name: types.EventName("chain.block")})
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("post-close publish: got %v, want ErrBusClosed", err)
	}
}

func TestBus_CloseClosesSubscriberChannels(t *testing.T) {
	bus, _ := newTestBus()
	sub := bus.Subscribe()
	if err := bus.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// Channel should close (recv returns zero value, ok=false).
	select {
	case _, ok := <-sub:
		if ok {
			t.Error("expected channel closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("subscriber channel not closed after bus.Close")
	}
}

func TestBus_ConcurrentPublish_RaceSafe(t *testing.T) {
	bus, _ := newTestBus()
	defer bus.Close()
	bus.Subscribe() // one passive sub

	const goroutines = 50
	const perGoroutine = 20
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				_ = bus.Publish(Event{Name: types.EventName("chain.block")})
			}
		}()
	}
	wg.Wait()
}
```

- [ ] **Step 6.2: Run test — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/events/... 2>&1 | tail -5
```
Expected: FAIL — `no Go files` or `undefined: NewBus`.

- [ ] **Step 6.3: Write package doc**

Create `network/internal/events/doc.go`:

```go
// Package events provides an in-process pub/sub layer built on top of
// wire.Emitter. Publishers call Bus.Publish; subscribers receive events via
// buffered channels obtained from Bus.Subscribe. The Bus also delegates each
// published event to the underlying wire.Emitter so it is written as NDJSON
// on the process stdout.
//
// Publish is non-blocking: if a subscriber's channel buffer is full, the
// event is dropped for that subscriber. This prevents a slow consumer from
// stalling the publisher.
package events
```

- [ ] **Step 6.4: Implement `bus.go`**

Create `network/internal/events/bus.go`:

```go
package events

import (
	"errors"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

// ErrBusClosed is returned by Publish after Close has been called.
var ErrBusClosed = errors.New("events: bus closed")

// DefaultSubscriberBuffer is the channel capacity for each Subscribe() result.
const DefaultSubscriberBuffer = 16

// Event is the in-process representation of a published event.
// The corresponding wire-level message is emitted by Publish via the
// bound wire.Emitter.
type Event struct {
	Name types.EventName
	Data map[string]any
	TS   time.Time
}

// Bus is a non-blocking pub/sub that also mirrors events to a wire.Emitter.
type Bus struct {
	mu      sync.RWMutex
	subs    []chan Event
	emitter *wire.Emitter
	clock   func() time.Time
	closed  bool
}

// NewBus creates a Bus that delegates wire emission to emitter.
func NewBus(emitter *wire.Emitter) *Bus {
	return &Bus{
		emitter: emitter,
		clock:   time.Now,
	}
}

// Publish fans out ev to in-process subscribers (non-blocking; dropped for any
// subscriber whose buffer is full) and writes it to the bound wire.Emitter.
// Returns ErrBusClosed after Close().
func (b *Bus) Publish(ev Event) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}
	// Fan-out under read lock (subs slice only mutated under write lock).
	for _, ch := range b.subs {
		select {
		case ch <- ev:
		default:
			// drop: subscriber is slow or not draining
		}
	}
	b.mu.RUnlock()

	// Emit outside the lock — wire.Emitter has its own mutex.
	return b.emitter.EmitEvent(ev.Name, ev.Data)
}

// Subscribe returns a new buffered channel that will receive published events
// until Close is called.
func (b *Bus) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, DefaultSubscriberBuffer)
	b.subs = append(b.subs, ch)
	return ch
}

// Close closes all subscriber channels and rejects subsequent Publish calls.
// Safe to call multiple times.
func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	for _, ch := range b.subs {
		close(ch)
	}
	b.subs = nil
	return nil
}
```

- [ ] **Step 6.5: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/events/... -v
```
Expected: PASS — 6 subtests green.

- [ ] **Step 6.6: Race + coverage**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race -cover ./internal/events/...
```
Expected: PASS, coverage ≥ 80%.

- [ ] **Step 6.7: Full module green**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./... && go test -race ./... && go vet ./... && gofmt -l .
```
Expected: all pass, gofmt empty.

- [ ] **Step 6.8: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/events/
git commit -m "network: add events Bus with non-blocking pub/sub"
```

---

## Final verification

- [ ] **Module-wide green**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network
go build ./... && go test -race ./... && go vet ./... && gofmt -l .
go build -tags tools ./...
```
Expected: all exit 0, gofmt empty.

- [ ] **Coverage summary**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network
go test -coverprofile=/tmp/cov.out ./internal/wire/... ./internal/events/...
go tool cover -func=/tmp/cov.out | tail -5
```
Expected: `wire` ≥ 85%, `events` ≥ 80% overall.

- [ ] **Commit list**

```bash
git log --oneline a828799..HEAD
```
Expected 6 commits, one per task:
1. `network: add wire command envelope decoder`
2. `network: add wire structured logger setup`
3. `network: add wire NDJSON emitter with terminator guard`
4. `network: add wire stream message decoder`
5. `network: cross-validate wire output against event schema`
6. `network: add events Bus with non-blocking pub/sub`

---

## Out of scope (explicit)

- No edits to `network/cmd/chainbench-net/main.go`
- No subprocess management / driver code
- No bash client
- No changes to existing schema files, types, or inventory scripts
