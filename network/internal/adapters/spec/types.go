// Package spec holds shared adapter types + interface + error sentinels.
// Lives at the leaf of the import graph so each chain adapter and the top
// level Load() can import it without cycles.
package spec

import (
	"context"
	"errors"
)

// Role identifies the operational role of a node within the local network.
type Role string

const (
	RoleValidator Role = "validator"
	RoleEndpoint  Role = "endpoint"
)

// Profile is the decoded chain profile (config/profiles/<name>.yaml -> JSON-ish).
type Profile map[string]any

// Metadata is the decoded per-network metadata emitted by the init flow.
type Metadata map[string]any

// GenesisInput carries everything needed to materialize a genesis.json for
// a chain-specific adapter.
type GenesisInput struct {
	Profile       Profile
	Metadata      Metadata
	TemplateBytes []byte
	OutputPath    string
	NumValidators int
	BaseP2P       int
}

// TomlInput carries everything needed to materialize per-node TOML configs.
type TomlInput struct {
	Metadata      Metadata
	TemplateBytes []byte
	WriteDir      string
	TotalNodes    int
	NumValidators int
	BaseP2P       int
	BaseHTTP      int
	BaseWS        int
	BaseAuth      int
	BaseMetrics   int
}

// FeeDelegateDynamicFeeTxType is the go-stablenet FeeDelegateDynamicFeeTx type
// byte (0x16). Defined here so each adapter can declare support for it via
// SupportedTxTypes without importing the handler package (which owns the
// envelope-building logic). Distinct from go-ethereum's standard typed-tx
// prefixes (0x01/0x02/0x03/0x04).
const FeeDelegateDynamicFeeTxType byte = 0x16

// Adapter is the chain-type-specific strategy for initializing a local
// chainbench network.
type Adapter interface {
	GenerateGenesis(ctx context.Context, input GenesisInput) error
	GenerateToml(ctx context.Context, input TomlInput) error
	ExtraStartFlags(role Role) string
	ConsensusRpcNamespace() string
	// SupportedTxTypes reports the chain-specific tx type bytes this chain
	// accepts beyond the Ethereum baseline (e.g. go-stablenet's 0x16
	// FeeDelegateDynamicFeeTx). Empty/nil means Ethereum-baseline only.
	SupportedTxTypes() []byte
}

var (
	// ErrUnknownChainType is returned by Load when no adapter handles the
	// requested chain type.
	ErrUnknownChainType = errors.New("adapters: unknown chain type")
	// ErrNotImplemented is returned by adapter methods that are present as
	// skeletons but whose real implementation has not been ported yet.
	ErrNotImplemented = errors.New("adapters: operation not implemented for this chain type")
)
