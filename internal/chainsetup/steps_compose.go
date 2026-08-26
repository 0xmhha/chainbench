package chainsetup

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/topology"
	netmapmod "github.com/0xmhha/chainbench/internal/netmap"
)

// Composition steps: keys, allocate, genesis, config, launchopts, filestore.
// Each reads the accumulated state, fails fast when a prerequisite step has
// not run, performs its one concern through the same core packages the engine
// uses, and records itself in the step table. Lifecycle steps (init, start,
// stop, ...) live in steps_lifecycle.go.

// Placement bounds applied when the caller did not supply a server set.
// The port bands themselves live in serverset (built-in defaults, or the
// operator's gitignored server set) — never here.
const (
	portBandSize              = 100
	minValidatorsForPlacement = 1
)

// plugin resolves the workspace's chain plugin, requiring `new` to have run.
func (w *Workspace) plugin() (registry.ChainPlugin, error) {
	if w.state.Chain == "" && w.state.ManifestPath == "" {
		return nil, fmt.Errorf("chainsetup: no chain set — run `net new` first")
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
// count, through the same KeySource seam `chainbench run` uses.
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
		return "", fmt.Errorf("chainsetup: keys: node count unknown — run `net allocate` first or pass --nodes")
	}

	var src KeySource
	switch opts.Source {
	case "", "preset":
		src = PresetKeySource{Path: w.state.KeysDir}
	case "generate":
		src = GeneratedKeySource{Path: w.state.KeysDir, Validators: opts.Validators}
	default:
		return "", fmt.Errorf("chainsetup: keys: unknown source %q (want preset or generate)", opts.Source)
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
	// Peering is the peer graph to wire ("mesh" default, "proxied" for
	// bp <-> pn <-> en). It is recorded now and consumed by the config step,
	// so the graph a network runs is decided where its layout is.
	Peering string
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
	// server set passes that server's placement instead, which is the
	// only way site-specific ports enter the composition.
	Placement netmapmod.Placement
	// SetPath is the server-set file Placement came from, persisted so later
	// steps resolve the same file (and, in docker mode, its sibling localmap).
	SetPath string
}

// placements resolves the requested layout into one placement request per node,
// in launch order. A topology is authoritative when given; otherwise the counts
// produce validators first, then endpoints.
func (o AllocateOpts) placements() ([]place.NodeReq, []string, error) {
	if o.Topology != nil {
		sorted := o.Topology.Sorted()
		if len(sorted) == 0 {
			return nil, nil, fmt.Errorf("chainsetup: allocate: topology has no nodes")
		}
		reqs := make([]place.NodeReq, len(sorted))
		modes := make([]string, len(sorted))
		for i, n := range sorted {
			role := n.NodeRole()
			reqs[i] = place.NodeReq{Role: role}
			// A topology's per-node mode wins; a validator is still pinned to
			// full, since the topology cannot make a sealing node stateless.
			modes[i] = syncModeFor(role, n.EffectiveSyncMode())
		}
		return reqs, modes, nil
	}
	if o.Validators < 1 {
		return nil, nil, fmt.Errorf("chainsetup: allocate: at least one validator is required")
	}
	reqs := make([]place.NodeReq, 0, o.Validators+o.Endpoints)
	modes := make([]string, 0, cap(reqs))
	for i := 0; i < o.Validators; i++ {
		reqs = append(reqs, place.NodeReq{Role: node.RoleValidator})
		modes = append(modes, syncModeFull)
	}
	for i := 0; i < o.Endpoints; i++ {
		reqs = append(reqs, place.NodeReq{Role: node.RoleEndpoint})
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
	plugin, err := w.plugin()
	if err != nil {
		return "", err
	}
	reqs, modes, err := opts.placements()
	if err != nil {
		return "", err
	}
	pl := opts.Placement
	if pl.Source == "" {
		pl = netmapmod.Builtin(minValidatorsForPlacement, portBandSize)
	}
	if opts.SetPath != "" {
		w.state.ServerSet = opts.SetPath
	}
	pool := pl.Pool
	if pool.Slots < 1 {
		pool.Slots = 1
	}
	// The family says how much room a node needs; a wemix node's embedded etcd
	// takes two ports beyond p2p, and sizing the step for a wbft node would put
	// the next node on top of it.
	pool.Reservation = plugin.Family().PortReservation()
	assigned, err := netmap.Assign(pool, netmapRequests(reqs))
	if err != nil {
		return "", err
	}
	placements := assigned.Placements()

	// A server set naming a data root wins over the workspace default:
	// it is where that machine keeps node data.
	root := w.state.Target.DataRoot
	if pl.DataRoot != "" {
		root = pl.DataRoot
		w.state.Target.DataRoot = root
	}
	layout := netmap.Layout{Root: root}
	// On a fleet each node's machine is a server-set entry; record its name so
	// every later step opens THAT machine. Addresses came from the pool, so
	// the name is the pool's word for the address.
	nameOf := map[string]string{}
	if pl.Remote {
		for _, h := range pl.Pool.Hosts {
			if h.Name != "" && h.Name != h.Addr {
				nameOf[h.Addr] = h.Name
			}
		}
	}
	nodes := make([]NodeState, len(placements))
	validators := 0
	for i, p := range placements {
		if netmap.Is(p.Role, node.RoleBP) {
			validators++
		}
		nodes[i] = NodeState{
			Server:     nameOf[p.Host],
			Index:      p.Index,
			Label:      string(p.Label),
			Role:       string(reqs[i].Role),
			SyncMode:   modes[i],
			DataDir:    layout.DataDir(p.Label),
			ConfigPath: layout.ConfigPath(p.Label),
			LogPath:    layout.LogPath(p.Label),
			Host:       p.Host,
			Endpoints:  p.Ports,
		}
	}
	// Reject an impossible graph here rather than at config time: the operator
	// is choosing the layout in this step.
	peering, err := netmap.ParsePeering(opts.Peering)
	if err != nil {
		return "", err
	}
	w.state.Peering = string(peering)
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
		return "", fmt.Errorf("chainsetup: genesis: no validators — run `net allocate` first")
	}
	// A family whose genesis its binary writes takes a different source: the
	// generic dispatch builds a genesis by substituting a template, and for
	// wemix that produces a file that initializes cleanly and runs the wrong
	// consensus.
	art, err := w.genesisArtifacts(ctx, p, opts)
	if err != nil {
		return "", err
	}
	gen := art.Genesis
	// Every machine gets the genesis (and its by-products): each node's init
	// reads it locally, and on a fleet "locally" is that node's server.
	path := filepath.Join(w.state.Target.DataRoot, "genesis.json")
	err = w.eachMachine(func(t *machine.Access, _ []NodeState) error {
		p := filepath.Join(t.DataRoot, "genesis.json")
		if err := t.Files.Write(ctx, p, gen, 0o644); err != nil {
			return fmt.Errorf("chainsetup: genesis: write: %w", err)
		}
		// The step's by-products go beside the genesis: a wemix bring-up
		// reads its governance config back during deploy-governance.
		for name, content := range art.Extra {
			if err := t.Files.Write(ctx, filepath.Join(t.DataRoot, name), content, 0o644); err != nil {
				return fmt.Errorf("chainsetup: genesis: write %s: %w", name, err)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("chainsetup: config: no node table — run `net allocate` first")
	}
	preset, err := store.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("chainsetup: config: %w", err)
	}
	placed, err := w.Netmap()
	if err != nil {
		return "", fmt.Errorf("chainsetup: config: %w", err)
	}
	peering, err := netmap.ParsePeering(w.state.Peering)
	if err != nil {
		return "", fmt.Errorf("chainsetup: config: %w", err)
	}
	if err := peering.Validate(placed, p.Family().SupportsRole); err != nil {
		return "", fmt.Errorf("chainsetup: config: %w", err)
	}
	// The peer's own recorded address: on a fleet each node lives on a
	// different host, and a static-node list pointing at this machine would
	// leave every node unable to find its peers. Keys reach the composition
	// as inputs — the netmap module joins them to placements.
	pubkey := func(index int) (string, bool) {
		nk, ok := preset.Node(index)
		if !ok {
			return "", false
		}
		return nk.PublicKey, true
	}
	m := p.Manifest()
	for _, ns := range w.state.Nodes {
		t, err := w.machineFor(ns)
		if err != nil {
			return "", err
		}
		staticNodes, err := netmapmod.PeerList(placed, peering, ns.NodeLabel(), pubkey)
		if err != nil {
			return "", fmt.Errorf("chainsetup: config: node%d peers: %w", ns.Index, err)
		}
		content := nodeconfig.Generate(nodeconfig.Params{
			Role:          node.Role(ns.Role),
			Ports:         ns.Endpoints,
			KeystoreDir:   filepath.Join(w.keysBase(), fmt.Sprintf("node%d", ns.Index), "keystore"),
			RPCNamespace:  m.Consensus.RPCNamespace,
			SyncMode:      ns.SyncMode,
			MinerRecommit: m.MinerRecommit,
			StaticNodes:   staticNodes,
		})
		if err := t.Files.Write(ctx, ns.ConfigPath, content, 0o644); err != nil {
			return "", fmt.Errorf("chainsetup: config: node%d: %w", ns.Index, err)
		}
	}
	detail := fmt.Sprintf("%d config(s) under %s", len(w.state.Nodes), w.state.Target.DataRoot)
	w.markStep("config", detail)
	return detail, nil
}

// LaunchOptsOpts customizes the assembled argv.
type LaunchOptsOpts struct {
	// Set are high-precedence launch knobs ("key=value", bare key for boolean
	// flags), applied through the launchopt Builder's override layer.
	Set []string
}

// LaunchOpts assembles each node's launch argv through NodeLaunchArgs —
// the single argv assembly site — and records it in the node table, so `start`
// launches exactly what this step showed.
func (w *Workspace) LaunchOpts(opts LaunchOptsOpts) (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if len(w.state.Nodes) == 0 {
		return "", fmt.Errorf("chainsetup: launchopts: no node table — run `net allocate` first")
	}
	preset, err := store.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("chainsetup: launchopts: %w", err)
	}
	overrides, err := ParseOverrides(opts.Set)
	if err != nil {
		return "", err
	}
	for i, ns := range w.state.Nodes {
		args, err := NodeLaunchArgs(p, preset, driverSpec(ns), w.keysBase(), overrides)
		if err != nil {
			return "", fmt.Errorf("chainsetup: launchopts: node%d: %w", ns.Index, err)
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
		return "", fmt.Errorf("chainsetup: provision: no genesis — run `net genesis` first")
	}
	if len(w.state.Nodes) == 0 {
		return "", fmt.Errorf("chainsetup: provision: no node table — run `net allocate` first")
	}
	present, shipped := 0, 0
	err := w.eachMachine(func(t *machine.Access, nodes []NodeState) error {
		check := func(path string) error {
			exists, err := t.Files.Exists(ctx, path)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("chainsetup: provision: %s missing — run the genesis/config steps first", path)
			}
			present++
			return nil
		}
		if err := check(w.state.GenesisPath); err != nil {
			return err
		}
		for _, ns := range nodes {
			if err := check(ns.ConfigPath); err != nil {
				return err
			}
		}
		n, err := w.shipIdentities(ctx, t, nodes)
		shipped += n
		return err
	})
	if err != nil {
		return "", err
	}
	detail := fmt.Sprintf("%d launch input(s) present on the target (reused, not rewritten)", present)
	if shipped > 0 {
		detail += fmt.Sprintf(", %d identity file(s) shipped to %s", shipped, w.keysBase())
	}
	w.markStep("provision", detail)
	return detail, nil
}

// shipIdentities uploads each node's identity files — the devp2p nodekey, the
// validator keystore, and the shared password — from the local key set to
// keysBase on a remote target, upload-if-absent like the rest of filestore.
// The rendered config and the launch argv point at keysBase, so without this
// a remote node would look for its keys on the operator's machine. A local
// target ships nothing: keysBase is the key set itself.
func (w *Workspace) shipIdentities(ctx context.Context, t *machine.Access, nodes []NodeState) (int, error) {
	if !t.Spec.IsRemote() {
		return 0, nil
	}
	shipped := 0
	put := func(src, dst string, mode fs.FileMode) error {
		b, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // e.g. an endpoint node with no keystore
			}
			return err
		}
		exists, err := t.Files.Exists(ctx, dst)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		if err := t.Files.Write(ctx, dst, b, mode); err != nil {
			return err
		}
		shipped++
		return nil
	}
	base := w.keysBase()
	if err := put(filepath.Join(w.state.KeysDir, "password"), filepath.Join(base, "password"), 0o600); err != nil {
		return shipped, fmt.Errorf("chainsetup: provision: password: %w", err)
	}
	for _, ns := range nodes {
		src := filepath.Join(w.state.KeysDir, fmt.Sprintf("node%d", ns.Index))
		dst := filepath.Join(base, fmt.Sprintf("node%d", ns.Index))
		if err := put(filepath.Join(src, "nodekey"), filepath.Join(dst, "nodekey"), 0o600); err != nil {
			return shipped, fmt.Errorf("chainsetup: provision: node%d nodekey: %w", ns.Index, err)
		}
		entries, err := os.ReadDir(filepath.Join(src, "keystore"))
		if err != nil {
			continue // no keystore for this node
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if err := put(filepath.Join(src, "keystore", e.Name()),
				filepath.Join(dst, "keystore", e.Name()), 0o600); err != nil {
				return shipped, fmt.Errorf("chainsetup: provision: node%d keystore: %w", ns.Index, err)
			}
		}
	}
	return shipped, nil
}

// ParseOverrides maps "key=value" strings (bare key for booleans) onto typed
// launchopt overrides. Whether a key exists for the target binary is checked
// at assembly by the Builder.
func ParseOverrides(sets []string) ([]launchopt.Override, error) {
	out := make([]launchopt.Override, 0, len(sets))
	for _, s := range sets {
		k, v, _ := strings.Cut(s, "=")
		if k == "" {
			return nil, fmt.Errorf("chainsetup: bad --set %q (want key=value or a bare boolean key)", s)
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
		Ports:      ns.Endpoints,
	}
}

// Netmap reads the workspace's node table as a placement map, so the peer
// policy and the address lookups run off one representation. The host is the
// node's own recorded address, which on a fleet is not this machine.
func (w *Workspace) Netmap() (*netmap.Map, error) {
	placements := make([]netmap.Placement, 0, len(w.state.Nodes))
	ordinals := map[node.Role]int{}
	for _, ns := range w.state.Nodes {
		role, err := netmap.NormalizeRole(ns.Role)
		if err != nil {
			return nil, fmt.Errorf("node%d: %w", ns.Index, err)
		}
		ordinals[role]++
		placements = append(placements, netmap.Placement{
			Index:   ns.Index,
			Label:   ns.NodeLabel(),
			Role:    role,
			Ord:     ordinals[role],
			Host:    nodeHost(ns),
			Ports:   ns.Endpoints,
			DataDir: ns.DataDir,
		})
	}
	return netmap.NewMap(placements)
}

// netmapRequests turns the composed node list into placement requests. Only the
// role travels: position comes from the order, which is also the node's
// identity.
func netmapRequests(reqs []place.NodeReq) []netmap.Request {
	out := make([]netmap.Request, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, netmap.Request{Role: r.Role})
	}
	return out
}

// genesisArtifacts builds the genesis through the one composition every surface
// uses. The wemix source runs the chain binary, so the request also carries the
// placement: the governance config names the producer by host and p2p port.
func (w *Workspace) genesisArtifacts(ctx context.Context, p registry.ChainPlugin, opts GenesisOpts) (GenesisArtifacts, error) {
	req := GenesisRequest{Validators: w.state.Validators}
	if p.Family().ID() == poa.FamilyID {
		placed, err := w.Netmap()
		if err != nil {
			return GenesisArtifacts{}, fmt.Errorf("chainsetup: genesis: %w", err)
		}
		req.Nodes = placed
	}
	return BuildGenesis(ctx, p, req, GenesisConfig{
		KeysDir:         w.state.KeysDir,
		Binary:          w.state.Binary,
		ChainID:         opts.ChainID,
		ConfigOverrides: opts.Overrides,
		Overlay:         opts.Overlay,
	})
}
