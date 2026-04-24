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

// Adapter is the chain-type-specific strategy for initializing a local
// chainbench network.
type Adapter interface {
	GenerateGenesis(ctx context.Context, input GenesisInput) error
	GenerateToml(ctx context.Context, input TomlInput) error
	ExtraStartFlags(role Role) string
	ConsensusRpcNamespace() string
}

var (
	// ErrUnknownChainType is returned by Load when no adapter handles the
	// requested chain type.
	ErrUnknownChainType = errors.New("adapters: unknown chain type")
	// ErrNotImplemented is returned by adapter methods that are present as
	// skeletons but whose real implementation has not been ported yet.
	ErrNotImplemented = errors.New("adapters: operation not implemented for this chain type")
)
