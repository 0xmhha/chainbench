package upgrade

import (
	"strconv"

	"github.com/0xmhha/chainbench/internal/core/netid"
)

// LaunchArgs builds the full launch argv (excluding the binary itself) for one
// node of a handoff network. It differs from the plain setup launch args in the
// two ways the live handoff proved to require:
//
//   - --networkid is set explicitly on every node. go-wemix otherwise defaults
//     its devp2p network id (independent of chain id) while go-wbft derives it
//     from chain id, so without this the two binaries never peer.
//   - --authrpc.port is pinned per node. The engine API port otherwise defaults
//     to 8551 on every node and collides when several run on one machine.
//
// familyFlags are the consensus family's role flags (e.g. --mine for a
// producer/validator), supplied by the caller from the node's own chain family
// so this stays engine-agnostic. The handoff passes every setting on the command
// line (no --config file), so the two binaries need no pre-written node config.
func LaunchArgs(n NodeSpec, dataDir string, familyFlags []string) []string {
	args := []string{
		"--datadir", dataDir,
		"--port", strconv.Itoa(n.Ports.P2P),
		"--http",
		"--http.addr", "127.0.0.1",
		"--http.port", strconv.Itoa(n.Ports.HTTP),
		"--ws",
		"--ws.port", strconv.Itoa(n.Ports.WS),
		"--authrpc.port", strconv.Itoa(n.Ports.Auth),
	}
	args = append(args, netid.Flag(n.NetworkID)...)
	return append(args, familyFlags...)
}
