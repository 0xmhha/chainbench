package main

import (
	"os/exec"
	"strings"
	"testing"
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
