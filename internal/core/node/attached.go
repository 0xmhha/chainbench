package node

import "fmt"

// Attached networks (requirement #7): chainbench did not create these nodes,
// so there is nothing to plan, provision, or launch — only RPC endpoints to
// address. Building the NodeSet is a pure constructor over this package's own
// types, which is why it lives here rather than in the launch pipeline it used
// to sit in.

// RPCEndpoint is one already-running node addressed by its RPC URL. It is
// named for the JSON-RPC endpoint to keep it distinct from Endpoints, this
// package's per-node port map.
type RPCEndpoint struct {
	// RPCURL is required: the node's JSON-RPC HTTP endpoint.
	RPCURL string
	// Host and HTTPPort are optional metadata (recorded on the Node when set).
	Host     string
	HTTPPort int
}

// AttachedSet constructs a NodeSet for an already-running network. Nodes are
// indexed from 1 in the order given and marked as endpoints — attach makes no
// producer assumptions, because chainbench cannot know which of someone else's
// nodes mine. At least one endpoint with an RPC URL is required.
func AttachedSet(chain, network string, eps []RPCEndpoint) (NodeSet, error) {
	if len(eps) == 0 {
		return NodeSet{}, fmt.Errorf("node: attach: no endpoints given")
	}
	ns := NodeSet{Chain: chain, Network: network, Capabilities: []string{"rpc"}}
	for i, ep := range eps {
		if ep.RPCURL == "" {
			return NodeSet{}, fmt.Errorf("node: attach: endpoint %d has empty rpc url", i+1)
		}
		ns.Nodes = append(ns.Nodes, Node{
			Index:  i + 1,
			Role:   RoleEndpoint,
			Host:   ep.Host,
			RPCURL: ep.RPCURL,
			Ports:  Endpoints{HTTP: ep.HTTPPort},
		})
	}
	return ns, nil
}
