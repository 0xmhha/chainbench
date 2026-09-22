package blueprint

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/origin"
)

// ResolvedNetwork is the network with every value decided.
//
// Nothing downstream asks a question again: the genesis, the per-node configs,
// the argv and the deploy set are all derived from this one snapshot. That is
// why it is serialised next to them — an artifact and the description it was
// built from should not be separable.
//
// It is immutable by convention. Something to change means changing the
// Blueprint and resolving again, so the snapshot always corresponds to a
// document that exists.
type ResolvedNetwork struct {
	// Chain is the chain id this network runs ("wemix").
	Chain string `json:"chain"`
	// ChainID is the EVM chain id in the genesis.
	ChainID int64 `json:"chain_id"`
	// Nodes are every node with every field decided, in node order.
	Nodes []ResolvedNode `json:"nodes"`
	// Validators names the nodes that seal, in the order the genesis records
	// them.
	Validators []string `json:"validators,omitempty"`
	// Peering is the shape of the peer graph.
	Peering node.Peering `json:"peering"`
	// Alloc is the genesis pre-funding, with node references left as written:
	// an account's address is not known until the key set is read, and this
	// snapshot does not open one.
	Alloc []Alloc `json:"alloc,omitempty"`
	// Genesis carries what shapes the built genesis beyond the chain id.
	Genesis *GenesisDecl `json:"genesis,omitempty"`
	// Governance is the poa family's declaration, carried through untouched.
	Governance *GovernanceDecl `json:"governance,omitempty"`

	// Origins records where each value came from, keyed by the path of the
	// field it decided ("nodes[1].ports.p2p").
	Origins map[string]origin.Origin `json:"origins,omitempty"`
}

// ResolvedNode is one node with nothing left to decide.
type ResolvedNode struct {
	// Index is the node's 1-based identity, the number its paths are named
	// from.
	Index int `json:"index"`
	// Name is what the document calls it, or the role label the placement
	// gave it ("bp1").
	Name string    `json:"name"`
	Role node.Role `json:"role"`
	Host string    `json:"host"`
	// Ports is the full port set, the pinned fields included.
	Ports node.Endpoints `json:"ports"`
	// NodeKey is this node's devp2p identity, as a file path or 0x-hex.
	NodeKey NodeKeyRef `json:"nodekey"`
	// Account is the sealing account, nil for a node that does not seal.
	Account *AccountRef `json:"account,omitempty"`
	// Binary is the executable this node runs, after any override.
	Binary string `json:"binary"`
	// DataDir is the node's directory on the target.
	DataDir string `json:"data_dir"`
	// ConfigPath and LogPath are derived the same way, from the same layout,
	// so nothing recomputes them from a format string.
	ConfigPath string `json:"config_path"`
	LogPath    string `json:"log_path"`
	SyncMode   string `json:"sync_mode"`
	// Launch is this node's own config overrides.
	Launch map[string]any `json:"launch,omitempty"`
}
