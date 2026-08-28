package testengine

import (
	"context"
	"github.com/0xmhha/chainbench/internal/core/launcher"
	"github.com/0xmhha/chainbench/internal/testspec"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// The composition side lives in chainsetup — building an environment,
// sourcing keys, bootstrapping a producer — and the runner consumes it
// through this bridge. It is the V6 boundary: the local engine still asks the
// setup module to build its environment; the workflow layer (V6.2) will
// compose the two from above, and what remains here then is only what the
// runner genuinely needs by name.
type (
	// BuildEnvFunc provisions and brings up a network for a spec.
	BuildEnvFunc = chainsetup.BuildEnvFunc
	// TeardownFunc tears a built environment down at the end of a session.
	TeardownFunc = chainsetup.TeardownFunc
	// KeySource supplies the key material an environment is built from.
	KeySource = chainsetup.KeySource
	// PresetKeySource reads keys from a preset ring.
	PresetKeySource = chainsetup.PresetKeySource
	// GeneratedKeySource derives fresh keys per run.
	GeneratedKeySource = chainsetup.GeneratedKeySource
	// GenesisConfig shapes the genesis the environment composes.
	GenesisConfig = chainsetup.GenesisConfig
	// LocalLauncher launches nodes on this machine.
	LocalLauncher = launcher.Direct
	// WemixBootstrap runs the wemix producer bring-up between phases.
	WemixBootstrap = chainsetup.WemixBootstrap
	// BuildDeps are what the environment builder needs.
	BuildDeps = chainsetup.BuildDeps
)

// NewBuildEnv builds the environment builder from its dependencies.
func NewBuildEnv(d chainsetup.BuildDeps) BuildEnvFunc { return chainsetup.NewBuildEnv(d) }

// GenesisSourceFor picks the genesis source for a chain plugin.
func GenesisSourceFor(plugin registry.ChainPlugin, cfg GenesisConfig) chainsetup.GenesisSource {
	return chainsetup.GenesisSourceFor(plugin, cfg)
}

// NewNodeController controls launched nodes over the launcher.
func NewNodeController(direct LocalLauncher, procs *process.Manager) *launcher.Controller {
	return launcher.NewController(direct, procs)
}

// The controller is what the DSL's fault steps drive.
var _ testspec.NodeControl = (*launcher.Controller)(nil)

// KeySet is a resolved set of key material.
type KeySet = chainsetup.KeySet

// RegisterIdentities loads a key set's identities into a ring.
func RegisterIdentities(ctx context.Context, ring *store.KeySet, ks KeySet, n int) error {
	return chainsetup.RegisterIdentities(ctx, ring, ks, n)
}

// BuildLocalPlan assembles the local run plan for a spec's chain.
var BuildLocalPlan = chainsetup.BuildLocalPlan

// LocalSetup is the local environment setup a plan runs through.
type LocalSetup = chainsetup.LocalSetup

// NodeLaunchArgs assembles one node's launch argv.
var NodeLaunchArgs = launcher.NodeLaunchArgs
