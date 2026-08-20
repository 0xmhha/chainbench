package keygen_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/keygen"
)

// presetNode is one entry of the shipped keys/preset/metadata.json.
type presetNode struct {
	Index        int    `json:"index"`
	Nodekey      string `json:"nodekey"`
	PublicKey    string `json:"publicKey"`
	Address      string `json:"address"`
	BLSPublicKey string `json:"blsPublicKey"`
	BLSPoP       string `json:"blsPoP"`
}

// presetNodes reads the shipped fixture. DeriveIdentity used to parse the
// go-wbft bootnode tool's output, so checking the in-process replacement
// against the very values that tool produced is what makes this a replacement
// rather than a second implementation.
func presetNodes(t *testing.T) []presetNode {
	t.Helper()
	path := filepath.Join("..", "..", "keys", "preset", "metadata.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var meta struct {
		Nodes []presetNode `json:"nodes"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if len(meta.Nodes) == 0 {
		t.Fatalf("%s has no nodes", path)
	}
	return meta.Nodes
}

func TestDeriveIdentity_MatchesShippedPreset(t *testing.T) {
	for _, want := range presetNodes(t) {
		t.Run(want.Address, func(t *testing.T) {
			got, err := keygen.DeriveIdentity(want.Nodekey)
			if err != nil {
				t.Fatalf("DeriveIdentity: %v", err)
			}
			if got.PublicKey != want.PublicKey {
				t.Errorf("publicKey\n got %s\nwant %s", got.PublicKey, want.PublicKey)
			}
			if !strings.EqualFold(got.Address, want.Address) {
				t.Errorf("address\n got %s\nwant %s", got.Address, want.Address)
			}
			if got.BLSPubKey != want.BLSPublicKey {
				t.Errorf("blsPubKey\n got %s\nwant %s", got.BLSPubKey, want.BLSPublicKey)
			}
			if got.BLSPoP != want.BLSPoP {
				t.Errorf("blsPoP\n got %s\nwant %s", got.BLSPoP, want.BLSPoP)
			}
		})
	}
}

func TestDeriveIdentity_AcceptsPrefixedAndRejectsGarbage(t *testing.T) {
	valid := presetNodes(t)[0].Nodekey
	if _, err := keygen.DeriveIdentity("0x" + valid); err != nil {
		t.Errorf("0x-prefixed key rejected: %v", err)
	}
	for _, bad := range []string{"", "0x", "zz", valid[:60]} {
		if _, err := keygen.DeriveIdentity(bad); err == nil {
			t.Errorf("accepted invalid key %q", bad)
		}
	}
}

// TestGeneratePreset_RunsNoExternalProcess is the K1 gate. Generation used to
// require the go-wbft bootnode binary; PATH is emptied here so that any
// surviving shell-out fails rather than silently finding a binary the
// developer happens to have installed.
func TestGeneratePreset_RunsNoExternalProcess(t *testing.T) {
	t.Setenv("PATH", "")
	if _, err := exec.LookPath("bootnode"); err == nil {
		t.Fatal("PATH is not empty; the test cannot prove a binary was unused")
	}

	const nodes = 2
	out := t.TempDir()
	meta, err := keygen.GeneratePreset(keygen.PresetOpts{
		Nodes:      nodes,
		Validators: nodes,
		Out:        out,
		Password:   "1",
		Balance:    "0x1",
	}, nil)
	if err != nil {
		t.Fatalf("GeneratePreset with no PATH: %v", err)
	}
	if len(meta.Nodes) != nodes || len(meta.Validators) != nodes {
		t.Fatalf("got %d nodes / %d validators", len(meta.Nodes), len(meta.Validators))
	}
	// Extra-data is not stored: it encodes the validator set and would go stale
	// the moment a network used a subset of it.
	//
	// Every generated node must re-derive from the nodekey written to disk, or
	// the metadata and the datadir describe different identities.
	for _, n := range meta.Nodes {
		again, err := keygen.DeriveIdentity(n.Nodekey)
		if err != nil {
			t.Fatalf("re-derive node %d: %v", n.Index, err)
		}
		if again.Address != n.Address || again.BLSPubKey != n.BLSPubKey {
			t.Errorf("node %d does not re-derive from its own nodekey", n.Index)
		}
		onDisk, err := os.ReadFile(filepath.Join(out, "node"+strconv.Itoa(n.Index), "nodekey"))
		if err != nil {
			t.Fatalf("read nodekey: %v", err)
		}
		if strings.TrimSpace(string(onDisk)) != n.Nodekey {
			t.Errorf("node %d: metadata nodekey differs from the file on disk", n.Index)
		}
	}
}
