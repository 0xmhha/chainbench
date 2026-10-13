package testengine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestAttachedLocalSignerUsesTheEndpointAuthentication(t *testing.T) {
	t.Setenv("CHAINBENCH_TEST_SIGNING_TOKEN", "local-signing-test-token")
	var signed, unauthenticated atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer local-signing-test-token" {
			unauthenticated.Add(1)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		var result any = "0x1"
		switch request.Method {
		case "eth_chainId":
			result = "0x205b"
		case "eth_getProof":
			result = map[string]string{"extra": "0x0"}
		case "eth_getTransactionCount":
			result = "0x0"
		case "eth_sendRawTransaction":
			signed.Add(1)
			result = "0x" + strings.Repeat("ab", 32)
		case "eth_getTransactionReceipt":
			result = map[string]string{"status": "0x1"}
		case "eth_sendTransaction":
			t.Error("local signing called the node's unlocked account API")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}))
	defer server.Close()
	key := filepath.Join(t.TempDir(), "private")
	if err := os.WriteFile(key, []byte(strings.Repeat("0", 63)+"1"), 0o600); err != nil {
		t.Fatal(err)
	}
	engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", ArtifactRoot: t.TempDir(), Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1, Role: node.RoleEN, RPCURL: server.URL, Auth: node.Auth{"type": "jwt", "env": "CHAINBENCH_TEST_SIGNING_TOKEN"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "authenticated-signer", "requires": []string{"rpc"}, "chainPreset": map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "external", "chain": "stablenet", "attach": map[string]any{"rpc": []string{server.URL}}, "accounts": map[string]any{"payer": map[string]any{"keyFile": key}}}, "steps": []map[string]any{{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "1"}, {"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "1"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dsl.Parse(raw); err != nil {
		t.Fatalf("signing fixture must reach execution: %v", err)
	}
	root, err := engine.Run(context.Background(), [][]byte{raw})
	if err != nil {
		t.Fatal(err)
	}
	status, err := os.ReadFile(filepath.Join(root, "tests", "001_authenticated-signer", "status.json"))
	if err != nil {
		t.Fatal(err)
	}
	var verdict struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(status, &verdict); err != nil || verdict.Result != "pass" || signed.Load() != 1 || unauthenticated.Load() != 0 {
		t.Fatalf("profile authentication was lost during local signing: %v %s signed=%d unauthenticated=%d", err, status, signed.Load(), unauthenticated.Load())
	}
}
