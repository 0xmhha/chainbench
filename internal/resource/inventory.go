package resource

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrFull is returned by Take when the set has no free slot left. The error
// value carries the usage, so the message can say who holds what rather than
// leaving the operator to guess which network to remove.
var ErrFull = errors.New("resource: the set is full")

// Slot is one unit of the resource: an address and which port step it uses.
// The ports themselves are computed from the bands; the slot is the key.
type Slot struct {
	// Host is the server-set entry name (or the bare address when unnamed).
	Host string
	// Index is the 1-based slot number on that host.
	Index int
}

// Holder is who is using a slot.
type Holder struct {
	// Network names the composition that took the slot — the workspace.
	Network string
	// Node is the node's label within that network.
	Node string
}

// Allocation is one node's claim on the resource, as a workspace records it.
// Adopt reads these; it never writes them. The workspace stays the record and
// the inventory derives from it, so there is one fact and one place.
type Allocation struct {
	Network string
	Node    string
	Host    string
	P2P     int
}

// Inventory is one server set's live view: what it offers, and what has been
// handed out. One instance per set, held in memory for the life of a process.
//
// Memory is the source of truth while a process runs. A fresh process starts
// empty and Adopts the allocations that existing workspaces recorded; it does
// not keep a file of its own, because a second file holding the same facts is
// exactly how copies drift (module-plan §2a). Persisting the inventory — so a
// crashed run can be recovered — is the final work item, not this one.
//
// It is not safe for concurrent use; a process's commands take it in turn.
type Inventory struct {
	pool  Pool
	taken map[Slot]Holder
}

// NewInventory opens an inventory over a pool with nothing taken.
func NewInventory(pool Pool) (*Inventory, error) {
	if err := pool.Validate(); err != nil {
		return nil, err
	}
	return &Inventory{pool: pool, taken: map[Slot]Holder{}}, nil
}

// Pool is the resource the inventory manages.
func (i *Inventory) Pool() Pool { return i.pool }

// Adopt records allocations that already exist — read from workspaces, not
// invented here. An allocation on a host the pool does not know, or on a port
// outside the p2p band, is not this set's and is skipped; a slot adopted twice
// keeps its first holder, which is the older workspace.
func (i *Inventory) Adopt(allocs []Allocation) {
	for _, a := range allocs {
		slot, ok := i.slotOf(a)
		if !ok {
			continue
		}
		if _, held := i.taken[slot]; held {
			continue
		}
		i.taken[slot] = Holder{Network: a.Network, Node: a.Node}
	}
}

// slotOf maps an allocation onto a slot of this pool by its host and p2p port.
func (i *Inventory) slotOf(a Allocation) (Slot, bool) {
	host := ""
	for _, h := range i.pool.Hosts {
		if h.Addr == a.Host || h.Name == a.Host {
			host = h.Name
			break
		}
	}
	if host == "" {
		return Slot{}, false
	}
	band := i.pool.Ports.P2P
	if band.Step < 1 || a.P2P < band.Base || (a.P2P-band.Base)%band.Step != 0 {
		return Slot{}, false
	}
	idx := (a.P2P-band.Base)/band.Step + 1
	if idx > i.pool.Slots {
		return Slot{}, false
	}
	return Slot{Host: host, Index: idx}, true
}

// Take hands out n free slots to network, hosts before slots — the same order
// Assign places in, so a plan and a take agree. It fails with ErrFull (wrapped
// with the usage) when fewer than n are free, taking nothing.
func (i *Inventory) Take(n int, network string) ([]Slot, error) {
	if n < 1 {
		return nil, fmt.Errorf("resource: take needs at least one slot, got %d", n)
	}
	free := i.free()
	if len(free) < n {
		return nil, fmt.Errorf("%w: %d requested, %d free\n%s", ErrFull, n, len(free), i.Usage().Holders())
	}
	out := free[:n]
	for k, s := range out {
		i.taken[s] = Holder{Network: network, Node: fmt.Sprintf("node%d", k+1)}
	}
	return out, nil
}

// Release returns every slot network holds. Stopping a node is not a release
// — the node keeps its resource across a restart, only its pid changes — so
// this is what `rm` calls, and nothing else.
func (i *Inventory) Release(network string) int {
	n := 0
	for s, h := range i.taken {
		if h.Network == network {
			delete(i.taken, s)
			n++
		}
	}
	return n
}

// free lists the slots not taken, hosts before slots.
func (i *Inventory) free() []Slot {
	var out []Slot
	for idx := 1; idx <= i.pool.Slots; idx++ {
		for _, h := range i.pool.Hosts {
			s := Slot{Host: h.Name, Index: idx}
			if _, held := i.taken[s]; !held {
				out = append(out, s)
			}
		}
	}
	return out
}

// Usage is the set's capacity, what is taken, and by whom.
type Usage struct {
	Cap  int
	Used int
	Free int
	// ByNetwork counts the slots each network holds, so a full set can say
	// what to remove.
	ByNetwork map[string]int
}

// Usage reports the inventory's state.
func (i *Inventory) Usage() Usage {
	u := Usage{Cap: i.pool.Cap(), Used: len(i.taken), ByNetwork: map[string]int{}}
	u.Free = u.Cap - u.Used
	for _, h := range i.taken {
		u.ByNetwork[h.Network]++
	}
	return u
}

// Full reports whether no slot is left: every port slot on every host has
// been handed out.
func (u Usage) Full() bool { return u.Free == 0 }

// Holders renders who holds what, one network per line, sorted so the output
// is stable. It is what a full-set error prints, because "15 of 15 taken"
// tells an operator nothing about what to remove.
func (u Usage) Holders() string {
	names := make([]string, 0, len(u.ByNetwork))
	for n := range u.ByNetwork {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		fmt.Fprintf(&b, "  %s holds %d\n", n, u.ByNetwork[n])
	}
	return strings.TrimRight(b.String(), "\n")
}
