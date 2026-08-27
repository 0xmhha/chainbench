package chainsetup

import (
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"regexp"
	"strings"
	"testing"

	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

var enodeLine = regexp.MustCompile(`"(enode://[^"]+)"`)

// TestArmSpecs_StaticNodesMatchNetmapMesh held the launcher's own assembly
// against node.Mesh while the two coexisted; the launcher now calls netmap,
// so what it pins today is the contract on the far side of the rendering: the
// config a node is handed still carries the whole network, in map order, with
// the node's own entry included.
//
// That list is the launched network's peering. A change of one entry or one
// position shows up as nodes that cannot find each other, far from the commit
// that caused it — which is why it is asserted against the rendered config
// rather than against the slice that produced it.
func TestArmSpecs_StaticNodesMatchNetmapMesh(t *testing.T) {
	const n = 4
	plugin := registry.StaticPlugin{
		M: registry.Manifest{
			ID: "stablenet", Binary: "go-stablenet", MinerRecommit: "duration",
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}

	preset := keyring.Preset{Network: keyring.Network{Validators: []string{"0xval1"}}}
	plan := driver.Plan{DataRoot: "/d"}
	for i := 1; i <= n; i++ {
		role := node.RoleValidator
		if i > 2 {
			role = node.RoleEndpoint
		}
		preset.Nodes = append(preset.Nodes, keyring.Entry{
			Index:    i,
			Identity: derive.Identity{PublicKey: fmt.Sprintf("%02x%02x", i, i), Address: fmt.Sprintf("0xnode%d", i)},
		})
		plan.Nodes = append(plan.Nodes, driver.NodeSpec{
			Index: i, Role: role, Host: "127.0.0.1", DataDir: fmt.Sprintf("/d/node%d", i),
			Ports: node.Endpoints{P2P: 31000 + (i-1)*10, HTTP: 8600 + (i-1)*10},
		})
	}

	specs, err := armSpecs(plugin, preset, plan, "go-stablenet", "/keys", "", nil)
	if err != nil {
		t.Fatalf("armSpecs: %v", err)
	}

	// The same network as a netmap, assigned from the pool the launcher's
	// ports came from.
	m, err := netmap.Assign(netmap.Pool{
		Hosts: []netmap.Host{{Name: "local", Addr: "127.0.0.1"}},
		Slots: n,
		Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10},
	}, []netmap.Request{
		{Role: node.RoleValidator}, {Role: node.RoleValidator},
		{Role: node.RoleEndpoint}, {Role: node.RoleEndpoint},
	})
	if err != nil {
		t.Fatalf("netmap.Assign: %v", err)
	}
	// The caller holds both the map and the key material; netmap holds neither.
	enode := func(p node.Placement) (string, bool) {
		e, ok := preset.Node(p.Index)
		if !ok {
			return "", false
		}
		return node.Enode(e.PublicKey, p.Host, p.Ports.P2P), true
	}

	for i, spec := range specs {
		label := node.LabelFor(i + 1)
		want, err := node.Mesh.StaticNodes(m, label, enode)
		if err != nil {
			t.Fatalf("StaticNodes(%s): %v", label, err)
		}
		got := staticNodesOf(string(spec.ConfigContent))
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("%s static nodes differ\n launcher: %v\n netmap:   %v", label, got, want)
		}
		if len(got) != n {
			t.Fatalf("%s lists %d peers, want the whole network (%d, self included)", label, len(got), n)
		}
	}
}

func staticNodesOf(config string) []string {
	var out []string
	for _, m := range enodeLine.FindAllStringSubmatch(config, -1) {
		out = append(out, m[1])
	}
	return out
}
