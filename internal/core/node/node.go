// Package node defines the chain-agnostic node model that flows between the
// three pipeline phases (setup -> verify -> test). NodeSet is the single
// hand-off object: every phase takes a NodeSet and returns a NodeSet, so a
// phase can run standalone against nodes it did not create (e.g. attach, which
// skips setup entirely — see docs/CHAINBENCH_GO_REDESIGN.md §3).
package node

// Role is a node's operational role within a network. The BFT chains
// (stablenet/wbft) use validator/endpoint; the wemix (poa) bootstrap adds the
// boot/en distinction seen in ../script/wemix-upgrade (boot-node deploys
// governance). Roles are orthogonal facts a driver may key launch flags off.
type Role string

// The canonical vocabulary is bp / en / pn. The two legacy
// spellings survive because they are written into persisted state
// (workspace.json, topology files) and launch flows; netmap.NormalizeRole folds
// them onto the canonical three, and new code should not emit them.
const (
	// RoleBP is a block producer: a BFT validator, or a staked poa producer.
	RoleBP Role = "bp"
	// RoleEN is a non-producing RPC endpoint.
	RoleEN Role = "en"
	// RolePN is a peer node: the proxy tier between producers and endpoints.
	// It is expressed through the static-nodes graph, not a binary flag, and
	// the poa family does not have it (etcd takes that place).
	RolePN Role = "pn"

	// RoleValidator is the legacy spelling of RoleBP.
	RoleValidator Role = "validator"
	// RoleEndpoint is the legacy spelling of RoleEN.
	RoleEndpoint Role = "endpoint"
	// RoleBoot marks the node that runs the poa governance bootstrap. In the
	// target model this is an attribute of a bp, not a role of its own;
	// it remains a role until the poa bring-up moves to that model.
	RoleBoot Role = "boot"
)

// Endpoints holds a node's reachable ports on its host. For nodes on the same
// host these are offset per node; across hosts they may repeat while Host
// varies (docs §7, requirement #6).
type Endpoints struct {
	P2P int `json:"p2p"`
	// Etcd is not a launch flag: a wemix node's embedded etcd derives its peer
	// port as P2P+1 and its client port as P2P+2. Both are carried so a running
	// node can be asked for them and so collision checks see them — the ports
	// whose step rule exists because of them used to disappear the moment a
	// plan became a running network.
	//
	// A family that does not embed etcd leaves them zero rather than reserving
	// ports it will not listen on.
	Etcd       int `json:"etcd,omitempty"`
	EtcdClient int `json:"etcdClient,omitempty"`
	HTTP       int `json:"http"`
	WS         int `json:"ws"`
	Auth       int `json:"auth"`
	Metrics    int `json:"metrics"`
}

// Node is one chain node, whether locally launched, remotely launched, or
// attached to an already-running endpoint.
type Node struct {
	// Index is the 1-based node number within the set.
	Index int `json:"index"`
	// Role is the node's operational role.
	Role Role `json:"role"`
	// Host is the address the node is reachable at ("127.0.0.1" for local,
	// a hostname/IP for remote).
	Host string `json:"host"`
	// RPCURL is the JSON-RPC endpoint used for verify/test. For attached
	// nodes this is the only field that must be set.
	RPCURL string `json:"rpc_url"`
	// Ports holds the node's port map (empty for pure-attach nodes whose
	// ports are unknown/irrelevant).
	Ports Endpoints `json:"ports"`
	// PID is the launched process id (0 for attached nodes chainbench did not
	// start). Used by `stop` and hardfork execution.
	PID int `json:"pid,omitempty"`
	// Auth is the optional authentication descriptor for reaching a remote
	// attached endpoint. Empty for local or unauthenticated nodes.
	Auth Auth `json:"auth,omitempty"`
}

// Auth is a node's authentication descriptor for reaching a remote attached
// endpoint. It is a flexible map (converted to/from pkg/core/remote.Auth at the
// boundary, so node need not import remote) with a fixed key convention: "type"
// (e.g. "api_key" | "bearer") and the name of the env var holding the secret —
// never the secret value itself. A named type so the seam is documented and
// greppable rather than an anonymous map[string]any.
type Auth map[string]any

// NodeSet is the collection of nodes for one network plus its identity and the
// capabilities its provider exposes. It is the only object passed between
// pipeline phases.
type NodeSet struct {
	// Chain is the chain id this set belongs to ("stablenet"|"wbft"|"wemix").
	Chain string `json:"chain"`
	// Network is the network name ("local" or an attached network's name).
	Network string `json:"network"`
	// Nodes are the member nodes, ordered by Index.
	Nodes []Node `json:"nodes"`
	// Capabilities is the effective capability set (e.g. "process","rpc",
	// "ws","consensus") the driver/provider supports for this set.
	Capabilities []string `json:"capabilities"`
}

// Primary returns the first node (lowest Index), the conventional RPC target
// for whole-network queries, and false if the set is empty.
func (s NodeSet) Primary() (Node, bool) {
	if len(s.Nodes) == 0 {
		return Node{}, false
	}
	best := s.Nodes[0]
	for _, n := range s.Nodes[1:] {
		if n.Index < best.Index {
			best = n
		}
	}
	return best, true
}

// HasCapability reports whether cap is in the set's capability list.
func (s NodeSet) HasCapability(cap string) bool {
	for _, c := range s.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// Offset returns base with each port advanced by i. Used to allocate distinct
// ports for co-located nodes on the same host (requirement #6): node i gets
// base+i. Across hosts, callers keep i=0 and vary Host instead.
func Offset(base Endpoints, i int) Endpoints {
	return Endpoints{
		P2P:     base.P2P + i,
		HTTP:    base.HTTP + i,
		WS:      base.WS + i,
		Auth:    base.Auth + i,
		Metrics: base.Metrics + i,
	}
}
