package derive_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
)

// This package had no test of its own. That mattered more here than the line
// count suggests: the BLS path has to reproduce what blst's blst_keygen
// produces, and a derivation that drifts still yields a well-formed key — one
// that every node rejects at seal time, so the fault presents as a consensus
// problem and the search starts in the wrong subsystem.
//
// keys/preset is the oracle. It ships each node's nodekey next to the address,
// devp2p public key, BLS public key and proof of possession that key produced,
// so re-deriving from the nodekey and comparing is a byte-for-byte check
// against known-good output — the check Derive's own doc comment claimed was
// happening.

// presetNode is one entry of keys/preset/metadata.json.
type presetNode struct {
	Index        int    `json:"index"`
	NodeKey      string `json:"nodekey"`
	PublicKey    string `json:"publicKey"`
	Address      string `json:"address"`
	BLSPublicKey string `json:"blsPublicKey"`
	BLSPoP       string `json:"blsPoP"`
}

func loadPreset(t *testing.T) []presetNode {
	t.Helper()
	// ../../../../keys/preset from internal/core/keyring/derive.
	path := filepath.Join("..", "..", "..", "..", "keys", "preset", "metadata.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the preset fixture is the oracle for this package: %v", err)
	}
	var meta struct {
		Nodes []presetNode `json:"nodes"`
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatalf("preset metadata: %v", err)
	}
	if len(meta.Nodes) == 0 {
		t.Fatal("preset metadata declares no nodes")
	}
	return meta.Nodes
}

// TestDerive_ReproducesTheShippedPreset is the pin. Every field the preset
// records must come back out of Derive unchanged.
func TestDerive_ReproducesTheShippedPreset(t *testing.T) {
	for _, n := range loadPreset(t) {
		t.Run(n.Address, func(t *testing.T) {
			k, err := derive.ParsePrivateKey(n.NodeKey)
			if err != nil {
				t.Fatalf("ParsePrivateKey: %v", err)
			}
			id, err := derive.Derive(k, derive.WithBLS)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			if !strings.EqualFold(id.Address, n.Address) {
				t.Errorf("address\n got %s\nwant %s", id.Address, n.Address)
			}
			if !strings.EqualFold(id.PublicKey, n.PublicKey) {
				t.Errorf("devp2p public key\n got %s\nwant %s", id.PublicKey, n.PublicKey)
			}
			if id.BLS == nil {
				t.Fatal("WithBLS produced no BLS material")
			}
			// The two that would drift silently. A BLS key that does not match
			// blst's output is well-formed and useless.
			if !strings.EqualFold(id.BLS.PublicKey, n.BLSPublicKey) {
				t.Errorf("BLS public key does not match blst's output\n got %s\nwant %s", id.BLS.PublicKey, n.BLSPublicKey)
			}
			if !strings.EqualFold(id.BLS.PoP, n.BLSPoP) {
				t.Errorf("BLS proof of possession\n got %s\nwant %s", id.BLS.PoP, n.BLSPoP)
			}
		})
	}
}

// TestDerive_IsDeterministic states the property the preset check depends on:
// the same key must give the same identity every time. A PoP built over fresh
// randomness would pass a single comparison against a value derived in the same
// run and fail against the fixture, so this separates "wrong" from "unstable".
func TestDerive_IsDeterministic(t *testing.T) {
	n := loadPreset(t)[0]
	k, err := derive.ParsePrivateKey(n.NodeKey)
	if err != nil {
		t.Fatal(err)
	}
	first, err := derive.Derive(k, derive.WithBLS)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		again, err := derive.Derive(k, derive.WithBLS)
		if err != nil {
			t.Fatal(err)
		}
		if again != first && *again.BLS != *first.BLS {
			t.Fatalf("run %d differs from the first", i)
		}
		if again.Address != first.Address || again.PublicKey != first.PublicKey ||
			again.BLS.PublicKey != first.BLS.PublicKey || again.BLS.PoP != first.BLS.PoP {
			t.Fatalf("run %d differs from the first:\n %+v\n %+v", i, *again.BLS, *first.BLS)
		}
	}
}

// TestDerive_AccountOnlyLeavesBLSAbsent pins the distinction the type makes on
// purpose: a chain without BLS has no BLS key, and that is absence rather than
// a zero value, so a consumer cannot mistake "not derived" for "derived to
// zeroes" and write zeroes into a genesis.
func TestDerive_AccountOnlyLeavesBLSAbsent(t *testing.T) {
	n := loadPreset(t)[0]
	k, err := derive.ParsePrivateKey(n.NodeKey)
	if err != nil {
		t.Fatal(err)
	}
	id, err := derive.Derive(k, derive.AccountOnly)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if id.BLS != nil {
		t.Errorf("AccountOnly derived BLS material: %+v", *id.BLS)
	}
	// The cheap half must still be identical to the WithBLS run — asking for
	// less must not change what you get.
	if !strings.EqualFold(id.Address, n.Address) || !strings.EqualFold(id.PublicKey, n.PublicKey) {
		t.Errorf("AccountOnly changed the account half\n got %s / %s\nwant %s / %s",
			id.Address, id.PublicKey, n.Address, n.PublicKey)
	}
}

// TestDerive_ShapesAreWhatConsumersParse guards the encodings downstream code
// slices by hand: an enode takes the 128-hex public key with no prefix, while
// the BLS values are 0x-prefixed compressed points of a fixed width.
func TestDerive_ShapesAreWhatConsumersParse(t *testing.T) {
	n := loadPreset(t)[0]
	k, _ := derive.ParsePrivateKey(n.NodeKey)
	id, err := derive.Derive(k, derive.WithBLS)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(id.PublicKey, "0x") {
		t.Error("the devp2p public key must be bare hex: it is concatenated into an enode URL")
	}
	if len(id.PublicKey) != 128 {
		t.Errorf("devp2p public key is %d hex chars, want 128", len(id.PublicKey))
	}
	// 48-byte G1 point and 96-byte G2 point, both 0x-prefixed.
	if !strings.HasPrefix(id.BLS.PublicKey, "0x") || len(id.BLS.PublicKey) != 2+96 {
		t.Errorf("BLS public key %q is not a 0x-prefixed 48-byte point", id.BLS.PublicKey)
	}
	if !strings.HasPrefix(id.BLS.PoP, "0x") || len(id.BLS.PoP) != 2+192 {
		t.Errorf("BLS PoP %q is not a 0x-prefixed 96-byte point", id.BLS.PoP)
	}
}

// TestParsePrivateKey_AcceptsBothSpellingsAndRefusesJunk covers the one input
// the package takes from a file an operator may have edited.
func TestParsePrivateKey_AcceptsBothSpellingsAndRefusesJunk(t *testing.T) {
	n := loadPreset(t)[0]
	bare, err := derive.ParsePrivateKey(n.NodeKey)
	if err != nil {
		t.Fatalf("bare hex: %v", err)
	}
	prefixed, err := derive.ParsePrivateKey("0x" + n.NodeKey)
	if err != nil {
		t.Fatalf("0x-prefixed hex: %v", err)
	}
	if bare.Bytes() == nil || string(bare.Bytes()) != string(prefixed.Bytes()) {
		t.Error("the 0x prefix changed the key")
	}
	// Whitespace is what a nodekey file carries when it ends with a newline.
	if _, err := derive.ParsePrivateKey(" " + n.NodeKey + "\n"); err != nil {
		t.Errorf("a nodekey read from a file ends with a newline: %v", err)
	}
	for name, bad := range map[string]string{
		"empty":      "",
		"too short":  "0a0b",
		"not hex":    strings.Repeat("z", 64),
		"one nibble": n.NodeKey[:63],
		"all zeroes": strings.Repeat("0", 64),
	} {
		if _, err := derive.ParsePrivateKey(bad); err == nil {
			t.Errorf("%s: ParsePrivateKey(%q) accepted an unusable key", name, bad)
		}
	}
}
