package node

import (
	"fmt"
)

// Peering is the shape of the peer graph a network is wired into. It is
// derived from roles rather than declared per node: which nodes a producer may
// talk to is a property of the topology, and writing it out node by node is
// how four copies of "everyone dials everyone" came to exist.
type Peering string

const (
	// Mesh connects every node to every other. It is the default and what
	// every composition did before this type existed.
	Mesh Peering = "mesh"
	// Proxied is the tiered graph a production network runs: bp <-> pn <-> en.
	// Endpoints do not know the producers, which is the point — transactions
	// reach the chain through en and travel inward, so an exposed RPC endpoint
	// is not a route to a validator.
	Proxied Peering = "proxied"
)

// ParsePeering resolves a peering name; an empty name is Mesh, so a caller that
// does not care keeps the behaviour it already had.
func ParsePeering(s string) (Peering, error) {
	switch Peering(s) {
	case "", Mesh:
		return Mesh, nil
	case Proxied:
		return Proxied, nil
	default:
		return "", fmt.Errorf("node: unknown peering %q (want %s or %s)", s, Mesh, Proxied)
	}
}

// RoleSupport answers whether a family can run a role. It is injected because
// This package does not know chains: the poa family has no proxy tier — etcd occupies
// that place — and only the family can say so.
type RoleSupport func(Role) bool

// Validate rejects a peering this network cannot express, before anything is
// written or launched.
//
// A pn declared on a family that has none is an error rather than a silent
// demotion to mesh: the operator asked for a tier that will not exist, and a
// network that quietly ignores half a topology is worse than one that refuses
// to start.
func (p Peering) Validate(m *Map, supports RoleSupport) error {
	if m == nil {
		return fmt.Errorf("node: peering: no map")
	}
	counts := map[Role]int{}
	for _, pl := range m.Placements() {
		counts[pl.Role]++
	}
	if supports != nil {
		for role, n := range counts {
			if n > 0 && !supports(role) {
				return fmt.Errorf("node: this chain family has no %q role, but %d node(s) declare it", role, n)
			}
		}
	}
	if p == Proxied && counts[RolePN] == 0 {
		return fmt.Errorf("node: peering %q needs at least one pn — with no proxy tier there is nothing between bp and en", Proxied)
	}
	return nil
}

// Peers returns the labels that appear in label's peer list, in map order.
//
// Mesh returns every node **including label itself**: the static-nodes file has
// always listed the whole network, the client ignores its own entry, and
// dropping it here would change the launch arguments of every existing network
// while claiming to be a refactor.
func (p Peering) Peers(m *Map, label Label) ([]Label, error) {
	if m == nil {
		return nil, fmt.Errorf("node: peering: no map")
	}
	self, ok := m.Lookup(label)
	if !ok {
		return nil, fmt.Errorf("node: %q is not in this network", label)
	}
	all := m.Placements()

	if p == Mesh {
		out := make([]Label, 0, len(all))
		for _, pl := range all {
			out = append(out, pl.Label)
		}
		return out, nil
	}

	// Proxied: endpoints are kept away from producers, and producers stay
	// connected to each other.
	//
	// The second half is not symmetry for its own sake — it was measured. A
	// graph where each bp dialled only the pn left every producer broadcasting
	// ROUND-CHANGE and seeing nothing but its own message (round 5, sequence 1,
	// currentRoundChanges.count=1, no block ever sealed): a pn is not a
	// validator, so it does not carry consensus traffic between the nodes that
	// are. The tier proxies transactions and blocks, not consensus.
	//
	// So bp <-> bp is direct, bp <-> pn and pn <-> en go through the tier, and
	// en never learns a producer — which is the property the shape exists for.
	var wants func(Role) bool
	switch {
	case Is(self.Role, RoleBP):
		wants = func(r Role) bool { return Is(r, RoleBP) || Is(r, RolePN) }
	case Is(self.Role, RoleEN):
		wants = func(r Role) bool { return Is(r, RolePN) }
	case Is(self.Role, RolePN):
		wants = func(Role) bool { return true }
	default:
		return nil, fmt.Errorf("node: peering %q has no place for role %q (%s)", p, self.Role, label)
	}

	out := make([]Label, 0, len(all))
	for _, pl := range all {
		if pl.Label == label {
			continue
		}
		if wants(pl.Role) {
			out = append(out, pl.Label)
		}
	}
	return out, nil
}

// StaticNodes assembles label's peer list into the entries a node config
// carries, formatting each peer through enode.
//
// The formatter is injected because an enode needs a public key, and key
// material belongs to the keyring while addresses belong here. Neither owner
// has to import the other: the caller holds both and hands over a function.
// A peer the formatter cannot express (no key yet) is skipped, matching what
// the assemblies this replaces did.
func (p Peering) StaticNodes(m *Map, label Label, enode func(Placement) (string, bool)) ([]string, error) {
	if enode == nil {
		return nil, fmt.Errorf("node: static nodes for %q: no enode formatter", label)
	}
	peers, err := p.Peers(m, label)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(peers))
	for _, peer := range peers {
		pl, ok := m.Lookup(peer)
		if !ok {
			return nil, fmt.Errorf("node: %q lists %q, which is not in this network", label, peer)
		}
		if e, ok := enode(pl); ok {
			out = append(out, e)
		}
	}
	return out, nil
}
