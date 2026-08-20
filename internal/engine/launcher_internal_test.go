package engine

import (
	"context"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// recordingSink is a provision.FileStore capturing writes and reporting a
// preset existing set (to exercise upload-if-absent) in memory.
type recordingSink struct {
	written  map[string][]byte
	existing map[string]bool
}

func newRecordingSink() *recordingSink {
	return &recordingSink{written: map[string][]byte{}, existing: map[string]bool{}}
}

func (s *recordingSink) Exists(_ context.Context, path string) (bool, error) {
	return s.existing[path], nil
}

func (s *recordingSink) Read(_ context.Context, path string) ([]byte, error) {
	b, ok := s.written[path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return b, nil
}

func (s *recordingSink) Write(_ context.Context, path string, content []byte, _ fs.FileMode) error {
	s.written[path] = content
	return nil
}

func TestMaterialize(t *testing.T) {
	plan := driver.Plan{
		DataRoot:    "/d",
		GenesisPath: "/d/genesis.json",
		Genesis:     []byte(`{"g":1}`),
		Nodes: []driver.NodeSpec{
			{Index: 1, ConfigPath: "/d/config_node1.toml", ConfigContent: []byte("cfg1")},
			{Index: 2, ConfigPath: "/d/config_node2.toml", ConfigContent: []byte("cfg2")},
		},
	}

	sink := newRecordingSink()
	if err := materialize(context.Background(), provision.New(sink), plan, plan.Nodes); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	for _, want := range []string{"/d/genesis.json", "/d/config_node1.toml", "/d/config_node2.toml"} {
		if _, ok := sink.written[want]; !ok {
			t.Fatalf("expected %s to be written, got %v", want, keysOf(sink.written))
		}
	}

	// Upload-if-absent: an existing genesis is not rewritten.
	sink2 := newRecordingSink()
	sink2.existing["/d/genesis.json"] = true
	if err := materialize(context.Background(), provision.New(sink2), plan, plan.Nodes); err != nil {
		t.Fatalf("materialize (reuse): %v", err)
	}
	if _, ok := sink2.written["/d/genesis.json"]; ok {
		t.Fatal("existing genesis should be reused, not rewritten")
	}
	if _, ok := sink2.written["/d/config_node1.toml"]; !ok {
		t.Fatal("absent config should still be written")
	}
}

func keysOf(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestArmSpecs(t *testing.T) {
	plugin := registry.StaticPlugin{
		M: registry.Manifest{
			ID: "stablenet", Binary: "go-stablenet", MinerRecommit: "duration",
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}
	preset := keys.Preset{
		Validators: []string{"0xval1"},
		Nodes: []keys.NodeKey{
			{Index: 1, PublicKey: "aa11", Address: "0xval1"},
			{Index: 2, PublicKey: "bb22", Address: "0xen2"},
		},
	}
	plan := driver.Plan{
		DataRoot: "/d",
		Nodes: []driver.NodeSpec{
			{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", DataDir: "/d/node1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}},
			{Index: 2, Role: node.RoleEndpoint, Host: "127.0.0.1", DataDir: "/d/node2", Ports: node.Endpoints{P2P: 31010, HTTP: 8610}},
		},
	}

	specs, err := armSpecs(plugin, preset, plan, "go-stablenet", "/keys", nil)
	if err != nil {
		t.Fatalf("armSpecs: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("specs = %d, want 2", len(specs))
	}

	v := specs[0]
	if v.Binary != "go-stablenet" {
		t.Fatalf("binary = %q", v.Binary)
	}
	if len(v.ConfigContent) == 0 {
		t.Fatal("validator has no rendered config content")
	}
	if !argsHas(v.Args, "--nodekey") {
		t.Fatalf("validator missing --nodekey: %v", v.Args)
	}
	if !argsHas(v.Args, "--unlock", "0xval1", "--miner.etherbase") {
		t.Fatalf("validator missing unlock/etherbase: %v", v.Args)
	}
	// The static-node enode must use the plan's (allocator) p2p port, not a
	// preset default, so peering matches the launched layout.
	if !strings.Contains(string(v.ConfigContent), "31000") {
		t.Fatalf("static node should use plan p2p port 31000:\n%s", v.ConfigContent)
	}

	// An endpoint gets a nodekey but no validator unlock.
	if argsHas(specs[1].Args, "--unlock") {
		t.Fatalf("endpoint should not unlock an account: %v", specs[1].Args)
	}
	if !argsHas(specs[1].Args, "--nodekey") {
		t.Fatalf("endpoint missing --nodekey: %v", specs[1].Args)
	}
}

// argsHas reports whether every val appears somewhere in args.
func argsHas(args []string, vals ...string) bool {
	set := make(map[string]bool, len(args))
	for _, a := range args {
		set[a] = true
	}
	for _, v := range vals {
		if !set[v] {
			return false
		}
	}
	return true
}

// TestArmSpecsOverrides pins the customization seam: a network id pin and a
// user launch knob flow through the Builder's override layer into every
// node's argv.
func TestArmSpecsOverrides(t *testing.T) {
	plugin := registry.StaticPlugin{
		M: registry.Manifest{
			ID: "stablenet", Binary: "go-stablenet", MinerRecommit: "duration",
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}
	preset := keys.Preset{Nodes: []keys.NodeKey{{Index: 1, PublicKey: "aa11", Address: "0xval1"}}}
	plan := driver.Plan{
		DataRoot: "/d",
		Nodes: []driver.NodeSpec{
			{Index: 1, Role: node.RoleEndpoint, Host: "127.0.0.1", DataDir: "/d/node1",
				ConfigPath: "/d/c1.toml", Ports: node.Endpoints{P2P: 31000, HTTP: 8600, WS: 8700}},
		},
	}
	specs, err := armSpecs(plugin, preset, plan, "go-stablenet", "/keys", []launchopt.Override{
		{Key: launchopt.KeyNetworkID, Value: "4242"},
		{Key: launchopt.KeyMaxPeers, Value: "7"},
	})
	if err != nil {
		t.Fatalf("armSpecs: %v", err)
	}
	if !argsHas(specs[0].Args, "--networkid", "4242", "--maxpeers", "7") {
		t.Fatalf("overrides missing from argv: %v", specs[0].Args)
	}

	// An override the dialect cannot express must fail assembly, not vanish.
	_, err = armSpecs(plugin, preset, plan, "go-stablenet", "/keys", []launchopt.Override{
		{Key: launchopt.KeyBlockInterval, Value: "1"},
	})
	if err == nil {
		t.Fatal("wemix-only knob on geth114 must fail arming")
	}
}

// TestArmSpecsLaunchoptEquivalence is the golden gate for the launchopt
// conversion: for every role, the Builder's argv must carry exactly the same
// flag/value pairs as the legacy composition (nodeconfig.LaunchArgs + family
// StartFlags + armSpecs identity appends). Order is not compared — the legacy
// argv interleaved identity and mining flags twice, which concern-contiguous
// emission cannot reproduce, and geth-family flag parsing is
// position-independent. Pair equality is the semantic contract.
func TestArmSpecsLaunchoptEquivalence(t *testing.T) {
	plugin := registry.StaticPlugin{
		M: registry.Manifest{
			ID: "stablenet", Binary: "go-stablenet", MinerRecommit: "duration",
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}
	preset := keys.Preset{
		Validators: []string{"0xval1"},
		Nodes: []keys.NodeKey{
			{Index: 1, PublicKey: "aa11", Address: "0xval1"},
			{Index: 2, PublicKey: "bb22", Address: "0xen2"},
		},
	}
	plan := driver.Plan{
		DataRoot: "/d",
		Nodes: []driver.NodeSpec{
			{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", DataDir: "/d/node1",
				ConfigPath: "/d/config_node1.toml", Ports: node.Endpoints{P2P: 31000, HTTP: 8600, WS: 8700}},
			{Index: 2, Role: node.RoleEndpoint, Host: "127.0.0.1", DataDir: "/d/node2",
				ConfigPath: "/d/config_node2.toml", Ports: node.Endpoints{P2P: 31010, HTTP: 8610, WS: 8710}},
		},
	}

	specs, err := armSpecs(plugin, preset, plan, "go-stablenet", "/keys", nil)
	if err != nil {
		t.Fatalf("armSpecs: %v", err)
	}

	fam := plugin.Family()
	for i, spec := range plan.Nodes {
		legacy := nodeconfig.LaunchArgs(spec.DataDir, spec.ConfigPath, spec.Ports, fam.StartFlags(spec.Role))
		legacy = append(legacy, "--nodekey", "/keys/node"+strconv.Itoa(spec.Index)+"/nodekey")
		if spec.Role == node.RoleValidator {
			if nk, ok := preset.Node(spec.Index); ok {
				legacy = append(legacy,
					"--unlock", nk.Address,
					"--password", "/keys/password",
					"--miner.etherbase", nk.Address,
				)
			}
		}
		if diff := pairDiff(legacy, specs[i].Args); diff != "" {
			t.Fatalf("node%d argv pairs diverge from legacy:\n%s\nlegacy: %v\n   new: %v",
				spec.Index, diff, legacy, specs[i].Args)
		}
	}
}

// pairDiff compares two argvs as flag->value maps (bool flags map to "").
// Returns "" when equal, else a description of the first difference.
func pairDiff(a, b []string) string {
	pa, pb := pairsOf(a), pairsOf(b)
	for k, v := range pa {
		if w, ok := pb[k]; !ok {
			return "missing " + k
		} else if v != w {
			return k + " = " + w + ", want " + v
		}
	}
	for k := range pb {
		if _, ok := pa[k]; !ok {
			return "extra " + k
		}
	}
	return ""
}

func pairsOf(args []string) map[string]string {
	out := map[string]string{}
	for i := 0; i < len(args); i++ {
		flag := args[i]
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			out[flag] = args[i+1]
			i++
		} else {
			out[flag] = ""
		}
	}
	return out
}
