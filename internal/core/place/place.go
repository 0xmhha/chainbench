// Package place holds the request that describes a node to be launched.
//
// It used to own allocation as well — an Allocator with three modes deciding
// which host and ports each node took. netmap.Assign replaced that: the two
// deterministic modes turned out to be one grid of addresses and port slots
// read two ways, and the third asked the OS for free ports and had no callers.
// What remains is the request itself, which is a launch concern (role, sync
// mode, binary) rather than a placement one.
package place

import "github.com/0xmhha/chainbench/internal/core/node"

// NodeReq is one node to launch: its role, and the launch choices that are not
// derivable from the network's shape.
//
// It carries no name. The name a node answers to is netmap's — it used to be
// invented here in four different spellings by different callers and read by
// none, which is why a node's label is now assigned once and persisted.
type NodeReq struct {
	Role   node.Role
	Sync   string
	Binary string
}
