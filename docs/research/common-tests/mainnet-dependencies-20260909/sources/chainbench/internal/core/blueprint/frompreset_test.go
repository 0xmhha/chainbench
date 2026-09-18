package blueprint_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/blueprint"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
)

const presetDir = "../../../keys/preset"

// TestFromPreset_ComposesTheSameNetwork is N5's gate.
//
// The generated document must resolve to the identities the preset holds. If it
// did not, `net blueprint --from-preset` would hand an operator a file that
// describes a different network from the one the same preset composes, and both
// would look correct until their genesis files disagreed.
func TestFromPreset_ComposesTheSameNetwork(t *testing.T) {
	want, err := store.PresetKeys{Path: presetDir}.Ensure(context.Background(), 5)
	if err != nil {
		t.Skipf("no preset fixture: %v", err)
	}

	bp, err := blueprint.FromPreset(want, blueprint.FromPresetIn{
		Dir: presetDir, Chain: "wbft", Producers: 4, Endpoints: 1,
	})
	if err != nil {
		t.Fatalf("from preset: %v", err)
	}

	// Through the file, not just in memory: the document an operator edits is
	// the one that has to work.
	raw, err := blueprint.Marshal(bp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// A generated document must not carry key material. It is meant to be
	// committed and shared, and a generator that inlined secrets would make
	// that unsafe by default.
	for i, e := range want.Nodes {
		if strings.Contains(string(raw), e.Nodekey.Hex()) {
			t.Fatalf("node%d's private key was written into the document", i+1)
		}
	}

	reread, err := blueprint.Parse(raw)
	if err != nil {
		t.Fatalf("the generated document does not parse: %v\n%s", err, raw)
	}

	roles := []node.Role{node.RoleBP, node.RoleBP, node.RoleBP, node.RoleBP, node.RoleEN}
	r, err := blueprint.Resolve(reread, blueprint.Inputs{
		Placed: placements(roles...), Chain: blueprint.ChainFacts{ID: "wbft"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got, err := blueprint.PresetFrom(r, derive.WithBLS, os.ReadFile)
	if err != nil {
		t.Fatalf("preset from the generated document: %v", err)
	}

	if len(got.Nodes) != len(want.Nodes) {
		t.Fatalf("the document yields %d identities, the preset holds %d", len(got.Nodes), len(want.Nodes))
	}
	for i := range want.Nodes {
		if got.Nodes[i].Address != want.Nodes[i].Address {
			t.Errorf("node%d address %s, preset %s", i+1, got.Nodes[i].Address, want.Nodes[i].Address)
		}
		if got.Nodes[i].PublicKey != want.Nodes[i].PublicKey {
			t.Errorf("node%d devp2p key differs from the preset's", i+1)
		}
		if got.Nodes[i].BLS == nil || want.Nodes[i].BLS == nil {
			continue
		}
		if got.Nodes[i].BLS.PublicKey != want.Nodes[i].BLS.PublicKey {
			t.Errorf("node%d BLS key differs from the preset's", i+1)
		}
	}
	// The sealing set and its order: extraData is built from this list, so the
	// same keys in a different order are a different genesis.
	if len(got.Network.Validators) != len(want.Network.Validators) {
		t.Fatalf("the document declares %d validators, the preset %d",
			len(got.Network.Validators), len(want.Network.Validators))
	}
	for i := range want.Network.Validators {
		if !strings.EqualFold(got.Network.Validators[i], want.Network.Validators[i]) {
			t.Errorf("validator %d = %s, preset %s", i, got.Network.Validators[i], want.Network.Validators[i])
		}
	}
}

// TestFromPreset_SizesFromTheValidatorSet: asked for nothing, the document
// describes the network the preset already declares.
func TestFromPreset_SizesFromTheValidatorSet(t *testing.T) {
	set, err := store.PresetKeys{Path: presetDir}.Ensure(context.Background(), 5)
	if err != nil {
		t.Skipf("no preset fixture: %v", err)
	}
	bp, err := blueprint.FromPreset(set, blueprint.FromPresetIn{Dir: presetDir, Chain: "wbft"})
	if err != nil {
		t.Fatalf("from preset: %v", err)
	}
	if len(bp.Nodes) != len(set.Network.Validators) {
		t.Errorf("wrote %d nodes, the preset declares %d validators", len(bp.Nodes), len(set.Network.Validators))
	}
	for i, n := range bp.Nodes {
		if n.Role != "bp" {
			t.Errorf("node %d role = %q, want bp", i+1, n.Role)
		}
	}
}

// TestFromPreset_Refuses covers what must not be invented.
func TestFromPreset_Refuses(t *testing.T) {
	set, err := store.PresetKeys{Path: presetDir}.Ensure(context.Background(), 5)
	if err != nil {
		t.Skipf("no preset fixture: %v", err)
	}
	for name, c := range map[string]struct {
		in    blueprint.FromPresetIn
		wants string
	}{
		"no directory to point the keys at": {blueprint.FromPresetIn{Chain: "wbft"}, "points its keys at"},
		"more nodes than the set holds": {
			blueprint.FromPresetIn{Dir: presetDir, Chain: "wbft", Producers: 9}, "holds 5 identities"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := blueprint.FromPreset(set, c.in)
			if err == nil {
				t.Fatal("accepted")
			}
			if !strings.Contains(err.Error(), c.wants) {
				t.Errorf("error %q does not say %q", err, c.wants)
			}
		})
	}
}

// placements builds the allocation the way the allocator would, so the test
// resolves against a node table rather than against nothing.
func placements(roles ...node.Role) []node.Placement {
	var out []node.Placement
	ord := map[node.Role]int{}
	for i, r := range roles {
		ord[r]++
		out = append(out, node.Placement{
			Index: i + 1, Label: node.LabelFor(i + 1), Role: r, Ord: ord[r],
			Host:  "127.0.0.1",
			Ports: node.Endpoints{P2P: 30300 + i*10, HTTP: 8500 + i*10},
		})
	}
	return out
}
