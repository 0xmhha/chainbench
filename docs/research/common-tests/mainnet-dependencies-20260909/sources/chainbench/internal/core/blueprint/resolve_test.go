package blueprint

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// placed builds a placement the way the allocator would: node order, role
// ordinals, one host, ports stepped per node.
func placed(roles ...node.Role) []node.Placement {
	var out []node.Placement
	ord := map[node.Role]int{}
	for i, r := range roles {
		ord[r]++
		out = append(out, node.Placement{
			Index: i + 1, Label: node.LabelFor(i + 1), Role: r, Ord: ord[r],
			Host: "127.0.0.1",
			Ports: node.Endpoints{
				P2P: 30300 + i*10, HTTP: 8500 + i*10, WS: 8600 + i*10,
				Auth: 8700 + i*10, Metrics: 6000 + i*10,
			},
		})
	}
	return out
}

// facts is a chain that says the two things Resolve asks it.
var facts = ChainFacts{ID: "wbft", Binary: "/opt/gwbft", ChainID: 5000}

// keyed is a blueprint whose nodes carry their own keys, so a test resolves
// without a key set — the raw path N3 exists to complete.
func keyed(n int) []Node {
	var out []Node
	for i := 0; i < n; i++ {
		out = append(out, Node{NodeKey: &NodeKeyRef{Hex: "0x" + strings.Repeat("ab", 32)}})
	}
	return out
}

// TestResolve_IsDeterministic is N2's gate.
//
// Resolving twice must produce the identical snapshot. It matters because the
// genesis, the configs and the argv are all derived from it: a snapshot that
// differs between two runs of one document produces a network that does not
// match its own description, and the difference would show up as a chain that
// will not form rather than as an error anyone could read.
func TestResolve_IsDeterministic(t *testing.T) {
	bp, err := Parse([]byte(`chain: wbft
peering: mesh
nodes:
  - {name: bp1, role: bp, ports: {p2p: 40001}}
  - {name: bp2, role: bp}
  - {name: en1, role: en, syncmode: snap}
alloc:
  - {account: bp1, balance: 200000000000000000000000}
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for i := range bp.Nodes {
		bp.Nodes[i].NodeKey = &NodeKeyRef{Hex: "0x" + strings.Repeat("cd", 32)}
	}
	in := Inputs{Placed: placed(node.RoleBP, node.RoleBP, node.RoleEN), Chain: facts, Layout: node.Layout{Root: "/srv/cb"}}

	first, err := Resolve(bp, in)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	for i := 0; i < 20; i++ {
		again, err := Resolve(bp, in)
		if err != nil {
			t.Fatalf("resolve %d: %v", i, err)
		}
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("run %d differs\n first: %+v\nsecond: %+v", i, first, again)
		}
	}
}

// TestResolve_AnExplicitValueWins walks the source chain field by field. It is
// the one promise a declaration keeps: writing a value down decides it, or the
// document is a suggestion and the reader cannot tell what took effect.
func TestResolve_AnExplicitValueWins(t *testing.T) {
	bp := Blueprint{
		Chain:   "stablenet",
		Peering: "proxied",
		Genesis: &GenesisDecl{ChainID: 9999},
		Binaries: &Binaries{
			Node:      "/opt/declared",
			Overrides: []BinaryOverride{{Nodes: []string{"pn1"}, Node: "/opt/other"}},
		},
		Nodes: []Node{
			{Name: "bp1", Role: "bp", SyncMode: "archive", Ports: &node.Endpoints{P2P: 40001},
				NodeKey: &NodeKeyRef{File: "./bp1.key"},
				Account: &AccountRef{Keystore: "./bp1.json", Password: "./pw"}},
			{Name: "pn1", Role: "pn", NodeKey: &NodeKeyRef{File: "./pn1.key"}},
		},
	}
	r, err := Resolve(bp, Inputs{
		Placed: placed(node.RoleBP, node.RolePN), Chain: facts, Layout: node.Layout{Root: "/srv/cb"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	for _, c := range []struct{ what, got, want string }{
		{"chain", r.Chain, "stablenet"},
		{"peering", string(r.Peering), "proxied"},
		{"name", r.Nodes[0].Name, "bp1"},
		{"role", string(r.Nodes[0].Role), "bp"},
		{"sync mode", r.Nodes[0].SyncMode, "archive"},
		{"binary", r.Nodes[0].Binary, "/opt/declared"},
		{"overridden binary", r.Nodes[1].Binary, "/opt/other"},
		{"nodekey", r.Nodes[0].NodeKey.File, "./bp1.key"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want the declared %q", c.what, c.got, c.want)
		}
	}
	if r.ChainID != 9999 {
		t.Errorf("chain id = %d, want the declared 9999", r.ChainID)
	}
	if r.Nodes[0].Ports.P2P != 40001 {
		t.Errorf("p2p = %d, want the pinned 40001", r.Nodes[0].Ports.P2P)
	}
	if r.Nodes[0].Account == nil || r.Nodes[0].Account.Keystore != "./bp1.json" {
		t.Errorf("account = %+v, want the declared keystore", r.Nodes[0].Account)
	}
}

// TestResolve_PinningOnePortKeepsTheOthers: ports merge per field. Taking the
// whole set from whichever source spoke first would make pinning p2p mean
// discarding the other six, and the node would come up with no HTTP port.
func TestResolve_PinningOnePortKeepsTheOthers(t *testing.T) {
	bp := Blueprint{Nodes: []Node{{
		Name: "bp1", Role: "bp", Ports: &node.Endpoints{P2P: 40001},
		NodeKey: &NodeKeyRef{File: "./k"},
	}}}
	r, err := Resolve(bp, Inputs{Placed: placed(node.RoleBP), Chain: facts})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	p := r.Nodes[0].Ports
	if p.P2P != 40001 {
		t.Errorf("p2p = %d, want the pinned 40001", p.P2P)
	}
	if p.HTTP != 8500 || p.WS != 8600 || p.Auth != 8700 || p.Metrics != 6000 {
		t.Errorf("the allocated ports were lost: %+v", p)
	}
	if got := r.Sources["nodes[0].ports.p2p"]; got != FromBlueprint {
		t.Errorf("p2p source = %q, want %q", got, FromBlueprint)
	}
	if got := r.Sources["nodes[0].ports.http"]; got != FromInventory {
		t.Errorf("http source = %q, want %q", got, FromInventory)
	}
}

// TestMergePorts_CoversEveryPort holds the hand-written field list to
// node.Endpoints. A port added there and forgotten here would silently ignore
// what the document pinned — the failure the whole package exists to prevent.
func TestMergePorts_CoversEveryPort(t *testing.T) {
	want := reflect.TypeOf(node.Endpoints{}).NumField()
	if got := len(portFields()); got != want {
		t.Errorf("mergePorts knows %d ports, node.Endpoints has %d — a pinned port would be ignored", got, want)
	}
}

// TestResolve_FillsFromThePlacementWhatTheDocumentLeavesOut is the other half:
// a document that says almost nothing still resolves, and every value it did
// not state is attributed to whatever supplied it.
func TestResolve_FillsFromThePlacementWhatTheDocumentLeavesOut(t *testing.T) {
	bp := Blueprint{Nodes: keyed(2)}
	r, err := Resolve(bp, Inputs{
		Placed: placed(node.RoleBP, node.RoleEN), Chain: facts, Layout: node.Layout{Root: "/srv/cb"},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if r.Chain != "wbft" || r.ChainID != 5000 {
		t.Errorf("chain = %q/%d, want the plugin's wbft/5000", r.Chain, r.ChainID)
	}
	// The name comes from the role label, so a document that names nothing
	// still addresses its nodes the way a test definition does.
	if r.Nodes[0].Name != "bp1" || r.Nodes[1].Name != "en1" {
		t.Errorf("names = %q, %q; want bp1, en1", r.Nodes[0].Name, r.Nodes[1].Name)
	}
	// Paths are named after the identity label, never the name: a name moves
	// with a role and a datadir that moved with it would lose its chain.
	if r.Nodes[1].DataDir != "/srv/cb/node2" {
		t.Errorf("datadir = %q, want /srv/cb/node2", r.Nodes[1].DataDir)
	}
	if r.Nodes[0].SyncMode != "full" {
		t.Errorf("sync mode = %q, want the default full", r.Nodes[0].SyncMode)
	}
	for key, want := range map[string]Source{
		"chain":              FromChain,
		"chain_id":           FromChain,
		"peering":            FromDefault,
		"nodes[0].name":      FromInventory,
		"nodes[0].host":      FromInventory,
		"nodes[0].sync_mode": FromDefault,
		"nodes[0].binary":    FromChain,
		"nodes[0].nodekey":   FromBlueprint,
		"validators":         FromDefault,
	} {
		if got := r.Sources[key]; got != want {
			t.Errorf("source[%s] = %q, want %q", key, got, want)
		}
	}
}

// TestResolve_ValidatorsAreInNodeOrder: the genesis records the sealing set as
// a list, so a set that came out in a different order on two runs would build
// two different genesis files from one document.
func TestResolve_ValidatorsAreInNodeOrder(t *testing.T) {
	bp := Blueprint{Nodes: keyed(4)}
	bp.Nodes[0].Role, bp.Nodes[1].Role, bp.Nodes[2].Role, bp.Nodes[3].Role = "bp", "en", "bp", "bp"
	r, err := Resolve(bp, Inputs{Placed: placed(node.RoleBP, node.RoleEN, node.RoleBP, node.RoleBP), Chain: facts})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if want := []string{"bp1", "bp2", "bp3"}; !reflect.DeepEqual(r.Validators, want) {
		t.Errorf("validators = %v, want %v", r.Validators, want)
	}
}

// TestResolve_Refuses covers what cannot be decided and must not be guessed.
func TestResolve_Refuses(t *testing.T) {
	for name, c := range map[string]struct {
		bp    Blueprint
		in    Inputs
		wants string
	}{
		"a node with no key and no key set": {
			Blueprint{Nodes: []Node{{Name: "bp1", Role: "bp"}}},
			Inputs{Placed: placed(node.RoleBP), Chain: facts},
			"no nodekey",
		},
		"a document and a placement of different sizes": {
			Blueprint{Nodes: keyed(3)},
			Inputs{Placed: placed(node.RoleBP), Chain: facts},
			"same network",
		},
		"validators naming a label the network does not have": {
			Blueprint{Nodes: keyed(1), Validators: &ValidatorsDecl{Explicit: []string{"bp9"}}},
			Inputs{Placed: placed(node.RoleBP), Chain: facts},
			"no node in the resolved network",
		},
		"an override naming a label the network does not have": {
			Blueprint{Nodes: keyed(1), Binaries: &Binaries{Overrides: []BinaryOverride{{Nodes: []string{"bp9"}, Node: "/x"}}}},
			Inputs{Placed: placed(node.RoleBP), Chain: facts},
			"binaries override 1",
		},
		"an alloc naming a label the network does not have": {
			Blueprint{Nodes: keyed(1), Alloc: []Alloc{{Account: "en1", Balance: "1"}}},
			Inputs{Placed: placed(node.RoleBP), Chain: facts},
			"alloc 1",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Resolve(c.bp, c.in)
			if err == nil {
				t.Fatal("resolved anyway")
			}
			if !strings.Contains(err.Error(), c.wants) {
				t.Errorf("error %q does not say %q", err, c.wants)
			}
		})
	}
}

// TestResolve_AnOverrideReachesAnUnnamedNode is the defect the reference check
// exposed: matching an override against the DECLARED name means a document that
// names no nodes can never override one, and it changes nothing while saying
// nothing.
func TestResolve_AnOverrideReachesAnUnnamedNode(t *testing.T) {
	bp := Blueprint{
		Nodes:    keyed(2),
		Binaries: &Binaries{Node: "/opt/base", Overrides: []BinaryOverride{{Nodes: []string{"bp2"}, Node: "/opt/next"}}},
	}
	bp.Nodes[0].Role, bp.Nodes[1].Role = "bp", "bp"
	r, err := Resolve(bp, Inputs{Placed: placed(node.RoleBP, node.RoleBP), Chain: facts})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if r.Nodes[0].Binary != "/opt/base" {
		t.Errorf("bp1 binary = %q, want /opt/base", r.Nodes[0].Binary)
	}
	if r.Nodes[1].Binary != "/opt/next" {
		t.Errorf("bp2 binary = %q, want the override /opt/next — the override addressed it by role label", r.Nodes[1].Binary)
	}
}
