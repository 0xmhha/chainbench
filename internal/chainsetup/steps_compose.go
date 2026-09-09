package chainsetup

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/blueprint"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
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
		return nil, fmt.Errorf("chainsetup: no chain set — run `chain new` first")
	}
	return external.ResolveChain(w.state.Chain, w.state.ManifestPath, w.state.TemplatePath)
}

// KeysOpts selects where node identities come from (algorithm steps 2-3).
type KeysOpts struct {
	// Source is "preset" (default), "generate", or "declared".
	Source string
	// Blueprint is the declaration the keys come from when Source is
	// "declared". Its nodes carry their own nodekeys, which is what lets a
	// network be composed with no preset directory anywhere (N3).
	Blueprint *blueprint.Blueprint
	// Nodes is how many identities the set must cover; <=0 uses the node table
	// length, falling back to the validator count.
	Nodes int
	// Validators is how many identities join the validator set (generate only);
	// <=0 means all.
	Validators int
}

// Keys ensures the workspace's key set exists and covers the requested node
// count, through the same KeySource boundary `chainbench run` uses.
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
		return "", fmt.Errorf("chainsetup: keys: node count unknown — run `chain place` first or pass --nodes")
	}

	// A node table that names any per-node key builds the set from the table:
	// declared where a key is given, generated where not. This fixes each
	// producer's genesis validator address to its key, and hands a non-producer
	// its nodekey and enode without making it a validator. It takes precedence
	// over the source string because a table that declares keys has said where
	// its identities come from.
	var src store.KeySource
	if set, keyed, kerr := w.nodeTableKeys(ctx, n); kerr != nil {
		return "", kerr
	} else if keyed {
		src = store.DeclaredKeys{Path: w.state.KeysDir, Set: set}
	} else {
		switch opts.Source {
		case "", "preset":
			src = store.PresetKeys{Path: w.state.KeysDir}
		case "declared":
			// The declaration is the origin, and the ring is materialised from it
			// so that the genesis source, the launcher and provision keep reading
			// keys the one way they already do.
			if opts.Blueprint == nil {
				return "", fmt.Errorf("chainsetup: keys: source %q needs a blueprint to take the keys from", opts.Source)
			}
			set, err := w.declaredKeys(*opts.Blueprint, n)
			if err != nil {
				return "", err
			}
			src = store.DeclaredKeys{Path: w.state.KeysDir, Set: set}
		case "generate":
			// A generated set must declare exactly the topology's validators, not
			// make every node one: a network with endpoints (4 bp + 11 en) whose key
			// set claims 15 validators fails genesis, where the governance contract
			// requires members and validators to match. The allocated count is the
			// authority; an explicit opts.Validators still wins.
			validators := opts.Validators
			if validators <= 0 {
				validators = w.state.Validators
			}
			src = store.GeneratedKeys{Path: w.state.KeysDir, Validators: validators}
		default:
			return "", fmt.Errorf("chainsetup: keys: unknown source %q (want preset, generate or declared)", opts.Source)
		}
	}
	ks, err := src.Ensure(ctx, n)
	if err != nil {
		return "", err
	}
	detail := fmt.Sprintf("%s: %d identities, %d declared validators",
		src.Describe(), len(ks.Nodes), len(ks.Network.Validators))
	w.markStep("keys", detail)
	return detail, nil
}

// declaredKeys derives the ring a blueprint declares, for the network the
// placement has already decided.
//
// It resolves against the node table this workspace allocated rather than
// against the document alone: the identity that matters is the node's index,
// which is what its datadir, its keyring entry and its enode are all named
// from. Deriving against a different table would produce keys that are correct
// in isolation and attached to the wrong nodes.
func (w *Workspace) declaredKeys(bp blueprint.Blueprint, n int) (keyring.Preset, error) {
	placed, err := w.Netmap()
	if err != nil {
		return keyring.Preset{}, fmt.Errorf("chainsetup: keys: %w — run `chain place` first", err)
	}
	r, err := blueprint.Resolve(bp, blueprint.Inputs{
		Placed: placed.Placements(),
		Chain:  blueprint.ChainFacts{ID: w.state.Chain, Binary: w.state.Binary},
		Layout: node.Layout{Root: w.state.Target.DataRoot},
	})
	if err != nil {
		return keyring.Preset{}, err
	}
	// BLS material is derived for every family, which is what the generated
	// source already does. Only wbft reads it, and asking the family instead
	// would be the better answer, but there is no method that says so today and
	// inventing one here would put the question in two places. Recorded as N3
	// debt rather than guessed at.
	set, err := blueprint.PresetFrom(r, derive.WithBLS, os.ReadFile)
	if err != nil {
		return keyring.Preset{}, err
	}
	if len(set.Nodes) < n {
		return keyring.Preset{}, fmt.Errorf("chainsetup: keys: the blueprint declares %d identities and the network has %d nodes", len(set.Nodes), n)
	}
	return set, nil
}

// nodeTableKeys builds the key set a node table with per-node keys asks for:
// a node that names a key uses it (case a), and a node that names none is
// generated (case b). A producer's key fixes its genesis validator address; a
// non-producer takes the key as its nodekey and enode without becoming a
// validator (case c). The second result is false when no node names a key, so
// the caller falls back to the source string.
//
// The generated identities come from store.Generate — the one module that
// creates keys, with the entropy, BLS derivation, keystores and password the
// rest of the system expects — rather than being hand-rolled here. Only their
// key material is taken; DeclaredKeys re-writes the ring (keystores, password,
// metadata) at the workspace's key dir, the way the declared source already does.
func (w *Workspace) nodeTableKeys(ctx context.Context, n int) (keyring.Preset, bool, error) {
	byIndex := make(map[int]node.Record, len(w.state.Nodes))
	keyed := false
	for _, r := range w.state.Nodes {
		byIndex[r.Index] = r
		if r.Key != "" {
			keyed = true
		}
	}
	if !keyed {
		return keyring.Preset{}, false, nil
	}

	// Generate a full set once, to a throwaway dir, for the entropy of the nodes
	// that name no key. Its keystores are not reused — DeclaredKeys re-writes
	// them from the key material below — so the dir is temporary.
	tmp, err := os.MkdirTemp("", "cb-nodekeys-")
	if err != nil {
		return keyring.Preset{}, true, fmt.Errorf("chainsetup: keys: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	gen, err := store.GenerateAt(ctx, store.GenerateOpts{Nodes: n, Out: tmp, Derive: derive.WithBLS}, nil)
	if err != nil {
		return keyring.Preset{}, true, fmt.Errorf("chainsetup: keys: generate node identities: %w", err)
	}

	var set keyring.Preset
	for i := 1; i <= n; i++ {
		r, ok := byIndex[i]
		if !ok {
			return keyring.Preset{}, true, fmt.Errorf("chainsetup: keys: node table has no node%d", i)
		}
		var key derive.PrivateKey
		if r.Key != "" {
			key, err = parseNodeKey(r.Key)
			if err != nil {
				return keyring.Preset{}, true, fmt.Errorf("chainsetup: keys: node%d: %w", i, err)
			}
		} else {
			key = gen.Nodes[i-1].Nodekey
		}
		id, derr := derive.Derive(key, derive.WithBLS)
		if derr != nil {
			return keyring.Preset{}, true, fmt.Errorf("chainsetup: keys: node%d: %w", i, derr)
		}
		set.Nodes = append(set.Nodes, keyring.Entry{
			Label:    keyring.Label(node.LabelFor(i)),
			Index:    i,
			Nodekey:  key,
			Identity: id,
		})
	}
	// The validator set is the producers, in index order — the same rule the
	// genesis source applies. A non-producer holds a key but is not listed.
	for i := 1; i <= n; i++ {
		if node.Is(node.Role(byIndex[i].Role), node.RoleBP) {
			set.Network.Validators = append(set.Network.Validators, set.Nodes[i-1].Address)
		}
	}
	return set, true, nil
}

// parseNodeKey reads a node's declared key. A path that exists is read as a key
// file; anything else is parsed as 0x-hex, so a network can pin a key inline or
// point at a file, the way a blueprint's nodekey does.
func parseNodeKey(ref string) (derive.PrivateKey, error) {
	if _, err := os.Stat(ref); err == nil {
		b, rerr := os.ReadFile(ref)
		if rerr != nil {
			return derive.PrivateKey{}, fmt.Errorf("read key file %q: %w", ref, rerr)
		}
		return derive.ParsePrivateKey(string(b))
	}
	return derive.ParsePrivateKey(ref)
}

// AllocateOpts sizes the network.
type AllocateOpts struct {
	// Validators is the validator node count (>=1).
	Validators int
	// Endpoints is the non-validator (endpoint) node count.
	Endpoints int
	// Proxies is the pn (proxy-tier) node count. A family with no proxy tier
	// (poa, where etcd occupies that place) refuses a pn at peering validation.
	Proxies int
	// Peering is the peer graph to wire ("mesh" default, "proxied" for
	// bp <-> pn <-> en). It is recorded now and consumed by the config step,
	// so the graph a network runs is decided where its layout is.
	Peering string
	// EndpointSyncMode is the geth sync mode endpoints render into their config
	// ("snap" or "archive" instead of the default "full"), so a re-sync test can
	// exercise a path other than full sync. Validators ignore it: a node that
	// seals blocks must hold full state.
	EndpointSyncMode string
	// Binaries maps a per-node binary name (as the topology references it) to
	// its resolved path, recorded on the workspace so launch resolves each
	// node's binary. Empty means every node runs the single binary.
	Binaries map[string]string
	// Topology, when set, gives the layout explicitly — one entry per node, in
	// launch order, each with its own role and sync mode. It replaces the
	// Validators/Endpoints counts and EndpointSyncMode, which cannot express a
	// per-node choice. Its Nodes must already be Validate()d.
	Topology *node.Topology
	// Blueprint, when set, is the network declaration the layout comes from
	// (N1-N3). It is the widest of the three sources — a topology says role and
	// sync mode per node, a blueprint says those and the keys, ports, server and
	// binary too — so it wins over both.
	Blueprint *blueprint.Blueprint
	// Pool decides the port bands and the capacity
	// bound. Its zero value is the built-in local plan; a caller that read a
	// server set passes that server's placement instead, which is the
	// only way site-specific ports enter the composition.
	Pool resource.Pool
	// AutoSize fills the network to the server set: the count is one node per
	// server (len(Pool.Hosts)), not a figure the spec named. Proxies and
	// Endpoints still say how many of those the operator wants (one each by
	// default); the rest are validators. It needs a resolved server set — with
	// no pool there is no capacity to fill — and the pn is placed last so it
	// lands on the last server (the discovery hub the model puts there).
	AutoSize bool
	// SetPath is the server-set file Pool came from, persisted so later
	// steps resolve the same file (and, in docker mode, its sibling localmap).
	SetPath string
}

// placements resolves the requested layout into one placement request per node,
// in launch order. A topology is authoritative when given; otherwise the counts
// produce validators first, then endpoints.
func (o AllocateOpts) placements() ([]node.LaunchReq, []string, error) {
	if o.Blueprint != nil {
		return blueprintPlacements(*o.Blueprint)
	}
	if o.Topology != nil {
		sorted := o.Topology.Sorted()
		if len(sorted) == 0 {
			return nil, nil, fmt.Errorf("chainsetup: allocate: topology has no nodes")
		}
		reqs := make([]node.LaunchReq, len(sorted))
		modes := make([]string, len(sorted))
		for i, n := range sorted {
			role := n.NodeRole()
			reqs[i] = node.LaunchReq{Role: role, Binary: n.Binary, Config: n.Config, Key: n.Key}
			// A topology's per-node mode wins; a validator is still pinned to
			// full, since the topology cannot make a sealing node stateless.
			modes[i] = syncModeFor(role, n.EffectiveSyncMode())
		}
		return reqs, modes, nil
	}
	validators := o.Validators
	if o.AutoSize {
		v, err := o.autoValidators()
		if err != nil {
			return nil, nil, err
		}
		validators = v
	}
	if validators < 1 {
		return nil, nil, fmt.Errorf("chainsetup: allocate: at least one validator is required")
	}
	reqs := make([]node.LaunchReq, 0, validators+o.Proxies+o.Endpoints)
	modes := make([]string, 0, cap(reqs))
	for i := 0; i < validators; i++ {
		reqs = append(reqs, node.LaunchReq{Role: node.RoleBP})
		modes = append(modes, syncModeFull)
	}
	// Order differs by path. A named count keeps bp, pn, en — the ordering
	// existing specs address by index. AutoSize instead ends on the pn, so the
	// last node lands on the last server: that node is the discovery hub the
	// unified model puts at the highest index (and, on wemix, still leaves the
	// etcd seed as the highest-index bp, which comes before it either way).
	appendProxies := func() {
		for i := 0; i < o.Proxies; i++ {
			reqs = append(reqs, node.LaunchReq{Role: node.RolePN})
			modes = append(modes, syncModeFor(node.RolePN, o.EndpointSyncMode))
		}
	}
	appendEndpoints := func() {
		for i := 0; i < o.Endpoints; i++ {
			reqs = append(reqs, node.LaunchReq{Role: node.RoleEN})
			modes = append(modes, syncModeFor(node.RoleEN, o.EndpointSyncMode))
		}
	}
	if o.AutoSize {
		appendEndpoints()
		appendProxies()
	} else {
		appendProxies()
		appendEndpoints()
	}
	return reqs, modes, nil
}

// autoValidators sizes the validator count from the server set: one node per
// server, less the proxies and endpoints the operator asked for. It is the
// count form's dynamic default (bp: "max"), so the same spec fills a 6-server
// set with 6 nodes and a 15-server set with 15.
func (o AllocateOpts) autoValidators() (int, error) {
	servers := len(o.Pool.Hosts)
	if servers == 0 {
		return 0, fmt.Errorf("chainsetup: allocate: dynamic sizing (bp: \"max\") needs a server-set target — there is no capacity to fill without one")
	}
	validators := servers - o.Proxies - o.Endpoints
	if validators < 1 {
		return 0, fmt.Errorf("chainsetup: allocate: %d server(s) cannot hold %d pn + %d en and still leave a validator", servers, o.Proxies, o.Endpoints)
	}
	return validators, nil
}

// blueprintPlacements turns a declaration's node table into one placement
// request per node.
//
// Only what the ALLOCATION needs is read here: the role decides where a node
// lands and how much port room it takes, and the sync mode is recorded with the
// layout. Keys, accounts and pinned ports are the resolver's business, and
// reading them twice is how two places come to disagree about one document.
func blueprintPlacements(bp blueprint.Blueprint) ([]node.LaunchReq, []string, error) {
	if len(bp.Nodes) == 0 {
		return nil, nil, fmt.Errorf("chainsetup: allocate: the blueprint declares no nodes")
	}
	reqs := make([]node.LaunchReq, len(bp.Nodes))
	modes := make([]string, len(bp.Nodes))
	for i, n := range bp.Nodes {
		// An unstated role is a producer. A declaration whose nodes say nothing
		// is the smallest network anyone writes, and it has to be one that
		// seals.
		role := node.RoleBP
		if n.Role != "" {
			r, err := node.NormalizeRole(n.Role)
			if err != nil {
				return nil, nil, fmt.Errorf("chainsetup: allocate: blueprint node %d: %w", i+1, err)
			}
			role = r
		}
		// A field this step cannot honour is refused by name rather than
		// dropped. Silently ignoring a declared value is the failure this whole
		// track exists to end: the network comes up looking right and running
		// something the document does not describe.
		if n.Server != "" {
			return nil, nil, fmt.Errorf("chainsetup: allocate: blueprint node %d declares server %q, and per-node server placement is not wired yet (N3) — remove it or use a server set", i+1, n.Server)
		}
		reqs[i] = node.LaunchReq{Role: role}
		// A sealing node is pinned to full whatever the document says: it must
		// hold full state, and the declaration cannot make it stateless.
		modes[i] = syncModeFor(role, n.SyncMode)
	}
	return reqs, modes, nil
}

// syncModeFor returns the sync mode a node of this role renders. Only endpoints
// are configurable — see AllocateOpts.EndpointSyncMode.
func syncModeFor(role node.Role, endpointMode string) string {
	if node.Is(role, node.RoleEN) && endpointMode != "" {
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
	pool := opts.Pool
	if pool.Source == "" {
		pool = resource.Builtin(minValidatorsForPlacement, portBandSize)
	}
	if opts.SetPath != "" {
		w.state.ServerSet = opts.SetPath
	}
	if pool.Slots < 1 {
		pool.Slots = 1
	}
	// The family says how much room a node needs; a wemix node's embedded etcd
	// takes two ports beyond p2p, and sizing the step for a wbft node would put
	// the next node on top of it.
	pool.Reservation = plugin.Family().PortReservation()
	// Draw from the set's inventory, not from an empty pool: every other
	// composition on this set already holds its slots, and the second network
	// must not be handed the first one's ports.
	inv, err := Inventory(pool, w.Dir())
	if err != nil {
		return "", err
	}
	assigned, err := inv.Assign(netmapRequests(reqs), w.Dir())
	if err != nil {
		return "", err
	}
	placements := assigned.Placements()

	// The data root is the target's: a server set naming one reached the
	// workspace through Retarget before this step ran, so there is one answer
	// rather than a copy that can disagree with it.
	layout := node.Layout{Root: w.state.Target.DataRoot}
	// Spread across a set, each node's machine is a server-set entry; record
	// its name so every later step opens THAT resource. Addresses came from the
	// pool, so the name is the pool's word for the address.
	nameOf := map[string]string{}
	if w.state.Target.IsRemote() {
		for _, h := range pool.Hosts {
			if h.Name != "" && h.Name != h.Addr {
				nameOf[h.Addr] = h.Name
			}
		}
	}
	nodes := make([]node.Record, len(placements))
	validators := 0
	for i, p := range placements {
		if node.Is(p.Role, node.RoleBP) {
			validators++
		}
		nodes[i] = node.Record{
			Server:     nameOf[p.Host],
			Index:      p.Index,
			Label:      string(p.Label),
			Role:       string(reqs[i].Role),
			SyncMode:   modes[i],
			Binary:     reqs[i].Binary,
			DataDir:    layout.DataDir(p.Label),
			ConfigPath: layout.ConfigPath(p.Label),
			LogPath:    layout.LogPath(p.Label),
			Host:       p.Host,
			Endpoints:  p.Ports,
			Config:     reqs[i].Config,
			Key:        reqs[i].Key,
		}
	}
	// Reject an impossible graph here rather than at config time: the operator
	// is choosing the layout in this step.
	peering, err := node.ParsePeering(opts.Peering)
	if err != nil {
		return "", err
	}
	w.state.Peering = string(peering)
	w.state.Nodes = nodes
	if len(opts.Binaries) > 0 {
		w.state.Binaries = opts.Binaries
	}
	// Counted from the resolved placements, not the requested count: a topology
	// decides the validator set, and the genesis step sizes itself from this.
	w.state.Validators = validators
	if opts.Topology != nil {
		w.state.Bootnode = opts.Topology.BootnodeIndex()
	}

	w.state.PortSource = pool.Source

	detail := fmt.Sprintf("%d node(s): %d validator(s) + %d endpoint(s); ports: %s; p2p from %d, http from %d",
		len(nodes), validators, len(nodes)-validators, pool.Source, nodes[0].P2P, nodes[0].HTTP)
	if opts.Topology != nil {
		detail += " (topology)"
	}
	w.markStep("place", detail)
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
	if err := w.require("genesis"); err != nil {
		return "", err
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
	// reads it locally, and spread across a set "locally" is that node's server.
	path := filepath.Join(w.state.Target.DataRoot, "genesis.json")
	err = w.eachMachine(func(t *resource.Access, _ []node.Record) error {
		p := filepath.Join(t.DataRoot, "genesis.json")
		if err := t.Files.Write(ctx, p, gen, 0o644); err != nil {
			return fmt.Errorf("chainsetup: genesis: write: %w", err)
		}
		w.recordInput(p, gen)
		// The step's by-products go beside the genesis: a wemix bring-up
		// reads its governance config back during deploy-governance.
		for name, content := range art.Extra {
			extra := filepath.Join(t.DataRoot, name)
			if err := t.Files.Write(ctx, extra, content, 0o644); err != nil {
				return fmt.Errorf("chainsetup: genesis: write %s: %w", name, err)
			}
			w.recordInput(extra, content)
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
	if err := w.require("config"); err != nil {
		return "", err
	}
	preset, placed, peering, pubkey, err := w.peerPlan(p)
	if err != nil {
		return "", fmt.Errorf("chainsetup: config: %w", err)
	}
	// Recomputed for this config revision: a re-run reflects the current
	// overrides, not a stale record.
	w.state.ConfigProvenance = w.state.ConfigProvenance[:0]
	overridden := 0
	for _, ns := range w.state.Nodes {
		prov, err := w.writeNodeConfig(ctx, p, preset, placed, peering, pubkey, ns, "")
		if err != nil {
			return "", err
		}
		w.state.ConfigProvenance = append(w.state.ConfigProvenance, prov)
		if len(prov.Overrides) > 0 {
			overridden++
		}
	}
	detail := fmt.Sprintf("%d config(s) under %s", len(w.state.Nodes), w.state.Target.DataRoot)
	if overridden > 0 {
		detail += fmt.Sprintf(", %d with overrides", overridden)
	}
	w.markStep("config", detail)
	return detail, nil
}

// writeNodeConfig renders one node's config with its overrides, writes it,
// verifies the readback checksum, and returns its provenance. The Config step
// and a mid-test config swap share it, so both produce the same config and the
// same provenance record. purpose, when set, names the swap's config fixture
// (config-<purpose>); the initial compose passes "".
func (w *Workspace) writeNodeConfig(ctx context.Context, p registry.ChainPlugin, preset keyring.Preset, placed *node.Map, peering node.Peering, pubkey func(int) (string, bool), ns node.Record, purpose string) (ConfigProvenance, error) {
	t, err := w.machineFor(ns)
	if err != nil {
		return ConfigProvenance{}, err
	}
	// A node that names its own config file uses it verbatim: the file is the
	// whole config, so the composition renders nothing and applies no overrides
	// for it. It still goes through the same write + readback as a rendered one,
	// so a truncated copy is caught here rather than at boot.
	if ns.Config != "" {
		toml, rerr := os.ReadFile(ns.Config)
		if rerr != nil {
			return ConfigProvenance{}, fmt.Errorf("chainsetup: config: node%d: read pinned config %s: %w", ns.Index, ns.Config, rerr)
		}
		return w.writeConfigFile(ctx, t, ns, toml, purpose, nil)
	}
	staticNodes, err := node.PeerList(placed, peering, ns.NodeLabel(), pubkey)
	if err != nil {
		return ConfigProvenance{}, fmt.Errorf("chainsetup: config: node%d peers: %w", ns.Index, err)
	}
	spec := process.NodeConfig(p, preset, process.SpecOf(ns), w.keysBase(), staticNodes)
	overrides := w.configOverridesFor(ns.Index)
	if err := w.applyConfigOverrides(&spec, ns.Index); err != nil {
		return ConfigProvenance{}, fmt.Errorf("chainsetup: config: node%d: %w", ns.Index, err)
	}
	toml := nodeconfig.TOML(spec)
	return w.writeConfigFile(ctx, t, ns, toml, purpose, overrides)
}

// writeConfigFile writes one node's config to its target and reads it back:
// the config on the target must hash to what was written, so a truncated or
// clobbered write is caught here rather than at node boot. The checksum runs
// on the target (a remote store runs sha256sum), so it does not download the
// file back. overrides is nil for a pinned config file — it applied none.
func (w *Workspace) writeConfigFile(ctx context.Context, t *resource.Access, ns node.Record, toml []byte, purpose string, overrides []string) (ConfigProvenance, error) {
	w.recordInput(ns.ConfigPath, toml)
	if err := t.Files.Write(ctx, ns.ConfigPath, toml, 0o644); err != nil {
		return ConfigProvenance{}, fmt.Errorf("chainsetup: config: node%d: %w", ns.Index, err)
	}
	want := filestore.Hash(toml)
	got, err := t.Files.Checksum(ctx, ns.ConfigPath)
	if err != nil {
		return ConfigProvenance{}, fmt.Errorf("chainsetup: config: node%d readback: %w", ns.Index, err)
	}
	if got != want {
		return ConfigProvenance{}, fmt.Errorf("chainsetup: config: node%d config did not read back intact (wrote %s, target has %s)", ns.Index, want, got)
	}
	prov := ConfigProvenance{Node: ns.Index, Overrides: overrides, Checksum: want}
	if purpose != "" {
		prov.Fixture = "config-" + purpose
	}
	return prov, nil
}

// LaunchOpts assembles each node's launch argv through nodeconfig.Argv —
// the single argv renderer — and records it in the node table, so `start`
// launches exactly what this step showed. Each node's overrides come from the
// scopes recorded on the workspace (all, then its role, then the node), so one
// argv renderer serves a network of mixed launch flags.
func (w *Workspace) LaunchOpts() (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if err := w.require("build"); err != nil {
		return "", err
	}
	preset, placed, peering, pubkey, err := w.peerPlan(p)
	if err != nil {
		return "", fmt.Errorf("chainsetup: launchopts: %w", err)
	}
	scoped := false
	for i, ns := range w.state.Nodes {
		overrides, err := ParseOverrides(w.launchOverridesFor(ns.Role, ns.Index))
		if err != nil {
			return "", err
		}
		if len(overrides) > 0 {
			scoped = true
		}
		staticNodes, err := node.PeerList(placed, peering, ns.NodeLabel(), pubkey)
		if err != nil {
			return "", fmt.Errorf("chainsetup: launchopts: node%d peers: %w", ns.Index, err)
		}
		args, err := nodeconfig.Argv(process.NodeConfig(p, preset, process.SpecOf(ns), w.keysBase(), staticNodes), overrides...)
		if err != nil {
			return "", fmt.Errorf("chainsetup: launchopts: node%d: %w", ns.Index, err)
		}
		w.state.Nodes[i].Args = args
	}
	detail := fmt.Sprintf("%d argv(s) assembled", len(w.state.Nodes))
	if scoped {
		detail += ", with launch overrides"
	}
	w.markStep("build", detail)
	return detail, nil
}

// Provision materializes the shared launch inputs on the target with
// upload-if-absent semantics: the genesis (as built by the genesis step) and
// the per-node configs. Re-running it reuses what already exists.
func (w *Workspace) Provision(ctx context.Context) (string, error) {
	if err := w.require("deploy"); err != nil {
		return "", err
	}
	present, shipped := 0, 0
	err := w.eachMachine(func(t *resource.Access, nodes []node.Record) error {
		check := func(path string) error {
			exists, err := t.Files.Exists(ctx, path)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("chainsetup: provision: %s missing — run the genesis/config steps first", path)
			}
			// Present is not the same as ours. A genesis someone edited, or a
			// config left by a previous composition, is present and would be
			// launched from.
			if want, known := w.state.LaunchInputs[path]; known {
				have, err := t.Files.Checksum(ctx, path)
				if err != nil {
					return err
				}
				if have != want {
					return fmt.Errorf("chainsetup: provision: %s is not the file this workspace built "+
						"(built %s, found %s) — something else wrote it; re-run the step that makes it "+
						"(`chain genesis` or `chain config`) to put yours back", path, short(want), short(have))
				}
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
	w.markStep("deploy", detail)
	return detail, nil
}

// shipIdentities uploads each node's identity files — the devp2p nodekey, the
// validator keystore, and the shared password — from the local key set to
// keysBase on a remote target, upload-if-absent like the rest of filestore.
// The rendered config and the launch argv point at keysBase, so without this
// a remote node would look for its keys on the operator's resource. A local
// target ships nothing: keysBase is the key set itself.
func (w *Workspace) shipIdentities(ctx context.Context, t *resource.Access, nodes []node.Record) (int, error) {
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
			have, err := t.Files.Checksum(ctx, dst)
			if err != nil {
				return err
			}
			if have == filestore.Hash(b) {
				return nil // identical content already on the target: not re-sent
			}
			// A stale key file under the same name is not the one we mean; ship
			// the current content over it rather than launch with the wrong key.
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
func ParseOverrides(sets []string) ([]nodeconfig.Override, error) {
	out := make([]nodeconfig.Override, 0, len(sets))
	for _, s := range sets {
		k, v, _ := strings.Cut(s, "=")
		if k == "" {
			return nil, fmt.Errorf("chainsetup: bad --set %q (want key=value or a bare boolean key)", s)
		}
		out = append(out, nodeconfig.Override{Key: nodeconfig.Key(k), Value: v})
	}
	return out, nil
}

// nodeHost is the address a composed node is reachable at: the one the
// allocator recorded, falling back to this machine for a plan that predates
// per-node hosts.
func nodeHost(ns node.Record) string {
	if ns.Host != "" {
		return ns.Host
	}
	return localHost
}

// Netmap reads the workspace's node table as a placement map, so the peer
// policy and the address lookups run off one representation. The host is the
// node's own recorded address, which spread across a set is not this resource.
func (w *Workspace) Netmap() (*node.Map, error) {
	placements := make([]node.Placement, 0, len(w.state.Nodes))
	ordinals := map[node.Role]int{}
	for _, ns := range w.state.Nodes {
		role, err := node.NormalizeRole(ns.Role)
		if err != nil {
			return nil, fmt.Errorf("node%d: %w", ns.Index, err)
		}
		ordinals[role]++
		placements = append(placements, node.Placement{
			Index:   ns.Index,
			Label:   ns.NodeLabel(),
			Role:    role,
			Ord:     ordinals[role],
			Host:    nodeHost(ns),
			Ports:   ns.Endpoints,
			DataDir: ns.DataDir,
		})
	}
	return node.NewMap(placements)
}

// netmapRequests turns the composed node list into placement requests. Only the
// role travels: position comes from the order, which is also the node's
// identity.
func netmapRequests(reqs []node.LaunchReq) []resource.Request {
	out := make([]resource.Request, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, resource.Request{Role: r.Role})
	}
	return out
}

// genesisArtifacts builds the genesis through the one composition every surface
// uses. The wemix source runs the chain binary, so the request also carries the
// placement: the governance config names the producer by host and p2p port.
func (w *Workspace) genesisArtifacts(ctx context.Context, p registry.ChainPlugin, opts GenesisOpts) (genesis.Artifacts, error) {
	// The placement always travels with the request: a family whose genesis
	// names the producer's address reads it, and one whose genesis carries the
	// validator set ignores it. Branching here on the family was the second
	// place the wemix path had to be special-cased.
	placed, err := w.Netmap()
	if err != nil {
		return genesis.Artifacts{}, fmt.Errorf("chainsetup: genesis: %w", err)
	}
	req := genesis.Request{Validators: w.state.Validators, Nodes: placed}
	cfg := genesis.Config{
		KeysDir:         w.state.KeysDir,
		Binary:          w.state.Binary,
		ChainID:         opts.ChainID,
		ConfigOverrides: opts.Overrides,
		Overlay:         opts.Overlay,
	}
	// A family whose genesis its own binary writes (wemix) runs that binary. On
	// a remote target the binary lives there, not here, so stage the generator's
	// inputs on the target and run it over the same access init/start use. A
	// family whose genesis is in-process ignores these.
	if w.state.Target.IsRemote() {
		boot, ok := firstProducer(w.state.Nodes)
		if !ok {
			return genesis.Artifacts{}, fmt.Errorf("chainsetup: genesis: no producer to generate the genesis on")
		}
		access, err := w.machineFor(boot)
		if err != nil {
			return genesis.Artifacts{}, fmt.Errorf("chainsetup: genesis: %w", err)
		}
		cmdr, ok := access.Driver.(process.Commander)
		if !ok {
			return genesis.Artifacts{}, fmt.Errorf("chainsetup: genesis: the target cannot run a command, so a binary-written genesis cannot be generated there")
		}
		cfg.Files = access.Files
		cfg.WorkDir = path.Join(access.DataRoot, genesisWorkDir)
		cfg.Runner = commanderRunner(cmdr)
	}
	return genesis.Compose(ctx, p, req, cfg)
}

// genesisWorkDir is where a binary-written genesis stages its config, template,
// and output on the target, under the data root.
const genesisWorkDir = "genesis-work"

// firstProducer returns the first block-producing node in the table — the node
// a binary-written genesis is generated on and the boot phase launches alone.
func firstProducer(nodes []node.Record) (node.Record, bool) {
	for _, ns := range nodes {
		if node.Is(node.Role(ns.Role), node.RoleBP) {
			return ns, true
		}
	}
	return node.Record{}, false
}

// commanderRunner adapts a process.Commander (the local or remote driver's
// arbitrary-command capability) into the runner a binary-written genesis and the
// poa bootstrap take, so both run on the same transport as init and start. The
// binary and its args are shell-quoted into one command line.
func commanderRunner(c process.Commander) genesis.CommandRunner {
	return func(ctx context.Context, name string, args ...string) ([]byte, error) {
		out, err := c.Run(ctx, shellCommand(name, args...))
		return []byte(out), err
	}
}

// shellCommand quotes a binary and its args into one command line for a shell.
func shellCommand(name string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(name))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

// shellQuote single-quotes one argument so a shell takes it literally.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// peerPlan is what every per-node rendering needs beyond the record: the key
// set (identity and public keys), the placement, the validated peering, and a
// public-key lookup by index. Config, launchopts and start all render from
// the same four, so they are gathered once.
func (w *Workspace) peerPlan(p registry.ChainPlugin) (keyring.Preset, *node.Map, node.Peering, func(int) (string, bool), error) {
	preset, err := store.LoadPreset(w.state.KeysDir)
	if err != nil {
		return keyring.Preset{}, nil, "", nil, err
	}
	placed, err := w.Netmap()
	if err != nil {
		return keyring.Preset{}, nil, "", nil, err
	}
	peering, err := node.ParsePeering(w.state.Peering)
	if err != nil {
		return keyring.Preset{}, nil, "", nil, err
	}
	if err := peering.Validate(placed, p.Family().SupportsRole); err != nil {
		return keyring.Preset{}, nil, "", nil, err
	}
	// The peer's own recorded address: spread across a set each node lives on
	// a different host, and a static-node list pointing at this machine would
	// leave every node unable to find its peers. Keys reach the composition
	// as inputs — the node module joins them to placements.
	pubkey := func(index int) (string, bool) {
		nk, ok := preset.Node(index)
		if !ok {
			return "", false
		}
		return nk.PublicKey, true
	}
	return preset, placed, peering, pubkey, nil
}

// recordInput remembers what a launch input hashed to when this workspace wrote
// it, so deploy can tell the file it built from one that merely occupies the
// same path.
func (w *Workspace) recordInput(path string, content []byte) {
	if w.state.LaunchInputs == nil {
		w.state.LaunchInputs = map[string]string{}
	}
	w.state.LaunchInputs[path] = filestore.Hash(content)
}

// short renders a hash the way a reader compares two of them: enough to tell
// them apart, not so much that the message wraps.
func short(hash string) string {
	if i := strings.IndexByte(hash, ':'); i >= 0 && len(hash) > i+13 {
		return hash[:i+13]
	}
	return hash
}

// composeNeeds is the composition's resolution order, declared once.
//
// It was a convention before: each step hand-rolled a check on whatever state
// field it happened to need and wrote its own "run X first" message. Three
// things went wrong with that. The checks disagreed about what a step needs —
// genesis looked at the validator count and never at the key set, though it
// cannot build extraData without one. The messages named different steps for
// the same missing prerequisite. And the order existed nowhere a reader could
// see it, so N9's question ("what has to resolve before what") could only be
// answered by reading six functions.
//
// The order is NOT keyring → netmap, which is how the worklist recorded it.
// `keys` takes its node count from the placement, so `place` runs first and
// `chain up` has always run them that way; the note was written from the
// intended design rather than from the code.
//
// genesis needs place and NOT keys, which is the second thing writing this down
// corrected. It hands the key directory to the family and the family decides:
// one that seals extraData from the validator keys reads it, one that
// substitutes a supplied template never opens it. Requiring a key set here
// broke composing an external chain from its own template, which is a thing
// that worked, so the dependency belongs to the family and not to the step.
//
// Each entry lists what a step reaches for DIRECTLY. deploy needs a key set as
// much as config does, and gets it by way of config rather than by claiming it
// here, so a change in what config needs does not have to be copied.
//
// enode is absent because it produces nothing and marks no step: it derives a
// view from place and keys, and verbs_enode.go asks for both directly.
var composeNeeds = map[string][]string{
	"place":   {"new"},
	"keys":    {"new"},
	"genesis": {"place"},
	"config":  {"place", "keys"},
	"build":   {"place", "keys"},
	"deploy":  {"place", "genesis", "config"},
}

// require reports whether every step that has to resolve before step has run,
// naming the first one that has not.
//
// It reads the recorded steps rather than the state fields they leave behind.
// A field can be non-empty because something else filled it, and a step that
// half-ran leaves exactly that: state that looks composed and was not.
func (w *Workspace) require(step string) error {
	for _, need := range composeNeeds[step] {
		if _, done := w.state.Steps[need]; !done {
			return fmt.Errorf("chainsetup: %s: %s has not run — run `chain %s` first", step, need, need)
		}
	}
	return nil
}
