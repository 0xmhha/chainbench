package registry

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/pkg/core/node"
)

// GenesisParams are the per-network genesis values a family's BuildGenesis
// substitutes into a chain's template. It is the union of what the families
// need — the wbft family uses the validator set / BLS / extra-data / members /
// alloc; the poa family uses only ChainID / Coinbase (its membership is set at
// bootstrap). Defined here (not in pkg/core/genesis) so the ConsensusFamily
// contract stays the single dispatch seam and core need not import a family.
type GenesisParams struct {
	ChainID    int64
	Validators []string        // validator addresses (0x-hex) — wbft family
	BLSKeys    []string        // BLS public keys (0x-hex), aligned with Validators
	ExtraData  string          // RLP validator extra-data (0x-hex) — wbft family
	Members    []string        // governance council addresses (0x-hex) — anzeon system contracts
	Alloc      json.RawMessage // raw pre-funded accounts (address -> account) — wbft family
	Coinbase   string          // block coinbase (0x-hex) — poa family; default zero
}

// ConsensusFamily is the consensus-algorithm-centric strategy shared by chains
// of the same family (docs §4, D9): "wbft" (stablenet+wbft) and "poa" (wemix).
// It owns the parts of setup/verify that depend on the consensus algorithm.
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
	// BuildGenesis substitutes the family's placeholders in template with
	// params and returns the genesis.json bytes. This is the dispatch seam that
	// lets pkg/core/genesis build a genesis without importing any family.
	BuildGenesis(template []byte, params GenesisParams) ([]byte, error)
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
	// GenesisTemplate returns the chain's embedded genesis template bytes, or
	// nil for chains without a static template (poa/registry family).
	GenesisTemplate() []byte
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
