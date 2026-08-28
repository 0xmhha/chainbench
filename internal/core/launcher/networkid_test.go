package launcher_test

import (
	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/launcher"
	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"testing"
)

// TestNodeLaunchArgs_EmitsTheManifestNetworkID: the devp2p network id was never
// on the command line. A chain whose network id differs from its genesis chain
// id — which the handoff produces, because it forces the chain id — ran on
// whichever the binary inferred, and two networks that should not see each
// other could agree to.
func TestNodeLaunchArgs_EmitsTheManifestNetworkID(t *testing.T) {
	plugin := registry.StaticPlugin{
		M: registry.Manifest{
			ID: "stablenet", Binary: "go-stablenet", NetworkID: 8283,
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}
	preset := keyring.Preset{Nodes: []keyring.Entry{{Index: 1, Identity: derive.Identity{PublicKey: "aa", Address: "0x1"}}}}
	spec := driver.NodeSpec{Index: 1, Role: node.RoleEN, Host: "127.0.0.1", DataDir: "/d/node1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}}

	args, err := launcher.NodeLaunchArgs(plugin, preset, spec, "/keys", nil)
	if err != nil {
		t.Fatalf("launcher.NodeLaunchArgs: %v", err)
	}
	if !argsHasPair(args, "--networkid", "8283") {
		t.Fatalf("argv does not carry the manifest network id: %v", args)
	}

	// An operator's override arrives on a later layer and wins.
	args, err = launcher.NodeLaunchArgs(plugin, preset, spec, "/keys", []launchopt.Override{
		{Key: launchopt.KeyNetworkID, Value: "99", Layer: launchopt.LayerEnv},
	})
	if err != nil {
		t.Fatalf("launcher.NodeLaunchArgs with override: %v", err)
	}
	if !argsHasPair(args, "--networkid", "99") {
		t.Fatalf("override did not win: %v", args)
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
