package netmap

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/portplan"
)

// Bands are the two disjoint port bands a network draws from. A node takes one
// step of each: p2p (with etcd derived at p2p+1) and rpc (http, ws, auth, and
// metrics when the step leaves room). The separation is not cosmetic — packing
// p2p ports one apart makes the wemix binary's derived etcd port land on the
// next node's p2p port, which stalls block production with no obvious cause
// (portplan).
type Bands struct {
	P2PBase int
	P2PStep int
	RPCBase int
	RPCStep int
}

// Host is one address the pool can place nodes on. The name is how an operator
// and a path reference it (srv://<name>/...); an entry that gives only an
// address is named by it.
type Host struct {
	Name string
	Addr string
}

// Pool is the resource a network is allocated from: the addresses it may use,
// and how many port slots each address can hold.
//
// It is a grid, not a list of servers. One host with four slots is a laptop
// running four nodes on stepped ports; four hosts with one slot each is a
// fleet running one node per machine on identical ports. Both were separate
// allocation modes before, and both are this grid read differently.
type Pool struct {
	// Hosts are the addresses in the order they are consumed.
	Hosts []Host
	// Slots is how many port slots one host may hold (>= 1).
	Slots int
	// Ports are the bands each slot steps through.
	Ports Bands
	// Reservation is how many consecutive ports one node needs, which the
	// chain family decides — a wemix node's embedded etcd takes two more than
	// a wbft one. The zero value takes portplan's default, so a caller that
	// has not asked a family still gets a usable plan.
	Reservation portplan.Reservation
	// Source names where the pool was read from, so a port number is never a
	// guess ("remote-server-config.yaml", "built-in defaults").
	Source string
}

// Cap is how many nodes the pool can place: every host, every slot.
func (p Pool) Cap() int { return len(p.Hosts) * p.Slots }

// Validate rejects a pool that cannot produce a usable placement. The port
// rules are portplan's, checked here so the operator hears about the inventory
// rather than about a node that failed to bind.
func (p Pool) Validate() error {
	if len(p.Hosts) == 0 {
		return fmt.Errorf("netmap: pool has no hosts")
	}
	for i, h := range p.Hosts {
		if h.Addr == "" {
			return fmt.Errorf("netmap: pool host %d (%q) has no address", i+1, h.Name)
		}
	}
	if p.Slots < 1 {
		return fmt.Errorf("netmap: pool must allow at least one slot per host, got %d", p.Slots)
	}
	// The last slot is the one that can run off the end of a band, so checking
	// it checks every earlier one.
	if _, err := portplan.Plan(p.Slots, p.Ports.P2PBase, p.Ports.P2PStep, p.Ports.RPCBase, p.Ports.RPCStep, p.Reservation); err != nil {
		return fmt.Errorf("netmap: pool ports: %w", err)
	}
	return nil
}

// Request is one node to place.
//
// The role is required; the label is not. Position in the request list is the
// node's identity, and LabelFor spells it — but an operator who names a node
// should have that name kept, because the name is how they will refer to it in
// a log, a path, and a test definition. The previous placement type carried a
// name too, invented in four different spellings by different callers and then
// read by none.
type Request struct {
	Role node.Role
	// Label overrides the conventional identity label for this node. Empty
	// takes LabelFor(position).
	Label NodeLabel
}

// Assign allocates the pool to the requests, deterministically: node i takes
// host i mod len(hosts), and the slot it wraps onto. Hosts are consumed before
// slots, so a five-host pool places node1..node5 on distinct machines before
// node6 returns to the first host on the next port slot.
//
// Determinism is the point. The same pool and the same request list always
// produce the same map, so re-running a composition does not silently move a
// node to a different address, and a recorded placement can be reproduced.
//
// A request list with no producing node is refused: a network where nothing
// seals never advances, and that is far easier to read here than as a chain
// that starts and then does nothing.
//
// Requests beyond the pool's capacity are an error naming the shortfall. The
// alternative — wrapping onto ports already handed out — produces a network
// where two nodes cannot both bind, discovered much later and much less
// clearly.
func Assign(pool Pool, reqs []Request) (*Map, error) {
	if err := pool.Validate(); err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("netmap: no nodes requested")
	}
	if n := len(reqs); n > pool.Cap() {
		return nil, fmt.Errorf(
			"netmap: %d nodes exceed the pool: %d host(s) x %d slot(s) = %d (%d short)",
			n, len(pool.Hosts), pool.Slots, pool.Cap(), n-pool.Cap())
	}

	hosts := len(pool.Hosts)
	ordinals := make(map[node.Role]int, 3)
	placements := make([]Placement, 0, len(reqs))
	for i, r := range reqs {
		role, err := NormalizeRole(string(r.Role))
		if err != nil {
			return nil, fmt.Errorf("netmap: node %d: %w", i+1, err)
		}
		slot := i/hosts + 1 // 1-based: portplan counts slots from one
		ports, err := portplan.Plan(slot, pool.Ports.P2PBase, pool.Ports.P2PStep, pool.Ports.RPCBase, pool.Ports.RPCStep, pool.Reservation)
		if err != nil {
			return nil, fmt.Errorf("netmap: node %d: %w", i+1, err)
		}
		ordinals[role]++
		label := r.Label
		if label == "" {
			label = LabelFor(i + 1)
		}
		placements = append(placements, Placement{
			Index: i + 1,
			Label: label,
			Role:  role,
			Ord:   ordinals[role],
			Host:  pool.Hosts[i%hosts].Addr,
			Ports: ports,
		})
	}
	return NewMap(placements)
}
