package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// The read-only questions: what this bench knows, and what a running node
// says. They pass through here for the same reason the state-changing verbs do
// — so that a question asked at the CLI and the same question asked by a tool
// are answered by one piece of code.

type (
	// ChainPlugin is a registered chain: its manifest, capabilities and rules.
	ChainPlugin = registry.ChainPlugin
	// Capability is one advertised capability of a chain.
	Capability = registry.Capability
	// Descriptor is one system contract: its address and what it is for.
	Descriptor = registry.Descriptor
)

// CommonChain is the pseudo-chain holding capabilities every chain shares.
const CommonChain = registry.CommonChain

// Chains names every registered chain.
func Chains(_ Deps) []string { return registry.Names() }

// Chain looks a chain up by id.
func Chain(_ Deps, id string) (ChainPlugin, error) { return registry.Get(id) }

// Capabilities lists every capability every chain advertises.
func Capabilities(_ Deps) []Capability { return registry.All() }

// CapabilitiesFor lists what one chain advertises.
func CapabilitiesFor(_ Deps, chain string) []Capability { return registry.For(chain) }

// CapabilityByAddress finds the capability a system-contract address belongs
// to, which is how a caller turns an address in a log into a name.
func CapabilityByAddress(_ Deps, addr string) (Descriptor, bool) {
	return registry.GetByAddress(addr)
}

// CapabilityByName finds a capability by its version.chain.name spelling.
func CapabilityByName(_ Deps, name string) (Capability, bool) { return registry.Lookup(name) }

// ValidatorsOut is a chain's validator set as a node reports it, with the RPC
// method that was asked so a caller can say where the answer came from.
type ValidatorsOut struct {
	Chain      string   `json:"chain"`
	Method     string   `json:"method"`
	Validators []string `json:"validators"`
}

// Validators reports a chain's validator set as the node sees it.
//
// Which RPC namespace holds the answer differs per consensus family, and the
// chain's manifest knows. Looking it up here means neither surface has to, and
// neither can look it up differently.
func Validators(ctx context.Context, _ Deps, chain, manifest, template, rpcURL string) (ValidatorsOut, error) {
	if rpcURL == "" {
		return ValidatorsOut{}, fmt.Errorf("a node endpoint is required")
	}
	p, err := ResolveChain(chain, manifest, template)
	if err != nil {
		return ValidatorsOut{}, err
	}
	method := p.Manifest().Consensus.ValidatorsMethod
	vals, err := registry.Validators(ctx, rpc.Dial(rpcURL), method)
	if err != nil {
		return ValidatorsOut{}, err
	}
	return ValidatorsOut{Chain: p.Manifest().ID, Method: method, Validators: vals}, nil
}

// NodeCallIn forwards a raw JSON-RPC call to a node, so an operator can ask a
// node anything the bench has no verb for.
type NodeCallIn struct {
	RPC    string
	Method string
	Params []any
}

// NodeCall makes the call and returns the raw JSON result.
func NodeCall(ctx context.Context, _ Deps, in NodeCallIn) ([]byte, error) {
	if in.RPC == "" || in.Method == "" {
		return nil, fmt.Errorf("an endpoint and a method are required")
	}
	var out any
	if err := rpc.Dial(in.RPC).Call(ctx, in.Method, &out, in.Params...); err != nil {
		return nil, err
	}
	return json.Marshal(out)
}
