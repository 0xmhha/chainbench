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
	RPCURL     string        `json:"rpc"`
	Role       Role          `json:"role,omitempty"`
	WSURL      string        `json:"ws,omitempty"`
	MetricsURL string        `json:"metrics,omitempty"`
	Auth       *EndpointAuth `json:"auth,omitempty"`
	APIs       []string      `json:"apis,omitempty"`
	// Host and HTTPPort are optional metadata (recorded on the Node when set).
	Host     string `json:"host,omitempty"`
	HTTPPort int    `json:"httpPort,omitempty"`
}

// AttachedSet constructs a NodeSet for an already-running network. Nodes are
// indexed from 1 in the order given. Bare URLs default to endpoints; a
// structured profile preserves the operator-declared role without owning any
// process or implying that the role has been observed. At least one endpoint with an RPC URL is required.
func AttachedSet(chain, network string, eps []RPCEndpoint) (NodeSet, error) {
	if len(eps) == 0 {
		return NodeSet{}, fmt.Errorf("node: attach: no endpoints given")
	}
	ns := NodeSet{Chain: chain, Network: network, Capabilities: []string{"rpc"}}
	for i, ep := range eps {
		if ep.RPCURL == "" {
			return NodeSet{}, fmt.Errorf("node: attach: endpoint %d has empty rpc url", i+1)
		}
		role := ep.Role
		if role == "" {
			role = RoleEN
		}
		normalized, err := NormalizeRole(string(role))
		if err != nil {
			return NodeSet{}, fmt.Errorf("node: attach: endpoint %d: %w", i+1, err)
		}
		ns.Nodes = append(ns.Nodes, Node{
			Index: i + 1,
			Role:  normalized,
			WSURL: ep.WSURL, MetricsURL: ep.MetricsURL, Auth: ep.Auth.descriptor(), APIs: append([]string(nil), ep.APIs...),
			Host:   ep.Host,
			RPCURL: ep.RPCURL,
			Ports:  Endpoints{HTTP: ep.HTTPPort},
		})
	}
	return ns, nil
}

// EndpointAuth permits references only; literal passwords/tokens are not input fields.
type EndpointAuth struct {
	Type   string `json:"type"`
	Env    string `json:"env"`
	Header string `json:"header,omitempty"`
}

func (auth *EndpointAuth) descriptor() Auth {
	if auth == nil {
		return nil
	}
	out := Auth{"type": auth.Type, "env": auth.Env}
	if auth.Header != "" {
		out["header"] = auth.Header
	}
	return out
}
