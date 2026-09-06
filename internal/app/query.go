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
	// Param is one input a capability takes.
	Param = registry.Param
)

// RegisterCapability records a capability a surface itself provides, so that
// what the bench advertises includes the surface's own tools rather than only
// the chains'.
func RegisterCapability(_ Deps, version, chain, group, name, desc string, params []Param) {
	registry.RegisterFlat(version, chain, group, name, desc, params)
}

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

// NodeReadings is what a single node says about itself in one pass: its head,
// its chain, its peers, and whether it is still catching up.
type NodeReadings struct {
	BlockNumber uint64 `json:"blockNumber"`
	ChainID     uint64 `json:"chainId"`
	PeerCount   uint64 `json:"peerCount"`
	Syncing     bool   `json:"syncing"`
}

// ReadNode takes those four readings.
//
// Four calls in one place because they are one question — "what is this node
// doing" — and a surface that took three of them would answer it differently
// from one that took four.
//
// Only the head is required. The rest are best-effort: a node with the admin or
// net namespace switched off still answers about its head, and refusing to
// report anything because it will not name its peers would turn a working node
// into an error. That was the behaviour of both surfaces before this call
// existed, and keeping it is why the failures are dropped rather than returned.
func ReadNode(ctx context.Context, _ Deps, rpcURL string) (NodeReadings, error) {
	if rpcURL == "" {
		return NodeReadings{}, fmt.Errorf("a node endpoint is required")
	}
	c := rpc.Dial(rpcURL)
	head, err := c.BlockNumber(ctx)
	if err != nil {
		return NodeReadings{}, err
	}
	r := NodeReadings{BlockNumber: head}
	r.ChainID, _ = c.ChainID(ctx)
	r.PeerCount, _ = c.PeerCount(ctx)
	r.Syncing, _ = c.Syncing(ctx)
	return r, nil
}
