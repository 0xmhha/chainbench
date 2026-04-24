// Package adapters exposes a chain-type-agnostic init layer for future
// chainbench-net subcommands.
//
// The Adapter interface (defined in the sibling `spec` package and
// re-exported here so callers need only one import) defines:
//
//   - GenerateGenesis
//   - GenerateToml
//   - ExtraStartFlags
//   - ConsensusRpcNamespace
//
// Bash adapters under lib/adapters/ remain the active `chainbench init`
// execution path — this package is additive and grows into a drop-in
// replacement over Sprint 3c.
package adapters

import (
	"github.com/0xmhha/chainbench/network/internal/adapters/spec"
	"github.com/0xmhha/chainbench/network/internal/adapters/stablenet"
	"github.com/0xmhha/chainbench/network/internal/adapters/wbft"
	"github.com/0xmhha/chainbench/network/internal/adapters/wemix"
)

// Re-exports so callers only need to import this package.
type (
	Adapter      = spec.Adapter
	GenesisInput = spec.GenesisInput
	TomlInput    = spec.TomlInput
	Role         = spec.Role
	Profile      = spec.Profile
	Metadata     = spec.Metadata
)

const (
	RoleValidator = spec.RoleValidator
	RoleEndpoint  = spec.RoleEndpoint
)

var (
	ErrUnknownChainType = spec.ErrUnknownChainType
	ErrNotImplemented   = spec.ErrNotImplemented
)

// Load returns the Adapter registered for chainType, or ErrUnknownChainType
// when no adapter handles it.
func Load(chainType string) (Adapter, error) {
	switch chainType {
	case "stablenet":
		return stablenet.New(), nil
	case "wbft":
		return wbft.New(), nil
	case "wemix":
		return wemix.New(), nil
	default:
		return nil, ErrUnknownChainType
	}
}
