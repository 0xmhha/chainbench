package app

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// defaultPortBand is how many port slots a host is assumed to offer when no
// server set says otherwise. It matches what the composition uses.
const defaultPortBand = 100

// NetMapIn asks the composed network where a node is. The selectors are
// alternatives: give one, or none for the whole map.
type NetMapIn struct {
	DataDir string
	// Node selects by identity (the 1-based node number).
	Node int
	// Label selects by identity ("node7") or role alias ("en2").
	Label string
	// Host selects every node on an address.
	Host string
	// Port selects whichever node listens on a port — the reverse question an
	// address in a log line or an error message asks.
	Port int
	// Addr is the same question in the form a log line actually prints it,
	// "host:port", so it can be pasted rather than split by hand.
	Addr string
}

// MapEntry is one node's placement as an operator reads it. Both names are
// present because they answer different questions: the identity is what
// reaches disk, the alias is what a test definition addresses.
type MapEntry struct {
	Node  int    `json:"node"`
	Label string `json:"label"`
	Alias string `json:"alias"`
	Role  string `json:"role"`
	Host  string `json:"host"`
	// Endpoints is embedded rather than listed field by field. Writing the
	// ports out is how this projection came to omit the etcd client port the
	// moment a family started reserving one — the fifth time in this series
	// that a hand-written copy of a port set lost a member of it.
	node.Endpoints
	DataDir string `json:"dataDir,omitempty"`
}

// NetMapOut is the answer, plus the counts an operator scanning a network wants
// without having to tally the rows.
type NetMapOut struct {
	Entries []MapEntry     `json:"entries"`
	Roles   map[string]int `json:"roles"`
	Total   int            `json:"total"`
}

// NetMap answers where nodes are, in either direction: label to address, or
// address to label. It reads the composed workspace and never dials — a node
// that is not running still has a place in the map.
func NetMap(_ context.Context, d Deps, in NetMapIn) (NetMapOut, error) {
	ws, err := chainsetup.Open(in.DataDir, d.Clock)
	if err != nil {
		return NetMapOut{}, err
	}
	m, err := ws.Netmap()
	if err != nil {
		return NetMapOut{}, err
	}
	all := m.Placements()
	if len(all) == 0 {
		return NetMapOut{}, fmt.Errorf("app: netmap show: no node table — run `net allocate` first")
	}

	roles := map[string]int{}
	for _, p := range all {
		roles[string(p.Role)]++
	}
	out := NetMapOut{Roles: roles, Total: len(all)}

	match, err := mapFilter(m, in)
	if err != nil {
		return NetMapOut{}, err
	}
	for _, p := range all {
		if !match(p) {
			continue
		}
		out.Entries = append(out.Entries, MapEntry{
			Node: p.Index, Label: string(p.Label), Alias: string(p.RoleLabel()),
			Role: string(p.Role), Host: p.Host,
			Endpoints: p.Ports, DataDir: p.DataDir,
		})
	}
	if len(out.Entries) == 0 {
		return NetMapOut{}, fmt.Errorf("app: netmap show: nothing matches (the network has %d node(s))", len(all))
	}
	return out, nil
}

// mapFilter turns the selectors into one predicate, rejecting a combination
// that asks two questions at once rather than silently honouring one.
func mapFilter(m *node.Map, in NetMapIn) (func(node.Placement) bool, error) {
	given := 0
	for _, on := range []bool{in.Node > 0, in.Label != "", in.Host != "", in.Port > 0, in.Addr != ""} {
		if on {
			given++
		}
	}
	if given > 1 {
		return nil, fmt.Errorf("app: netmap show: give at most one of --node, --label, --host, --port, --addr")
	}
	if in.Addr != "" {
		host, portStr, err := net.SplitHostPort(in.Addr)
		if err != nil {
			return nil, fmt.Errorf("app: netmap show: %q is not host:port: %w", in.Addr, err)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("app: netmap show: %q has no port number", in.Addr)
		}
		if label, ok := m.At(host, port); ok {
			return func(p node.Placement) bool { return p.Label == label }, nil
		}
		return func(node.Placement) bool { return false }, nil
	}
	switch {
	case in.Node > 0:
		return func(p node.Placement) bool { return p.Index == in.Node }, nil
	case in.Label != "":
		want := node.Label(in.Label)
		if _, ok := m.Lookup(want); ok {
			return func(p node.Placement) bool { return p.Label == want }, nil
		}
		// Not an identity, so read it as a role alias ("en2").
		role, ord, err := node.ParseRoleLabel(want)
		if err != nil {
			return nil, fmt.Errorf("app: netmap show: %q is neither a node in this network nor a role label: %w", in.Label, err)
		}
		return func(p node.Placement) bool { return node.Is(p.Role, role) && p.Ord == ord }, nil
	case in.Host != "":
		return func(p node.Placement) bool { return p.Host == in.Host }, nil
	case in.Port > 0:
		return func(p node.Placement) bool {
			for _, port := range []int{p.Ports.P2P, p.Ports.Etcd, p.Ports.EtcdClient, p.Ports.HTTP, p.Ports.WS, p.Ports.Auth, p.Ports.Metrics} {
				if port != 0 && port == in.Port {
					return true
				}
			}
			return false
		}, nil
	default:
		return func(node.Placement) bool { return true }, nil
	}
}

// NetPoolIn asks what the network may be allocated from. A workspace is
// optional: without one the answer is the server set (or the built-ins) alone.
type NetPoolIn struct {
	DataDir string
	Server  ServerRef
}

// NetPoolOut is the resource, how much of it is spoken for, and by whom.
//
// It deliberately carries no credentials. The pool says where nodes may run;
// how to log in belongs to the server set and the environment, and a summary an
// agent can read should not be the place a password leaks.
type NetPoolOut struct {
	Source string   `json:"source"`
	Hosts  []string `json:"hosts"`
	Slots  int      `json:"slots"`
	Cap    int      `json:"cap"`
	Used   int      `json:"used"`
	Free   int      `json:"free"`
	// ByNetwork counts the slots each workspace holds, so a full set says
	// what to remove rather than only that it is full.
	ByNetwork map[string]int `json:"byNetwork,omitempty"`
}

// NetPool reports the addresses and port slots a network may be composed from,
// and how many are already taken. It is what answers "why was 15 refused".
//
// Taken is what the resource module's inventory says after adopting every
// workspace it can see: the one named (optional) and every composition under
// the default root. A workspace composed somewhere else is counted only when
// it is named — there is no registry of workspaces, only the directories.
func NetPool(_ context.Context, d Deps, in NetPoolIn) (NetPoolOut, error) {
	resolved, err := ResolveServer(d, in.Server, 1, defaultPortBand)
	if err != nil {
		return NetPoolOut{}, err
	}
	pool := resolved.Pool
	if pool.Slots < 1 {
		pool.Slots = 1
	}
	inv, err := resource.NewInventory(pool)
	if err != nil {
		return NetPoolOut{}, err
	}
	for _, dir := range poolWorkspaces(in.DataDir) {
		if ws, err := chainsetup.Open(dir, d.Clock); err == nil {
			inv.Adopt(chainsetup.Allocations(ws))
		}
	}
	u := inv.Usage()
	out := NetPoolOut{Source: pool.Source, Slots: pool.Slots, Cap: u.Cap, Used: u.Used, Free: u.Free, ByNetwork: u.ByNetwork}
	for _, h := range pool.Hosts {
		name := h.Name
		if name == "" || name == h.Addr {
			out.Hosts = append(out.Hosts, h.Addr)
			continue
		}
		out.Hosts = append(out.Hosts, fmt.Sprintf("%s (%s)", name, h.Addr))
	}
	return out, nil
}

// poolWorkspaces is every workspace whose claims count against the pool: the
// named one first, then the default root's. A missing default root is simply
// nothing more to count.
func poolWorkspaces(named string) []string {
	var dirs []string
	if named != "" {
		dirs = append(dirs, named)
	}
	root, err := chainsetup.DefaultRoot()
	if err != nil {
		return dirs
	}
	found, err := chainsetup.Discover(root)
	if err != nil {
		return dirs
	}
	for _, f := range found {
		if f != named {
			dirs = append(dirs, f)
		}
	}
	return dirs
}

// NetPlanIn asks what placement a network of this shape would get, before any
// workspace exists. The chain matters because a family reserves a different
// number of ports per node; the default is stablenet.
type NetPlanIn struct {
	Chain      string
	Validators int
	Endpoints  int
	Server     ServerRef
}

// NetPlan runs the allocator as a question: the same deterministic assignment
// a composition would record, computed from the server set (or the built-in
// pool) and the requested shape, with nothing written anywhere. It is how a
// placement change is inspected — and tested — without composing a network.
func NetPlan(_ context.Context, d Deps, in NetPlanIn) (NetMapOut, error) {
	if in.Validators < 1 {
		return NetMapOut{}, fmt.Errorf("app: netmap plan: a network needs at least one validator — nothing seals without one")
	}
	chain := in.Chain
	if chain == "" {
		chain = "stablenet"
	}
	plugin, err := registry.Get(chain)
	if err != nil {
		return NetMapOut{}, err
	}
	resolved, err := ResolveServer(d, in.Server, in.Validators, defaultPortBand)
	if err != nil {
		return NetMapOut{}, err
	}
	pool := resolved.Pool
	if pool.Slots < 1 {
		pool.Slots = 1
	}
	pool.Reservation = plugin.Family().PortReservation()
	reqs := make([]resource.Request, 0, in.Validators+in.Endpoints)
	for range in.Validators {
		reqs = append(reqs, resource.Request{Role: node.RoleBP})
	}
	for range in.Endpoints {
		reqs = append(reqs, resource.Request{Role: node.RoleEN})
	}
	m, err := resource.Assign(pool, reqs)
	if err != nil {
		return NetMapOut{}, err
	}
	all := m.Placements()
	out := NetMapOut{Roles: map[string]int{}, Total: len(all)}
	for _, p := range all {
		out.Roles[string(p.Role)]++
		out.Entries = append(out.Entries, MapEntry{
			Node: p.Index, Label: string(p.Label), Alias: string(p.RoleLabel()),
			Role: string(p.Role), Host: p.Host, Endpoints: p.Ports,
		})
	}
	return out, nil
}
