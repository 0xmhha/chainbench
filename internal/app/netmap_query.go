package app

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/netcompose"
)

// defaultPortBand is how many port slots a host is assumed to offer when no
// inventory says otherwise. It matches what the composition uses.
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
}

// MapEntry is one node's placement as an operator reads it. Both names are
// present because they answer different questions: the identity is what
// reaches disk, the alias is what a test definition addresses.
type MapEntry struct {
	Node    int    `json:"node"`
	Label   string `json:"label"`
	Alias   string `json:"alias"`
	Role    string `json:"role"`
	Host    string `json:"host"`
	P2P     int    `json:"p2p"`
	Etcd    int    `json:"etcd,omitempty"`
	HTTP    int    `json:"http"`
	WS      int    `json:"ws,omitempty"`
	Auth    int    `json:"auth,omitempty"`
	Metrics int    `json:"metrics,omitempty"`
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
	ws, err := netcompose.Open(in.DataDir, d.Clock)
	if err != nil {
		return NetMapOut{}, err
	}
	m, err := ws.Netmap()
	if err != nil {
		return NetMapOut{}, err
	}
	all := m.Placements()
	if len(all) == 0 {
		return NetMapOut{}, fmt.Errorf("app: net map: no node table — run `net allocate` first")
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
			P2P: p.Ports.P2P, Etcd: p.Ports.Etcd, HTTP: p.Ports.HTTP,
			WS: p.Ports.WS, Auth: p.Ports.Auth, Metrics: p.Ports.Metrics,
			DataDir: p.DataDir,
		})
	}
	if len(out.Entries) == 0 {
		return NetMapOut{}, fmt.Errorf("app: net map: nothing matches (the network has %d node(s))", len(all))
	}
	return out, nil
}

// mapFilter turns the selectors into one predicate, rejecting a combination
// that asks two questions at once rather than silently honouring one.
func mapFilter(m *netmap.Map, in NetMapIn) (func(netmap.Placement) bool, error) {
	given := 0
	for _, on := range []bool{in.Node > 0, in.Label != "", in.Host != "", in.Port > 0} {
		if on {
			given++
		}
	}
	if given > 1 {
		return nil, fmt.Errorf("app: net map: give at most one of --node, --label, --host, --port")
	}
	switch {
	case in.Node > 0:
		return func(p netmap.Placement) bool { return p.Index == in.Node }, nil
	case in.Label != "":
		want := netmap.NodeLabel(in.Label)
		if _, ok := m.Lookup(want); ok {
			return func(p netmap.Placement) bool { return p.Label == want }, nil
		}
		// Not an identity, so read it as a role alias ("en2").
		role, ord, err := netmap.ParseRoleLabel(want)
		if err != nil {
			return nil, fmt.Errorf("app: net map: %q is neither a node in this network nor a role label: %w", in.Label, err)
		}
		return func(p netmap.Placement) bool { return netmap.Is(p.Role, role) && p.Ord == ord }, nil
	case in.Host != "":
		return func(p netmap.Placement) bool { return p.Host == in.Host }, nil
	case in.Port > 0:
		return func(p netmap.Placement) bool {
			for _, port := range []int{p.Ports.P2P, p.Ports.Etcd, p.Ports.HTTP, p.Ports.WS, p.Ports.Auth, p.Ports.Metrics} {
				if port != 0 && port == in.Port {
					return true
				}
			}
			return false
		}, nil
	default:
		return func(netmap.Placement) bool { return true }, nil
	}
}

// NetPoolIn asks what the network may be allocated from. A workspace is
// optional: without one the answer is the inventory (or the built-ins) alone.
type NetPoolIn struct {
	DataDir string
	Server  ServerRef
}

// NetPoolOut is the resource and how much of it is spoken for.
//
// It deliberately carries no credentials. The pool says where nodes may run;
// how to log in belongs to the inventory and the environment, and a summary an
// agent can read should not be the place a password leaks.
type NetPoolOut struct {
	Source string   `json:"source"`
	Hosts  []string `json:"hosts"`
	Slots  int      `json:"slots"`
	Cap    int      `json:"cap"`
	Used   int      `json:"used"`
}

// NetPool reports the addresses and port slots a network may be composed from,
// and how many are already taken. It is what answers "why was 15 refused".
func NetPool(_ context.Context, d Deps, in NetPoolIn) (NetPoolOut, error) {
	placement, err := ResolveServer(d, in.Server, 1, defaultPortBand)
	if err != nil {
		return NetPoolOut{}, err
	}
	pool := placement.Placement.Pool
	if pool.Slots < 1 {
		pool.Slots = 1
	}
	out := NetPoolOut{Source: pool.Source, Slots: pool.Slots, Cap: pool.Cap()}
	for _, h := range pool.Hosts {
		name := h.Name
		if name == "" || name == h.Addr {
			out.Hosts = append(out.Hosts, h.Addr)
			continue
		}
		out.Hosts = append(out.Hosts, fmt.Sprintf("%s (%s)", name, h.Addr))
	}
	// A workspace is optional: reporting the resource is useful before one
	// exists, which is exactly when an operator is sizing a network.
	if in.DataDir != "" {
		if ws, err := netcompose.Open(in.DataDir, d.Clock); err == nil {
			out.Used = len(ws.State().Nodes)
		}
	}
	return out, nil
}
