package node

import (
	"fmt"
)

// Enode composition is this module's passive role: keyring derives the keys,
// this package says where each node runs and who dials whom, and the enode is the
// two joined — a public key at a  The keys come in as inputs here;
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
func PeerList(m *Map, p Peering, label Label, pubkey func(index int) (string, bool)) ([]string, error) {
	return p.StaticNodes(m, label, func(pl Placement) (string, bool) {
		pk, ok := pubkey(pl.Index)
		if !ok {
			return "", false
		}
		return Enode(pk, pl.Host, pl.Ports.P2P), true
	})
}
