package netmapcmd_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/netmapcmd"
	"github.com/0xmhha/chainbench/internal/netcompose"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// run executes a netmap command line the way an operator types it. The group
// is mounted on a bare root, so these tests exercise exactly what the real
// root command mounts, without depending on package main.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(netmapcmd.New())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

// TestPlan_ComputesWithoutComposing pins the group's reason to exist: the
// allocator answers as a question, no workspace, nothing written anywhere.
func TestPlan_ComputesWithoutComposing(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	out, err := run(t, "netmap", "plan", "--validators", "2", "--endpoints", "1")
	if err != nil {
		t.Fatalf("plan: %v\n%s", err, out)
	}
	for _, want := range []string{"node1", "node2", "node3", "2 bp", "1 en", "3 node(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan output lost %q:\n%s", want, out)
		}
	}
	// Nothing composed: the working directory stays empty.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("plan wrote files: %v", entries)
	}
}

// TestPlan_IsDeterministic pins the allocator's core promise through the CLI:
// the same inputs always place the same.
func TestPlan_IsDeterministic(t *testing.T) {
	first, err := run(t, "netmap", "plan", "--validators", "3")
	if err != nil {
		t.Fatal(err)
	}
	second, err := run(t, "netmap", "plan", "--validators", "3")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Errorf("same inputs placed differently:\n%s\nvs\n%s", first, second)
	}
}

// TestPlan_RefusesAProducerlessNetwork: a network where nothing seals never
// advances, and the refusal belongs here, not at chain start.
func TestPlan_RefusesAProducerlessNetwork(t *testing.T) {
	if _, err := run(t, "netmap", "plan", "--validators", "0", "--endpoints", "2"); err == nil {
		t.Fatal("planned a network with no validator")
	}
}

// TestPlan_ReadsTheInventory pins the placement source: name a server from an
// server-set file and the plan lands on its address, not the built-in pool.
func TestPlan_ReadsTheInventory(t *testing.T) {
	inv := filepath.Join(t.TempDir(), "server-set.yaml")
	body := "version: 2\n" +
		"pool:\n" +
		"  hosts: [{name: box1, addr: 10.9.9.9}]\n" +
		"  slots: 4\n" +
		"ssh: {user: dev, password: pw}\n" +
		"dataRoot: /data/cb\n"
	if err := os.WriteFile(inv, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "netmap", "plan", "--server-set", inv, "--server", "box1", "--validators", "2")
	if err != nil {
		t.Fatalf("plan over inventory: %v\n%s", err, out)
	}
	if !strings.Contains(out, "10.9.9.9") {
		t.Errorf("plan ignored the server set host:\n%s", out)
	}
	if strings.Contains(out, "pw") {
		t.Errorf("plan output carries a credential:\n%s", out)
	}
}

// composedWorkspace builds the smallest real workspace show can read. The
// fixture goes through netcompose because the map show reads is the one a
// composition records — but only new+allocate run; nothing launches.
func composedWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ws, err := netcompose.Open(dir, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(netcompose.NewOpts{Chain: "stablenet"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Allocate(netcompose.AllocateOpts{Validators: 2, Endpoints: 1}); err != nil {
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
	dir := composedWorkspace(t)

	whole, err := run(t, "netmap", "show", "--data-dir", dir)
	if err != nil {
		t.Fatalf("show: %v\n%s", err, whole)
	}
	if !strings.Contains(whole, "3 node(s)") || !strings.Contains(whole, "2 bp") {
		t.Errorf("whole map lost the composition:\n%s", whole)
	}

	byLabel, err := run(t, "netmap", "show", "--data-dir", dir, "--label", "en1")
	if err != nil {
		t.Fatalf("show --label: %v\n%s", err, byLabel)
	}
	if !strings.Contains(byLabel, "node3") {
		t.Errorf("alias en1 did not resolve to its identity:\n%s", byLabel)
	}

	// Reverse lookup: take node1's http port from the whole map's row.
	fields := strings.Fields(strings.Split(whole, "\n")[1])
	port := fields[6]
	byPort, err := run(t, "netmap", "show", "--data-dir", dir, "--port", port)
	if err != nil {
		t.Fatalf("show --port: %v\n%s", err, byPort)
	}
	if !strings.Contains(byPort, "node1") {
		t.Errorf("port %s did not resolve to node1:\n%s", port, byPort)
	}
}

// TestShow_RequiresAWorkspace: show reads a composed map; without one the
// answer is a refusal, not an empty table.
func TestShow_RequiresAWorkspace(t *testing.T) {
	if _, err := run(t, "netmap", "show", "--data-dir", t.TempDir()); err == nil {
		t.Fatal("showed a map for an empty directory")
	}
}

// TestPool_CountsUsedSlots pins the arithmetic an operator otherwise does by
// hand: capacity from the pool, used from the workspace.
func TestPool_CountsUsedSlots(t *testing.T) {
	dir := composedWorkspace(t)
	out, err := run(t, "netmap", "pool", "--data-dir", dir)
	if err != nil {
		t.Fatalf("pool: %v\n%s", err, out)
	}
	if !strings.Contains(out, "3 used") {
		t.Errorf("pool did not count the workspace's nodes:\n%s", out)
	}
}
