package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/types"
)

func TestSaveRemote_WritesNetworksFile(t *testing.T) {
	dir := t.TempDir()
	net := &types.Network{
		Name:      "sepolia",
		ChainType: types.NetworkChainType("ethereum"),
		ChainId:   11155111,
		Nodes: []types.Node{{
			Id:       "node1",
			Provider: types.NodeProvider("remote"),
			Http:     "https://rpc.sepolia.test",
		}},
	}
	if err := SaveRemote(dir, net); err != nil {
		t.Fatalf("SaveRemote: %v", err)
	}
	path := filepath.Join(dir, "networks", "sepolia.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("empty file written")
	}
}

func TestSaveRemote_RejectsLocal(t *testing.T) {
	dir := t.TempDir()
	net := &types.Network{Name: "local", ChainType: "stablenet", ChainId: 8283}
	if err := SaveRemote(dir, net); err == nil {
		t.Fatal("expected error for reserved name 'local'")
	}
}

func TestSaveRemote_RejectsBadName(t *testing.T) {
	dir := t.TempDir()
	// Note: "trailing-" is NOT included here even though the plan snippet
	// listed it. The schema pattern ^[a-z0-9][a-z0-9_-]*$ (SSoT in
	// schema/network.json, mirrored by remoteNameRE) explicitly permits a
	// trailing hyphen, so the regex-based validator accepts it. The plan
	// snippet test case conflicted with the plan's stated regex; we honor
	// the schema SSoT.
	cases := []string{"", "Has-Upper", "has/slash", "..", ".hidden"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			net := &types.Network{Name: name, ChainType: "ethereum", ChainId: 1}
			if err := SaveRemote(dir, net); err == nil {
				t.Errorf("expected error for bad name %q", name)
			}
		})
	}
}

func TestSaveRemote_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	orig := &types.Network{
		Name:      "mynet",
		ChainType: types.NetworkChainType("wbft"),
		ChainId:   31337,
		Nodes: []types.Node{{
			Id:       "node1",
			Provider: types.NodeProvider("remote"),
			Http:     "https://rpc.example.com",
		}},
	}
	if err := SaveRemote(dir, orig); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := loadRemote(dir, "mynet")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Name != orig.Name || got.ChainId != orig.ChainId || string(got.ChainType) != string(orig.ChainType) {
		t.Errorf("roundtrip mismatch: got %+v want %+v", got, orig)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].Http != orig.Nodes[0].Http {
		t.Errorf("nodes mismatch: got %+v", got.Nodes)
	}
}

func TestLoadRemote_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := loadRemote(dir, "missing")
	if err == nil {
		t.Fatal("expected error for missing network")
	}
}

func TestSaveRemote_AtomicOnExistingFile(t *testing.T) {
	dir := t.TempDir()
	orig := &types.Network{Name: "foo", ChainType: "ethereum", ChainId: 1, Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: "http://a"}}}
	if err := SaveRemote(dir, orig); err != nil {
		t.Fatal(err)
	}
	// Overwrite with new content
	updated := *orig
	updated.Nodes = []types.Node{{Id: "node1", Provider: "remote", Http: "http://b"}}
	if err := SaveRemote(dir, &updated); err != nil {
		t.Fatal(err)
	}
	got, err := loadRemote(dir, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if got.Nodes[0].Http != "http://b" {
		t.Errorf("overwrite failed: %q", got.Nodes[0].Http)
	}
	// No orphan temp file
	entries, _ := os.ReadDir(filepath.Join(dir, "networks"))
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("orphan temp file: %s", e.Name())
		}
	}
}
