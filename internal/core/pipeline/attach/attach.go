// Package attach is the pipeline entry point for already-running networks
// (requirement #7): it skips setup entirely and builds a NodeSet from existing
// RPC endpoints so the verify (and then test) phases can run against nodes
// chainbench did not create (docs/CHAINBENCH_GO_REDESIGN.md §3).
package attach

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Endpoint is one already-running node addressed by its RPC URL.
type Endpoint struct {
	// RPCURL is required: the node's JSON-RPC HTTP endpoint.
	RPCURL string
	// Host and HTTPPort are optional metadata (recorded on the Node when set).
	Host     string
	HTTPPort int
}

// Build constructs a NodeSet for an attached network. Nodes are indexed from 1
// in the order given and marked as endpoints (attach makes no producer
// assumptions). At least one endpoint with an RPC URL is required.
func Build(chain, network string, eps []Endpoint) (node.NodeSet, error) {
	if len(eps) == 0 {
		return node.NodeSet{}, fmt.Errorf("attach: no endpoints given")
	}
	ns := node.NodeSet{Chain: chain, Network: network, Capabilities: []string{"rpc"}}
	for i, ep := range eps {
		if ep.RPCURL == "" {
			return node.NodeSet{}, fmt.Errorf("attach: endpoint %d has empty rpc url", i+1)
		}
		ns.Nodes = append(ns.Nodes, node.Node{
			Index:  i + 1,
			Role:   node.RoleEndpoint,
			Host:   ep.Host,
			RPCURL: ep.RPCURL,
			Ports:  node.Endpoints{HTTP: ep.HTTPPort},
		})
	}
	return ns, nil
}
