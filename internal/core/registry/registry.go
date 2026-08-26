package registry

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/portplan"
)

// GenesisParams are the per-network genesis values a family's BuildGenesis
// substitutes into a chain's template. It is the union of what the families
// need — the wbft family uses the validator set / BLS / extra-data / members /
// alloc; the poa family uses only ChainID / Coinbase (its membership is set at
// bootstrap). Defined here (not in pkg/core/genesis) so the ConsensusFamily
// contract stays the single dispatch boundary and core need not import a family.
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
	// BringUpPhases orders the launch: which nodes start together, and what
	// must complete between one group and the next.
	//
	// Only one upper-layer assumption actually differs between the families —
	// that every node in a plan starts at once. A wemix network cannot: its
	// etcd cluster only forms while the producer is alone, so the bootstrap
	// runs in the gap between two groups. Saying that as data keeps the
	// supervisor's ownership intact: it still decides timing, retries and how a
	// failure is classified, and only asks the family for the order.
	//
	// Actions are names, not functions. The core does not know what
	// "deploy-governance" is, for the same reason the DSL puts chain vocabulary
	// in the spec rather than in the interpreter.
	BringUpPhases(roles []node.Role) []Phase
	// PortReservation is how many consecutive ports one of this family's nodes
	// needs from each band. It is asked rather than assumed because the answer
	// differs: a wemix node's embedded etcd listens on two ports beyond p2p,
	// and a global rule sized for one of them is wrong for the other.
	PortReservation() portplan.Reservation
	// SupportsRole reports whether this family can run a role. The proxy tier
	// (pn) is the case that matters: poa has no such tier — etcd occupies that
	// place — so a topology declaring one is asking for something that will not
	// exist, and only the family can say so (netmap-design 2.6).
	SupportsRole(role node.Role) bool
	// BuildGenesis substitutes the family's placeholders in template with
	// params and returns the genesis.json bytes. This is the dispatch boundary that
	// lets pkg/core/genesis build a genesis without importing any family.
	BuildGenesis(template []byte, params GenesisParams) ([]byte, error)
}

// Phase is one ordered group of a bring-up: nodes that start together, then
// the actions that must complete before the next group may start.
type Phase struct {
	// Name identifies the phase in diagnostics ("all", "boot", "rest").
	Name string
	// Nodes are the 1-based indices launched in this phase. Empty means every
	// node, which is what a single-phase family declares.
	Nodes []int
	// Actions are the named bring-up steps that run after this phase's nodes
	// are up, before the next phase starts. An action a phase names but that
	// the caller has not wired is an error, not a silent pass.
	Actions []string
	// ActionsOn is the 1-based node the phase's actions run against. Zero
	// means the first node this phase launched, which is what a bootstrap
	// phase wants.
	//
	// A phase whose actions concern a node it did not launch names it here.
	// The rest joining the cluster the boot node formed is that case: without
	// this, every executor would re-derive which node the boot node was, and
	// a rule copied into three places is the shape of bug this package exists
	// to remove.
	ActionsOn int
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

// StaticPlugin is a ChainPlugin assembled from already-resolved parts. It backs
// a chain whether its manifest came from the embedded set or an external file,
// so both paths produce the same object. The family and protocol are supplied by
// the composition layer (which may import concrete families), keeping this core
// type free of any consensus/chain import.
type StaticPlugin struct {
	M     Manifest
	Fam   ConsensusFamily
	Proto protocol.Protocol
	Tmpl  []byte
}

func (p StaticPlugin) Manifest() Manifest          { return p.M }
func (p StaticPlugin) Family() ConsensusFamily     { return p.Fam }
func (p StaticPlugin) Protocol() protocol.Protocol { return p.Proto }
func (p StaticPlugin) GenesisTemplate() []byte     { return p.Tmpl }

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
