package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNodeStopCmd_RefusesAPIDThatIsNotTheNode: a recorded pid is not proof of
// whose process it is. When it belongs to something other than the node the
// record describes, `node stop` refuses rather than signal it, and the process
// keeps running.
func TestNodeStopCmd_RefusesAPIDThatIsNotTheNode(t *testing.T) {
	stranger := exec.Command("/bin/sleep", "30")
	if err := stranger.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stranger.Process.Kill(); _ = stranger.Wait() })
	dir := t.TempDir()
	n := wsNode(dir, 1, "bp", 8501, stranger.Process.Pid)
	n.Args = []string{"--datadir", n.DataDir}
	writeWorkspace(t, dir, "stablenet", n)

	_, err := run(t, "node", "stop", "--workspace-dir", dir, "--index", "1")
	if err == nil || !strings.Contains(err.Error(), "refus") {
		t.Fatalf("stopping a pid that runs something else gave %v, want a refusal", err)
	}
	if !alive(stranger.Process.Pid) {
		t.Fatal("the unrelated process was signalled")
	}
}

// TestNodeStopCmd_StopsTheRecordedProcess: when the live process is the one
// the record launched, the same command stops it.
func TestNodeStopCmd_StopsTheRecordedProcess(t *testing.T) {
	owned := exec.Command("/bin/sleep", "30")
	if err := owned.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owned.Process.Kill(); _ = owned.Wait() })
	dir := t.TempDir()
	n := wsNode(dir, 1, "bp", 8501, owned.Process.Pid)
	n.Args = []string{"30"}
	writeWorkspace(t, dir, "stablenet", n)
	useInputs(t, dir, "/bin/sleep", "")

	if out, err := run(t, "node", "stop", "--workspace-dir", dir, "--index", "1"); err != nil {
		t.Fatalf("node stop: %v\n%s", err, out)
	}
	if err := owned.Wait(); err == nil {
		t.Fatal("the recorded process was not stopped")
	}
}

// TestHardforkCmd_RefusesAPIDThatIsNotTheNode: a hardfork stops every node
// before relaunching it on the new binary, and a recorded pid running
// something else is refused rather than signalled.
func TestHardforkCmd_RefusesAPIDThatIsNotTheNode(t *testing.T) {
	stranger := exec.Command("/bin/sleep", "30")
	if err := stranger.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stranger.Process.Kill(); _ = stranger.Wait() })
	dir := t.TempDir()
	n := wsNode(dir, 1, "validator", 8501, stranger.Process.Pid)
	n.Args = []string{"--datadir", dir}
	writeWorkspace(t, dir, "wemix", n)
	bin := fakeNodeBinary(t, dir, "faketo")

	_, err := run(t, "hardfork", "--workspace-dir", dir, "--to-chain", "wbft", "--block", "100", "--to-binary", bin, "--dry-run=false")
	if err == nil || !strings.Contains(err.Error(), "refus") {
		t.Fatalf("a hardfork over a pid that runs something else gave %v, want a refusal", err)
	}
	if !alive(stranger.Process.Pid) {
		t.Fatal("the unrelated process was signalled")
	}
}

// TestControlCommandsTakeServerLogins: every command the Web runs against a
// recorded network takes the job's own logins apart from the recorded set.
func TestControlCommandsTakeServerLogins(t *testing.T) {
	logins := t.TempDir() + "/logins.yaml"
	for _, args := range [][]string{
		{"node", "start", "--index", "1"}, {"node", "stop", "--index", "1"}, {"node", "reset", "--index", "1"},
		{"node", "swap", "--index", "1", "--binary", "/bin/true"},
		{"chain", "restart", "--node", "1"}, {"chain", "stop"}, {"chain", "rm"},
	} {
		_, err := run(t, append(args, "--workspace-dir", t.TempDir(), "--server-logins", logins)...)
		if err != nil && strings.Contains(err.Error(), "unknown flag") {
			t.Errorf("%v: %v", args, err)
		}
	}
}

// TestChainObserveCmd_ReportsWhatEachNodeActuallyIs: `chain observe` reads the
// machine for every recorded node without changing anything: the recorded
// launch running, a pid now running something else, and a node with no pid.
func TestChainObserveCmd_ReportsWhatEachNodeActuallyIs(t *testing.T) {
	// Use our regular test executable instead of a system utility that may
	// be a symbolic alias. Its name also avoids unrelated sleep processes.
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ownedArgs := []string{"-test.run=^TestChainObserveProcessFixture$", "--", "owned"}
	strangerArgs := []string{"-test.run=^TestChainObserveProcessFixture$", "--", "stranger"}
	owned, stranger := exec.Command(binary, ownedArgs...), exec.Command(binary, strangerArgs...)
	for _, p := range []*exec.Cmd{owned, stranger} {
		p.Env = append(os.Environ(), "CHAINBENCH_OBSERVE_PROCESS_FIXTURE=1")
		if err := p.Start(); err != nil {
			t.Fatal(err)
		}
		p := p
		t.Cleanup(func() { _ = p.Process.Kill(); _ = p.Wait() })
	}
	dir := t.TempDir()
	n1 := wsNode(dir, 1, "bp", 8501, owned.Process.Pid)
	n1.Args = ownedArgs
	n2 := wsNode(dir, 2, "bp", 8502, stranger.Process.Pid)
	n2.Args = ownedArgs
	n3 := wsNode(dir, 3, "en", 8503, 0)
	n3.Args = []string{"-test.run=^TestChainObserveProcessFixture$", "--", "stopped"}
	writeWorkspace(t, dir, "stablenet", n1, n2, n3)
	useInputs(t, dir, binary, "")

	out, err := run(t, "chain", "observe", "--workspace-dir", dir, "--json")
	if err != nil {
		t.Fatalf("chain observe: %v\n%s", err, out)
	}
	var doc struct {
		Nodes []struct {
			Index      int    `json:"index"`
			State      string `json:"state"`
			PID        int    `json:"observedPid"`
			Executable string `json:"executable"`
		} `json:"nodes"`
	}
	if err = json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not a document: %v\n%s", err, out)
	}
	want := map[int]string{1: "running", 2: "ownership_mismatch", 3: "stopped"}
	if len(doc.Nodes) != 3 {
		t.Fatalf("nodes = %+v", doc.Nodes)
	}
	for _, n := range doc.Nodes {
		if n.State != want[n.Index] {
			t.Errorf("node%d = %s, want %s", n.Index, n.State, want[n.Index])
		}
		if n.Executable != "ready" {
			t.Errorf("node%d executable = %q, want ready for %s", n.Index, n.Executable, binary)
		}
	}
	if doc.Nodes[0].PID != owned.Process.Pid {
		t.Errorf("node1 observed pid = %d, want %d", doc.Nodes[0].PID, owned.Process.Pid)
	}
	if !alive(owned.Process.Pid) || !alive(stranger.Process.Pid) {
		t.Fatal("observing changed a process")
	}
}

// This subprocess supplies a stable argv without relying on host utilities.
func TestChainObserveProcessFixture(t *testing.T) {
	if os.Getenv("CHAINBENCH_OBSERVE_PROCESS_FIXTURE") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

// A symbolic deployment alias remains unavailable even when its target has
// execute permission. Fixing a runner fixture must not weaken this refusal.
func TestChainObserveCmd_ReportsSymlinkExecutableUnavailable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "owned-executable")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(binary, alias); err != nil {
		t.Fatal(err)
	}
	n := wsNode(dir, 1, "en", 8501, 0)
	n.Args = []string{"--datadir", n.DataDir}
	writeWorkspace(t, dir, "stablenet", n)
	useInputs(t, dir, alias, "")
	out, err := run(t, "chain", "observe", "--workspace-dir", dir, "--json")
	if err != nil {
		t.Fatalf("chain observe: %v\n%s", err, out)
	}
	var doc struct {
		Nodes []struct {
			Executable string `json:"executable"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Nodes) != 1 || doc.Nodes[0].Executable != "unavailable" {
		t.Fatalf("symbolic executable reported as ready: %s", out)
	}
	info, err := os.Lstat(alias)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("observation changed the alias: %v", err)
	}
}
