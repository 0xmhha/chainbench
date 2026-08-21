package netmap

import (
	"fmt"
	"path"
	"slices"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// PortSpan is how many consecutive ports one node consumes from its base:
// p2p, etcd (=p2p+1, reserved even where unused so a wemix node never
// collides), http, ws, auth, metrics. Port bases in a pool must therefore be
// at least this far apart.
const PortSpan = 6

// Pool is the assignable address space, read from the operator's gitignored
// inventory (schema v2): the usable host addresses, the port bases a node's
// port set is carved from, and the server-side root that node data dirs are
// created under.
//
// It is a pool rather than per-server entries because nodes may outnumber
// hosts: the walk reuses addresses at the next base, and two nodes on one host
// always differ by base.
type Pool struct {
	// Hosts are the usable addresses, in assignment order.
	Hosts []string
	// PortBases are the usable port-set bases, in assignment order. Each base
	// yields one node's PortSpan consecutive ports via FromBase.
	PortBases []int
	// DataRoot is the configuration root on each host; a node's data dir is
	// <DataRoot>/<label>. It is not the node's datadir itself.
	DataRoot string
}

// Capacity is how many nodes the pool can place without a collision.
func (p Pool) Capacity() int { return len(p.Hosts) * len(p.PortBases) }

// Validate rejects a pool that cannot assign safely: empty axes, duplicate
// hosts or bases, and bases closer together than one node's span — which would
// hand two nodes overlapping port sets on every host.
func (p Pool) Validate() error {
	if len(p.Hosts) == 0 {
		return fmt.Errorf("netmap: pool has no hosts")
	}
	if len(p.PortBases) == 0 {
		return fmt.Errorf("netmap: pool has no port bases")
	}
	seen := make(map[string]bool, len(p.Hosts))
	for _, h := range p.Hosts {
		if h == "" {
			return fmt.Errorf("netmap: pool has an empty host")
		}
		if seen[h] {
			return fmt.Errorf("netmap: host %s appears twice in the pool", h)
		}
		seen[h] = true
	}
	bases := slices.Clone(p.PortBases)
	slices.Sort(bases)
	for i, b := range bases {
		if b <= 0 {
			return fmt.Errorf("netmap: port base %d is not a valid port", b)
		}
		if i > 0 && b-bases[i-1] < PortSpan {
			return fmt.Errorf("netmap: port bases %d and %d are %d apart; a node's port set spans %d",
				bases[i-1], b, b-bases[i-1], PortSpan)
		}
	}
	return nil
}

// FromBase carves one node's port set out of a base: PortSpan consecutive
// ports. Etcd is reserved even for families that do not run it, so a layout
// that works for wbft cannot collide when reused for wemix.
func FromBase(base int) Ports {
	return Ports{
		P2P:     base,
		Etcd:    base + 1,
		HTTP:    base + 2,
		WS:      base + 3,
		Auth:    base + 4,
		Metrics: base + 5,
	}
}

// Assign places one node per role onto the pool, deterministically: node 1
// walks the host list first, and when the hosts are exhausted the base index
// advances and the walk restarts. Two nodes on one host therefore always
// differ by base, and the same pool with the same roles always yields the same
// map — a re-run cannot produce a different layout.
//
// Overflow is a counted refusal, never a silent collision: asking for more
// nodes than Capacity says how many placements are missing.
func Assign(pool Pool, roles []node.Role) (*Map, error) {
	if err := pool.Validate(); err != nil {
		return nil, err
	}
	if len(roles) > pool.Capacity() {
		return nil, fmt.Errorf(
			"netmap: %d nodes exceed the pool's capacity of %d (%d hosts × %d port bases) — %d short; add hosts or port bases to the inventory",
			len(roles), pool.Capacity(), len(pool.Hosts), len(pool.PortBases), len(roles)-pool.Capacity())
	}

	placements := make([]Placement, 0, len(roles))
	for i, role := range roles {
		label := LabelFor(i + 1)
		placements = append(placements, Placement{
			Label:   label,
			Role:    role,
			Host:    pool.Hosts[i%len(pool.Hosts)],
			Ports:   FromBase(pool.PortBases[i/len(pool.Hosts)]),
			DataDir: path.Join(pool.DataRoot, string(label)),
		})
	}
	// NewMap re-checks address uniqueness. With a valid pool it cannot fail,
	// but the map is the contract every consumer relies on, so the guarantee
	// comes from the check rather than from this comment.
	return NewMap(placements)
}
