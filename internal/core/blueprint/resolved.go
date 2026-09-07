package blueprint

import (
	"github.com/0xmhha/chainbench/internal/core/node"
)

// Source names where a resolved value came from.
//
// It is recorded rather than inferred because the value's origin is the first
// question asked when a network does not behave like its document: a port that
// is not the one written down came either from the inventory or from a default,
// and those are two different mistakes.
type Source string

// The source chain, strongest first (design §3.4). A value found earlier is
// never overwritten by one found later — that is what "an explicit value wins"
// means in code.
const (
	// FromBlueprint is a value the document states.
	FromBlueprint Source = "blueprint"
	// FromInventory is a value the placement decided (host, ports, paths).
	FromInventory Source = "inventory"
	// FromKeySet is a value a key set supplied (nodekey, sealing account).
	FromKeySet Source = "keyset"
	// FromChain is a value the chain plugin knows (binary, chain id).
	FromChain Source = "chain"
	// FromDefault is a built-in fallback.
	FromDefault Source = "default"
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

	// Sources records where each value came from, keyed by the path of the
	// field it decided ("nodes[1].ports.p2p").
	Sources map[string]Source `json:"sources,omitempty"`
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
