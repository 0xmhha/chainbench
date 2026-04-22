package wire_test

import (
	"bufio"
	"bytes"
	"testing"

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
