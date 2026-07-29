// This file adds the admin_peers network case (ported from the legacy bash
// regression suite regression/g-api/g4-04-admin-peers.sh).
//
// # Test: admin-peers-populated
//
// Intent:   in a multi-node network the admin_peers RPC must report the P2P
//
//	peer table — at least one connected peer whose object carries a
//	non-empty id (the enode/node identity). An empty peer list or a
//	peer with no id signals a broken P2P layer even when net_peerCount
//	looks non-zero.
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   read admin_peers on the primary node; assert the result has at
//
//	least one entry and its first peer object has a non-empty id field
//	(vacuous for a single-node network, which has no peers to expect).
//
// Pass:     admin_peers returns >= 1 peer and the first peer has a non-empty id.
//
// This is chainbench TEST CODE (requirement #16): registered at init and run by
// the testrun phase against a live NodeSet, not by `go test` (the sibling
// _test.go validates registration/convention).
package network

import (
	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "admin-peers-populated",
		Category:     "network",
		RequiresCaps: []string{"rpc"},
		Fn:           adminPeersPopulated,
	})
}

func adminPeersPopulated(t *testkit.T) {
	if len(t.NodeSet().Nodes) < 2 {
		return // single-node network has no peers to expect
	}
	var peers []struct {
		ID string `json:"id"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "admin_peers", &peers), "admin_peers")
	t.Truef(len(peers) >= 1, "admin_peers returned at least one peer (got %d)", len(peers))
	if len(peers) >= 1 {
		t.Truef(peers[0].ID != "", "first peer has a non-empty id")
	}
}
