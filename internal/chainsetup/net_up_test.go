package chainsetup_test

import (
	"github.com/0xmhha/chainbench/internal/chainsetup"

	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chainsetup.NetUp is the whole point of the step stack replacing the setup stack: one
// call has to compose what nine hand-run steps do. These cover the composition
// half — everything up to (not including) launching processes, which needs a
// real node binary.

func TestNetUp_ProvisionStageComposesEverything(t *testing.T) {
	dir := t.TempDir()
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}

	out, err := chainsetup.NetUp(context.Background(), chainsetup.Deps{Clock: fixedClock()}, chainsetup.NetUpIn{
		DataDir: dir, Stage: chainsetup.UpProvision,
		Chain: "stablenet", KeysDir: keysAbs,
		Validators: 2, Endpoints: 1,
		LaunchSet: []string{"networkid=4242"},
	})
	if err != nil {
		t.Fatalf("chainsetup.NetUp: %v", err)
	}

	// Every composition step ran, in order, and each recorded something.
	wantSteps := []string{"new", "allocate", "keys", "genesis", "config", "launchopts", "provision"}
	if len(out.Steps) != len(wantSteps) {
		t.Fatalf("ran %d steps, want %d: %v", len(out.Steps), len(wantSteps), out.Steps)
	}
	for i, want := range wantSteps {
		if !strings.HasPrefix(out.Steps[i], want+": ") {
			t.Errorf("step %d = %q, want %s", i, out.Steps[i], want)
		}
	}

	// The artifacts a launcher boots from are on disk.
	for _, f := range []string{"genesis.json", "config_node1.toml", "config_node3.toml"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s not written: %v", f, err)
		}
	}
	// And the network reads back as a NodeSet with no processes started.
	if len(out.Nodes.Nodes.Nodes) != 3 || !out.Nodes.Composed {
		t.Fatalf("node set = %+v composed=%v", out.Nodes.Nodes, out.Nodes.Composed)
	}
	for _, n := range out.Nodes.Nodes.Nodes {
		if n.PID != 0 {
			t.Errorf("node%d has PID %d after a provision-only run", n.Index, n.PID)
		}
	}
	// The launch override reached the assembled argv.
	st := stateOf(t, dir, chainsetup.Deps{Clock: fixedClock()})
	if !strings.Contains(strings.Join(st.Nodes[0].Args, " "), "4242") {
		t.Errorf("launch override missing from argv: %v", st.Nodes[0].Args)
	}
}

func TestNetUp_CarriesTheGenesisAndLayoutCustomizations(t *testing.T) {
	dir := t.TempDir()
	keysAbs, _ := filepath.Abs(presetDir)

	out, err := chainsetup.NetUp(context.Background(), chainsetup.Deps{Clock: fixedClock()}, chainsetup.NetUpIn{
		DataDir: dir, Stage: chainsetup.UpProvision,
		Chain: "stablenet", KeysDir: keysAbs,
		Validators: 2, Endpoints: 1, EndpointSyncMode: "snap",
		ChainID: 7777, GenesisSet: []string{"bohoBlock=10"},
	})
	if err != nil {
		t.Fatalf("chainsetup.NetUp: %v", err)
	}
	cfg := genesisConfig(t, filepath.Join(dir, "genesis.json"))
	if cfg["chainId"] != float64(7777) || cfg["bohoBlock"] != float64(10) {
		t.Errorf("genesis config = %v", cfg)
	}
	if !hasCapability(out.Nodes.Nodes.Capabilities, "delayed-boho") {
		t.Errorf("capabilities = %v, want delayed-boho", out.Nodes.Nodes.Capabilities)
	}
	st := stateOf(t, dir, chainsetup.Deps{Clock: fixedClock()})
	if st.Nodes[2].SyncMode != "snap" {
		t.Errorf("endpoint sync mode = %q, want snap", st.Nodes[2].SyncMode)
	}
}

func TestNetUp_StartStageNeedsABinary(t *testing.T) {
	// The stage that runs processes cannot guess the executable, and for a
	// remote target it would not be a local path anyway.
	_, err := chainsetup.NetUp(context.Background(), chainsetup.Deps{Clock: fixedClock()}, chainsetup.NetUpIn{
		DataDir: t.TempDir(), Chain: "stablenet", Validators: 1,
	})
	if err == nil {
		t.Fatal("want an error without a binary")
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Errorf("error should name the missing binary, got: %v", err)
	}
}

func TestNetUp_StopsAtTheFirstFailingStepAndReportsProgress(t *testing.T) {
	// A run that dies part way still has to say how far it got: the workspace
	// is resumable by hand from exactly there.
	dir := t.TempDir()
	keysAbs, _ := filepath.Abs(presetDir)

	out, err := chainsetup.NetUp(context.Background(), chainsetup.Deps{Clock: fixedClock()}, chainsetup.NetUpIn{
		DataDir: dir, Stage: chainsetup.UpProvision,
		Chain: "stablenet", KeysDir: keysAbs, Validators: 2,
		GenesisSet: []string{"malformed-no-value"},
	})
	if err == nil {
		t.Fatal("want an error from the genesis step")
	}
	if !strings.Contains(err.Error(), "genesis") {
		t.Errorf("error should name the failing step, got: %v", err)
	}
	// new and allocate and keys ran; genesis is where it stopped.
	if len(out.Steps) != 3 {
		t.Errorf("recorded %d steps before the failure: %v", len(out.Steps), out.Steps)
	}
}

func TestNetUp_RejectsAnUnknownStage(t *testing.T) {
	_, err := chainsetup.NetUp(context.Background(), chainsetup.Deps{Clock: fixedClock()}, chainsetup.NetUpIn{
		DataDir: t.TempDir(), Stage: "halfway", Chain: "stablenet",
	})
	if err == nil {
		t.Fatal("want an error for an unknown stage")
	}
}

func TestNetUp_NeedsAWorkspaceDirectory(t *testing.T) {
	if _, err := chainsetup.NetUp(context.Background(), chainsetup.Deps{}, chainsetup.NetUpIn{Chain: "stablenet"}); err == nil {
		t.Error("want an error without a data dir")
	}
}
