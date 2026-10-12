package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// sleeper starts a long-lived process that stands in for a launched node, and
// kills it when the test ends unless something else already did.
func sleeper(t *testing.T) *exec.Cmd {
	t.Helper()
	p := exec.Command("sleep", "30")
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = p.Process.Kill()
		_ = p.Wait()
	})
	return p
}

// fakeNodeBinary answers `init` and otherwise runs like a node until killed.
func fakeNodeBinary(t *testing.T, dir, name string) string {
	t.Helper()
	bin := filepath.Join(dir, name)
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nif [ \"$1\" = \"init\" ]; then exit 0; fi\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

// useInputs points the written workspace at a real binary and genesis, since
// writeWorkspace records neither.
func useInputs(t *testing.T, dir, bin, genesis string) {
	t.Helper()
	comp, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	var st chainsetup.State
	if _, err = comp.Load(&st); err != nil {
		t.Fatal(err)
	}
	st.Binary, st.GenesisPath = bin, genesis
	if err = comp.Save(st); err != nil {
		t.Fatal(err)
	}
}

func recorded(t *testing.T, dir string, index int) node.Record {
	t.Helper()
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range ws.State().Nodes {
		if n.Index == index {
			return n
		}
	}
	t.Fatalf("node%d is not recorded", index)
	return node.Record{}
}

func alive(pid int) bool { return syscall.Kill(pid, 0) == nil }

// TestNodeSwapCmd_RelaunchesOneNodeOnANewBinary: the CLI swaps the binary of
// the selected node only. Its old process ends, it runs again from the new
// binary, and the other node keeps its process.
func TestNodeSwapCmd_RelaunchesOneNodeOnANewBinary(t *testing.T) {
	dir := t.TempDir()
	one, two := sleeper(t), sleeper(t)
	n1 := wsNode(dir, 1, "bp", 8501, one.Process.Pid)
	n1.Args = []string{"--datadir", n1.DataDir}
	n2 := wsNode(dir, 2, "bp", 8502, two.Process.Pid)
	n2.Args = []string{"--datadir", n2.DataDir}
	writeWorkspace(t, dir, "stablenet", n1, n2)
	bin := fakeNodeBinary(t, dir, "gstable-next")

	out, err := run(t, "node", "swap", "--workspace-dir", dir, "--index", "1", "--binary", bin)
	if err != nil {
		t.Fatalf("node swap: %v\n%s", err, out)
	}
	after := recorded(t, dir, 1)
	t.Cleanup(func() { _ = syscall.Kill(after.PID, syscall.SIGKILL) })
	if !strings.Contains(out, "swapped node1") {
		t.Errorf("no swap confirmation:\n%s", out)
	}
	if after.PID == 0 || after.PID == one.Process.Pid || !alive(after.PID) {
		t.Errorf("node1 pid after swap = %d (was %d), want a new live process", after.PID, one.Process.Pid)
	}
	if err := one.Wait(); err == nil {
		t.Error("node1's old process was not stopped")
	}
	if !alive(two.Process.Pid) || recorded(t, dir, 2).PID != two.Process.Pid {
		t.Error("node2 lost its process during node1's swap")
	}
}

// TestNodeResetCmd_RefusesAProducer: a producer is refused before anything is
// stopped, so the network keeps its sealer.
func TestNodeResetCmd_RefusesAProducer(t *testing.T) {
	dir := t.TempDir()
	bp := sleeper(t)
	writeWorkspace(t, dir, "stablenet", wsNode(dir, 1, "bp", 8501, bp.Process.Pid))

	out, err := run(t, "node", "reset", "--workspace-dir", dir, "--index", "1")
	if err == nil || !strings.Contains(err.Error(), "block producer") {
		t.Fatalf("reset of a producer gave %v, want a refusal naming the producer\n%s", err, out)
	}
	if !alive(bp.Process.Pid) || recorded(t, dir, 1).PID != bp.Process.Pid {
		t.Error("the producer was stopped before the refusal")
	}
}

// TestNodeResetCmd_LeavesAnEndpointStopped: an endpoint is stopped, its chain
// data is removed and it stays down for a later start.
func TestNodeResetCmd_LeavesAnEndpointStopped(t *testing.T) {
	dir := t.TempDir()
	bp, en := sleeper(t), sleeper(t)
	n2 := wsNode(dir, 2, "en", 8502, en.Process.Pid)
	n2.Args = []string{"--datadir", n2.DataDir}
	writeWorkspace(t, dir, "stablenet", wsNode(dir, 1, "bp", 8501, bp.Process.Pid), n2)
	genesis := filepath.Join(dir, "genesis.json")
	if err := os.WriteFile(genesis, []byte(`{"config":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	useInputs(t, dir, fakeNodeBinary(t, dir, "gstable"), genesis)
	if err := os.MkdirAll(n2.DataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(n2.DataDir, "chaindata-marker")
	if err := os.WriteFile(marker, []byte("blocks"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, "node", "reset", "--workspace-dir", dir, "--index", "2")
	if err != nil {
		t.Fatalf("node reset: %v\n%s", err, out)
	}
	if !strings.Contains(out, "node2 reset") {
		t.Errorf("no reset confirmation:\n%s", out)
	}
	if err := en.Wait(); err == nil {
		t.Error("the endpoint's process was not stopped")
	}
	if got := recorded(t, dir, 2).PID; got != 0 {
		t.Errorf("endpoint pid after reset = %d, want 0", got)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("the endpoint's chain data survived the reset: %v", err)
	}
	if !alive(bp.Process.Pid) {
		t.Error("the producer lost its process during the endpoint's reset")
	}
}
