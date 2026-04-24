// Package wemix is the adapter skeleton for the wemix chain type.
// Real GenerateGenesis / GenerateToml implementations are deferred to a
// follow-up sprint; ExtraStartFlags + ConsensusRpcNamespace mirror the
// values emitted by lib/adapters/wemix.sh.
package wemix

import (
	"context"

	"github.com/0xmhha/chainbench/network/internal/adapters/spec"
)

// Adapter is the wemix implementation of spec.Adapter.
type Adapter struct{}

// New returns a zero-configuration wemix adapter.
func New() *Adapter { return &Adapter{} }

// GenerateGenesis is a placeholder — real implementation deferred.
func (*Adapter) GenerateGenesis(_ context.Context, _ spec.GenesisInput) error {
	return spec.ErrNotImplemented
}

// GenerateToml is a placeholder — real implementation deferred.
func (*Adapter) GenerateToml(_ context.Context, _ spec.TomlInput) error {
	return spec.ErrNotImplemented
}

// ExtraStartFlags returns extra CLI flags appended to the client start
// command for this chain type. Role is accepted for interface parity; wemix
// currently returns the same flag set for every role.
func (*Adapter) ExtraStartFlags(_ spec.Role) string { return "--allow-insecure-unlock" }

// ConsensusRpcNamespace returns the consensus RPC module name exposed by
// this chain type.
func (*Adapter) ConsensusRpcNamespace() string { return "wemix" }
