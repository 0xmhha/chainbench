// Package node owns what is known about a node: what it is called, what role
// it plays, where it runs, on which ports, at which paths, and — derived from
// the roles — who dials whom.
//
// It is the vocabulary every other module speaks. Nothing here reaches the
// outside world: no files, no sockets, no processes. That is what lets it be
// imported from anywhere without dragging a dependency along, and the rule is
// enforced by measurement — this package imports nothing (see the code graph
// in docs/dev/architecture/code-graph.md).
//
// The facts arrive from elsewhere and are recorded here. netmap allocates a
// host and ports and hands back a Map; keyring derives the keys an Enode joins
// to a placement. The division is deliberate: a module that decides something
// about a node should not also be the place that remembers it, or the memory
// forks the moment a second decider appears — which is how ten types came to
// mean "one node" and four places came to compute the same paths (formerly
// recorded in docs/dev/architecture/module-plan.md §2, a plan since closed and
// retired; `git show cde3a08f:docs/dev/architecture/module-plan.md`).
//
// NodeSet remains the hand-off object between the three pipeline phases
// (setup -> verify -> test): every phase takes a NodeSet and returns a
// NodeSet, so a phase can run standalone against nodes it did not create.
package node

// Role is what a node does in a network. A driver keys launch flags off it,
// the peering graph decides who dials whom by it, and a spec addresses a node
// by it ("bp1", "en2").
type Role string

// The role vocabulary is three words, and these are the three. Nothing else is
// accepted anywhere: not in a declaration, not in a record.
const (
	// RoleBP is a block producer. It builds a block and proposes it.
	//
	// When it is not this node's turn to propose, it verifies the block
	// another producer proposed; that verifying is what "validator" names. A
	// validator is therefore something a bp does, not a fourth role, which is
	// why the word is not in this vocabulary.
	RoleBP Role = "bp"
	// RoleEN is an endpoint. It serves RPC and never produces a block. It
	// reaches the producers through a pn rather than dialling one directly.
	RoleEN Role = "en"
	// RolePN is a proxy node: the tier that carries traffic between producers
	// and endpoints. It is expressed through the static-nodes graph rather than
	// a binary flag, and it is what connects nodes on every chain this harness
	// supports — both families run one, and there is no separate "boot" role
	// for that job.
	RolePN Role = "pn"
)

// Endpoints holds a node's reachable ports on its host. For nodes on the same
// host these are offset per node; across hosts they may repeat while Host
// varies (docs §7, requirement #6).
//
// The yaml tags are here so a blueprint declares ports in the one representation
// the rest of the system uses. Three spellings of a port map is what NM7 ended,
// and a declaration format is exactly where a fourth would have started.
type Endpoints struct {
	P2P int `json:"p2p" yaml:"p2p,omitempty"`
	// Etcd is not a launch flag: a wemix node's embedded etcd derives its peer
	// port as P2P+1 and its client port as P2P+2. Both are carried so a running
	// node can be asked for them and so collision checks see them — the ports
	// whose step rule exists because of them used to disappear the moment a
	// plan became a running network.
	//
	// A family that does not embed etcd leaves them zero rather than reserving
	// ports it will not listen on.
	Etcd       int `json:"etcd,omitempty" yaml:"etcd,omitempty"`
	EtcdClient int `json:"etcdClient,omitempty" yaml:"etcd_client,omitempty"`
	HTTP       int `json:"http" yaml:"http,omitempty"`
	WS         int `json:"ws" yaml:"ws,omitempty"`
	Auth       int `json:"auth" yaml:"auth,omitempty"`
	Metrics    int `json:"metrics" yaml:"metrics,omitempty"`
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
	// MetricsURL is the full URL a caller GETs to scrape this node's metrics —
	// scheme, reachable host and port, AND the Prometheus path.
	//
	// It exists for the same reason RPCURL does: Host and Ports hold the node's
	// OWN address, which is what peers use and what goes into the genesis and
	// static-nodes list, and that is not always the address this tool can reach
	// it at. A docker container publishes on loopback; composing the pair by
	// hand is how the metric assertion came to dial a container-internal address
	// and time out. Empty when the composition did not record one (an attached
	// node, or a node with no metrics port).
	MetricsURL string `json:"metrics_url,omitempty"`
	// WSURL is the WebSocket endpoint a subscription dials, recorded for the
	// same reason MetricsURL is: composing ws://Host:Ports.WS by hand reaches
	// the node's OWN address, which under docker is the container-internal one
	// this tool cannot route to. Measured 2026-09-19: ws-subscribe-new-heads
	// timed out on ws://172.30.0.11:8701 while every HTTP dial in the same run
	// went to the published 127.0.0.1 port. Empty when the composition recorded
	// none (an attached node, or a node with no ws port).
	WSURL string `json:"ws_url,omitempty"`
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
// endpoint. It is a flexible map (converted to/from internal/core/remote.Auth at the
// boundary, so node need not import remote) with a fixed key convention: "type"
// (e.g. "api_key" | "bearer") and the name of the env var holding the secret —
// never the secret value itself. A named type so the boundary is documented and
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
