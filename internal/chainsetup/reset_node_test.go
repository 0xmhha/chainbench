package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// upWithEndpoint brings up two producers and one endpoint on the stub driver.
func upWithEndpoint(t *testing.T) (string, *stubDriver, chainsetup.Deps) {
	t.Helper()
	dir := t.TempDir()
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	stub := initStub{stubDriver: &stubDriver{}}
	deps := chainsetup.Deps{
		Clock:  fixedClock(),
		Driver: func() (process.Driver, error) { return stub, nil },
	}
	binary := filepath.Join(dir, "gstable")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	up := chainsetup.ChainUpIn{
		DataDir: dir, Stage: chainsetup.UpStart,
		Chain: "stablenet", KeysDir: keysAbs, Binary: binary,
		BPCount: 2, ENCount: 1, WorkspaceConfigPath: reuseWorkspaceConfig(t, dir, t.TempDir()),
	}
	if _, err := verb.ChainUp(context.Background(), deps, up); err != nil {
		t.Fatalf("up: %v", err)
	}
	return dir, stub.stubDriver, deps
}

func nodeOfRole(t *testing.T, st chainsetup.State, role node.Role) node.Record {
	t.Helper()
	for _, n := range st.Nodes {
		if node.Is(node.Role(n.Role), role) {
			return n
		}
	}
	t.Fatalf("no %s node", role)
	return node.Record{}
}

// TestNodeReset_LeavesAnEndpointStoppedAtGenesis: a reset endpoint is down, its
// chain data is gone and its datadir is initialised again, and it keeps the
// arming a later start relaunches it with. That is what a snap-sync case needs:
// a node whose head is 0 when it first dials the network.
func TestNodeReset_LeavesAnEndpointStoppedAtGenesis(t *testing.T) {
	dir, stub, deps := upWithEndpoint(t)
	en := nodeOfRole(t, stateOf(t, dir, deps), node.RoleEN)
	chain := filepath.Join(en.DataDir, "chaindata-marker")
	if err := os.WriteFile(chain, []byte("blocks"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := verb.NodeReset(context.Background(), deps, verb.NodeResetIn{DataDir: dir, Index: en.Index}); err != nil {
		t.Fatalf("reset: %v", err)
	}

	after := nodeOfRole(t, stateOf(t, dir, deps), node.RoleEN)
	if after.PID != 0 {
		t.Errorf("pid = %d after a reset, want 0", after.PID)
	}
	if !containsInt(stub.stopped, en.Index) {
		t.Errorf("node%d was not stopped (stopped %v)", en.Index, stub.stopped)
	}
	if _, err := os.Stat(chain); !os.IsNotExist(err) {
		t.Errorf("the chain data survived the reset: %v", err)
	}
	if _, err := os.Stat(en.DataDir); err != nil {
		t.Errorf("the datadir was not initialised again: %v", err)
	}
	if len(after.Args) == 0 {
		t.Error("the reset dropped the arming a start relaunches with")
	}

	if _, err := verb.NodeStart(context.Background(), deps, verb.NodeStartIn{DataDir: dir, Index: en.Index}); err != nil {
		t.Fatalf("start after reset: %v", err)
	}
}

// TestNodeReset_RefusesAProducer: wiping a producer's chain would take a
// sealer out of the quorum and make it rebuild from genesis; that is not what
// the step is for, so it is refused before anything is stopped.
func TestNodeReset_RefusesAProducer(t *testing.T) {
	dir, stub, deps := upWithEndpoint(t)
	bp := nodeOfRole(t, stateOf(t, dir, deps), node.RoleBP)
	err := verb.NodeReset(context.Background(), deps, verb.NodeResetIn{DataDir: dir, Index: bp.Index})
	if err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("resetting node%d (bp) gave %v, want a refusal naming the producer", bp.Index, err)
	}
	if containsInt(stub.stopped, bp.Index) {
		t.Error("the producer was stopped before the refusal")
	}
}

func containsInt(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
