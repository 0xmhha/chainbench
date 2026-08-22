package netcompose

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/topology"
	"github.com/0xmhha/chainbench/internal/engine"
	"github.com/0xmhha/chainbench/internal/serverset"
)

// Composition steps: keys, allocate, genesis, config, launchopts, provision.
// Each reads the accumulated state, fails fast when a prerequisite step has
// not run, performs its one concern through the same core packages the engine
// uses, and records itself in the step table. Lifecycle steps (init, start,
// stop, ...) live in steps_lifecycle.go.

// Placement bounds applied when the caller did not supply a server inventory.
// The port bands themselves live in serverset (built-in defaults, or the
// operator's gitignored inventory) — never here.
const (
	portBandSize              = 100
	minValidatorsForPlacement = 1
)

// plugin resolves the workspace's chain plugin, requiring `new` to have run.
func (w *Workspace) plugin() (registry.ChainPlugin, error) {
	if w.state.Chain == "" && w.state.ManifestPath == "" {
		return nil, fmt.Errorf("netcompose: no chain set — run `net new` first")
	}
	return external.ResolveChain(w.state.Chain, w.state.ManifestPath, w.state.TemplatePath)
}

// KeysOpts selects where node identities come from (algorithm steps 2-3).
type KeysOpts struct {
	// Source is "preset" (default) or "generate".
	Source string
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
		src = engine.GeneratedKeySource{Path: w.state.KeysDir, Validators: opts.Validators}
	default:
		return "", fmt.Errorf("netcompose: keys: unknown source %q (want preset or generate)", opts.Source)
	}
	ks, err := src.Ensure(ctx, n)
	if err != nil {
		return "", err
	}
	detail := fmt.Sprintf("%s: %d identities, %d declared validators",
		src.Describe(), len(ks.Preset.Nodes), len(ks.Preset.Network.Validators))
	w.markStep("keys", detail)
	return detail, nil
}

// AllocateOpts sizes the network.
type AllocateOpts struct {
	// Validators is the validator node count (>=1).
	Validators int
	// Endpoints is the non-validator (endpoint) node count.
	Endpoints int
	// EndpointSyncMode is the geth sync mode endpoints render into their config
	// ("snap" or "archive" instead of the default "full"), so a re-sync test can
	// exercise a path other than full sync. Validators ignore it: a node that
	// seals blocks must hold full state.
	EndpointSyncMode string
	// Topology, when set, gives the layout explicitly — one entry per node, in
	// launch order, each with its own role and sync mode. It replaces the
	// Validators/Endpoints counts and EndpointSyncMode, which cannot express a
	// per-node choice. Its Nodes must already be Validate()d.
	Topology *topology.Topology
	// Placement decides the port bands, the addressing mode, and the capacity
	// bound. Its zero value is the built-in local plan; a caller that read a
	// server inventory passes that server's placement instead, which is the
	// only way site-specific ports enter the composition.
	Placement serverset.Placement
}

// placements resolves the requested layout into one placement request per node,
// in launch order. A topology is authoritative when given; otherwise the counts
// produce validators first, then endpoints.
func (o AllocateOpts) placements() ([]place.NodeReq, []string, error) {
	if o.Topology != nil {
		sorted := o.Topology.Sorted()
		if len(sorted) == 0 {
			return nil, nil, fmt.Errorf("netcompose: allocate: topology has no nodes")
		}
		reqs := make([]place.NodeReq, len(sorted))
		modes := make([]string, len(sorted))
		for i, n := range sorted {
			role := n.NodeRole()
			reqs[i] = place.NodeReq{Name: fmt.Sprintf("node%d", i+1), Role: role}
			// A topology's per-node mode wins; a validator is still pinned to
			// full, since the topology cannot make a sealing node stateless.
			modes[i] = syncModeFor(role, n.EffectiveSyncMode())
		}
		return reqs, modes, nil
	}
	if o.Validators < 1 {
		return nil, nil, fmt.Errorf("netcompose: allocate: at least one validator is required")
	}
	reqs := make([]place.NodeReq, 0, o.Validators+o.Endpoints)
	modes := make([]string, 0, cap(reqs))
	for i := 0; i < o.Validators; i++ {
		reqs = append(reqs, place.NodeReq{Name: fmt.Sprintf("val%d", i+1), Role: node.RoleValidator})
		modes = append(modes, syncModeFull)
	}
	for i := 0; i < o.Endpoints; i++ {
		reqs = append(reqs, place.NodeReq{Name: fmt.Sprintf("ep%d", i+1), Role: node.RoleEndpoint})
		modes = append(modes, syncModeFor(node.RoleEndpoint, o.EndpointSyncMode))
	}
	return reqs, modes, nil
}

// syncModeFor returns the sync mode a node of this role renders. Only endpoints
// are configurable — see AllocateOpts.EndpointSyncMode.
func syncModeFor(role node.Role, endpointMode string) string {
	if netmap.Is(role, node.RoleEN) && endpointMode != "" {
		return endpointMode
	}
	return syncModeFull
}

// syncModeFull is the sync mode every validator uses and the default for
// endpoints.
const syncModeFull = "full"

// Allocate builds the node table: roles, target-side paths, and deterministic
// ports through the same allocator the engine uses. Where the nodes land and on
// what ports comes from the placement, not from this package.
func (w *Workspace) Allocate(opts AllocateOpts) (string, error) {
	if _, err := w.plugin(); err != nil {
		return "", err
	}
	reqs, modes, err := opts.placements()
	if err != nil {
		return "", err
	}
	pl := opts.Placement
	if pl.Source == "" {
		pl = serverset.Builtin(minValidatorsForPlacement, portBandSize)
	}
	placements, err := place.New(pl.Config).Allocate(reqs, pl.Mode, pl.Capacity)
	if err != nil {
		return "", err
	}

	// A server inventory naming a data root wins over the workspace default:
	// it is where that machine keeps node data.
	root := w.state.Target.DataRoot
	if pl.DataRoot != "" {
		root = pl.DataRoot
		w.state.Target.DataRoot = root
	}
	nodes := make([]NodeState, len(placements))
	validators := 0
	for i, p := range placements {
		idx := i + 1
		if netmap.Is(reqs[i].Role, node.RoleBP) {
			validators++
		}
		nodes[i] = NodeState{
			Index:      idx,
			Role:       string(reqs[i].Role),
			SyncMode:   modes[i],
			DataDir:    filepath.Join(root, fmt.Sprintf("node%d", idx)),
			ConfigPath: filepath.Join(root, fmt.Sprintf("config_node%d.toml", idx)),
			LogPath:    filepath.Join(root, "logs", fmt.Sprintf("node%d.log", idx)),
			Host:       p.Host,
			P2P:        p.Ports.P2P, HTTP: p.Ports.HTTP, WS: p.Ports.WS, Auth: p.Ports.Auth, Metrics: p.Ports.Metrics,
		}
	}
	w.state.Nodes = nodes
	// Counted from the resolved placements, not the requested count: a topology
	// decides the validator set, and the genesis step sizes itself from this.
	w.state.Validators = validators
	if opts.Topology != nil {
		w.state.Bootnode = opts.Topology.BootnodeIndex()
	}

	w.state.PortSource = pl.Source

	detail := fmt.Sprintf("%d node(s): %d validator(s) + %d endpoint(s); ports: %s; p2p from %d, http from %d",
		len(nodes), validators, len(nodes)-validators, pl.Source, nodes[0].P2P, nodes[0].HTTP)
	if opts.Topology != nil {
		detail += " (topology)"
	}
	w.markStep("allocate", detail)
	return detail, nil
}

// GenesisOpts customizes the built genesis.
type GenesisOpts struct {
	// ChainID, when non-zero, overrides the manifest chain id.
	ChainID int64
	// Overrides sets bare keys in the genesis `config` object, e.g.
	// {"bohoBlock": "10"} to move a fork off genesis. The fork ordering of the
	// result is validated, so a bad delayed-fork request fails here rather than
	// at node boot.
	Overrides map[string]string
	// Overlay is a genesis JSON fragment deep-merged into the built genesis
	// (extra alloc accounts, config bits). Fork ordering is re-validated after
	// the merge.
	Overlay []byte
	// Capabilities are advertised alongside the network so capability-gated
	// cases run — an overlay declares what it enables.
	Capabilities []string
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
	preset, err := keyring.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("netcompose: genesis: %w", err)
	}
	net := preset.NetworkFor(w.state.Validators)
	gen, err := genesis.Build(p, genesis.Inputs{
		Validators: net.Validators,
		BLSKeys:    net.BLSKeys,
		ExtraData:  net.ExtraData,
		Members:    net.Members,
		Alloc:      net.Alloc,
		ChainID:    opts.ChainID,
	})
	if err != nil {
		return "", err
	}
	gen, err = customizeGenesis(gen, opts)
	if err != nil {
		return "", err
	}
	t, err := w.state.Target.Resolve(w.env)
	if err != nil {
		return "", err
	}
	path := filepath.Join(t.DataRoot, "genesis.json")
	if err := t.Files.Write(ctx, path, gen, 0o644); err != nil {
		return "", fmt.Errorf("netcompose: genesis: write: %w", err)
	}
	w.state.GenesisPath = path
	w.state.Capabilities = networkCapabilities(p.Manifest().Capabilities, opts)

	detail := fmt.Sprintf("%d bytes at %s, %d validator(s)", len(gen), path, w.state.Validators)
	if opts.ChainID != 0 {
		detail += fmt.Sprintf(", chain id %d (override)", opts.ChainID)
	}
	if len(opts.Overrides) > 0 {
		detail += fmt.Sprintf(", %d config override(s)", len(opts.Overrides))
	}
	if len(opts.Overlay) > 0 {
		detail += ", overlay merged"
	}
	w.markStep("genesis", detail)
	return detail, nil
}

// delayedForkSuffix marks a config override that moves a fork off genesis. Such
// a network is advertised as delayed-<fork> so the fork-transition cases gate on
// it and skip on a normal network where the fork is active at genesis.
const delayedForkSuffix = "Block"

// customizeGenesis applies the config overrides and the overlay, re-validating
// fork ordering after each so a bad request fails while composing rather than
// when a node refuses to boot.
func customizeGenesis(gen []byte, opts GenesisOpts) ([]byte, error) {
	var err error
	if len(opts.Overrides) > 0 {
		if gen, err = genesis.ApplyConfigOverrides(gen, opts.Overrides); err != nil {
			return nil, fmt.Errorf("netcompose: genesis overrides: %w", err)
		}
		if err := genesis.ValidateForks(gen); err != nil {
			return nil, fmt.Errorf("netcompose: genesis overrides: %w", err)
		}
	}
	if len(opts.Overlay) > 0 {
		if gen, err = genesis.MergeOverride(gen, opts.Overlay); err != nil {
			return nil, fmt.Errorf("netcompose: genesis overlay: %w", err)
		}
		if err := genesis.ValidateForks(gen); err != nil {
			return nil, fmt.Errorf("netcompose: genesis overlay: %w", err)
		}
	}
	return gen, nil
}

// networkCapabilities is what the composed network advertises: the chain's own
// capabilities, "ws" (composed nodes always serve a WebSocket endpoint), a
// delayed-<fork> marker per fork moved off genesis, and whatever the caller
// declared for its overlay.
func networkCapabilities(manifest []string, opts GenesisOpts) []string {
	caps := append([]string(nil), manifest...)
	caps = append(caps, "ws")
	for _, key := range slices.Sorted(maps.Keys(opts.Overrides)) {
		fork, ok := strings.CutSuffix(key, delayedForkSuffix)
		if !ok || fork == "" {
			continue
		}
		if n, err := strconv.Atoi(opts.Overrides[key]); err == nil && n > 0 {
			caps = append(caps, "delayed-"+strings.ToLower(fork))
		}
	}
	return append(caps, opts.Capabilities...)
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
	preset, err := keyring.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("netcompose: config: %w", err)
	}
	staticNodes := make([]string, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		if nk, ok := preset.Node(ns.Index); ok {
			// The node's own recorded address: on a fleet each node lives on a
			// different host, and a static-node list pointing at this machine
			// would leave every node unable to find its peers.
			staticNodes = append(staticNodes, nodeconfig.Enode(nk.PublicKey, nodeHost(ns), ns.P2P))
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
			Ports:         node.Endpoints{P2P: ns.P2P, HTTP: ns.HTTP, WS: ns.WS, Auth: ns.Auth, Metrics: ns.Metrics},
			KeystoreDir:   filepath.Join(w.state.KeysDir, fmt.Sprintf("node%d", ns.Index), "keystore"),
			RPCNamespace:  m.Consensus.RPCNamespace,
			SyncMode:      ns.SyncMode,
			MinerRecommit: m.MinerRecommit,
			StaticNodes:   staticNodes,
		})
		if err := t.Files.Write(ctx, ns.ConfigPath, content, 0o644); err != nil {
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
	preset, err := keyring.LoadPreset(w.state.KeysDir)
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
		exists, err := t.Files.Exists(ctx, path)
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

// nodeHost is the address a composed node is reachable at: the one the
// allocator recorded, falling back to this machine for a plan that predates
// per-node hosts.
func nodeHost(ns NodeState) string {
	if ns.Host != "" {
		return ns.Host
	}
	return localHost
}

// driverSpec maps a persisted node row onto the driver spec shape the argv
// assembly and lifecycle steps consume.
func driverSpec(ns NodeState) driver.NodeSpec {
	return driver.NodeSpec{
		Index:      ns.Index,
		Role:       node.Role(ns.Role),
		Host:       nodeHost(ns),
		DataDir:    ns.DataDir,
		ConfigPath: ns.ConfigPath,
		LogPath:    ns.LogPath,
		Args:       ns.Args,
		Ports:      node.Endpoints{P2P: ns.P2P, HTTP: ns.HTTP, WS: ns.WS, Auth: ns.Auth, Metrics: ns.Metrics},
	}
}
