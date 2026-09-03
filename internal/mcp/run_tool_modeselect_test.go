package mcp_test

import (
	"strings"
	"testing"
)

// TestRunTool_ModeSelection pins that chainbench_run picks its mode the way the
// CLI `run` does: rpc selects attach (which requires a chain), and either mode
// requires specs — so the two surfaces validate the same way (E9).
func TestRunTool_ModeSelection(t *testing.T) {
	s := newServer()

	// Either mode needs specs.
	if text, isErr := callText(t, s, "chainbench_run", map[string]any{"rpc": []any{"http://127.0.0.1:1"}, "chain": "stablenet"}); !isErr || !strings.Contains(text, "spec") {
		t.Errorf("run without specs: err=%v text=%q, want a spec-required error", isErr, text)
	}

	// rpc present selects attach, which requires a chain.
	if text, isErr := callText(t, s, "chainbench_run", map[string]any{"rpc": []any{"http://127.0.0.1:1"}, "spec": `{"schemaVersion":"2","kind":"case","id":"c","env":{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet"},"steps":[]}`}); !isErr || !strings.Contains(text, "chain") {
		t.Errorf("attach without chain: err=%v text=%q, want a chain-required error", isErr, text)
	}
}
