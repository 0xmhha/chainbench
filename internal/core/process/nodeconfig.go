package process

import (
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/preset"
)

// NodeConfig assembles one node's configuration from the chain, the key set,
// and the node's spec: the one place a nodeconfig.Spec is built from a plan.
// The step surface builds the same Spec from a workspace record through here,
// so a step-composed node launches with exactly the argv and config it renders.
//
// plugin is THIS NODE's chain and net is THE NETWORK's, and they are two
// parameters rather than one because a network of mixed builds has one of the
// first per node and one of the second in total. A caller that passes the same
// plugin's facts for both is describing a network of one build, which is the
// ordinary case; a caller that passes the node's plugin for net has said every
// node is its own network, which is the defect this signature exists to stop.
func NodeConfig(plugin registry.ChainPlugin, net nodeconfig.Network, keys preset.Key, spec NodeSpec, keysDir string, staticNodes []string) nodeconfig.Spec {
	nodeDir := filepath.Join(keysDir, fmt.Sprintf("node%d", spec.Index))
	cfg := nodeconfig.Spec{
		Chain:       nodeconfig.ChainOf(plugin, spec.Role),
		Network:     net,
		Role:        spec.Role,
		Ports:       spec.Ports,
		SyncMode:    spec.SyncMode,
		DataDir:     spec.DataDir,
		ConfigPath:  spec.ConfigPath,
		NodekeyPath: filepath.Join(nodeDir, "nodekey"),
		KeystoreDir: filepath.Join(nodeDir, "keystore"),
		StaticNodes: staticNodes,
	}
	if node.Is(spec.Role, node.RoleBP) {
		if nk, ok := keys.Node(spec.Index); ok {
			// The account it seals with, which is its keystore's when the ring
			// says those differ. Unlocking the address its nodekey derives sent
			// a producer to "no key for given address or file" on a ring whose
			// keystore holds another account.
			cfg.Unlock = nk.SealingAccount()
			cfg.PasswordFile = filepath.Join(keysDir, "password")
		}
	}
	return cfg
}
