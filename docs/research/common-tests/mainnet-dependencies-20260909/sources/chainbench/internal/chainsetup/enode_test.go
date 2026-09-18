package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// TestNetEnodes covers stage ③: enodes derive from keys and place, in node
// order, and the verb names the missing prerequisite rather than returning an
// empty list.
func TestNetEnodes(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := chainsetup.Deps{Clock: fixedClock()}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := chainsetup.NetNew(ctx, d, chainsetup.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}

	// Before place: no node table, so enode says which step to run first.
	if _, err := chainsetup.NetEnodes(ctx, d, chainsetup.NetEnodesIn{DataDir: dir}); err == nil ||
		!strings.Contains(err.Error(), "chain place") {
		t.Fatalf("enode before place: %v", err)
	}

	if _, err := chainsetup.NetAllocate(ctx, d, chainsetup.NetAllocateIn{DataDir: dir, Validators: 3}); err != nil {
		t.Fatalf("place: %v", err)
	}
	if _, err := chainsetup.NetKeys(ctx, d, chainsetup.NetKeysIn{DataDir: dir}); err != nil {
		t.Fatalf("keys: %v", err)
	}

	out, err := chainsetup.NetEnodes(ctx, d, chainsetup.NetEnodesIn{DataDir: dir})
	if err != nil {
		t.Fatalf("enode: %v", err)
	}
	if len(out.Enodes) != 3 {
		t.Fatalf("got %d enodes, want 3", len(out.Enodes))
	}
	for i, e := range out.Enodes {
		if e.Index != i+1 {
			t.Errorf("enode %d has index %d, want %d (node order)", i, e.Index, i+1)
		}
		if !strings.HasPrefix(e.Enode, "enode://") || !strings.Contains(e.Enode, "@") {
			t.Errorf("node%d enode is malformed: %q", e.Index, e.Enode)
		}
	}

	// --node filters to one, keeping its identity.
	one, err := chainsetup.NetEnodes(ctx, d, chainsetup.NetEnodesIn{DataDir: dir, Node: 2})
	if err != nil {
		t.Fatalf("enode --node 2: %v", err)
	}
	if len(one.Enodes) != 1 || one.Enodes[0].Index != 2 || one.Enodes[0].Enode != out.Enodes[1].Enode {
		t.Fatalf("--node 2 = %+v, want just node2 matching the full list", one.Enodes)
	}

	// A node the workspace does not have is an error, not an empty list.
	if _, err := chainsetup.NetEnodes(ctx, d, chainsetup.NetEnodesIn{DataDir: dir, Node: 9}); err == nil {
		t.Fatal("enode --node 9 must fail on a 3-node workspace")
	}
}

// TestNetConfigNodeScoped covers per-node config overrides: an "all" override
// reaches every node, a "node<N>" override only that node (and wins), and an
// unknown knob is refused.
func TestNetConfigNodeScoped(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := chainsetup.Deps{Clock: fixedClock()}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chainsetup.NetNew(ctx, d, chainsetup.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := chainsetup.NetAllocate(ctx, d, chainsetup.NetAllocateIn{DataDir: dir, Validators: 3}); err != nil {
		t.Fatalf("place: %v", err)
	}
	if _, err := chainsetup.NetKeys(ctx, d, chainsetup.NetKeysIn{DataDir: dir}); err != nil {
		t.Fatalf("keys: %v", err)
	}
	if _, err := chainsetup.NetGenesis(ctx, d, chainsetup.NetGenesisIn{DataDir: dir}); err != nil {
		t.Fatalf("genesis: %v", err)
	}

	// node2 only: syncMode=snap.
	if _, err := chainsetup.NetConfig(ctx, d, chainsetup.NetConfigIn{DataDir: dir, Node: 2, Set: []string{"syncMode=snap"}}); err != nil {
		t.Fatalf("config node2: %v", err)
	}
	readSync := func(idx int) string {
		b, rerr := os.ReadFile(filepath.Join(dir, "node"+itoa(idx), "..", "config_node"+itoa(idx)+".toml"))
		if rerr != nil {
			b, rerr = os.ReadFile(filepath.Join(dir, "config_node"+itoa(idx)+".toml"))
			if rerr != nil {
				t.Fatalf("read node%d config: %v", idx, rerr)
			}
		}
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "SyncMode = ") {
				return line
			}
		}
		return ""
	}
	if got := readSync(2); !strings.Contains(got, "snap") {
		t.Errorf("node2 SyncMode = %q, want snap", got)
	}
	if got := readSync(1); !strings.Contains(got, "full") {
		t.Errorf("node1 SyncMode = %q, want the default full (node2 override must not leak)", got)
	}

	// An unknown knob is refused.
	if _, err := chainsetup.NetConfig(ctx, d, chainsetup.NetConfigIn{DataDir: dir, Set: []string{"bogus=1"}}); err == nil {
		t.Fatal("an unknown config knob must be refused")
	}
}
