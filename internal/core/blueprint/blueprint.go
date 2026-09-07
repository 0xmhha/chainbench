package blueprint

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Version is the schema version this package reads. A document that names a
// different one is refused rather than read with today's meaning: the whole
// point of the field is that a future shape fails loudly on an old binary.
const Version = 1

// Blueprint is a network declaration. Every field is optional; what is absent
// is filled by Resolve from the inventory, the key set, the plugin, the family
// or the built-in defaults, in that order (design §3.4).
type Blueprint struct {
	// Version is the schema version. Zero means "the current one", so a hand
	// written document need not carry it.
	Version int `yaml:"version,omitempty"`

	// Chain names a registered chain plugin; Manifest points at an external
	// one. They answer the same question, so a document may carry one.
	Chain    string `yaml:"chain,omitempty"`
	Manifest string `yaml:"manifest,omitempty"`

	Binaries *Binaries `yaml:"binaries,omitempty"`

	// Nodes is the network: its length is the node count. An empty list is a
	// legal document — a generator writes one and a person fills it in.
	Nodes []Node `yaml:"nodes,omitempty"`

	// Peering is the shape of the peer graph (mesh or proxied). Empty means
	// mesh, which is what node.ParsePeering already decides.
	Peering string `yaml:"peering,omitempty"`

	Validators *ValidatorsDecl `yaml:"validators,omitempty"`
	Alloc      []Alloc         `yaml:"alloc,omitempty"`
	Genesis    *GenesisDecl    `yaml:"genesis,omitempty"`

	// GovernanceDecl is the poa family's bootstrap declaration. Whether a chain
	// accepts one is the plugin's answer, not this package's.
	Governance *GovernanceDecl `yaml:"governance,omitempty"`
}

// Binaries says which executables the network runs.
//
// It exists because the binary path was spelled out in twenty files
// (design §1.3): one place to say it, and one way to say "these nodes run a
// different one", which is what a handoff and a hardfork both need.
type Binaries struct {
	Node      string           `yaml:"node,omitempty"`
	Bootnode  string           `yaml:"bootnode,omitempty"`
	Overrides []BinaryOverride `yaml:"overrides,omitempty"`
}

// BinaryOverride points some of the nodes at a different executable.
type BinaryOverride struct {
	// Nodes names the nodes this override covers, by their blueprint name.
	Nodes []string `yaml:"nodes"`
	Node  string   `yaml:"node,omitempty"`
}

// Node is one node's declaration. A node with nothing but a role is complete:
// the name, host, ports, key and paths are all derivable.
type Node struct {
	Name string `yaml:"name,omitempty"`
	Role string `yaml:"role,omitempty"`

	// Server names an entry in the server set this node runs on. Empty means
	// wherever the placement puts it.
	Server string `yaml:"server,omitempty"`

	// Ports pins some or all of this node's ports. What is left zero is
	// allocated. It is node.Endpoints so the document and the running network
	// describe ports the same way.
	Ports *node.Endpoints `yaml:"ports,omitempty"`

	// NodeKey is this node's devp2p identity. Every node has one; declaring it
	// is how a network is stood up without a key set at all.
	NodeKey *NodeKeyRef `yaml:"nodekey,omitempty"`

	// Account is the account a producer seals with. A node that does not seal
	// has none, and saying otherwise is refused rather than ignored.
	Account *AccountRef `yaml:"account,omitempty"`

	SyncMode string `yaml:"syncmode,omitempty"`

	// Launch is this node's own config overrides, as the flat dot-path values
	// nodeconfig resolves. The keys belong to the chain, so they are not
	// checked here.
	Launch map[string]any `yaml:"launch,omitempty"`
}

// NodeKeyRef locates a private key: in a file, or written out in the document.
type NodeKeyRef struct {
	File string `yaml:"file,omitempty"`
	Hex  string `yaml:"hex,omitempty"`
}

// AccountRef locates a sealing account's keystore and the password that opens
// it.
type AccountRef struct {
	Keystore string `yaml:"keystore,omitempty"`
	// Password is a path to the password file. A key set may supply it
	// instead, so it is optional here.
	Password string `yaml:"password,omitempty"`
}

// ValidatorsDecl says who seals.
type ValidatorsDecl struct {
	// From derives the set from the node table. "role" means every bp.
	From string `yaml:"from,omitempty"`
	// Explicit names the nodes instead, in the order the genesis records them.
	Explicit []string `yaml:"explicit,omitempty"`
	Stake    WeiText  `yaml:"stake,omitempty"`
}

// Alloc is one genesis pre-funded balance. It names either a declared node
// (whose account address is resolved later) or an address written out.
type Alloc struct {
	Account string  `yaml:"account,omitempty"`
	Address string  `yaml:"address,omitempty"`
	Balance WeiText `yaml:"balance,omitempty"`
}

// GenesisDecl is what the built genesis is shaped by.
type GenesisDecl struct {
	ChainID int64 `yaml:"chainId,omitempty"`
	// Overrides are genesis config keys. They belong to the chain, so this
	// package carries them without reading them.
	Overrides map[string]any `yaml:"overrides,omitempty"`
	// Overlay is a JSON document deep-merged into the built genesis.
	Overlay string `yaml:"overlay,omitempty"`
}

// GovernanceDecl is the poa family's bootstrap declaration: who holds each role and
// what the environment contract is initialised with.
type GovernanceDecl struct {
	// Roles maps a governance role to a node name or an address.
	Roles map[string]string `yaml:"roles,omitempty"`
	Env   map[string]any    `yaml:"env,omitempty"`
}

// WeiText is an amount in wei, kept as text.
//
// Text because these are chain amounts: a stake of 1500000000000000000000000
// does not fit in an int64 and becomes a different number as a float. YAML
// happily reads it as one of those, so the type takes the scalar as written and
// hands on exactly what the document said.
type WeiText string

// UnmarshalYAML reads a wei amount from a plain scalar, whether the document
// quoted it or not.
func (w *WeiText) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("blueprint: line %d: an amount is a number or a quoted string, not %s", n.Line, kindName(n.Kind))
	}
	*w = WeiText(n.Value)
	return nil
}

// MarshalYAML writes the amount as a string, so re-reading the document yields
// the same digits rather than a float that has lost the low ones.
func (w WeiText) MarshalYAML() (any, error) {
	if w == "" {
		return nil, nil
	}
	return string(w), nil
}

// kindName names a YAML node kind for an error a person has to act on.
func kindName(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "a document"
	case yaml.SequenceNode:
		return "a list"
	case yaml.MappingNode:
		return "a map"
	case yaml.AliasNode:
		return "an alias"
	default:
		return "a scalar"
	}
}
