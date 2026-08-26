package store_test

import (
	"bytes"
	"context"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// TestLoadPreset_ShippedFixture checks the reader against the file three chains
// actually consume, including the fields that used to need a second type to
// hold them.
func TestLoadPreset_ShippedFixture(t *testing.T) {
	p, err := store.LoadPreset(filepath.Join("..", "..", "..", "..", "keys", "preset"))
	if err != nil {
		t.Fatalf("LoadPreset: %v", err)
	}
	if len(p.Nodes) == 0 || len(p.Network.Validators) == 0 {
		t.Fatalf("empty preset: %d nodes, %d validators", len(p.Nodes), len(p.Network.Validators))
	}
	if len(p.Network.Validators) != len(p.Network.BLSKeys) {
		t.Errorf("%d validators but %d BLS keys", len(p.Network.Validators), len(p.Network.BLSKeys))
	}
	// Every entry carries its secret and its public identity in one value.
	for _, e := range p.Nodes {
		if e.Address == "" || e.PublicKey == "" || e.BLS == nil {
			t.Errorf("node %d is missing derived material: %+v", e.Index, e.Identity)
		}
		if err := e.Verify(); err != nil {
			t.Errorf("node %d: %v", e.Index, err)
		}
	}
	if _, ok := p.Node(1); !ok {
		t.Error("no node1")
	}
	if _, ok := p.Node(999); ok {
		t.Error("found a node that does not exist")
	}
}

// TestGenerate_RoundTrips pins the merged package's round trip: what Generate
// writes is exactly what LoadPreset reads back, with no second shape in
// between. Entropy is injected so the run is reproducible.
func TestGenerate_RoundTrips(t *testing.T) {
	const nodes = 3
	validators := 2
	dir := t.TempDir()
	written, err := store.Generate(store.GenerateOpts{
		Nodes: nodes, Validators: &validators, Out: dir, Password: "1", Balance: "0x1",
		Derive: store.WithBLS,
		Rand:   bytes.NewReader(bytes.Repeat([]byte{0x7f}, nodes*derive.PrivateKeyLen)),
	}, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	read, err := store.LoadPreset(dir)
	if err != nil {
		t.Fatalf("LoadPreset: %v", err)
	}
	if len(read.Nodes) != nodes || len(read.Network.Validators) != 2 {
		t.Fatalf("read back %d nodes / %d validators", len(read.Nodes), len(read.Network.Validators))
	}
	for i, want := range written.Nodes {
		got := read.Nodes[i]
		if got.Nodekey.Hex() != want.Nodekey.Hex() || got.Address != want.Address {
			t.Errorf("node %d did not round trip", want.Index)
		}
		if err := got.Verify(); err != nil {
			t.Errorf("node %d: %v", want.Index, err)
		}
	}
	// Derived values are not stored: extra-data is computed at genesis time for
	// whatever validator set is actually used.
	if read.Network.ExtraData != "" {
		t.Errorf("generated set stored extra-data: %q", read.Network.ExtraData)
	}
	// The nodekey file a node launches with must match the index.
	onDisk, err := os.ReadFile(filepath.Join(dir, "node1", "nodekey"))
	if err != nil {
		t.Fatalf("read nodekey: %v", err)
	}
	if strings.TrimSpace(string(onDisk)) != read.Nodes[0].Nodekey.Hex() {
		t.Error("node1's file and the index disagree")
	}
}

// TestImportRing_ClonesDeclarationAndRefusesTamper pins the whole-ring
// import: labels and the validator declaration travel with the keys, and a
// source whose index no longer matches its keys is refused before anything
// is written — the integrity check the transfer exists to provide.
func TestImportRing_ClonesDeclarationAndRefusesTamper(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "src")
	two := 2
	src, err := store.Generate(store.GenerateOpts{
		Nodes: 3, Validators: &two, Out: srcDir, Password: "pw",
		Derive: store.WithBLS,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	dstDir := filepath.Join(t.TempDir(), "dst")
	got, err := store.ImportRing(context.Background(), nil, dstDir, src, "")
	if err != nil {
		t.Fatalf("import-ring: %v", err)
	}
	if len(got.Nodes) != 3 || len(got.Network.Validators) != 2 {
		t.Fatalf("clone lost shape: %d nodes, %d validators", len(got.Nodes), len(got.Network.Validators))
	}
	back, err := store.LoadPreset(dstDir)
	if err != nil {
		t.Fatal(err)
	}
	for i := range src.Nodes {
		if back.Nodes[i].Address != src.Nodes[i].Address || back.Nodes[i].Label != src.Nodes[i].Label {
			t.Fatalf("entry %d changed in transit: %+v", i, back.Nodes[i])
		}
	}
	if len(back.Network.Validators) != 2 {
		t.Fatalf("validator declaration not carried: %v", back.Network.Validators)
	}

	// A tampered source is refused whole: flip one recorded address.
	src.Nodes[1].Address = src.Nodes[0].Address
	if _, err := store.ImportRing(context.Background(), nil, filepath.Join(t.TempDir(), "d2"), src, ""); err == nil {
		t.Fatal("import-ring accepted a source whose index does not match its keys")
	}

	// A second import onto the same destination is refused.
	if _, err := store.ImportRing(context.Background(), nil, dstDir, got, ""); err == nil {
		t.Fatal("import-ring overwrote an existing ring")
	}
}
