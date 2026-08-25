package netmap

import (
	"fmt"

	placement "github.com/0xmhha/chainbench/internal/core/netmap"
)

// Enode composition is this module's passive role: keyring derives the keys,
// netmap says where each node runs and who dials whom, and the enode is the
// two joined — a public key at a placement. The keys come in as inputs here;
// nothing in this module ever derives one.

// Enode builds a static-node enode URL from a devp2p public key and a
// placement's host and p2p port.
func Enode(publicKey, host string, p2pPort int) string {
	return fmt.Sprintf("enode://%s@%s:%d?discport=0", publicKey, host, p2pPort)
}

// PeerList composes label's static-node list: the peering graph decides who
// this node dials, the placement map says where each peer lives, and pubkey
// supplies the identity material by node index. A peer whose key the caller
// cannot supply is skipped, matching the graph's own contract.
func PeerList(m *placement.Map, p placement.Peering, label placement.NodeLabel, pubkey func(index int) (string, bool)) ([]string, error) {
	return p.StaticNodes(m, label, func(pl placement.Placement) (string, bool) {
		pk, ok := pubkey(pl.Index)
		if !ok {
			return "", false
		}
		return Enode(pk, pl.Host, pl.Ports.P2P), true
	})
}
