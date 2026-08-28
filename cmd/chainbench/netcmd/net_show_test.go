package netcmd_test

import (
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// showWorkspace builds the smallest real workspace `net show` can read: only
// new+allocate run; nothing launches.
func showWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Validators: 2, Endpoints: 1}); err != nil {
		t.Fatal(err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestShow_AnswersBothDirections pins the lookup: by label, and the reverse
// question a port in a log line asks.
func TestShow_AnswersBothDirections(t *testing.T) {
	dir := showWorkspace(t)

	whole, err := run(t, "net", "show", "--workspace-dir", dir)
	if err != nil {
		t.Fatalf("show: %v\n%s", err, whole)
	}
	if !strings.Contains(whole, "3 node(s)") || !strings.Contains(whole, "2 bp") {
		t.Errorf("whole map lost the composition:\n%s", whole)
	}

	byLabel, err := run(t, "net", "show", "--workspace-dir", dir, "--label", "en1")
	if err != nil {
		t.Fatalf("show --label: %v\n%s", err, byLabel)
	}
	if !strings.Contains(byLabel, "node3") {
		t.Errorf("alias en1 did not resolve to its identity:\n%s", byLabel)
	}

	// Reverse lookup: take node1's http port from the whole map's row.
	fields := strings.Fields(strings.Split(whole, "\n")[1])
	port := fields[6]
	byPort, err := run(t, "net", "show", "--workspace-dir", dir, "--port", port)
	if err != nil {
		t.Fatalf("show --port: %v\n%s", err, byPort)
	}
	if !strings.Contains(byPort, "node1") {
		t.Errorf("port %s did not resolve to node1:\n%s", port, byPort)
	}
}

// TestShow_RequiresAWorkspace: show reads a composed map; without one the
// answer is a refusal, not an empty table.

// TestShow_RequiresAWorkspace: show reads a composed map; without one the
// answer is a refusal, not an empty table.
func TestShow_RequiresAWorkspace(t *testing.T) {
	if _, err := run(t, "net", "show", "--workspace-dir", t.TempDir()); err == nil {
		t.Fatal("showed a map for an empty directory")
	}
}

// TestPool_CountsUsedSlots pins the arithmetic an operator otherwise does by
// hand: capacity from the pool, used from the workspace.
