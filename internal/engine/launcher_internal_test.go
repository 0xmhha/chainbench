package engine

import (
	"strings"
	"testing"

	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

func TestArmSpecs(t *testing.T) {
	plugin := registry.StaticPlugin{
		M: registry.Manifest{
			ID: "stablenet", Binary: "go-stablenet", MinerRecommit: "duration",
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}
	preset := keys.Preset{
		Validators: []string{"0xval1"},
		Nodes: []keys.NodeKey{
			{Index: 1, PublicKey: "aa11", Address: "0xval1"},
			{Index: 2, PublicKey: "bb22", Address: "0xen2"},
		},
	}
	plan := setup.Plan{
		DataRoot: "/d",
		Nodes: []driver.NodeSpec{
			{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", DataDir: "/d/node1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}},
			{Index: 2, Role: node.RoleEndpoint, Host: "127.0.0.1", DataDir: "/d/node2", Ports: node.Endpoints{P2P: 31010, HTTP: 8610}},
		},
	}

	specs := armSpecs(plugin, preset, plan, "go-stablenet", "/keys")
	if len(specs) != 2 {
		t.Fatalf("specs = %d, want 2", len(specs))
	}

	v := specs[0]
	if v.Binary != "go-stablenet" {
		t.Fatalf("binary = %q", v.Binary)
	}
	if len(v.ConfigContent) == 0 {
		t.Fatal("validator has no rendered config content")
	}
	if !argsHas(v.Args, "--nodekey") {
		t.Fatalf("validator missing --nodekey: %v", v.Args)
	}
	if !argsHas(v.Args, "--unlock", "0xval1", "--miner.etherbase") {
		t.Fatalf("validator missing unlock/etherbase: %v", v.Args)
	}
	// The static-node enode must use the plan's (allocator) p2p port, not a
	// preset default, so peering matches the launched layout.
	if !strings.Contains(string(v.ConfigContent), "31000") {
		t.Fatalf("static node should use plan p2p port 31000:\n%s", v.ConfigContent)
	}

	// An endpoint gets a nodekey but no validator unlock.
	if argsHas(specs[1].Args, "--unlock") {
		t.Fatalf("endpoint should not unlock an account: %v", specs[1].Args)
	}
	if !argsHas(specs[1].Args, "--nodekey") {
		t.Fatalf("endpoint missing --nodekey: %v", specs[1].Args)
	}
}

// argsHas reports whether every val appears somewhere in args.
func argsHas(args []string, vals ...string) bool {
	set := make(map[string]bool, len(args))
	for _, a := range args {
		set[a] = true
	}
	for _, v := range vals {
		if !set[v] {
			return false
		}
	}
	return true
}
