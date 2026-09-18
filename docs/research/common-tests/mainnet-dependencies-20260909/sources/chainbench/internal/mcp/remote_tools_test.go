package mcp_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

func TestRemoteRPCTool(t *testing.T) {
	srv := rpcMock(map[string]any{"eth_blockNumber": "0x2a"})
	defer srv.Close()
	dir := t.TempDir()
	// save a network pointing at the mock endpoint (as attach would).
	ns := node.NodeSet{
		Chain: "wbft", Network: "prod",
		Nodes: []node.Node{{Index: 1, Role: node.RoleEndpoint, RPCURL: srv.URL}},
	}
	if err := session.SaveNetwork(dir, ns); err != nil {
		t.Fatal(err)
	}

	text, isErr := callText(t, newServer(), "chainbench_remote_rpc", map[string]any{
		"name": "prod", "state_dir": dir, "method": "eth_blockNumber",
	})
	if isErr || !strings.Contains(text, "0x2a") {
		t.Fatalf("remote_rpc: err=%v text=%s", isErr, text)
	}

	// unknown network is an error.
	if _, isErr := callText(t, newServer(), "chainbench_remote_rpc", map[string]any{
		"name": "nope", "state_dir": dir, "method": "eth_blockNumber",
	}); !isErr {
		t.Error("unknown network should error")
	}
}
