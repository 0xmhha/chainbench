package chainsetup

import (
	"encoding/json"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register the chains a fork hands over between
	"github.com/0xmhha/chainbench/internal/core/node"
)

// The genesis a fork is scheduled on: a wemix network's, reduced to the fields
// the fork touches.
const forkBase = `{
  "config": {"chainId": 1111, "homesteadBlock": 0, "eip150Block": 0, "eip155Block": 0,
             "eip158Block": 0, "byzantiumBlock": 0, "constantinopleBlock": 0,
             "petersburgBlock": 0, "istanbulBlock": 0, "pangyoBlock": 0,
             "applepieBlock": 0, "briocheBlock": 0},
  "nonce": "0x42", "timestamp": "0x00", "gasLimit": "105000000",
  "difficulty": "0x1", "extraData": "0x00", "rewards": "0x",
  "alloc": {}
}`

// forkWorkspace is a composed wemix network whose second build is wbft: the
// shape every hardfork case has, with nothing in it but what the fork reads.
func forkWorkspace() *Workspace {
	return &Workspace{state: State{
		Chain:        "wemix",
		KeysDir:      "../../presets/keys",
		Binaries:     map[string]string{"from": "gwemix", "to": "gwbft"},
		BinaryChains: map[string]string{"to": "wbft"},
		Nodes: []node.Record{
			{Index: 1, Binary: "from"},
			{Index: 2, Binary: "to"},
			{Index: 3, Binary: "to"},
			{Index: 4, Binary: "to"},
			{Index: 5, Binary: "to"},
		},
	}}
}

func forkConfigOf(t *testing.T, gen []byte) map[string]json.RawMessage {
	t.Helper()
	var doc struct {
		Config map[string]json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(gen, &doc); err != nil {
		t.Fatalf("the fork step produced a genesis that is not JSON: %v", err)
	}
	return doc.Config
}

// TestApplyFork_CarriedByGenesisPutsTheSectionInTheGenesis.
//
// The default route, and the cheaper one. Both the activation block and the
// fork's own configuration go into the genesis every node initializes from; the
// pre-fork build ignores a section its config has no field for, because a
// genesis document is read by a decoder that skips what it cannot place.
func TestApplyFork_CarriedByGenesisPutsTheSectionInTheGenesis(t *testing.T) {
	w := forkWorkspace()
	gen, configs, err := w.applyFork([]byte(forkBase), GenesisFork{Name: "croissant", At: 100, Binary: "to"})
	if err != nil {
		t.Fatalf("applyFork: %v", err)
	}
	cfg := forkConfigOf(t, gen)
	if string(cfg["croissantBlock"]) != "100" {
		t.Errorf("croissantBlock = %s, want 100", cfg["croissantBlock"])
	}
	if len(cfg["croissant"]) == 0 {
		t.Error("the genesis carries no croissant section, so the post-fork build has no validator set")
	}
	if len(configs) != 0 {
		t.Errorf("a genesis-carried fork also wrote %d config(s)", len(configs))
	}
}

// TestApplyFork_CarriedByConfigLeavesTheGenesisWithOnlyTheBlock.
//
// The other route. Every node initializes from one genesis, which still has to
// say when the fork happens — that is what stops the pre-fork build sealing.
// The section itself goes to the post-fork build's config, so the network has
// one genesis document and two shapes of config instead of the reverse.
func TestApplyFork_CarriedByConfigLeavesTheGenesisWithOnlyTheBlock(t *testing.T) {
	w := forkWorkspace()
	gen, configs, err := w.applyFork([]byte(forkBase), GenesisFork{
		Name: "croissant", At: 100, Binary: "to", Carrier: ForkInConfig,
	})
	if err != nil {
		t.Fatalf("applyFork: %v", err)
	}
	cfg := forkConfigOf(t, gen)
	if string(cfg["croissantBlock"]) != "100" {
		t.Errorf("croissantBlock = %s, want 100 — without it the pre-fork build never stops sealing", cfg["croissantBlock"])
	}
	if len(cfg["croissant"]) != 0 {
		t.Error("the shared genesis still carries the croissant section, so it is not one genesis after all")
	}

	toml, ok := configs["to"]
	if !ok {
		t.Fatalf("no genesis config for the binary that seals after the fork; got %v", configs)
	}
	body := string(toml)
	for _, want := range []string{
		"[Eth.Genesis]",                  // the whole genesis, not a fragment
		"[Eth.Genesis.Config.Croissant]", // with the section the genesis no longer has
		"CroissantBlock = 100",           // and the block, so both files agree
		"ChainID = 1111",                 // spelled the way the config decoder matches
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the genesis config is missing %q\n%s", want, body)
		}
	}
	// The chain declares the keys its config decoder has no field for; leaving
	// one in is a node that dies naming a field rather than one that runs.
	if strings.Contains(body, "Rewards") {
		t.Errorf("the genesis config carries a key core.Genesis has no field for\n%s", body)
	}
}

// TestApplyFork_AForkWithNowhereToLandIsRefusedOnEitherRoute: the section is
// built from the ring of the nodes running the post-fork binary, so a fork
// naming a binary no node runs would hand over to nobody.
func TestApplyFork_AForkWithNowhereToLandIsRefusedOnEitherRoute(t *testing.T) {
	for _, carrier := range []ForkCarrier{ForkInGenesis, ForkInConfig} {
		w := forkWorkspace()
		w.state.Nodes = []node.Record{{Index: 1, Binary: "from"}}
		_, _, err := w.applyFork([]byte(forkBase), GenesisFork{
			Name: "croissant", At: 100, Binary: "to", Carrier: carrier,
		})
		if err == nil {
			t.Errorf("carried by %s: a fork with no post-fork node was accepted", carrier)
		}
	}
}

// TestGenesisConfigFor_ANodeReadsItsOwnBinarysGenesisConfig mirrors genesisFor:
// the binary decides, a binary with none carries none, and an ordinary node's
// config says nothing about the genesis at all.
func TestGenesisConfigFor_ANodeReadsItsOwnBinarysGenesisConfig(t *testing.T) {
	w := &Workspace{state: State{
		GenesisPath:        "/data/genesis.json",
		GenesisConfigPaths: map[string]string{"to": "/data/genesis-to.toml"},
		Nodes: []node.Record{
			{Index: 1},
			{Index: 2, Binary: "to"},
			{Index: 3, Binary: "other"},
		},
	}}
	want := map[int]string{1: "", 2: "/data/genesis-to.toml", 3: ""}
	for _, ns := range w.state.Nodes {
		if got := w.genesisConfigFor(ns); got != want[ns.Index] {
			t.Errorf("node%d genesis config = %q, want %q", ns.Index, got, want[ns.Index])
		}
	}
}
