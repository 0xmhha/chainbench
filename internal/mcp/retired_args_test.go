package mcp_test

import (
	"strings"
	"testing"
)

// TestCallTool_RejectsRetiredArgs: handlers read arguments permissively, so a
// renamed argument must be refused by name — silently dropping it would make
// credentials resolve from a different server-set file than the caller named.
func TestCallTool_RejectsRetiredArgs(t *testing.T) {
	s := newServer()
	text, isErr := callText(t, s, "chainbench_keyring_list", map[string]any{
		"serverConfig": "/ops/servers.yaml",
	})
	if !isErr {
		t.Fatalf("a retired argument was accepted: %q", text)
	}
	if !strings.Contains(text, "serverSet") {
		t.Errorf("error %q does not name the replacement argument", text)
	}
}
