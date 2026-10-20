package testengine

import "github.com/0xmhha/chainbench/internal/core/node"

// recordHandover replaces the pre-fork snapshot with the nodes the operation
// actually relaunched, before readiness and capability checks consume them.
func recordHandover(net *composed, nodes node.NodeSet) {
	net.nodes = &nodes
	net.fork.handedOver = true
}
