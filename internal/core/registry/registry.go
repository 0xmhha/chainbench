package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/internal/core/node"
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
	// launcher's ownership intact: it still decides timing, retries and how a
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
	PortReservation() node.Reservation
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

// GenesisValidatorReader is an optional ConsensusFamily capability: read the
// validator addresses a finished genesis encodes. A composition uses it to
// check an existing genesis against the keys the network will run with — a
// validator the running keys cannot produce would pass genesis validation and
// then stall consensus. A family that carries its validator set outside the
// genesis file (poa keeps it in a governance config) does not implement it, and
// the check is skipped for that family.
type GenesisValidatorReader interface {
	GenesisValidators(genesisJSON []byte) ([]string, error)
}

// The account roles a ring supplies. They are the roster's wire vocabulary: a
// family produces them and a reader matches on them, so they belong with the
// type that carries them rather than being spelled once in each.
//
// Spelled Account rather than Role because a node's role and an account's
// function are different questions about different things, and sharing the word
// made them look like one (A7).
const (
	AccountValidator  = "validator"
	AccountGovernance = "governance-member"
	AccountNode       = "node"
)

// RingAccount is one account a key set supplies to a chain, with what the chain
// uses it for.
type RingAccount struct {
	// Role is what this account does: a validator, a governance council member,
	// a node identity. It is spelled Role rather than reusing node.Role because
	// a node's role and an account's function are different questions about
	// different things.
	Role string
	// Index is the 1-based position within its role, or 0 when the role is not
	// numbered.
	Index int
	// Address is the account's address (0x-hex).
	Address string
	// Detail says something the operator needs that the role does not, such as
	// whether a validator carries BLS material.
	Detail string
}

// RingAccountReader is an optional ConsensusFamily capability: which accounts a
// family takes out of a key set, beyond the node identities every family uses.
//
// It exists because the answer is family-shaped and was being decided by a
// switch on the family's id in a package above — the same "branch on the target
// instead of asking it" the step ratchet forbids, with the same cost: adding a
// family meant editing that switch as well as the family, and a family the
// switch did not know fell into a default that reported it as unknown rather
// than as unimplemented.
//
// note carries what the roster cannot say in accounts — poa has no validators in
// genesis at all, and a roster that simply omits them looks like a key set that
// is missing some. A family that does not implement this capability supplies
// node identities only.
type RingAccountReader interface {
	RingAccounts(validators, blsKeys, members []string) (accounts []RingAccount, note string)
}

// RuntimeValidatorReader is an optional ConsensusFamily capability: ask a
// running chain, over RPC, which validators it currently recognizes. It exists
// because the answer is not one shape for every family — wbft returns the set
// from a single <ns>_getValidators method, while poa keeps it in a governance
// contract the first producer deploys and must be read by eth_call. Both go
// through the same RPC the harness already reaches a node by (no IPC, so a
// remote or docker node is reached the same way a local one is). A family that
// declares its method in the manifest and needs nothing family-specific does
// not implement it, and the caller falls back to that method.
type RuntimeValidatorReader interface {
	RuntimeValidators(ctx context.Context, c Caller) ([]string, error)
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
