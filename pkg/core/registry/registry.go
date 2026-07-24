package registry

import (
	"fmt"
	"sort"

	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/pkg/core/node"
)

// ConsensusFamily is the consensus-algorithm-centric strategy shared by chains
// of the same family (docs §4, D9): "wbft" (stablenet+wbft) and "poa" (wemix).
// It owns the parts of setup/verify that depend on the consensus algorithm.
//
// Genesis/Bootstrap (block-production bring-up) are added with the setup
// pipeline in phase G2; G0 defines the static, RPC-facing surface the verify
// and consensus layers need immediately.
type ConsensusFamily interface {
	// ID is the family identifier ("wbft" | "poa").
	ID() string
	// RPCNamespace is the JSON-RPC namespace exposing consensus methods
	// ("istanbul" | "wemix").
	RPCNamespace() string
	// ValidatorsMethod is the RPC method returning the validator/producer
	// set.
	ValidatorsMethod() string
	// StartFlags returns the node launch flags for a given role.
	StartFlags(role node.Role) []string
}

// ChainPlugin is one chain's registration. Most of a chain is data (Manifest)
// plus a consensus-family selection plus its account protocol; only genuinely
// chain-specific behavior needs code.
type ChainPlugin interface {
	// Manifest returns the chain's declarative profile.
	Manifest() Manifest
	// Family returns the consensus family this chain composes.
	Family() ConsensusFamily
	// Protocol returns the accounts SDK protocol profile (tx types, account
	// model, system contracts) for this chain.
	Protocol() protocol.Protocol
}

var chains = map[string]ChainPlugin{}

// Register adds a chain plugin. Intended to be called from a chain package's
// init(); panics on duplicate id so a wiring mistake fails loudly at startup.
func Register(p ChainPlugin) {
	id := p.Manifest().ID
	if id == "" {
		panic("registry: plugin with empty manifest id")
	}
	if _, dup := chains[id]; dup {
		panic(fmt.Sprintf("registry: duplicate chain plugin %q", id))
	}
	chains[id] = p
}

// Get returns the plugin registered for id, or an error naming the known set.
func Get(id string) (ChainPlugin, error) {
	if p, ok := chains[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("registry: unknown chain %q (known: %v)", id, Names())
}

// Names returns the sorted list of registered chain ids.
func Names() []string {
	names := make([]string, 0, len(chains))
	for n := range chains {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
