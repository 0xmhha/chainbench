// Package registry is the chain-agnostic plugin registry. It holds the
// declarative per-chain Manifest data and the ChainPlugin/ConsensusFamily
// contracts that core code (pipeline, drivers, mcp) uses without importing any
// specific chain — chain knowledge lives in pkg/chains/* and pkg/consensus/*,
// which register here at init (docs/CHAINBENCH_GO_REDESIGN.md §4).
package registry

import (
	"encoding/json"
	"fmt"
)

// Manifest is the declarative static profile of one chain, mirroring
// manifests/chains/<id>.json. It data-izes the facts that were previously
// hardcoded across lib/*.sh and network/internal/probe (binary name, consensus
// namespace, hardfork fields, supported tx types, probe signature) so adding a
// chain is mostly data, not code.
type Manifest struct {
	// ID is the chain identifier ("stablenet"|"wbft"|"wemix").
	ID string `json:"id"`
	// Binary is the node binary name (gstable|gwbft|gwemix).
	Binary string `json:"binary"`
	// Build describes how to build the binary.
	Build BuildSpec `json:"build"`
	// ConsensusFamily is the consensus algorithm family this chain uses
	// ("wbft" for stablenet+wbft, "poa" for wemix). It selects the
	// ConsensusFamily plugin (docs §4, decision D9).
	ConsensusFamily string `json:"consensus_family"`
	// Consensus holds the RPC-facing consensus facts.
	Consensus ConsensusSpec `json:"consensus"`
	// TxTypes is the set of EIP-2718 tx type bytes this chain accepts,
	// as hex strings (e.g. "0x16"). Mirrors accounts/protocol.
	TxTypes []string `json:"tx_types"`
	// Probe is the chain-detection signature (namespace probe method).
	Probe ProbeSpec `json:"probe"`
	// Capabilities is the provider-independent capability set the chain
	// supports (e.g. "consensus").
	Capabilities []string `json:"capabilities"`
}

// BuildSpec describes how to obtain the node binary.
type BuildSpec struct {
	Repo       string `json:"repo"`        // e.g. "go-wbft"
	MakeTarget string `json:"make_target"` // e.g. "gwbft"
}

// ConsensusSpec holds the RPC-facing consensus facts used by verify/consensus.
type ConsensusSpec struct {
	RPCNamespace     string `json:"rpc_namespace"`     // "istanbul" | "wemix"
	ValidatorsMethod string `json:"validators_method"` // "istanbul_getValidators" | ...
}

// ProbeSpec is the chain-detection signature (see network/internal/probe).
type ProbeSpec struct {
	Method   string  `json:"method"`              // RPC method whose presence identifies the chain
	ChainIDs []int64 `json:"chain_ids,omitempty"` // optional chain-id gate for disambiguation
}

// ParseManifest decodes a Manifest from JSON bytes and validates required
// fields.
func ParseManifest(b []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("registry: parse manifest: %w", err)
	}
	if err := m.validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func (m Manifest) validate() error {
	if m.ID == "" {
		return fmt.Errorf("registry: manifest missing id")
	}
	if m.Binary == "" {
		return fmt.Errorf("registry: manifest %q missing binary", m.ID)
	}
	if m.ConsensusFamily == "" {
		return fmt.Errorf("registry: manifest %q missing consensus_family", m.ID)
	}
	return nil
}
