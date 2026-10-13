package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// TestChainUp_InitStageInitialisesAndStartsNothing: --stage init composes the
// network, deploys it and initialises every datadir, then stops: no node is
// launched, and a later resume does not start one either. It is the setup a
// caller boots later, with the datadirs already at genesis.
func TestChainUp_InitStageInitialisesAndStartsNothing(t *testing.T) {
	dir := t.TempDir()
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	stub := initStub{stubDriver: &stubDriver{}}
	deps := chainsetup.Deps{Clock: fixedClock(), Driver: func() (process.Driver, error) { return stub, nil }}
	binary := filepath.Join(dir, "gstable")
	if err = os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	up := chainsetup.ChainUpIn{
		DataDir: dir, Stage: chainsetup.UpInit, Chain: "stablenet", KeysDir: keysAbs, Binary: binary,
		BPCount: 2, WorkspaceConfigPath: reuseWorkspaceConfig(t, dir, t.TempDir()),
	}
	if _, err = verb.ChainUp(context.Background(), deps, up); err != nil {
		t.Fatalf("up: %v", err)
	}
	st := stateOf(t, dir, deps)
	if !st.Steps["init"].Done {
		t.Fatal("init did not run")
	}
	if st.Steps["start"].Done || len(stub.launched) != 0 {
		t.Fatalf("an init-stage run launched nodes %v", stub.launched)
	}
	for _, n := range st.Nodes {
		if _, err = os.Stat(n.DataDir); err != nil {
			t.Fatalf("node%d datadir was not initialised: %v", n.Index, err)
		}
	}
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if next := ws.FirstUndone(); next != "" {
		t.Fatalf("an init-stage composition reports %q still to do", next)
	}
}
