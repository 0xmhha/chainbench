package chainsetup_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// TestNodeSwap_ConfigOnlyRewritesOneNode is what N11 was missing.
//
// The gate — restart some of the nodes on a different config — is met by
// swapNode, which takes a config with no binary through the action, the use
// case and the workspace. What had no test was the whole path against a real
// key set, because the lifecycle fixture seeds a workspace without one and
// swapNodeConfig has to render a node's config, which needs the ring.
//
// Composing for real is what closes that: preset keys, an allocation, a
// genesis, configs, and recorded argv, and only the driver is a stub. A swap
// that named a binary would prove nothing here — the point is that naming only
// a config is enough, and that it reaches one node.
func TestNodeSwap_ConfigOnlyRewritesOneNode(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	stub := &stubDriver{}
	d := chainsetup.Deps{
		Clock:  fixedClock(),
		Driver: func() (process.Driver, error) { return stub, nil },
	}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	// The binary is recorded here rather than left out: relaunching a node
	// needs an executable even when only its config changed, and a workspace
	// with none refuses the swap for that reason rather than for its config.
	if _, err := chainsetup.NetNew(ctx, d, chainsetup.NetNewIn{
		DataDir: dir, Chain: "stablenet", Binary: "/opt/gstable", KeysDir: keysAbs,
	}); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := chainsetup.NetAllocate(ctx, d, chainsetup.NetAllocateIn{
		DataDir: dir, Validators: 1, Endpoints: 1,
	}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if _, err := chainsetup.NetKeys(ctx, d, chainsetup.NetKeysIn{DataDir: dir}); err != nil {
		t.Fatalf("keys: %v", err)
	}

	for _, step := range []func() error{
		func() error {
			_, err := chainsetup.NetGenesis(ctx, d, chainsetup.NetGenesisIn{DataDir: dir})
			return err
		},
		func() error { _, err := chainsetup.NetConfig(ctx, d, chainsetup.NetConfigIn{DataDir: dir}); return err },
		func() error {
			_, err := chainsetup.NetLaunchOpts(ctx, d, chainsetup.NetLaunchOptsIn{DataDir: dir})
			return err
		},
	} {
		if err := step(); err != nil {
			t.Fatalf("compose: %v", err)
		}
	}

	before := readConfig(t, dir, 2)
	if strings.Contains(before, `SyncMode = "snap"`) {
		t.Fatal("node2 already renders snap, so the swap would prove nothing")
	}

	// No binary. This is the whole point of the case: a config change alone is
	// a reason to relaunch a node.
	out, err := chainsetup.NodeSwap(ctx, d, chainsetup.NodeSwapIn{
		DataDir: dir, Index: 2, Config: []string{"syncMode=snap"}, Purpose: "resync",
	})
	if err != nil {
		t.Fatalf("config-only swap: %v", err)
	}
	if out.Node.Index != 2 {
		t.Errorf("swapped node = %d, want 2", out.Node.Index)
	}
	if len(stub.launched) != 1 || stub.launched[0] != 2 {
		t.Errorf("driver launched %v, want just node2", stub.launched)
	}

	if got := readConfig(t, dir, 2); !strings.Contains(got, `SyncMode = "snap"`) {
		t.Errorf("node2's config was not rewritten:\n%s", got)
	}
	// The other node is untouched, which is what "some of the nodes" means.
	if got := readConfig(t, dir, 1); strings.Contains(got, `SyncMode = "snap"`) {
		t.Errorf("node1's config was rewritten too:\n%s", got)
	}

	// The override is recorded against that node's scope, so a later restart
	// uses it rather than reverting to what the composition first rendered.
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	scoped := ws.State().ConfigSet["node2"]
	if len(scoped) != 1 || scoped[0] != "syncMode=snap" {
		t.Errorf("node2's recorded overrides = %v, want [syncMode=snap]", scoped)
	}
	if _, ok := ws.State().ConfigSet["node1"]; ok {
		t.Errorf("node1 got an override recorded: %v", ws.State().ConfigSet["node1"])
	}
}
