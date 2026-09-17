package upgrade

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Where a handoff's nodes run.
//
// HandoffInputs has taken Host, Files, Driver, Exec, Placement and Machine
// since the family rework, so a handoff has never been local by construction —
// it was local because nothing resolved those. One surface then resolved them
// (`chainbench upgrade`) and the other did not (a handoff declared in the DSL),
// which is not a difference between two kinds of handoff. It is a difference
// between two callers, and it lived in the caller's package where only one of
// them could reach it. It lives here now, beside the inputs it fills.

// handoffPortBand is the span one server's port purposes occupy, matching what
// the composition path reserves.
const handoffPortBand = 100

// Target says where a handoff's data plane should live: on this machine when it
// names no server, or on the server set's machines when it does.
type Target struct {
	// Server names the machine, from the server set. Empty with All false runs
	// the handoff on this machine.
	Server resource.ServerRef
	// AllServers spreads the nodes across every server in the set.
	AllServers bool
	// WorkspaceConfigPath owns the target's data root. Required whenever a
	// server is named: a remote machine cannot resolve without knowing where
	// its data lives.
	WorkspaceConfigPath string
	// Docker translates this tool's dials through the localmap beside the
	// server set, for containers standing in for servers.
	Docker bool
	// Env reads the process environment; nil is the real one.
	Env func(string) string
	// Report narrates the placement; nil is silent.
	Report func(string, ...any)
}

// Local reports whether this target is this machine, which needs no resolution.
func (t Target) Local() bool { return t.Server.Name == "" && !t.AllServers }

// Apply fills the machine-facing half of in — placement, ports, per-node file
// stores and drivers, the data root, and the bootstrap runner — for a handoff
// of total nodes. A local target leaves in untouched.
//
// The workspace-config is returned because it is also the answer to "where does
// a bare binary name live over there": dataRoot plus paths.binaries, with
// binaryAliases applied. A caller that drops it has no answer to that.
func (t Target) Apply(in *HandoffInputs, total int) (*resource.WorkspaceConfig, error) {
	var wc *resource.WorkspaceConfig
	if t.WorkspaceConfigPath != "" {
		c, err := resource.LoadWorkspaceConfig(t.WorkspaceConfigPath)
		if err != nil {
			return nil, err
		}
		wc = &c
	}
	if t.Local() {
		// An environment file says where files are, and that is as true of this
		// machine as of any other: it owns the data root either way. A local
		// handoff given one used to ignore it and compose under the workspace
		// directory instead.
		if wc != nil {
			in.DataDir = wc.DataRoot
		}
		return wc, nil
	}
	// Named a server, so the data root has to come from somewhere, and the
	// environment file is the only thing that says where it is over there.
	if wc == nil {
		return nil, fmt.Errorf("upgrade: a workspace-config is required to run a handoff on a server (it owns the target data root)")
	}
	// Placement comes before the target: with --all-servers the set names every
	// machine, so the base target is not something the operator repeats but the
	// machine the first node landed on.
	placed, err := t.place(total)
	if err != nil {
		return nil, err
	}
	// The opener is the one place an address becomes reachable: it applies the
	// localmap under docker and leaves a real server's address alone.
	opener := resource.Opener{ServerSet: t.Server.SetPath, Docker: t.Docker, Env: t.Env, Report: t.Report}
	machines, err := t.machines(wc, placed, opener)
	if err != nil {
		return nil, err
	}
	server := t.Server.Name
	if server == "" {
		if server, err = firstPlacedServer(t.Server.SetPath, placed); err != nil {
			return nil, err
		}
	}
	acc, err := opener.Open(resource.Spec{Server: server, DataRoot: wc.DataRoot})
	if err != nil {
		return nil, err
	}
	ex, err := handoffExec(acc)
	if err != nil {
		return nil, err
	}
	in.Placement = placed
	in.MultiMachine = spansHosts(placed)
	in.DialURL = opener.HTTPEndpoint
	in.Machine = machines
	in.Host = acc.Spec.Host
	in.DataDir = acc.DataRoot
	in.Files = acc.Files
	in.Driver = acc.Driver
	in.Exec = ex
	return wc, nil
}

// place draws the handoff's nodes from the server set: which machine each one
// runs on and which port band it gets.
//
// Every node is requested as a producer role because a handoff's roles are
// producer/successor rather than bp/en, and the allocator only needs to know how
// many slots to hand out. The order it returns is the order the plan reads: the
// profile's producers come first, so nodes 1..P are the from-chain miners and
// the rest are the successors — which on a 15-server set puts one of each on
// every server, the second at the next port band.
func (t Target) place(total int) (*node.Map, error) {
	ref := t.Server
	ref.All = t.AllServers
	resolved, err := resource.ResolveServer(ref, 1, handoffPortBand)
	if err != nil {
		return nil, fmt.Errorf("upgrade: placing %d node(s): %w", total, err)
	}
	inv, err := resource.NewInventory(resolved.Pool)
	if err != nil {
		return nil, fmt.Errorf("upgrade: %w", err)
	}
	reqs := make([]resource.Request, total)
	for i := range reqs {
		reqs[i] = resource.Request{Role: node.RoleBP}
	}
	placed, err := inv.Assign(reqs, "")
	if err != nil {
		return nil, fmt.Errorf("upgrade: placing %d node(s) on the server set: %w", total, err)
	}
	if t.Report != nil {
		for _, pl := range placed.Placements() {
			t.Report("place: %s on %s p2p=%d http=%d", pl.Label, pl.Host, pl.Ports.P2P, pl.Ports.HTTP)
		}
	}
	return placed, nil
}

// machines resolves a file store and a driver for each placed node, by the
// server its address belongs to.
//
// The placement records an address, and a server set maps an address back to the
// entry that owns its credentials — so the lookup goes address -> server name ->
// Access, and each Access is opened once and shared by the nodes on that machine.
func (t Target) machines(wc *resource.WorkspaceConfig, placed *node.Map, opener resource.Opener) (
	func(int) (filestore.Store, process.Driver, error), error,
) {
	byIndex, err := serverNamesByIndex(t.Server.SetPath, placed)
	if err != nil {
		return nil, err
	}
	cache := map[string]*resource.Access{}
	return func(i int) (filestore.Store, process.Driver, error) {
		name, ok := byIndex[i]
		if !ok {
			return nil, nil, fmt.Errorf("upgrade: node%d has no placement", i+1)
		}
		if acc, hit := cache[name]; hit {
			return acc.Files, acc.Driver, nil
		}
		acc, err := opener.Open(resource.Spec{Server: name, DataRoot: wc.DataRoot})
		if err != nil {
			return nil, nil, fmt.Errorf("upgrade: node%d on %s: %w", i+1, name, err)
		}
		cache[name] = acc
		return acc.Files, acc.Driver, nil
	}, nil
}

// handoffExec is the bootstrap runner for a remote target: the poa helpers take
// a binary and arguments, and a remote target takes one command line, so the
// target's own Commander does the quoting through process.ShellRunner — the same
// adapter the composition path uses for the poa phase actions.
func handoffExec(acc *resource.Access) (poa.Runner, error) {
	cmdr, ok := acc.Driver.(process.Commander)
	if !ok {
		return nil, fmt.Errorf("upgrade: target %q cannot run commands, so governance and etcd cannot be bootstrapped on it", acc.Spec.Server)
	}
	return poa.Runner(process.ShellRunner(cmdr)), nil
}

// spansHosts reports whether a placement put nodes on more than one machine,
// which is what needs a file store and a driver per node.
func spansHosts(m *node.Map) bool {
	seen := map[string]bool{}
	for _, pl := range m.Placements() {
		seen[pl.Host] = true
	}
	return len(seen) > 1
}

// firstPlacedServer names the server the handoff's first node landed on. With
// every server in the set there is no named one to read the base target from,
// and every machine is equally a target, so the plan's own first node decides:
// its machine is the one whose workspace-config owns the data root and whose
// PATH a bare binary name is looked up on.
func firstPlacedServer(setPath string, placed *node.Map) (string, error) {
	names, err := serverNamesByIndex(setPath, placed)
	if err != nil {
		return "", err
	}
	name, ok := names[0]
	if !ok {
		return "", fmt.Errorf("upgrade: the placement has no node1, so there is no server to take the data root from")
	}
	return name, nil
}

// serverNamesByIndex maps each placed node's zero-based index to the name of the
// server set entry that owns its address. The placement records an address; the
// credentials, and therefore the Access, belong to the named entry.
func serverNamesByIndex(setPath string, placed *node.Map) (map[int]string, error) {
	if setPath == "" {
		setPath = resource.DefaultSetFile
	}
	set, err := resource.LoadSet(setPath)
	if err != nil {
		return nil, fmt.Errorf("upgrade: server set: %w", err)
	}
	nameOf := map[string]string{}
	for _, h := range set.PoolSpec.Hosts {
		nameOf[h.Addr] = h.Name
	}
	byIndex := map[int]string{}
	for _, pl := range placed.Placements() {
		name, ok := nameOf[pl.Host]
		if !ok {
			return nil, fmt.Errorf("upgrade: %s was placed at %s, which the server set does not name", pl.Label, pl.Host)
		}
		byIndex[pl.Index-1] = name
	}
	return byIndex, nil
}
