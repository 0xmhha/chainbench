package netmap

import (
	"fmt"
	"net"
	"strconv"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/portplan"
)

// Ports is one node's full port set. It is an alias for portplan.Ports — the
// one representation that never lost the etcd port — until the type moves here
// outright (design NM-b: node.Node has fan-in 25, so the swap is staged).
type Ports = portplan.Ports

// Placement is where and how one node runs.
type Placement struct {
	// Index is the node's 1-based number within the network. It is the
	// identity: the datadir, the log file, and the keyring entry are all named
	// from it, so it survives a change of role while an alias does not.
	Index int
	// Label names the node by its identity ("node7"). It is LabelFor(Index).
	Label NodeLabel
	// Role is the node's canonical role (bp / en / pn).
	Role node.Role
	// Ord is the node's 1-based ordinal within its role, so RoleLabel can spell
	// the alias a test definition addresses it by ("en2").
	Ord int
	// Host is the address the node binds and is dialled on.
	Host string
	// Ports is the node's full port set, etcd included.
	Ports Ports
	// DataDir is the node's data directory on Host.
	DataDir string
}

// Map is the resolved placement of a whole network: label → placement, plus
// the reverse lookup that turns an address from a log line back into a node.
type Map struct {
	byLabel map[NodeLabel]Placement
	byAddr  map[string]NodeLabel
	order   []NodeLabel
}

// NewMap builds a Map from placements, rejecting duplicate labels and
// colliding (host, port) pairs — a collision here is two nodes that cannot
// both start, and the composition should say so before anything launches.
func NewMap(placements []Placement) (*Map, error) {
	m := &Map{
		byLabel: make(map[NodeLabel]Placement, len(placements)),
		byAddr:  make(map[string]NodeLabel, len(placements)),
	}
	for _, p := range placements {
		if p.Label == "" {
			return nil, fmt.Errorf("netmap: a placement has no label")
		}
		if _, dup := m.byLabel[p.Label]; dup {
			return nil, fmt.Errorf("netmap: duplicate label %q", p.Label)
		}
		m.byLabel[p.Label] = p
		m.order = append(m.order, p.Label)
		for _, port := range portsOf(p.Ports) {
			if port == 0 {
				continue
			}
			addr := net.JoinHostPort(p.Host, strconv.Itoa(port))
			if holder, taken := m.byAddr[addr]; taken {
				return nil, fmt.Errorf("netmap: %s is assigned to both %q and %q", addr, holder, p.Label)
			}
			m.byAddr[addr] = p.Label
		}
	}
	return m, nil
}

// Lookup returns a node's placement.
func (m *Map) Lookup(label NodeLabel) (Placement, bool) {
	p, ok := m.byLabel[label]
	return p, ok
}

// At answers the reverse question: which node owns host:port? It is how an
// address in a log line or an error message is traced back to a node.
func (m *Map) At(host string, port int) (NodeLabel, bool) {
	l, ok := m.byAddr[net.JoinHostPort(host, strconv.Itoa(port))]
	return l, ok
}

// Labels returns the placements' labels in insertion order, so output derived
// from a Map is stable between runs.
func (m *Map) Labels() []NodeLabel {
	return append([]NodeLabel(nil), m.order...)
}

// Placements returns every placement in label order, so a caller rendering the
// map or feeding it to the launch flow sees nodes in their identity order.
func (m *Map) Placements() []Placement {
	out := make([]Placement, 0, len(m.order))
	for _, l := range m.order {
		out = append(out, m.byLabel[l])
	}
	return out
}

// portsOf flattens a port set for collision checking. A port the set does not
// use stays zero and is skipped by the caller.
func portsOf(p Ports) []int {
	return []int{p.P2P, p.Etcd, p.EtcdClient, p.HTTP, p.WS, p.Auth, p.Metrics}
}

// RoleLabel is the placement's role-scoped alias ("en2"). See netmap.RoleLabel
// for why a node carries both an identity and an alias.
func (p Placement) RoleLabel() NodeLabel { return RoleLabel(p.Role, p.Ord) }
