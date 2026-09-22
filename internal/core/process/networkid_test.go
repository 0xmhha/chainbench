package process_test

import (
	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/preset"
	"testing"
)

// The devp2p network id a node is told, and where it comes from.
//
// It was never on the command line at all, so each build inferred its own: a
// handoff's producer and its successors could be told nothing and end up on two
// networks that never peer. Emitting it fixed that, and then left a second
// question — whose id? — which these answer.

// TestNodeConfigArgv_NetworkIDFollowsTheChainID is the rule. A network's devp2p
// id is its chain id, so two builds composed into one network are told the same
// number whichever manifest each of them came with.
func TestNodeConfigArgv_NetworkIDFollowsTheChainID(t *testing.T) {
	plugin := registry.StaticPlugin{
		M: registry.Manifest{Dialect: "geth114",
			ID: "stablenet", Binary: "go-stablenet", ChainID: 8283, NetworkID: 8283,
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}
	keys := preset.Key{Nodes: []keyring.Entry{{Index: 1, Identity: derive.Identity{PublicKey: "aa", Address: "0x1"}}}}
	spec := process.NodeSpec{Index: 1, Role: node.RoleEN, Host: "127.0.0.1", DataDir: "/d/node1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}}

	cfg := process.NodeConfig(plugin, nodeconfig.NetworkOf(plugin, 0), keys, spec, "/keys", nil)
	args, err := nodeconfig.Argv(cfg)
	if err != nil {
		t.Fatalf("nodeconfig.Argv: %v", err)
	}
	if !argsHasPair(args, "--networkid", "8283") {
		t.Fatalf("argv does not carry the network id: %v", args)
	}

	// An operator's override arrives on a later layer and wins.
	args, err = nodeconfig.Argv(cfg, nodeconfig.Override{Key: nodeconfig.KeyNetworkID, Value: "99", Layer: nodeconfig.LayerEnv})
	if err != nil {
		t.Fatalf("nodeconfig.Argv with override: %v", err)
	}
	if !argsHasPair(args, "--networkid", "99") {
		t.Fatalf("override did not win: %v", args)
	}
}

// TestNetworkOf_AnOverriddenChainIDCarriesTheNetworkID: `--chain-id` changes the
// genesis the network runs on, and the devp2p id has to come with it. It did
// not — the id was read from the manifest and the override was left behind, so
// a node on a chain nobody else runs announced the number of the one it was
// derived from.
func TestNetworkOf_AnOverriddenChainIDCarriesTheNetworkID(t *testing.T) {
	plugin := registry.StaticPlugin{
		M:   registry.Manifest{ID: "stablenet", ChainID: 8283, NetworkID: 8283},
		Fam: wbftfam.New(),
	}
	got := nodeconfig.NetworkOf(plugin, 4242)
	if got.ChainID != 4242 || got.NetworkID != 4242 {
		t.Errorf("chain id 4242 gives %+v, want both 4242 — the devp2p id follows the chain", got)
	}
}

// TestNetworkOf_ADeclaredDifferenceIsKept is the only reason the manifest field
// exists. A chain whose devp2p id is genuinely not its chain id says so, and is
// taken at its word; a chain that repeats its chain id adds nothing and an
// override still carries.
func TestNetworkOf_ADeclaredDifferenceIsKept(t *testing.T) {
	plugin := registry.StaticPlugin{
		M:   registry.Manifest{ID: "odd", ChainID: 10, NetworkID: 77},
		Fam: wbftfam.New(),
	}
	if got := nodeconfig.NetworkOf(plugin, 0); got.NetworkID != 77 {
		t.Errorf("declared devp2p id 77 became %d", got.NetworkID)
	}
	if got := nodeconfig.NetworkOf(plugin, 4242); got.NetworkID != 77 || got.ChainID != 4242 {
		t.Errorf("overriding the chain id dropped the declared devp2p id: %+v", got)
	}
}

// TestNodeConfig_OneNetworkIsToldOneNumber is the defect this split exists for.
// A network of two builds asked its two plugins and got two answers: the config
// writer read each node's own chain and the argv assembler the composition's, so
// a successor's config file named one devp2p network and its command line
// another. Passing the network separately is what makes them the same value.
func TestNodeConfig_OneNetworkIsToldOneNumber(t *testing.T) {
	producer := registry.StaticPlugin{
		M:   registry.Manifest{ID: "wemix", Dialect: "geth110-wemix", ChainID: 8285, NetworkID: 8285},
		Fam: wbftfam.New(),
	}
	successor := registry.StaticPlugin{
		M:   registry.Manifest{ID: "wbft", Dialect: "geth114", ChainID: 8284, NetworkID: 8284},
		Fam: wbftfam.New(),
	}
	keys := preset.Key{Nodes: []keyring.Entry{{Index: 1, Identity: derive.Identity{PublicKey: "aa", Address: "0x1"}}}}
	spec := process.NodeSpec{Index: 1, Role: node.RoleEN, Host: "127.0.0.1", DataDir: "/d/node1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}}

	// The network is the producer's; the node runs the successor's build.
	net := nodeconfig.NetworkOf(producer, 0)
	cfg := process.NodeConfig(successor, net, keys, spec, "/keys", nil)

	if cfg.Chain.Dialect != "geth114" {
		t.Errorf("the node's flag vocabulary should follow its own build, got %q", cfg.Chain.Dialect)
	}
	if cfg.Network.NetworkID != 8285 {
		t.Errorf("the successor is told devp2p id %d, but the network it joins is 8285", cfg.Network.NetworkID)
	}
	args, err := nodeconfig.Argv(cfg)
	if err != nil {
		t.Fatalf("nodeconfig.Argv: %v", err)
	}
	if !argsHasPair(args, "--networkid", "8285") {
		t.Fatalf("argv does not carry the network's id: %v", args)
	}
}

func argsHasPair(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
