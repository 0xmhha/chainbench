package engine

import (
	"strings"
	"testing"

	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// TestArmSpecs_CanonicalRolesLaunchIdentically is the gate the spelling
// migration must pass before the composition may emit the canonical role
// words: a plan written with bp/en must produce the
// same launch arguments and the same config as the same plan written with
// validator/endpoint. Anything that differs is a decision still keyed on a word
// rather than on the role.
func TestArmSpecs_CanonicalRolesLaunchIdentically(t *testing.T) {
	plugin := registry.StaticPlugin{
		M: registry.Manifest{
			ID: "stablenet", Binary: "go-stablenet", MinerRecommit: "duration",
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}
	preset := keyring.Preset{
		Network: keyring.Network{Validators: []string{"0xnode1"}},
		Nodes: []keyring.Entry{
			{Index: 1, Identity: keyring.Identity{PublicKey: "aa11", Address: "0xnode1"}},
			{Index: 2, Identity: keyring.Identity{PublicKey: "bb22", Address: "0xnode2"}},
		},
	}
	plan := func(producer, endpoint node.Role) driver.Plan {
		return driver.Plan{
			DataRoot: "/d",
			Nodes: []driver.NodeSpec{
				{Index: 1, Role: producer, Host: "127.0.0.1", DataDir: "/d/node1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}},
				{Index: 2, Role: endpoint, Host: "127.0.0.1", DataDir: "/d/node2", Ports: node.Endpoints{P2P: 31010, HTTP: 8610}},
			},
		}
	}

	legacy, err := armSpecs(plugin, preset, plan(node.RoleValidator, node.RoleEndpoint), "go-stablenet", "/keys", "", nil)
	if err != nil {
		t.Fatalf("armSpecs(legacy): %v", err)
	}
	canonical, err := armSpecs(plugin, preset, plan(node.RoleBP, node.RoleEN), "go-stablenet", "/keys", "", nil)
	if err != nil {
		t.Fatalf("armSpecs(canonical): %v", err)
	}

	for i := range legacy {
		if got, want := strings.Join(canonical[i].Args, " "), strings.Join(legacy[i].Args, " "); got != want {
			t.Fatalf("node%d args differ\n canonical: %s\n legacy:    %s", i+1, got, want)
		}
		if got, want := string(canonical[i].ConfigContent), string(legacy[i].ConfigContent); got != want {
			t.Fatalf("node%d config differs\n canonical:\n%s\n legacy:\n%s", i+1, got, want)
		}
	}
	// The producer must actually be armed to seal — otherwise both spellings
	// could be identically wrong.
	if !strings.Contains(strings.Join(canonical[0].Args, " "), "--unlock") {
		t.Fatalf("producer not armed with --unlock: %v", canonical[0].Args)
	}
	if !strings.Contains(string(canonical[0].ConfigContent), "[Eth.Miner]") {
		t.Fatal("producer config has no miner section")
	}
	if strings.Contains(string(canonical[1].ConfigContent), "[Eth.Miner]") {
		t.Fatal("endpoint config should have no miner section")
	}
}

// TestCountValidators_CanonicalSpelling: this number sizes the genesis
// validator set. Reading zero from a canonical plan would build a chain with no
// producers.
func TestCountValidators_CanonicalSpelling(t *testing.T) {
	reqs := []place.NodeReq{
		{Role: node.RoleBP},
		{Role: node.RoleValidator},
		{Role: node.RoleEN},
		{Role: node.RolePN},
	}
	if got := countValidators(reqs); got != 2 {
		t.Fatalf("countValidators = %d, want 2 (both spellings of the producing role)", got)
	}
	specs := driver.Plan{Nodes: []driver.NodeSpec{
		{Index: 1, Role: node.RoleBP}, {Index: 2, Role: node.RoleEndpoint}, {Index: 3, Role: node.RoleValidator},
	}}
	if got := planValidatorCount(specs); got != 2 {
		t.Fatalf("planValidatorCount = %d, want 2", got)
	}
}
