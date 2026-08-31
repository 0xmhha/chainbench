package upgrade

import (
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
)

// LaunchArgs builds the full launch argv (excluding the binary itself) for one
// node of a handoff network, through the launchopt Builder (the single argv
// assembly boundary — docs/dev/architecture/code-graph.md §3). It differs from the
// plain setup launch profile in the two ways the live handoff proved to
// require:
//
//   - --networkid is set explicitly on every node. go-wemix otherwise defaults
//     its devp2p network id (independent of chain id) while go-wbft derives it
//     from chain id, so without this the two binaries never peer.
//   - --authrpc.port is pinned per node. The engine API port otherwise defaults
//     to 8551 on every node and collides when several run on one machine.
//
// familyFlags are the consensus family's role flags (e.g. --mine for a
// producer/validator), supplied by the caller from the node's own chain family
// so this stays engine-agnostic; they must fit the closed vocabulary
// nodeconfig.ParseFamilyFlags accepts. overrides are the per-node high-precedence
// knobs (the handoff's account and RPC-namespace layer). The handoff passes
// every setting on the command line (no --config file), so the two binaries
// need no pre-written node config.
func LaunchArgs(n NodeSpec, dataDir string, familyFlags []string, overrides ...nodeconfig.Override) ([]string, error) {
	// A handoff relaunch carries no config file, so the ports the file would
	// have named travel on the command line; nodeconfig applies that rule.
	return nodeconfig.Argv(nodeconfig.Spec{
		Chain:    nodeconfig.Chain{ID: n.Chain, NetworkID: n.NetworkID, FamilyFlags: familyFlags},
		Role:     n.Role,
		Ports:    n.Ports,
		DataDir:  dataDir,
		HTTPHost: "127.0.0.1",
	}, overrides...)
}
