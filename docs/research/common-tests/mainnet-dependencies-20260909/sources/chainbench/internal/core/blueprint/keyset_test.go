package blueprint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// testKey is a deterministic key per index. TEST MATERIAL ONLY — it is derived
// from a counter, which is exactly what a real key must never be.
func testKey(i int) string {
	return "0x" + strings.Repeat(fmt.Sprintf("%02x", i), 32)
}

// rawNetwork is a blueprint that carries its own keys, which is the shape the
// raw path exists for: no preset directory anywhere.
func rawNetwork(t *testing.T, roles ...node.Role) ResolvedNetwork {
	t.Helper()
	var nodes []Node
	for i := range roles {
		nodes = append(nodes, Node{NodeKey: &NodeKeyRef{Hex: testKey(i + 1)}, Role: string(roles[i])})
	}
	r, err := Resolve(Blueprint{Nodes: nodes}, Inputs{Placed: placed(roles...), Chain: facts})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return r
}

// TestKeySet_StandsUpWithoutAPreset is N3's premise.
//
// Before this, store.LoadPreset was the only way a keyring.Preset came into
// being, so "the preset is optional" was untrue whatever the documents said. A
// blueprint that writes its own keys now produces the same Preset the rest of
// the composition already consumes, so nothing downstream learns a second way
// to obtain keys.
func TestPresetFrom_StandsUpWithoutAPreset(t *testing.T) {
	r := rawNetwork(t, node.RoleBP, node.RoleBP, node.RoleEN)
	ks, err := PresetFrom(r, derive.AccountOnly, nil)
	if err != nil {
		t.Fatalf("keyset: %v", err)
	}
	if len(ks.Nodes) != 3 {
		t.Fatalf("got %d entries, want 3", len(ks.Nodes))
	}
	for i, e := range ks.Nodes {
		if e.Index != i+1 {
			t.Errorf("entry %d has index %d", i, e.Index)
		}
		if string(e.Label) != fmt.Sprintf("node%d", i+1) {
			t.Errorf("entry %d label = %q", i, e.Label)
		}
		if len(e.PublicKey) != 128 {
			t.Errorf("entry %d devp2p key is %d hex chars, want 128", i, len(e.PublicKey))
		}
		if !strings.HasPrefix(e.Address, "0x") || len(e.Address) != 42 {
			t.Errorf("entry %d address = %q", i, e.Address)
		}
	}
	// The sealing set is the network's, in the order Resolve fixed. Deriving it
	// again from the entries would let the ring disagree with the blueprint
	// about who seals.
	if len(ks.Network.Validators) != 2 {
		t.Fatalf("validators = %v, want the two bp addresses", ks.Network.Validators)
	}
	if ks.Network.Validators[0] != ks.Nodes[0].Address || ks.Network.Validators[1] != ks.Nodes[1].Address {
		t.Errorf("validators %v are not the bp addresses in node order", ks.Network.Validators)
	}
}

// TestKeySet_IsDeterministic: the same document yields the same ring, which is
// what lets a genesis built from it be rebuilt.
func TestPresetFrom_IsDeterministic(t *testing.T) {
	r := rawNetwork(t, node.RoleBP, node.RoleEN)
	first, err := PresetFrom(r, derive.AccountOnly, nil)
	if err != nil {
		t.Fatalf("keyset: %v", err)
	}
	for i := 0; i < 5; i++ {
		again, err := PresetFrom(r, derive.AccountOnly, nil)
		if err != nil {
			t.Fatalf("keyset %d: %v", i, err)
		}
		for j := range first.Nodes {
			if first.Nodes[j].Address != again.Nodes[j].Address ||
				first.Nodes[j].PublicKey != again.Nodes[j].PublicKey {
				t.Fatalf("run %d derived a different identity for node%d", i, j+1)
			}
		}
	}
}

// TestKeySet_DerivesBLSOnlyWhenAsked: BLS material costs real computation and
// only wbft consumes it, so absence has to stay distinguishable from zeroes.
func TestPresetFrom_DerivesBLSOnlyWhenAsked(t *testing.T) {
	r := rawNetwork(t, node.RoleBP)
	plain, err := PresetFrom(r, derive.AccountOnly, nil)
	if err != nil {
		t.Fatalf("keyset: %v", err)
	}
	if plain.Nodes[0].BLS != nil {
		t.Error("account-only derivation produced BLS material")
	}
	withBLS, err := PresetFrom(r, derive.WithBLS, nil)
	if err != nil {
		t.Fatalf("keyset with bls: %v", err)
	}
	if withBLS.Nodes[0].BLS == nil || withBLS.Nodes[0].BLS.PublicKey == "" || withBLS.Nodes[0].BLS.PoP == "" {
		t.Errorf("bls material = %+v, want a key and a proof", withBLS.Nodes[0].BLS)
	}
}

// TestKeySet_ReadsAKeyNamedByPath covers the other spelling, including the
// trailing newline geth writes. Trimming it is not leniency; it is the format.
func TestPresetFrom_ReadsAKeyNamedByPath(t *testing.T) {
	r, err := Resolve(
		Blueprint{Nodes: []Node{{Role: "bp", NodeKey: &NodeKeyRef{File: "./bp1.nodekey"}}}},
		Inputs{Placed: placed(node.RoleBP), Chain: facts},
	)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	read := func(p string) ([]byte, error) {
		if p != "./bp1.nodekey" {
			return nil, fmt.Errorf("unexpected path %q", p)
		}
		return []byte(strings.TrimPrefix(testKey(7), "0x") + "\n"), nil
	}
	ks, err := PresetFrom(r, derive.AccountOnly, read)
	if err != nil {
		t.Fatalf("keyset: %v", err)
	}
	// The same key read from a file and written in the document must derive
	// the same identity, or the two spellings are two networks.
	inline := rawNetwork(t, node.RoleBP)
	inline.Nodes[0].NodeKey = NodeKeyRef{Hex: testKey(7)}
	want, err := PresetFrom(inline, derive.AccountOnly, nil)
	if err != nil {
		t.Fatalf("keyset inline: %v", err)
	}
	if ks.Nodes[0].Address != want.Nodes[0].Address {
		t.Errorf("a key read from a file derived %s, written inline it derives %s",
			ks.Nodes[0].Address, want.Nodes[0].Address)
	}
}

// TestKeySet_Refuses covers what must not become an empty identity: a node
// launched with one joins nothing and reports nothing wrong.
func TestPresetFrom_Refuses(t *testing.T) {
	r := rawNetwork(t, node.RoleBP)
	r.Nodes[0].NodeKey = NodeKeyRef{Hex: "0xnothex"}
	if _, err := PresetFrom(r, derive.AccountOnly, nil); err == nil {
		t.Error("a malformed key was accepted")
	}

	r = rawNetwork(t, node.RoleBP)
	r.Nodes[0].NodeKey = NodeKeyRef{File: "./missing"}
	if _, err := PresetFrom(r, derive.AccountOnly, nil); err == nil || !strings.Contains(err.Error(), "no way to read") {
		t.Errorf("a file key with no reader gave %v, want a refusal that says so", err)
	}

	r = rawNetwork(t, node.RoleBP)
	r.Nodes[0].NodeKey = NodeKeyRef{}
	if _, err := PresetFrom(r, derive.AccountOnly, nil); err == nil {
		t.Error("a node with no key was accepted")
	}
}

// TestKeySet_MatchesThePresetPath is the proof that the raw path is not a
// second, slightly different way to build a network.
//
// It takes the committed preset's own nodekeys, writes them into a blueprint,
// and checks that every derived address, devp2p key, BLS key and proof matches
// what the preset recorded — including the sealing set and its order, which is
// what a genesis's extraData is built from. If these ever diverged, a network
// composed from a blueprint and one composed from the same keys via the preset
// would be two different chains that both looked correct.
func TestPresetFrom_MatchesThePresetPath(t *testing.T) {
	const presetDir = "../../../keys/preset"
	var meta struct {
		Validators    []string `json:"validators"`
		BLSPublicKeys []string `json:"blsPublicKeys"`
	}
	b, err := os.ReadFile(filepath.Join(presetDir, "metadata.json"))
	if err != nil {
		t.Skipf("no preset fixture: %v", err)
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatalf("preset metadata: %v", err)
	}

	// Four producers and one endpoint, which is what the fixture describes.
	roles := []node.Role{node.RoleBP, node.RoleBP, node.RoleBP, node.RoleBP, node.RoleEN}
	var nodes []Node
	for i := range roles {
		nodes = append(nodes, Node{
			Role:    string(roles[i]),
			NodeKey: &NodeKeyRef{File: filepath.Join(presetDir, fmt.Sprintf("node%d", i+1), "nodekey")},
		})
	}
	r, err := Resolve(Blueprint{Nodes: nodes}, Inputs{Placed: placed(roles...), Chain: facts})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	ks, err := PresetFrom(r, derive.WithBLS, os.ReadFile)
	if err != nil {
		t.Fatalf("keyset: %v", err)
	}

	for i, e := range ks.Nodes {
		dir := filepath.Join(presetDir, fmt.Sprintf("node%d", i+1))
		for _, c := range []struct{ file, got string }{
			{"address", e.Address},
			{"pubkey", e.PublicKey},
			{"bls_pubkey", e.BLS.PublicKey},
			{"pop", e.BLS.PoP},
		} {
			want, err := os.ReadFile(filepath.Join(dir, c.file))
			if err != nil {
				t.Fatalf("read %s: %v", c.file, err)
			}
			if !strings.EqualFold(strings.TrimSpace(string(want)), strings.TrimSpace(c.got)) {
				t.Errorf("node%d %s\n  derived %s\n  preset  %s", i+1, c.file, c.got, strings.TrimSpace(string(want)))
			}
		}
	}

	// The sealing set and its order too: extraData is built from this list, so
	// the same keys in a different order are a different genesis.
	if len(ks.Network.Validators) != len(meta.Validators) {
		t.Fatalf("derived %d validators, the preset records %d", len(ks.Network.Validators), len(meta.Validators))
	}
	for i, want := range meta.Validators {
		if !strings.EqualFold(ks.Network.Validators[i], want) {
			t.Errorf("validator %d = %s, the preset records %s", i, ks.Network.Validators[i], want)
		}
	}
}
