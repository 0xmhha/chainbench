package chainsetup

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/blueprint"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The place step: which node runs where, on which ports.
//
// The allocator is deterministic — the same request places the same way — and
// the port bands come from the server set rather than from here, so a site
// whose firewall groups ports by purpose is expressible without changing code.

type AllocateOpts struct {
	// BPCount is the bp (block-producing) node count (>=1).
	BPCount int
	// ENCount is the en (endpoint, non-producing) node count.
	ENCount int
	// PNCount is the pn (proxy-tier) node count. A family with no proxy tier
	// refuses a pn at peering validation.
	PNCount int
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
	// BinaryChains names, per binary, the chain that binary runs when it is not
	// the composition's. A name it does not hold runs the composition's chain,
	// which is every network of one build.
	BinaryChains map[string]string
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
			// Before the string is copied anywhere, not after.
			//
			// The refusal used to live in the keys step, which runs after place
			// has put this exact string into the node record and after
			// withWorkspace has saved it: the run stopped, and the key it
			// stopped for was already in chain-record.json — and in the --json
			// report, since a setup error is carried in it verbatim. Rejecting
			// a value the moment it is read is the only order in which "never
			// stored" is true.
			if err := checkNodeKeyRef(n.Index, n.Key); err != nil {
				return nil, nil, err
			}
			reqs[i] = node.LaunchReq{Role: role, Binary: n.Binary, Config: n.Config, Key: n.Key}
			// A topology's per-node mode wins; a validator is still pinned to
			// full, since the topology cannot make a sealing node stateless.
			modes[i] = syncModeFor(role, n.EffectiveSyncMode())
		}
		return reqs, modes, nil
	}
	validators := o.BPCount
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
	reqs := make([]node.LaunchReq, 0, validators+o.PNCount+o.ENCount)
	modes := make([]string, 0, cap(reqs))
	for i := 0; i < validators; i++ {
		reqs = append(reqs, node.LaunchReq{Role: node.RoleBP})
		modes = append(modes, syncModeFull)
	}
	// Both paths order bp, en, pn, so the network ends on the pn and that node
	// lands on the last server: the pn is the discovery hub every other node
	// dials, and the unified model puts it at the highest index. On wemix the
	// etcd seed stays the highest-index bp, which comes before it either way.
	//
	// The named-count path used to order bp, pn, en instead, on the ground that
	// existing specs address nodes by index. Measured 2026-09-19: of the seven
	// cases that compose a pn, none names a node as "nodeN" — they address the
	// tier by its role label (pn1, en1, bp1), which is stable under either
	// order. Two orders for one question is a fact in two places, so there is
	// one now.
	appendProxies := func() {
		for i := 0; i < o.PNCount; i++ {
			reqs = append(reqs, node.LaunchReq{Role: node.RolePN})
			modes = append(modes, syncModeFor(node.RolePN, o.EndpointSyncMode))
		}
	}
	appendEndpoints := func() {
		for i := 0; i < o.ENCount; i++ {
			reqs = append(reqs, node.LaunchReq{Role: node.RoleEN})
			modes = append(modes, syncModeFor(node.RoleEN, o.EndpointSyncMode))
		}
	}
	appendEndpoints()
	appendProxies()
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
	validators := servers - o.PNCount - o.ENCount
	if validators < 1 {
		return 0, fmt.Errorf("chainsetup: allocate: %d server(s) cannot hold %d pn + %d en and still leave a validator", servers, o.PNCount, o.ENCount)
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
	// A workspace-config composition isolates its node datadirs, generated
	// genesis/configs, and logs under its composition id, so two compositions
	// sharing one data root do not collide. Without one the layout stays flat.
	layout, err := w.layout()
	if err != nil {
		return "", err
	}
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
	if len(opts.BinaryChains) > 0 {
		w.state.BinaryChains = opts.BinaryChains
	}
	// Counted from the resolved placements, not the requested count: a topology
	// decides the validator set, and the genesis step sizes itself from this.
	w.state.BPCount = validators
	if opts.Topology != nil {
		w.state.Bootnode = opts.Topology.BootnodeIndex()
	}

	w.state.PortSource = pool.Source

	// Counted by role rather than as "producers and the rest": the rest is two
	// different jobs, and a pn reported as an endpoint is how a proxy tier goes
	// unnoticed in the one line that says what was placed.
	var ens, pns int
	for _, n := range nodes {
		switch {
		case node.Is(node.Role(n.Role), node.RoleEN):
			ens++
		case node.Is(node.Role(n.Role), node.RolePN):
			pns++
		}
	}
	detail := fmt.Sprintf("%d node(s): %d bp + %d en + %d pn; ports: %s; p2p from %d, http from %d",
		len(nodes), validators, ens, pns, pool.Source, nodes[0].P2P, nodes[0].HTTP)
	if opts.Topology != nil {
		detail += " (topology)"
	}
	w.markStep("place", detail)
	return detail, nil
}

// GenesisOpts customizes the built genesis.
