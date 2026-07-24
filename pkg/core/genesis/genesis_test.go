package genesis_test

import (
	"encoding/json"
	"testing"

	_ "github.com/0xmhha/chainbench/pkg/chains/all"
	"github.com/0xmhha/chainbench/pkg/core/genesis"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

var testInputs = genesis.Inputs{
	Validators: []string{"0xc17d493883eaa3b4cceb0f214b273392d562f9d8"},
	BLSKeys:    []string{"0xa00eb14731965f294993a2df1cf09e5b826193a41853fd9aaa7195922b8461c97b215a1181d4ddecc9f5981fdd47556f"},
	ExtraData:  "0xdeadbeef",
}

// decodeGenesis parses just enough of a genesis to check chain id and which
// engine field (anzeon vs croissant) is present.
type genesisView struct {
	Config struct {
		ChainID   int64           `json:"chainId"`
		Anzeon    json.RawMessage `json:"anzeon"`
		Croissant json.RawMessage `json:"croissant"`
	} `json:"config"`
	ExtraData string `json:"extraData"`
}

func TestBuild_StablenetIsAnzeon(t *testing.T) {
	p, _ := registry.Get("stablenet")
	b, err := genesis.Build(p, testInputs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var g genesisView
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatalf("invalid genesis JSON: %v\n%s", err, b)
	}
	if g.Config.ChainID != 8283 {
		t.Errorf("chainId: %d, want 8283", g.Config.ChainID)
	}
	if len(g.Config.Anzeon) == 0 {
		t.Error("stablenet genesis must have anzeon engine field")
	}
	if len(g.Config.Croissant) != 0 {
		t.Error("stablenet genesis must NOT have croissant field")
	}
	if g.ExtraData != "0xdeadbeef" {
		t.Errorf("extraData: %s", g.ExtraData)
	}
}

func TestBuild_WbftIsCroissant(t *testing.T) {
	p, _ := registry.Get("wbft")
	b, err := genesis.Build(p, testInputs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var g genesisView
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatalf("invalid genesis JSON: %v\n%s", err, b)
	}
	if g.Config.ChainID != 8284 {
		t.Errorf("chainId: %d, want 8284", g.Config.ChainID)
	}
	if len(g.Config.Croissant) == 0 {
		t.Error("wbft genesis must have croissant engine field")
	}
	if len(g.Config.Anzeon) != 0 {
		t.Error("wbft genesis must NOT have anzeon field")
	}
	// The validator must land inside the croissant.init set.
	var croissant struct {
		Init struct {
			Validators []string `json:"validators"`
		} `json:"init"`
	}
	if err := json.Unmarshal(g.Config.Croissant, &croissant); err != nil {
		t.Fatalf("croissant decode: %v", err)
	}
	if len(croissant.Init.Validators) != 1 || croissant.Init.Validators[0] != testInputs.Validators[0] {
		t.Errorf("croissant validators: %v", croissant.Init.Validators)
	}
}

func TestBuild_WemixBaseGenesis(t *testing.T) {
	p, _ := registry.Get("wemix")
	b, err := genesis.Build(p, genesis.Inputs{Coinbase: "0xb4388353fd0f3b3a017e09f2b857052ff219e663"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var g genesisView
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatalf("invalid genesis JSON: %v\n%s", err, b)
	}
	if g.Config.ChainID != 8285 {
		t.Errorf("chainId: %d, want 8285", g.Config.ChainID)
	}
	// poa genesis carries neither wbft-family engine field (validators come
	// from bootstrap, not genesis).
	if len(g.Config.Anzeon) != 0 || len(g.Config.Croissant) != 0 {
		t.Error("wemix base genesis must not have anzeon/croissant engine fields")
	}
	var full struct {
		Coinbase string `json:"coinbase"`
	}
	_ = json.Unmarshal(b, &full)
	if full.Coinbase != "0xb4388353fd0f3b3a017e09f2b857052ff219e663" {
		t.Errorf("coinbase: %s", full.Coinbase)
	}
}
