package inspector_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/inspector"
)

// presentStore is a store that knows which paths exist and nothing else.
type presentStore map[string]bool

func (s presentStore) Exists(_ context.Context, path string) (bool, error) { return s[path], nil }
func (presentStore) Read(context.Context, string) ([]byte, error)          { return nil, nil }
func (presentStore) Write(context.Context, string, []byte, fs.FileMode) error {
	return nil
}

// TestPaths_ReportsOnlyTheAbsent: the inspector says what is missing, in a
// stable order, and names each path by what it is for.
func TestPaths_ReportsOnlyTheAbsent(t *testing.T) {
	store := presentStore{"/bin/gstable": true, "/w/genesis.json": true, "/w/node1": true}
	missing, err := inspector.Paths(context.Background(), store, []inspector.Path{
		{Path: "/w/config_node2.toml", Node: 2, Purpose: "config"},
		{Path: "/bin/gstable", Purpose: "binary"},
		{Path: "/w/node2", Node: 2, Purpose: "datadir"},
		{Path: "/w/genesis.json", Purpose: "genesis"},
		{Path: "/w/node1", Node: 1, Purpose: "datadir"},
		{Path: "", Node: 9, Purpose: "unset paths are skipped"},
	})
	if err != nil {
		t.Fatalf("Paths: %v", err)
	}
	if len(missing) != 2 {
		t.Fatalf("missing = %v, want the two node2 paths", missing)
	}
	if missing[0].String() != "/w/config_node2.toml (node2 config)" || missing[1].Purpose != "datadir" {
		t.Errorf("missing = %v", missing)
	}
}
