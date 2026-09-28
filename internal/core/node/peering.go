package node

import (
	"fmt"
	"slices"
	"strings"
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

// groupsPrefix marks a declared peer graph: "groups:" followed by groups of
// node labels, groups separated by ";" and labels by ",". Every node in a
// group dials every other member, and nothing else.
//
// This one is declared, not derived. A test that isolates part of a network
// has to say which part, and no role rule knows that: a fault case splits
// nine producers into two sets joined by one of them, and stopping that one
// is the partition. Groups may overlap, which is how a bridge is written —
// the bridge is the node in both groups.
//
// It rides in the same string every other peering does, so the request, the
// workspace state, the run record and reuse's comparison all carry the graph
// with no new field, and two networks wired differently never compare equal.
const groupsPrefix = "groups:"

// GroupsPeering builds the declared peer graph from its groups. It checks the
// shape a declaration can get wrong on its own; whether the labels exist is
// Validate's question, asked against the network they are placed in.
func GroupsPeering(groups [][]string) (Peering, error) {
	if len(groups) == 0 {
		return "", fmt.Errorf("node: peering groups: none declared")
	}
	parts := make([]string, 0, len(groups))
	for i, g := range groups {
		if len(g) < 2 {
			return "", fmt.Errorf("node: peering group %d has %d node(s) — a group is the nodes that dial each other, so it needs two", i+1, len(g))
		}
		seen := map[string]bool{}
		for _, l := range g {
			if _, err := Label(l).Index(); err != nil {
				return "", fmt.Errorf("node: peering group %d: %w (name nodes by index, e.g. node5)", i+1, err)
			}
			if seen[l] {
				return "", fmt.Errorf("node: peering group %d lists %s twice", i+1, l)
			}
			seen[l] = true
		}
		parts = append(parts, strings.Join(g, ","))
	}
	return Peering(groupsPrefix + strings.Join(parts, ";")), nil
}

// groups splits a declared peer graph back into its groups; ok is false for
// any other peering.
func (p Peering) groups() ([][]Label, bool) {
	body, ok := strings.CutPrefix(string(p), groupsPrefix)
	if !ok {
		return nil, false
	}
	var out [][]Label
	for part := range strings.SplitSeq(body, ";") {
		var g []Label
		for l := range strings.SplitSeq(part, ",") {
			g = append(g, Label(l))
		}
		out = append(out, g)
	}
	return out, true
}

// neighbours is the set of labels that share a group with label.
func (p Peering) neighbours(label Label) map[Label]bool {
	gs, _ := p.groups()
	out := map[Label]bool{}
	for _, g := range gs {
		if !slices.Contains(g, label) {
			continue
		}
		for _, l := range g {
			if l != label {
				out[l] = true
			}
		}
	}
	return out
}

// DeclaredPeers is how many peers label should have under a declared peer
// graph; ok is false for a derived one (mesh, proxied), whose count the caller
// cannot know from the graph alone. A readiness check uses it to hold a
// network until every declared connection is up — a network wired otherwise
// than declared is not the network the test asked for.
func (p Peering) DeclaredPeers(label Label) (int, bool) {
	if _, ok := p.groups(); !ok {
		return 0, false
	}
	return len(p.neighbours(label)), true
}

// ParsePeering resolves a peering name; an empty name is Mesh, so a caller that
// does not care keeps the behaviour it already had.
func ParsePeering(s string) (Peering, error) {
	switch Peering(s) {
	case "", Mesh:
		return Mesh, nil
	case Proxied:
		return Proxied, nil
	}
	if gs, ok := Peering(s).groups(); ok {
		raw := make([][]string, len(gs))
		for i, g := range gs {
			for _, l := range g {
				raw[i] = append(raw[i], string(l))
			}
		}
		return GroupsPeering(raw)
	}
	return "", fmt.Errorf("node: unknown peering %q (want %s, %s or %s<groups>)", s, Mesh, Proxied, groupsPrefix)
}

// RoleSupport answers whether a family can run a role. It is injected because
// this package does not know chains, and whether a family can run a role is the
// family's answer to give.
//
// It said the poa family has no proxy tier. That has not been true since poa
// gained one: poa.Family.SupportsRole accepts pn, and a wemix bp/pn/en network
// comes up and produces blocks. Both families run all three roles today; the
// question stays the family's because the next one may not.
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
	if _, ok := p.groups(); ok {
		return p.validateGroups(m)
	}
	if p == Proxied && counts[RolePN] == 0 {
		return fmt.Errorf("node: peering %q needs at least one pn — with no proxy tier there is nothing between bp and en", Proxied)
	}
	// The mirror of the check above: a pn only means something under proxied
	// peering. Meshing a topology that declares a proxy tier would let endpoints
	// dial producers anyway, so refuse it loudly rather than quietly ignore the
	// tier the operator asked for.
	if p == Mesh && counts[RolePN] > 0 {
		return fmt.Errorf("node: peering %q with %d pn(s) — a proxy tier only takes effect under %s peering; under mesh endpoints would still dial producers", Mesh, counts[RolePN], Proxied)
	}
	return nil
}

// validateGroups holds a declared graph to the network it is placed in: every
// label is a node of it, every node is in some group, and the producers can
// reach each other through producers.
//
// The last rule is the one a declaration gets wrong without noticing. A pn
// does not carry consensus traffic (see Proxied below — measured), so
// producers joined only through a pn never seal a block, and the network
// would sit in round changes until the readiness wait ran out. It is refused
// here, before anything is written, with the reason.
func (p Peering) validateGroups(m *Map) error {
	gs, _ := p.groups()
	member := map[Label]bool{}
	for i, g := range gs {
		for _, l := range g {
			if _, ok := m.Lookup(l); !ok {
				return fmt.Errorf("node: peering group %d names %s, which is not in this network", i+1, l)
			}
			member[l] = true
		}
	}
	var bps []Label
	for _, pl := range m.Placements() {
		if !member[pl.Label] {
			return fmt.Errorf("node: peering groups leave %s out — a node in no group has no peers", pl.Label)
		}
		if Is(pl.Role, RoleBP) {
			bps = append(bps, pl.Label)
		}
	}
	if len(bps) < 2 {
		return nil
	}
	isBP := map[Label]bool{}
	for _, l := range bps {
		isBP[l] = true
	}
	reached := map[Label]bool{bps[0]: true}
	queue := []Label{bps[0]}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for n := range p.neighbours(cur) {
			if isBP[n] && !reached[n] {
				reached[n] = true
				queue = append(queue, n)
			}
		}
	}
	for _, l := range bps {
		if !reached[l] {
			return fmt.Errorf("node: peering groups leave producer %s unreachable from %s through producers — a pn does not relay consensus, so these producers could never seal a block together", l, bps[0])
		}
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

	// Declared groups: exactly the nodes that share a group with label. Both
	// ends of every connection list each other, because membership is shared —
	// a one-sided entry would still connect the pair, since a node accepts the
	// dial it did not make.
	if _, ok := p.groups(); ok {
		near := p.neighbours(label)
		out := make([]Label, 0, len(near))
		for _, pl := range all {
			if near[pl.Label] {
				out = append(out, pl.Label)
			}
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
