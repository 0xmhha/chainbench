// What a chain provides, said in the words a test asks in.
//
// This is NOT the capability registry in capability.go, which answers "what
// tools does the MCP surface offer for chain X". These are the facts a DSL case
// gates on, derived from the manifest's own data.

package registry

import (
	"sort"
	"strings"
)

// Capability prefixes. A prefix says which kind of fact the name after it is,
// so a requirement reads as a question about the chain rather than as a bare
// word whose meaning depends on where it came from.
//
// A capability with no prefix is one the harness or an overlay advertises
// (rpc, ws, account-extra, delayed-boho); those keep the spelling they have.
const (
	// CapContract asks for one of the chain's own contracts by name. The name is
	// what a chain calls it, not where it sits: the same address holds a
	// different contract on different chains.
	CapContract = "contract:"
	// CapFork asks for a hardfork the chain's genesis knows.
	CapFork = "fork:"
	// CapEngine asks for the consensus config block a chain's genesis carries
	// ("anzeon", "croissant"), which is what decides rules a test can only meet
	// on a chain that has it — a fee floor, an epoch layout.
	CapEngine = "engine:"
	// CapFamily asks for the consensus algorithm family.
	CapFamily = "family:"
	// CapTx asks for an EIP-2718 transaction type the chain accepts.
	CapTx = "tx:"
	// CapPrecompile asks for a precompiled contract the chain's EVM answers at.
	//
	// It is declared rather than derived, because whether a precompile is LIVE
	// is not a fact about the chain alone: go-stablenet and go-wbft both build
	// p256Verify, and each activates it at its own fork — boho and croissant.
	// A default wbft network has croissant at block 0 and answers; a default
	// stablenet network declares no bohoBlock at all and does not. So the
	// manifest says what its ordinary network provides, and a network that
	// turns the fork on declares the capability alongside its overlay.
	CapPrecompile = "precompile:"
)

// ForkActiveIn reports whether a genesis template switches the named fork on.
//
// Genesis.Hardforks is the list of forks a chain KNOWS, in activation order —
// that is what `chains hardforks` prints and what it should keep meaning. It is
// not the list a default network HAS. stablenet knows applepie and boho and its
// template turns on neither, so deriving a capability from the list advertised
// two forks no ordinary stablenet network answers for.
//
// Measured 2026-09-19: boho-crossed-by-restart requires fork:boho, the gate let
// it through on a network whose genesis has no bohoBlock, and it failed on an
// assertion about govMinter instead of skipping. The same file already takes the
// opposite care with CapPrecompile, and for this exact fork — so the rule was in
// the room, applied to one capability and not its neighbour.
//
// A network that does turn a fork on says so itself: an override moves it off
// genesis and is advertised as delayed-<fork>, and an overlay declares what it
// provides. Nothing here has to guess.
//
// The template is read as text rather than decoded. A fork appears either as
// "<fork>Block" (a block number) or as "<fork>": { ... } (an engine section),
// the shapes the chains actually use, and a decoder would need a config struct
// per chain generation to see either.
func ForkActiveIn(genesisTemplate []byte, fork string) bool {
	if len(genesisTemplate) == 0 || fork == "" {
		// No static template: the family writes the genesis, and what it turns
		// on is not knowable here. Advertising the declared list is what this
		// did before, and the chains in that position (poa) declare only forks
		// their genesis carries.
		return true
	}
	t := string(genesisTemplate)
	return strings.Contains(t, `"`+fork+`Block"`) || strings.Contains(t, `"`+fork+`"`)
}

// CapabilityPrefixes is every prefix a requirement may carry, for the message
// that refuses one it does not know.
var CapabilityPrefixes = []string{CapContract, CapFork, CapEngine, CapFamily, CapTx, CapPrecompile}

// DerivedCapabilities is what this manifest's own data says the chain provides.
//
// They are derived rather than listed because the facts are already here: the
// hardforks are in the genesis spec, the contracts in the contract table, the
// family in its own field. Writing them into Capabilities as well would put one
// fact in two places, and the one that is not read is the one that goes stale.
//
// The result is sorted so a composed network's advertised set is the same on
// every run, which is what lets a run record be compared with another.
// genesisTemplate is the chain's template bytes (ChainPlugin.GenesisTemplate).
// Only the forks that template actually switches on are advertised; nil means a
// chain with no static template, whose forks its family writes at genesis time.
func (m Manifest) DerivedCapabilities(genesisTemplate []byte) []string {
	out := make([]string, 0, len(m.SystemContracts)+len(m.Genesis.Hardforks)+len(m.TxTypes)+2)
	for name := range m.SystemContracts {
		out = append(out, CapContract+name)
	}
	for _, fork := range m.Genesis.Hardforks {
		if !ForkActiveIn(genesisTemplate, fork) {
			continue
		}
		out = append(out, CapFork+fork)
	}
	if m.Genesis.EngineField != "" {
		out = append(out, CapEngine+m.Genesis.EngineField)
	}
	if m.ConsensusFamily != "" {
		out = append(out, CapFamily+m.ConsensusFamily)
	}
	for _, t := range m.TxTypes {
		out = append(out, CapTx+strings.ToLower(t))
	}
	sort.Strings(out)
	return out
}

// MalformedCapability explains why req is not a capability anyone can provide,
// or "" when it is well formed.
//
// Only the shape is judged. Whether a given chain provides it is a different
// question with a different answer per chain, and the one a spec is skipped
// for; this one is a typo, and a typo should not read as "no chain has it".
func MalformedCapability(req string) string {
	name := strings.TrimSpace(req)
	if name == "" {
		return "a requirement cannot be empty"
	}
	i := strings.Index(name, ":")
	if i < 0 {
		return "" // unprefixed: advertised by the harness, an overlay, or a chain
	}
	prefix := name[:i+1]
	for _, p := range CapabilityPrefixes {
		if p != prefix {
			continue
		}
		if strings.TrimSpace(name[i+1:]) == "" {
			return prefix + " needs a name after it"
		}
		return ""
	}
	return "unknown requirement prefix " + prefix + " (want " + strings.Join(CapabilityPrefixes, " ") + ")"
}
