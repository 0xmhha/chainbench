package upgrade_test

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// a minimal pre-fork from-chain (wemix) base genesis: chain id + petersburg.
const fromGenesis = `{"config":{"chainId":8285,"petersburgBlock":0},"alloc":{}}`

func toInputs() genesis.Inputs {
	return genesis.Inputs{
		Validators: []string{
			"0xc17d493883eaa3b4cceb0f214b273392d562f9d8",
			"0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c",
			"0x8c4a10b9108d49b9d23f764464090831d9c17764",
			"0x8eb79036bc0f3aba136ef18b3a2fb8c1188939a6",
		},
		BLSKeys: []string{
			"0xa00eb14731965f294993a2df1cf09e5b826193a41853fd9aaa7195922b8461c97b215a1181d4ddecc9f5981fdd47556f",
			"0x929af9896092b61db0ead8931feaed3f77058825c3c82f20fd9557a244b8732303f2136b6acd06ba7e1b861bf5514449",
			"0x8c7faed16ab71ca6a3f8d82d643f6502e4c2dc3ecf48e86ed4d5dba42e67240313b84e911ad1bbf5783263284f09c1d0",
			"0xa63e51dd59a291b3cef9804c0790a0f95285297fb4fe141a587c6dda0784822c27e6e6b9404754e679ae4cca62d0ce4a",
		},
		ExtraData: "0xdeadbeef",
		Members:   []string{"0xc17d493883eaa3b4cceb0f214b273392d562f9d8"},
		Alloc:     json.RawMessage(`{}`),
	}
}

func goodInputs() upgrade.Inputs {
	return upgrade.Inputs{
		Roles:         upgrade.Roles{Producers: 1, Validators: 4},
		NetworkID:     8285,
		ForkBlock:     big.NewInt(20),
		FromGenesis:   []byte(fromGenesis),
		ToGenesis:     toInputs(),
		ProducerAddrs: []string{"0xf9593d5b8c0e6c1e2f3a4b5c6d7e8f9012345678"},
		P2PBase:       30010, P2PStep: 10, RPCBase: 40010, RPCStep: 10,
	}
}

func plugins(t *testing.T) (registry.ChainPlugin, registry.ChainPlugin) {
	t.Helper()
	from, err := registry.Get("wemix")
	if err != nil {
		t.Fatal(err)
	}
	to, err := registry.Get("wbft")
	if err != nil {
		t.Fatal(err)
	}
	return from, to
}

func TestBuildPlan_OK(t *testing.T) {
	from, to := plugins(t)
	p, err := upgrade.BuildPlan(from, to, goodInputs())
	if err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
	if len(p.Nodes) != 5 {
		t.Fatalf("want 5 nodes, got %d", len(p.Nodes))
	}
	// role assignment: first is the from-chain producer, rest to-chain validators.
	if !p.Nodes[0].Producer || p.Nodes[0].Chain != "wemix" {
		t.Errorf("node0 should be a wemix producer: %+v", p.Nodes[0])
	}
	for _, n := range p.Nodes[1:] {
		if n.Producer || n.Chain != "wbft" {
			t.Errorf("node %d should be a wbft validator: %+v", n.Index, n)
		}
		if n.NetworkID != 8285 {
			t.Errorf("node %d networkid not uniform: %d", n.Index, n.NetworkID)
		}
	}
	// the fork section from the to-chain genesis was merged in, with the block.
	if err := genesis.ValidateForks(p.Genesis); err != nil {
		t.Errorf("merged genesis fails fork check: %v", err)
	}
	var g struct {
		Config struct {
			CroissantBlock *big.Int        `json:"croissantBlock"`
			Croissant      json.RawMessage `json:"croissant"`
		} `json:"config"`
	}
	if err := json.Unmarshal(p.Genesis, &g); err != nil {
		t.Fatal(err)
	}
	if g.Config.CroissantBlock == nil || g.Config.CroissantBlock.Int64() != 20 {
		t.Errorf("croissantBlock not set to fork block: %v", g.Config.CroissantBlock)
	}
	if len(g.Config.Croissant) == 0 {
		t.Error("croissant section not merged")
	}
}

func TestBuildPlan_Rejects(t *testing.T) {
	from, to := plugins(t)
	cases := map[string]func(*upgrade.Inputs){
		"too few validators": func(in *upgrade.Inputs) { in.Roles.Validators = 3 },
		"no producer":        func(in *upgrade.Inputs) { in.Roles.Producers = 0 },
		"nil fork block":     func(in *upgrade.Inputs) { in.ForkBlock = nil },
		"empty from genesis": func(in *upgrade.Inputs) { in.FromGenesis = nil },
		"from genesis missing petersburg": func(in *upgrade.Inputs) {
			in.FromGenesis = []byte(`{"config":{"chainId":8285}}`)
		},
		"producer overlaps validator": func(in *upgrade.Inputs) {
			in.ProducerAddrs = []string{in.ToGenesis.Validators[0]}
		},
	}
	for name, mut := range cases {
		in := goodInputs()
		mut(&in)
		if _, err := upgrade.BuildPlan(from, to, in); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestBuildPlan_WrongPair(t *testing.T) {
	// wbft declares no upgrade; using it as the from-chain must fail loudly.
	from, to := plugins(t)
	if _, err := upgrade.BuildPlan(to, from, goodInputs()); err == nil ||
		!strings.Contains(err.Error(), "no upgrade") {
		t.Errorf("wbft-as-from should be rejected, got %v", err)
	}
}
