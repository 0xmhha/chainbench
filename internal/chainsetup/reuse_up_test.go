package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/process"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
)

// reuseWorkspaceConfig writes an environment file whose execution.chain is
// reuse-if-matching, rooted at the given data root.
func reuseWorkspaceConfig(t *testing.T, dir, dataRoot string) string {
	t.Helper()
	p := filepath.Join(dir, "workspace-config.yaml")
	body := "version: 1\ndataRoot: " + dataRoot + "\n" +
		"paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}\n" +
		"control: {artifactRoot: ~/.chainbench}\ninputs: {mode: generated}\nexecution: {chain: reuse-if-matching}\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestNetUp_ReuseRefusalLeavesTheRunningCompositionUntouched is MON-009.
//
// The genesis and config steps write to the target and re-record the input
// hashes. While the reuse verdict was computed after them, a refusal had already
// replaced the running network's genesis and configs — it announced that nothing
// could be reconciled while the files the live nodes were launched from had been
// swapped underneath. The verdict now runs before the first write, so this test
// reads the actual bytes back after a refusal.
func TestNetUp_ReuseRefusalLeavesTheRunningCompositionUntouched(t *testing.T) {
	dir := t.TempDir()
	dataRoot := t.TempDir()
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	wc := reuseWorkspaceConfig(t, dir, dataRoot)
	stub := initStub{stubDriver: &stubDriver{}}
	deps := chainsetup.Deps{
		Clock:  fixedClock(),
		Driver: func() (process.Driver, error) { return stub, nil },
	}
	// A stand-in binary: start checks the path exists, and this test never runs it.
	binary := filepath.Join(dir, "gstable")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	up := chainsetup.NetUpIn{
		DataDir: dir, Stage: chainsetup.UpStart,
		Chain: "stablenet", KeysDir: keysAbs, Binary: binary,
		Validators: 2, WorkspaceConfigPath: wc,
	}

	if _, err := chainsetup.NetUp(context.Background(), deps, up); err != nil {
		t.Fatalf("first up: %v", err)
	}

	// What the running composition is made of, as it stands.
	st := stateOf(t, dir, deps)
	genesisPath := st.GenesisPath
	genesisBefore := readFile(t, genesisPath)
	configsBefore := map[string][]byte{}
	pidsBefore := map[int]int{}
	for _, n := range st.Nodes {
		configsBefore[n.ConfigPath] = readFile(t, n.ConfigPath)
		pidsBefore[n.Index] = n.PID
	}
	hashesBefore := map[string]string{}
	for k, v := range st.LaunchInputs {
		hashesBefore[k] = v
	}

	// Ask for a different chain: the genesis changes, so the reuse must be
	// refused — a running network cannot be reconciled onto another chain.
	changed := up
	changed.ChainID = 424242
	_, err = chainsetup.NetUp(context.Background(), deps, changed)
	if err == nil {
		t.Fatal("a changed genesis must refuse the reuse")
	}
	if !strings.Contains(err.Error(), "genesis changed") {
		t.Fatalf("the refusal should name the genesis: %v", err)
	}

	// Nothing the running network reads may have moved.
	if got := readFile(t, genesisPath); string(got) != string(genesisBefore) {
		t.Fatal("the genesis on the target was replaced by a refused composition")
	}
	for path, want := range configsBefore {
		if got := readFile(t, path); string(got) != string(want) {
			t.Fatalf("%s was replaced by a refused composition", path)
		}
	}
	// And the workspace still describes what is actually running.
	after := stateOf(t, dir, deps)
	for k, want := range hashesBefore {
		if got := after.LaunchInputs[k]; got != want {
			t.Fatalf("recorded hash for %s changed on a refusal: %s -> %s", k, want, got)
		}
	}
	for _, n := range after.Nodes {
		if n.PID != pidsBefore[n.Index] {
			t.Fatalf("node%d pid changed on a refusal: %d -> %d", n.Index, pidsBefore[n.Index], n.PID)
		}
	}

	// Re-asking must keep refusing: the refusal did not quietly adopt the new
	// inputs as the baseline for next time.
	if _, err := chainsetup.NetUp(context.Background(), deps, changed); err == nil {
		t.Fatal("the second attempt must be refused too")
	}
}

// initStub is the stub driver plus the datadir initializer the start stage
// asks for. It writes nothing: this test is about the genesis and config files,
// not about what a node does with them.
type initStub struct{ *stubDriver }

func (initStub) InitDatadir(_ context.Context, spec process.NodeSpec, _ []byte) error {
	// The start stage checks the datadir exists before launching, so create it.
	return os.MkdirAll(spec.DataDir, 0o755)
}

// readFile is a t.Fatal-on-error read, for asserting bytes did not move.
func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}
