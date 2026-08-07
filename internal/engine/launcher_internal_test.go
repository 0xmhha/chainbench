package engine

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// recordingSink is a provision.FileSink capturing writes and reporting a
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

func (s *recordingSink) Write(_ context.Context, path string, content []byte, _ fs.FileMode) error {
	s.written[path] = content
	return nil
}

func TestMaterialize(t *testing.T) {
	plan := setup.Plan{
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
	plan := setup.Plan{
		DataRoot: "/d",
		Nodes: []driver.NodeSpec{
			{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", DataDir: "/d/node1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}},
			{Index: 2, Role: node.RoleEndpoint, Host: "127.0.0.1", DataDir: "/d/node2", Ports: node.Endpoints{P2P: 31010, HTTP: 8610}},
		},
	}

	specs := armSpecs(plugin, preset, plan, "go-stablenet", "/keys")
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
