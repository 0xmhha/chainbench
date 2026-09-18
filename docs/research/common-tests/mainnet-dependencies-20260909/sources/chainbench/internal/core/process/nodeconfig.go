package process

import (
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// NodeConfig assembles one node's configuration from the chain, the key set,
// and the node's spec: the one place a nodeconfig.Spec is built from a plan.
// The step surface builds the same Spec from a workspace record through here,
// so a step-composed node launches with exactly the argv and config it renders.
func NodeConfig(plugin registry.ChainPlugin, preset keyring.Preset, spec NodeSpec, keysDir string, staticNodes []string) nodeconfig.Spec {
	nodeDir := filepath.Join(keysDir, fmt.Sprintf("node%d", spec.Index))
	cfg := nodeconfig.Spec{
		Chain:       nodeconfig.ChainOf(plugin, spec.Role),
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
		if nk, ok := preset.Node(spec.Index); ok {
			cfg.Unlock = nk.Address
			cfg.PasswordFile = filepath.Join(keysDir, "password")
		}
	}
	return cfg
}
