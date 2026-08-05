package genesis_test

import (
	"bytes"
	"encoding/json"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

var testInputs = genesis.Inputs{
	Validators: []string{"0xc17d493883eaa3b4cceb0f214b273392d562f9d8"},
	BLSKeys:    []string{"0xa00eb14731965f294993a2df1cf09e5b826193a41853fd9aaa7195922b8461c97b215a1181d4ddecc9f5981fdd47556f"},
	ExtraData:  "0xdeadbeef",
	Members:    []string{"0xc17d493883eaa3b4cceb0f214b273392d562f9d8"},
	Alloc:      json.RawMessage(`{"c17d493883eaa3b4cceb0f214b273392d562f9d8":{"balance":"0x64"}}`),
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

	// The anzeon engine requires a full systemContracts section; assert all
	// five contracts are present and the dynamic lists were substituted with
	// the supplied validators/BLS keys/members (go-stablenet
	// params/config_wbft.go:96-111 rejects a genesis missing any of these).
	var anz struct {
		SystemContracts struct {
			GovValidator      *scView `json:"govValidator"`
			NativeCoinAdapter *scView `json:"nativeCoinAdapter"`
			GovMasterMinter   *scView `json:"govMasterMinter"`
			GovMinter         *scView `json:"govMinter"`
			GovCouncil        *scView `json:"govCouncil"`
		} `json:"systemContracts"`
	}
	if err := json.Unmarshal(g.Config.Anzeon, &anz); err != nil {
		t.Fatalf("anzeon decode: %v", err)
	}
	sc := anz.SystemContracts
	for name, c := range map[string]*scView{
		"govValidator": sc.GovValidator, "nativeCoinAdapter": sc.NativeCoinAdapter,
		"govMasterMinter": sc.GovMasterMinter, "govMinter": sc.GovMinter, "govCouncil": sc.GovCouncil,
	} {
		if c == nil {
			t.Fatalf("systemContracts.%s missing", name)
		}
	}
	if got := sc.GovValidator.Params["validators"]; got != testInputs.Validators[0] {
		t.Errorf("govValidator.validators = %q, want %q", got, testInputs.Validators[0])
	}
	if got := sc.GovValidator.Params["blsPublicKeys"]; got != testInputs.BLSKeys[0] {
		t.Errorf("govValidator.blsPublicKeys = %q, want %q", got, testInputs.BLSKeys[0])
	}
	if got := sc.GovCouncil.Params["members"]; got != testInputs.Members[0] {
		t.Errorf("govCouncil.members = %q, want %q", got, testInputs.Members[0])
	}
	// No placeholder token may survive substitution.
	if bytes.Contains(g.Config.Anzeon, []byte("__SC_")) {
		t.Errorf("unsubstituted system-contract placeholder in anzeon:\n%s", g.Config.Anzeon)
	}

	// The preset alloc must land in the genesis alloc field (funds accounts).
	var full struct {
		Alloc map[string]struct {
			Balance string `json:"balance"`
		} `json:"alloc"`
	}
	if err := json.Unmarshal(b, &full); err != nil {
		t.Fatalf("alloc decode: %v", err)
	}
	if acct, ok := full.Alloc["c17d493883eaa3b4cceb0f214b273392d562f9d8"]; !ok || acct.Balance != "0x64" {
		t.Errorf("alloc not substituted: %+v", full.Alloc)
	}
	if bytes.Contains(b, []byte("__ALLOC_JSON__")) {
		t.Errorf("unsubstituted alloc placeholder in genesis")
	}
}

// scView decodes one system contract for assertions.
type scView struct {
	Address string            `json:"address"`
	Version string            `json:"version"`
	Params  map[string]string `json:"params"`
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
