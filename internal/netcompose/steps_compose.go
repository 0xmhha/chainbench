package netcompose

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/engine"
)

// Composition steps: keys, allocate, genesis, config, launchopts, provision.
// Each reads the accumulated state, fails fast when a prerequisite step has
// not run, performs its one concern through the same core packages the engine
// uses, and records itself in the step table. Lifecycle steps (init, start,
// stop, ...) live in steps_lifecycle.go.

// Port allocation mirrors the local engine's constants so a step-composed
// network and an engine run land on the same layout.
const (
	localP2PBase  = 31000
	localRPCBase  = 8600
	localPortStep = 10
	portBandSize  = 100
)

// plugin resolves the workspace's chain plugin, requiring `new` to have run.
func (w *Workspace) plugin() (registry.ChainPlugin, error) {
	if w.state.Chain == "" {
		return nil, fmt.Errorf("netcompose: no chain set — run `net new` first")
	}
	return registry.Get(w.state.Chain)
}

// KeysOpts selects where node identities come from (algorithm steps 2-3).
type KeysOpts struct {
	// Source is "preset" (default) or "generate".
	Source string
	// Bootnode is the external bootnode binary (required for generate).
	Bootnode string
	// Nodes is how many identities the set must cover; <=0 uses the node table
	// length, falling back to the validator count.
	Nodes int
	// Validators is how many identities join the validator set (generate only);
	// <=0 means all.
	Validators int
}

// Keys ensures the workspace's key set exists and covers the requested node
// count, through the same engine.KeySource seam `chainbench run` uses.
func (w *Workspace) Keys(ctx context.Context, opts KeysOpts) (string, error) {
	if _, err := w.plugin(); err != nil {
		return "", err
	}
	n := opts.Nodes
	if n <= 0 {
		n = len(w.state.Nodes)
	}
	if n <= 0 {
		n = opts.Validators
	}
	if n <= 0 {
		return "", fmt.Errorf("netcompose: keys: node count unknown — run `net allocate` first or pass --nodes")
	}

	var src engine.KeySource
	switch opts.Source {
	case "", "preset":
		src = engine.PresetKeySource{Path: w.state.KeysDir}
	case "generate":
		if opts.Bootnode == "" {
			return "", fmt.Errorf("netcompose: keys: --keys-source=generate needs --bootnode (BLS derivation)")
		}
		src = engine.GeneratedKeySource{Path: w.state.KeysDir, Bootnode: opts.Bootnode, Validators: opts.Validators}
	default:
		return "", fmt.Errorf("netcompose: keys: unknown source %q (want preset or generate)", opts.Source)
	}
	ks, err := src.Ensure(ctx, n)
	if err != nil {
		return "", err
	}
	detail := fmt.Sprintf("%s: %d identities, %d validators", src.Describe(), len(ks.Preset.Nodes), len(ks.Preset.Validators))
	w.markStep("keys", detail)
	return detail, nil
}

// AllocateOpts sizes the network.
type AllocateOpts struct {
	// Validators is the validator node count (>=1).
	Validators int
	// Endpoints is the non-validator (endpoint) node count.
	Endpoints int
}

// Allocate builds the node table: roles, target-side paths, and deterministic
// stepped ports on 127.0.0.1 through the same allocator the engine uses.
func (w *Workspace) Allocate(opts AllocateOpts) (string, error) {
	if _, err := w.plugin(); err != nil {
		return "", err
	}
	if opts.Validators < 1 {
		return "", fmt.Errorf("netcompose: allocate: at least one validator is required")
	}
	reqs := make([]place.NodeReq, 0, opts.Validators+opts.Endpoints)
	for i := 0; i < opts.Validators; i++ {
		reqs = append(reqs, place.NodeReq{Name: fmt.Sprintf("val%d", i+1), Role: node.RoleValidator})
	}
	for i := 0; i < opts.Endpoints; i++ {
		reqs = append(reqs, place.NodeReq{Name: fmt.Sprintf("ep%d", i+1), Role: node.RoleEndpoint})
	}
	alloc := place.New(place.Config{P2PBase: localP2PBase, P2PStep: localPortStep, RPCBase: localRPCBase, RPCStep: localPortStep})
	placements, err := alloc.Allocate(reqs, place.LocalStepped, place.Capacity{MinValidators: 1, PortBandSize: portBandSize})
	if err != nil {
		return "", err
	}

	root := w.state.Target.DataRoot
	nodes := make([]NodeState, len(placements))
	for i, p := range placements {
		idx := i + 1
		nodes[i] = NodeState{
			Index:      idx,
			Role:       string(reqs[i].Role),
			DataDir:    filepath.Join(root, fmt.Sprintf("node%d", idx)),
			ConfigPath: filepath.Join(root, fmt.Sprintf("config_node%d.toml", idx)),
			LogPath:    filepath.Join(root, "logs", fmt.Sprintf("node%d.log", idx)),
			P2P:        p.Ports.P2P, HTTP: p.Ports.HTTP, WS: p.Ports.WS, Auth: p.Ports.Auth,
		}
	}
	w.state.Nodes = nodes
	w.state.Validators = opts.Validators

	detail := fmt.Sprintf("%d node(s): %d validator(s) + %d endpoint(s); p2p from %d, http from %d",
		len(nodes), opts.Validators, opts.Endpoints, nodes[0].P2P, nodes[0].HTTP)
	w.markStep("allocate", detail)
	return detail, nil
}

// GenesisOpts customizes the built genesis.
type GenesisOpts struct {
	// ChainID, when non-zero, overrides the manifest chain id.
	ChainID int64
}

// Genesis builds the genesis from the key set's validator material and writes
// it to the target's data root (upload-if-absent semantics are the provision
// step's concern; genesis always reflects the current inputs).
func (w *Workspace) Genesis(ctx context.Context, opts GenesisOpts) (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if w.state.Validators <= 0 {
		return "", fmt.Errorf("netcompose: genesis: no validators — run `net allocate` first")
	}
	preset, err := keys.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("netcompose: genesis: %w", err)
	}
	sub := preset.Take(w.state.Validators)
	gen, err := genesis.Build(p, genesis.Inputs{
		Validators: sub.Validators,
		BLSKeys:    sub.BLSKeys,
		ExtraData:  sub.ExtraData,
		Members:    sub.Members,
		Alloc:      sub.Alloc,
		ChainID:    opts.ChainID,
	})
	if err != nil {
		return "", err
	}
	t, err := w.state.Target.Resolve(w.env)
	if err != nil {
		return "", err
	}
	path := filepath.Join(t.DataRoot, "genesis.json")
	if err := t.Sink.Write(ctx, path, gen, 0o644); err != nil {
		return "", fmt.Errorf("netcompose: genesis: write: %w", err)
	}
	w.state.GenesisPath = path

	detail := fmt.Sprintf("%d bytes at %s, %d validator(s)", len(gen), path, w.state.Validators)
	if opts.ChainID != 0 {
		detail += fmt.Sprintf(", chain id %d (override)", opts.ChainID)
	}
	w.markStep("genesis", detail)
	return detail, nil
}

// Config renders each node's TOML config and writes it to the target.
func (w *Workspace) Config(ctx context.Context) (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if len(w.state.Nodes) == 0 {
		return "", fmt.Errorf("netcompose: config: no node table — run `net allocate` first")
	}
	preset, err := keys.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("netcompose: config: %w", err)
	}
	staticNodes := make([]string, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		if nk, ok := preset.Node(ns.Index); ok {
			staticNodes = append(staticNodes, nodeconfig.Enode(nk.PublicKey, "127.0.0.1", ns.P2P))
		}
	}
	t, err := w.state.Target.Resolve(w.env)
	if err != nil {
		return "", err
	}
	m := p.Manifest()
	for _, ns := range w.state.Nodes {
		content := nodeconfig.Generate(nodeconfig.Params{
			Role:          node.Role(ns.Role),
			Ports:         node.Endpoints{P2P: ns.P2P, HTTP: ns.HTTP, WS: ns.WS, Auth: ns.Auth},
			KeystoreDir:   filepath.Join(w.state.KeysDir, fmt.Sprintf("node%d", ns.Index), "keystore"),
			RPCNamespace:  m.Consensus.RPCNamespace,
			MinerRecommit: m.MinerRecommit,
			StaticNodes:   staticNodes,
		})
		if err := t.Sink.Write(ctx, ns.ConfigPath, content, 0o644); err != nil {
			return "", fmt.Errorf("netcompose: config: node%d: %w", ns.Index, err)
		}
	}
	detail := fmt.Sprintf("%d config(s) under %s", len(w.state.Nodes), t.DataRoot)
	w.markStep("config", detail)
	return detail, nil
}

// LaunchOptsOpts customizes the assembled argv.
type LaunchOptsOpts struct {
	// Set are high-precedence launch knobs ("key=value", bare key for boolean
	// flags), applied through the launchopt Builder's override layer.
	Set []string
}

// LaunchOpts assembles each node's launch argv through engine.NodeLaunchArgs —
// the single argv assembly site — and records it in the node table, so `start`
// launches exactly what this step showed.
func (w *Workspace) LaunchOpts(opts LaunchOptsOpts) (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if len(w.state.Nodes) == 0 {
		return "", fmt.Errorf("netcompose: launchopts: no node table — run `net allocate` first")
	}
	preset, err := keys.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("netcompose: launchopts: %w", err)
	}
	overrides, err := ParseOverrides(opts.Set)
	if err != nil {
		return "", err
	}
	for i, ns := range w.state.Nodes {
		args, err := engine.NodeLaunchArgs(p, preset, driverSpec(ns), w.state.KeysDir, overrides)
		if err != nil {
			return "", fmt.Errorf("netcompose: launchopts: node%d: %w", ns.Index, err)
		}
		w.state.Nodes[i].Args = args
	}
	detail := fmt.Sprintf("%d argv(s) assembled", len(w.state.Nodes))
	if len(opts.Set) > 0 {
		detail += ", overrides: " + strings.Join(opts.Set, " ")
	}
	w.markStep("launchopts", detail)
	return detail, nil
}

// Provision materializes the shared launch inputs on the target with
// upload-if-absent semantics: the genesis (as built by the genesis step) and
// the per-node configs. Re-running it reuses what already exists.
func (w *Workspace) Provision(ctx context.Context) (string, error) {
	if w.state.GenesisPath == "" {
		return "", fmt.Errorf("netcompose: provision: no genesis — run `net genesis` first")
	}
	if len(w.state.Nodes) == 0 {
		return "", fmt.Errorf("netcompose: provision: no node table — run `net allocate` first")
	}
	t, err := w.state.Target.Resolve(w.env)
	if err != nil {
		return "", err
	}
	present := 0
	check := func(ctx context.Context, path string) error {
		exists, err := t.Sink.Exists(ctx, path)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("netcompose: provision: %s missing — run the genesis/config steps first", path)
		}
		present++
		return nil
	}
	if err := check(ctx, w.state.GenesisPath); err != nil {
		return "", err
	}
	for _, ns := range w.state.Nodes {
		if err := check(ctx, ns.ConfigPath); err != nil {
			return "", err
		}
	}
	detail := fmt.Sprintf("%d launch input(s) present on the target (reused, not rewritten)", present)
	w.markStep("provision", detail)
	return detail, nil
}

// ParseOverrides maps "key=value" strings (bare key for booleans) onto typed
// launchopt overrides. Whether a key exists for the target binary is checked
// at assembly by the Builder.
func ParseOverrides(sets []string) ([]launchopt.Override, error) {
	out := make([]launchopt.Override, 0, len(sets))
	for _, s := range sets {
		k, v, _ := strings.Cut(s, "=")
		if k == "" {
			return nil, fmt.Errorf("netcompose: bad --set %q (want key=value or a bare boolean key)", s)
		}
		out = append(out, launchopt.Override{Key: launchopt.Key(k), Value: v})
	}
	return out, nil
}

// driverSpec maps a persisted node row onto the driver spec shape the argv
// assembly and lifecycle steps consume.
func driverSpec(ns NodeState) driver.NodeSpec {
	return driver.NodeSpec{
		Index:      ns.Index,
		Role:       node.Role(ns.Role),
		Host:       "127.0.0.1",
		DataDir:    ns.DataDir,
		ConfigPath: ns.ConfigPath,
		LogPath:    ns.LogPath,
		Args:       ns.Args,
		Ports:      node.Endpoints{P2P: ns.P2P, HTTP: ns.HTTP, WS: ns.WS, Auth: ns.Auth},
	}
}
