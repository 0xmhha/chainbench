package store_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// A ring's index is ONE file and it carries every nodekey, so reading a ring on
// a server brings the private keys across whatever the caller meant to do. The
// public read cannot undo the transfer, but it decides whether the value the
// caller then holds -- and can log, persist, or hand on -- contains a secret.

// ringDir writes a two-entry preset and returns its directory.
func ringDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Real keys from keys/preset, which are public test fixtures.
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "keys", "preset", "metadata.json"))
	if err != nil {
		t.Skipf("keys/preset not readable from here: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), src, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestLoadPublicPreset_DropsEveryPrivateKey is the property: the returned ring
// has identities and no secrets.
func TestLoadPublicPreset_DropsEveryPrivateKey(t *testing.T) {
	dir := ringDir(t)

	full, err := store.LoadPreset(dir)
	if err != nil {
		t.Fatalf("LoadPreset: %v", err)
	}
	if len(full.Nodes) == 0 {
		t.Fatal("the fixture ring has no entries")
	}
	// The keyed read is what it always was, or the comparison below proves nothing.
	var anyKey bool
	for _, e := range full.Nodes {
		if len(e.Nodekey.Bytes()) > 0 && e.Nodekey.Hex() != strings.Repeat("00", 32) {
			anyKey = true
		}
	}
	if !anyKey {
		t.Fatal("the keyed read returned no private keys, so this fixture cannot show the difference")
	}

	pub, err := store.LoadPublicPreset(dir)
	if err != nil {
		t.Fatalf("LoadPublicPreset: %v", err)
	}
	if len(pub.Nodes) != len(full.Nodes) {
		t.Fatalf("public read has %d entries, keyed read has %d", len(pub.Nodes), len(full.Nodes))
	}
	for i, e := range pub.Nodes {
		if e.Nodekey.Hex() != strings.Repeat("00", 32) {
			t.Errorf("entry %d still carries a private key after a public read", i+1)
		}
		// The identity survives: that is what the caller asked for.
		if e.Address == "" || e.Address != full.Nodes[i].Address {
			t.Errorf("entry %d lost its address: %q vs %q", i+1, e.Address, full.Nodes[i].Address)
		}
		if e.PublicKey != full.Nodes[i].PublicKey {
			t.Errorf("entry %d lost its devp2p public key", i+1)
		}
	}
	if len(pub.Network.Validators) != len(full.Network.Validators) {
		t.Errorf("the network's validator list did not survive the public read")
	}
}

// TestLoadPublicPreset_MarshalsWithoutASecret: the reason to drop the key rather
// than trust callers is that a ring gets logged and persisted. Whatever a public
// read returns must not carry one into a file.
func TestLoadPublicPreset_MarshalsWithoutASecret(t *testing.T) {
	dir := ringDir(t)
	full, err := store.LoadPreset(dir)
	if err != nil {
		t.Fatalf("LoadPreset: %v", err)
	}
	pub, err := store.LoadPublicPreset(dir)
	if err != nil {
		t.Fatalf("LoadPublicPreset: %v", err)
	}
	raw, err := json.Marshal(pub.Nodes)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, e := range full.Nodes {
		if hex := e.Nodekey.Hex(); hex != "" && strings.Contains(strings.ToLower(string(raw)), hex) {
			t.Fatalf("a public ring marshalled a private key (%s…)", hex[:8])
		}
	}
}

// TestPublicEntry_CannotBeVerified: verification re-derives the identity from
// the key, so a public-only entry has nothing to verify against. It has to say
// so rather than derive from the zero key and report a mismatch, which reads like
// corrupted material.
func TestPublicEntry_CannotBeVerified(t *testing.T) {
	pub, err := store.LoadPublicPreset(ringDir(t))
	if err != nil {
		t.Fatalf("LoadPublicPreset: %v", err)
	}
	err = pub.Nodes[0].Verify()
	if err == nil {
		t.Fatal("a public-only entry reported itself verified")
	}
	if !strings.Contains(err.Error(), "public-only") {
		t.Errorf("the refusal should say the read did not fetch a key: %v", err)
	}
}

// TestLoadPublicPresetAt_GoesThroughTheStore keeps the remote form usable: the
// public read is the one an identity question on a server should take.
func TestLoadPublicPresetAt_GoesThroughTheStore(t *testing.T) {
	pub, err := store.LoadPublicPresetAt(context.Background(), nil, ringDir(t))
	if err != nil {
		t.Fatalf("LoadPublicPresetAt: %v", err)
	}
	if len(pub.Nodes) == 0 || pub.Nodes[0].Address == "" {
		t.Fatal("the public read through a store returned no identities")
	}
}
