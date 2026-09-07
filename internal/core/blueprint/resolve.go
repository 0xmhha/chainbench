package blueprint

import (
	"fmt"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// Inputs are the sources Resolve fills an absent field from, in the order the
// source chain names them (design §3.4).
//
// They are passed in rather than fetched. Resolve opens no file, dials nothing
// and allocates nothing: given the same Inputs it always produces the same
// snapshot, which is the property the whole pipeline downstream depends on.
type Inputs struct {
	// Placed is the allocation, one placement per node in node order. It is
	// where a host and the ports a document did not pin come from.
	Placed []node.Placement
	// Keys is the key set, when the composition has one. A blueprint that
	// writes its own nodekeys resolves with none, which is the raw path.
	Keys *keyring.Preset
	// Chain is what the chain plugin knows. The caller reads it off the
	// plugin so this package need not import the registry.
	Chain ChainFacts
	// Layout names the paths on the target.
	Layout node.Layout
}

// ChainFacts is the part of a chain plugin Resolve needs. It is a plain struct
// so a test states four fields instead of implementing a plugin, and so the
// declaration layer does not depend on the registry.
type ChainFacts struct {
	// ID is the chain's registered id ("wemix").
	ID string
	// Binary is the executable the chain runs when nothing overrides it.
	Binary string
	// ChainID is the EVM chain id when the document does not state one.
	ChainID int64
}

// defaultSyncMode is what a node runs when neither the document nor its role
// says otherwise.
const defaultSyncMode = "full"

// Resolve turns a declaration into the snapshot everything downstream reads.
//
// Every field follows the same rule: the first source that has an answer wins,
// and the source is recorded. An explicit value is therefore never overwritten,
// which is the one promise a declaration has to keep — otherwise writing a port
// down is a suggestion, and the reader cannot tell which of their settings took
// effect.
//
// Ports merge per field, not per node. `ports: {p2p: 8589}` pins p2p and leaves
// the rest to the allocation; taking the whole port set from whichever source
// spoke first would make pinning one port mean discarding the other six.
func Resolve(bp Blueprint, in Inputs) (ResolvedNetwork, error) {
	if err := bp.Validate(); err != nil {
		return ResolvedNetwork{}, err
	}
	if len(bp.Nodes) != len(in.Placed) {
		return ResolvedNetwork{}, fmt.Errorf("blueprint: resolve: the document declares %d nodes and the placement holds %d — they describe the same network", len(bp.Nodes), len(in.Placed))
	}
	peering, err := node.ParsePeering(bp.Peering)
	if err != nil {
		return ResolvedNetwork{}, fmt.Errorf("blueprint: resolve: %w", err)
	}

	r := ResolvedNetwork{
		Chain:      in.Chain.ID,
		Peering:    peering,
		Alloc:      bp.Alloc,
		Genesis:    bp.Genesis,
		Governance: bp.Governance,
		Sources:    map[string]Source{},
	}
	if bp.Chain != "" {
		r.Chain = bp.Chain
		r.Sources["chain"] = FromBlueprint
	} else if r.Chain != "" {
		r.Sources["chain"] = FromChain
	}
	r.Sources["peering"] = FromDefault
	if bp.Peering != "" {
		r.Sources["peering"] = FromBlueprint
	}

	r.ChainID = in.Chain.ChainID
	r.Sources["chain_id"] = FromChain
	if bp.Genesis != nil && bp.Genesis.ChainID != 0 {
		r.ChainID = bp.Genesis.ChainID
		r.Sources["chain_id"] = FromBlueprint
	}

	for i, dn := range bp.Nodes {
		rn, err := resolveNode(i, dn, bp, in, r.Sources)
		if err != nil {
			return ResolvedNetwork{}, err
		}
		r.Nodes = append(r.Nodes, rn)
	}

	if err := checkRefs(bp, r.Nodes); err != nil {
		return ResolvedNetwork{}, err
	}
	r.Validators, err = resolveValidators(bp, r.Nodes, r.Sources)
	if err != nil {
		return ResolvedNetwork{}, err
	}
	return r, nil
}

// checkRefs holds every node reference in the document against the network that
// actually resolved.
//
// This check lives here rather than in Validate because a reference may be a
// role label, and whether there is a bp9 is a question about the node table,
// which the document alone does not have. Left unchecked, an override naming a
// node that is not there matches nothing and changes no binary — a silent
// no-op, which is the failure mode this whole track exists to end.
func checkRefs(bp Blueprint, nodes []ResolvedNode) error {
	names := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		names[n.Name] = true
	}
	check := func(refs []string, where string) error {
		for _, r := range refs {
			if names[r] {
				continue
			}
			return fmt.Errorf("blueprint: resolve: %s names %q, which is no node in the resolved network (%s)", where, r, strings.Join(sorted(names), ", "))
		}
		return nil
	}
	if bp.Binaries != nil {
		for i, o := range bp.Binaries.Overrides {
			if err := check(o.Nodes, fmt.Sprintf("binaries override %d", i+1)); err != nil {
				return err
			}
		}
	}
	for i, a := range bp.Alloc {
		if a.Account == "" {
			continue
		}
		if err := check([]string{a.Account}, fmt.Sprintf("alloc %d", i+1)); err != nil {
			return err
		}
	}
	if bp.Validators != nil {
		return check(bp.Validators.Explicit, "validators")
	}
	return nil
}

// sorted returns a name set in a fixed order, so an error message reads the
// same on two runs.
func sorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// resolveNode decides one node's every field and records where each came from.
func resolveNode(i int, dn Node, bp Blueprint, in Inputs, src map[string]Source) (ResolvedNode, error) {
	p := in.Placed[i]
	at := func(field string) string { return fmt.Sprintf("nodes[%d].%s", i, field) }

	rn := ResolvedNode{Index: p.Index, Host: p.Host, Launch: dn.Launch}
	src[at("index")] = FromInventory
	src[at("host")] = FromInventory

	rn.Role = p.Role
	src[at("role")] = FromInventory
	if dn.Role != "" {
		role, err := node.NormalizeRole(dn.Role)
		if err != nil {
			return ResolvedNode{}, fmt.Errorf("blueprint: resolve: nodes[%d]: %w", i, err)
		}
		rn.Role = role
		src[at("role")] = FromBlueprint
	}

	rn.Name = string(node.RoleLabel(rn.Role, p.Ord))
	src[at("name")] = FromInventory
	if dn.Name != "" {
		rn.Name = dn.Name
		src[at("name")] = FromBlueprint
	}

	rn.Ports = mergePorts(dn.Ports, p.Ports, at, src)

	// Paths are named after the node's identity label, never after its name:
	// a name may change with a role, and a datadir that moves with it loses
	// the chain it holds.
	label := node.LabelFor(p.Index)
	rn.DataDir = in.Layout.DataDir(label)
	rn.ConfigPath = in.Layout.ConfigPath(label)
	rn.LogPath = in.Layout.LogPath(label)
	src[at("data_dir")] = FromInventory

	rn.SyncMode = defaultSyncMode
	src[at("sync_mode")] = FromDefault
	if dn.SyncMode != "" {
		rn.SyncMode = dn.SyncMode
		src[at("sync_mode")] = FromBlueprint
	}

	rn.Binary = in.Chain.Binary
	src[at("binary")] = FromChain
	if bp.Binaries != nil {
		if bp.Binaries.Node != "" {
			rn.Binary = bp.Binaries.Node
			src[at("binary")] = FromBlueprint
		}
		// Matched against the RESOLVED name, not the declared one. A document
		// that names no nodes still addresses them by role label, and matching
		// the declared field would make `nodes: [bp1]` there match nothing and
		// change no binary, without saying so.
		//
		// A later override wins over an earlier one, so a document can state a
		// rule and then an exception to it.
		for _, o := range bp.Binaries.Overrides {
			for _, n := range o.Nodes {
				if n == rn.Name {
					rn.Binary = o.Node
					src[at("binary")] = FromBlueprint
				}
			}
		}
	}

	if err := resolveKeys(&rn, dn, in, i, at, src); err != nil {
		return ResolvedNode{}, err
	}
	return rn, nil
}

// mergePorts takes each port from the document when it is pinned there and from
// the placement otherwise.
func mergePorts(declared *node.Endpoints, placed node.Endpoints, at func(string) string, src map[string]Source) node.Endpoints {
	out := placed
	// Each row is one port, the field it lands in, and how to read it off a
	// declaration. Listing them is deliberate: a port added to node.Endpoints
	// and forgotten here would silently ignore what the document pinned, and
	// TestMergePorts_CoversEveryPort fails when the two lists disagree.
	fields := []struct {
		name string
		to   *int
		of   func(node.Endpoints) int
	}{
		{"p2p", &out.P2P, func(e node.Endpoints) int { return e.P2P }},
		{"etcd", &out.Etcd, func(e node.Endpoints) int { return e.Etcd }},
		{"etcd_client", &out.EtcdClient, func(e node.Endpoints) int { return e.EtcdClient }},
		{"http", &out.HTTP, func(e node.Endpoints) int { return e.HTTP }},
		{"ws", &out.WS, func(e node.Endpoints) int { return e.WS }},
		{"auth", &out.Auth, func(e node.Endpoints) int { return e.Auth }},
		{"metrics", &out.Metrics, func(e node.Endpoints) int { return e.Metrics }},
	}
	for _, f := range fields {
		key := at("ports." + f.name)
		src[key] = FromInventory
		if declared == nil {
			continue
		}
		if v := f.of(*declared); v != 0 {
			*f.to = v
			src[key] = FromBlueprint
		}
	}
	return out
}

// portFields names the ports mergePorts knows, for the test that holds it to
// node.Endpoints.
func portFields() []string {
	return []string{"p2p", "etcd", "etcd_client", "http", "ws", "auth", "metrics"}
}

// resolveKeys decides a node's devp2p identity and, for a sealer, its account.
//
// A missing key is an error naming the node and what would supply it. The
// alternative is a node launched with an empty identity, which joins nothing
// and reports nothing wrong.
func resolveKeys(rn *ResolvedNode, dn Node, in Inputs, i int, at func(string) string, src map[string]Source) error {
	switch {
	case dn.NodeKey != nil:
		rn.NodeKey = *dn.NodeKey
		src[at("nodekey")] = FromBlueprint
	case in.Keys != nil:
		e, ok := in.Keys.Node(rn.Index)
		if !ok {
			return fmt.Errorf("blueprint: resolve: nodes[%d] (%s): the key set has no entry %d — declare a nodekey or use a set that covers %d nodes", i, rn.Name, rn.Index, len(in.Placed))
		}
		rn.NodeKey = NodeKeyRef{Hex: e.Nodekey.Hex()}
		src[at("nodekey")] = FromKeySet
	default:
		return fmt.Errorf("blueprint: resolve: nodes[%d] (%s): no nodekey — declare one or resolve with a key set", i, rn.Name)
	}

	// Only a sealer holds an account, and only where one is actually stated:
	// a key set supplies identities, and whether its entry may seal is the
	// genesis's answer rather than this one's.
	if !node.Is(rn.Role, node.RoleBP) {
		return nil
	}
	if dn.Account != nil {
		acct := *dn.Account
		rn.Account = &acct
		src[at("account")] = FromBlueprint
	}
	return nil
}

// resolveValidators names the nodes that seal, in a fixed order.
//
// Node order, not the order a set iterates: the genesis records the sealing set
// as a list, and a list that comes out differently on two runs produces two
// different genesis files from one document.
func resolveValidators(bp Blueprint, nodes []ResolvedNode, src map[string]Source) ([]string, error) {
	if bp.Validators != nil && len(bp.Validators.Explicit) > 0 {
		// Already held against the resolved node table by checkRefs.
		src["validators"] = FromBlueprint
		return append([]string(nil), bp.Validators.Explicit...), nil
	}
	src["validators"] = FromDefault
	if bp.Validators != nil && bp.Validators.From != "" {
		src["validators"] = FromBlueprint
	}
	var out []string
	for _, n := range nodes {
		if node.Is(n.Role, node.RoleBP) {
			out = append(out, n.Name)
		}
	}
	return out, nil
}
