package mcp_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/state"
)

func TestNetworkTopologyTool(t *testing.T) {
	up := rpcMock(map[string]any{"net_peerCount": "0x3"})
	defer up.Close()
	dir := t.TempDir()
	ns := node.NodeSet{
		Chain: "wbft", Network: "grid",
		Nodes: []node.Node{
			{Index: 1, Role: node.RoleValidator, RPCURL: up.URL},
			{Index: 2, Role: node.RoleValidator, RPCURL: up.URL},
			// unreachable endpoint -> reported down
			{Index: 3, Role: node.RoleEndpoint, RPCURL: "http://127.0.0.1:1"},
		},
	}
	if err := state.SaveNetwork(dir, ns); err != nil {
		t.Fatal(err)
	}

	text, isErr := callText(t, newServer(), "chainbench_network_topology", map[string]any{
		"name": "grid", "state_dir": dir,
	})
	if isErr {
		t.Fatalf("topology error: %s", text)
	}
	if !strings.Contains(text, "node1 "+up.URL+" up peers=3") {
		t.Errorf("node1 should be up with 3 peers:\n%s", text)
	}
	if !strings.Contains(text, "node3") || !strings.Contains(text, "down") {
		t.Errorf("node3 should be down:\n%s", text)
	}
	if !strings.Contains(text, "up=2 down=1") {
		t.Errorf("summary wrong:\n%s", text)
	}
}
