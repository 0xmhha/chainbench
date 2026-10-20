package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestChainShowDialRPCSchemaAndExecution(t *testing.T) {
	tool := Default("dial", "test").tools["chainbench_chain_show"]
	t.Run("schema", func(t *testing.T) {
		props := tool.InputSchema["properties"].(map[string]any)
		property, ok := props["dialRPC"].(map[string]any)
		if !ok || property["type"] != "boolean" || !tool.ReadOnly {
			t.Fatalf("missing read-only boolean dial contract: %#v", property)
		}
	})
	dir := t.TempDir()
	state := chainsetup.State{FormatVersion: chainsetup.StateFormatVersion, Chain: "stablenet", Docker: true, ServerSet: filepath.Join(dir, "server-set.yaml"), Nodes: []node.Record{{Index: 1, Role: "bp", Host: "172.31.42.11", DataDir: "/data/node1", Endpoints: node.Endpoints{HTTP: 8601}}}}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "chain-record.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "localmap.yaml"), []byte("hosts:\n  172.31.42.11:\n    host: 127.0.0.1\n    ports:\n      8601: 46001\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Run("execution", func(t *testing.T) {
		out, err := tool.Handler(context.Background(), map[string]any{"workspaceDir": dir, "node": float64(1), "dialRPC": true})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, `"rpcURL": "http://127.0.0.1:46001"`) || !strings.Contains(out, `"host": "172.31.42.11"`) {
			t.Fatal(out)
		}
		after, _ := os.ReadFile(filepath.Join(dir, "chain-record.json"))
		if string(data) != string(after) {
			t.Fatal("query mutated record")
		}
	})
	t.Run("invalid-type", func(t *testing.T) {
		if _, err := tool.Handler(context.Background(), map[string]any{"workspaceDir": dir, "dialRPC": "true"}); err == nil {
			t.Fatal("non-boolean dial mode silently ignored")
		}
	})
}
