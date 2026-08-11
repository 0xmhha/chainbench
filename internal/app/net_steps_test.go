package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
)

// presetDir is the repository's shipped key set, used as a realistic fixture
// (same convention as the engine tests).
const presetDir = "../../keys/preset"

// TestNetStepPipeline composes a network step by step without a chain binary:
// new -> allocate -> keys -> genesis -> config -> launchopts -> provision.
// It pins that each step persists its state, that the argv comes from the
// single assembly site with overrides applied, and that the lifecycle steps
// fail with actionable errors when their prerequisites are missing.
func TestNetStepPipeline(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := app.Deps{Clock: fixedClock}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := app.NetAllocate(ctx, d, app.NetAllocateIn{DataDir: dir, Validators: 2, Endpoints: 1}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if out, err := app.NetKeys(ctx, d, app.NetKeysIn{DataDir: dir}); err != nil {
		t.Fatalf("keys: %v", err)
	} else if !strings.Contains(out.Detail, "identities") {
		t.Fatalf("keys detail = %q", out.Detail)
	}
	if _, err := app.NetGenesis(ctx, d, app.NetGenesisIn{DataDir: dir, ChainID: 9999}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "genesis.json")); err != nil {
		t.Fatalf("genesis file: %v", err)
	} else if !strings.Contains(string(b), "9999") {
		t.Fatal("genesis does not carry the chain-id override")
	}
	if _, err := app.NetConfig(ctx, d, app.NetConfigIn{DataDir: dir}); err != nil {
		t.Fatalf("config: %v", err)
	}

	lo, err := app.NetLaunchOpts(ctx, d, app.NetLaunchOptsIn{DataDir: dir, Set: []string{"networkid=4242"}})
	if err != nil {
		t.Fatalf("launchopts: %v", err)
	}
	if len(lo.Nodes) != 3 {
		t.Fatalf("node table = %d, want 3", len(lo.Nodes))
	}
	argv := strings.Join(lo.Nodes[0].Args, " ")
	for _, frag := range []string{"--datadir", "--networkid 4242", "--unlock"} {
		if !strings.Contains(argv, frag) {
			t.Fatalf("node1 argv missing %q:\n%s", frag, argv)
		}
	}
	// node3 is an endpoint: no unlock.
	if strings.Contains(strings.Join(lo.Nodes[2].Args, " "), "--unlock") {
		t.Fatal("endpoint node must not unlock an account")
	}

	if _, err := app.NetProvision(ctx, d, app.NetProvisionIn{DataDir: dir}); err != nil {
		t.Fatalf("provision: %v", err)
	}

	// State round-trips: a fresh open sees the accumulated composition.
	st, err := app.NetStatus(ctx, d, app.NetStatusIn{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []string{"new", "allocate", "keys", "genesis", "config", "launchopts", "provision"} {
		if s, ok := st.State.Steps[step]; !ok || !s.Done {
			t.Fatalf("step %q not recorded: %+v", step, st.State.Steps)
		}
	}
	if len(st.State.Nodes) != 3 || len(st.State.Nodes[0].Args) == 0 {
		t.Fatalf("persisted node table incomplete: %+v", st.State.Nodes)
	}

	// Lifecycle prerequisites fail with actionable messages (no binary set).
	if _, err := app.NetStart(ctx, d, app.NetStartIn{DataDir: dir}); err == nil ||
		!strings.Contains(err.Error(), "binary") {
		t.Fatalf("start without a binary must name the missing binary, got %v", err)
	}
	if _, err := app.NetRestart(ctx, d, app.NetRestartIn{DataDir: dir, Node: 9}); err == nil {
		t.Fatal("restart of an unknown node must fail")
	}

	// Rm clears the composed data plane and the node table.
	if _, err := app.NetRm(ctx, d, app.NetRmIn{DataDir: dir}); err != nil {
		t.Fatalf("rm: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "genesis.json")); !os.IsNotExist(err) {
		t.Fatal("rm must remove the genesis")
	}
	st, err = app.NetStatus(ctx, d, app.NetStatusIn{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(st.State.Nodes) != 0 {
		t.Fatalf("rm must clear the node table, got %+v", st.State.Nodes)
	}
}

// TestNetStepPrerequisites pins the fail-fast messages steps give before their
// prerequisites ran.
func TestNetStepPrerequisites(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := app.Deps{Clock: fixedClock}

	if _, err := app.NetAllocate(ctx, d, app.NetAllocateIn{DataDir: dir, Validators: 1}); err == nil ||
		!strings.Contains(err.Error(), "net new") {
		t.Fatalf("allocate before new: %v", err)
	}
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet"}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.NetGenesis(ctx, d, app.NetGenesisIn{DataDir: dir}); err == nil ||
		!strings.Contains(err.Error(), "net allocate") {
		t.Fatalf("genesis before allocate: %v", err)
	}
	if _, err := app.NetKeys(ctx, d, app.NetKeysIn{DataDir: dir}); err == nil {
		t.Fatal("keys with no node count must fail")
	}
	if _, err := app.NetHealth(ctx, d, app.NetHealthIn{DataDir: dir}); err == nil {
		t.Fatal("health before allocate must fail")
	}
}
