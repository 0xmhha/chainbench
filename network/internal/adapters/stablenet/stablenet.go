// Package stablenet is the adapter for the stablenet chain type.
//
// ExtraStartFlags + ConsensusRpcNamespace mirror the bash adapter
// (lib/adapters/stablenet.sh) byte-for-byte. GenerateGenesis and
// GenerateToml are stubs today and will be filled in by follow-up tasks
// in sprint 3c; until then they return spec.ErrNotImplemented and callers
// keep using the bash adapter via `chainbench init`.
package stablenet

import (
	"context"

	"github.com/0xmhha/chainbench/network/internal/adapters/spec"
)

// Adapter is the stablenet implementation of spec.Adapter.
type Adapter struct{}

// New returns a zero-configuration stablenet adapter.
func New() *Adapter { return &Adapter{} }

// GenerateGenesis will port lib/adapters/stablenet.sh's Python genesis
// builder; for now it returns spec.ErrNotImplemented.
//
// TODO(sprint-3c/task-2): implement.
func (*Adapter) GenerateGenesis(_ context.Context, _ spec.GenesisInput) error {
	return spec.ErrNotImplemented
}

// GenerateToml will port lib/adapters/stablenet.sh's per-node TOML emitter;
// for now it returns spec.ErrNotImplemented.
//
// TODO(sprint-3c/task-3): implement.
func (*Adapter) GenerateToml(_ context.Context, _ spec.TomlInput) error {
	return spec.ErrNotImplemented
}

// ExtraStartFlags returns the CLI flags appended to the stablenet client
// start command. Validators additionally get `--mine` so they participate
// in block production.
func (*Adapter) ExtraStartFlags(role spec.Role) string {
	flags := "--allow-insecure-unlock --rpc.enabledeprecatedpersonal --rpc.allow-unprotected-txs"
	if role == spec.RoleValidator {
		flags += " --mine"
	}
	return flags
}

// ConsensusRpcNamespace returns the consensus RPC module name exposed by
// stablenet (WBFT consensus uses the `istanbul` namespace).
func (*Adapter) ConsensusRpcNamespace() string { return "istanbul" }
