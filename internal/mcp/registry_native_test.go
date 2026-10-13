package mcp_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/mcp"
)

func nativeCall(ctx context.Context, server *mcp.Server, name string, args map[string]any) (string, error) {
	list, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	if err != nil {
		return "", err
	}
	var catalogue struct {
		Result struct {
			Tools []struct {
				Name  string `json:"name"`
				Input struct {
					Properties map[string]any `json:"properties"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(server.Handle(ctx, list), &catalogue); err != nil {
		return "", err
	}
	known := false
	for _, tool := range catalogue.Result.Tools {
		if tool.Name != name {
			continue
		}
		known = true
		for key := range args {
			if _, ok := tool.Input.Properties[key]; !ok {
				return "", fmt.Errorf("native fixture names unadvertised argument %s for %s", key, name)
			}
		}
	}
	if !known {
		return "", fmt.Errorf("native fixture names unknown tool %s", name)
	}
	raw, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
	if err != nil {
		return "", err
	}
	var response struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(server.Handle(ctx, raw), &response); err != nil {
		return "", err
	}
	if response.Error != nil || len(response.Result.Content) != 1 {
		return "", fmt.Errorf("unexpected MCP response for %s", name)
	}
	text := response.Result.Content[0].Text
	if response.Result.IsError {
		return text, fmt.Errorf("%s: %s", name, text)
	}
	return text, nil
}

func TestMCPNativeRegistryComposition(t *testing.T) {
	for _, sample := range []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}} {
		t.Run(sample.chain, func(t *testing.T) {
			binary := os.Getenv(sample.variable)
			if binary == "" {
				t.Skip("native MCP registry gate requires " + sample.variable)
			}
			bytes, err := os.ReadFile(binary)
			if err != nil {
				t.Fatal(err)
			}
			binarySHA := sha256.Sum256(bytes)
			keys, err := filepath.Abs(filepath.Join("..", "..", "presets", "keys"))
			if err != nil {
				t.Fatal(err)
			}
			root := os.Getenv("WEBUI_RUNTIME_ROOT")
			if root == "" {
				root = t.TempDir()
			}
			fixture, err := os.MkdirTemp(root, "mcp-native-")
			if err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(fixture, "workspace")
			server := newServer()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				for _, name := range []string{"chainbench_chain_stop", "chainbench_chain_rm"} {
					if _, err := nativeCall(ctx, server, name, map[string]any{"workspaceDir": workspace}); err != nil {
						t.Errorf("fixture cleanup: %v", err)
						return
					}
				}
			})
			setup, err := nativeCall(ctx, server, "chainbench_chain_up", map[string]any{"workspaceDir": workspace, "chain": sample.chain, "binary": binary, "keysDir": keys, "bp": 4})
			if err != nil {
				t.Fatal(err)
			}
			observed, err := nativeCall(ctx, server, "chainbench_chain_observe", map[string]any{"workspaceDir": workspace})
			if err != nil {
				t.Fatal(err)
			}
			// The oracle checks actual recorded processes, independently of the
			// MCP renderer's prose. Both must describe the four native nodes.
			state, err := app.NetworkStatus(ctx, app.Deps{}, app.NetworkStatusIn{DataDir: workspace})
			if err != nil {
				t.Fatal(err)
			}
			if len(state.Nodes.Nodes) != 4 {
				t.Fatal("MCP composed the wrong number of nodes")
			}
			for _, node := range state.Nodes.Nodes {
				if !state.Alive[node.Index] {
					t.Fatalf("native node %d is not live", node.Index)
				}
			}
			deadline := time.Now().Add(time.Minute)
			var status string
			for {
				status, err = nativeCall(ctx, server, "chainbench_consensus_status", map[string]any{"chain": sample.chain, "rpc": state.Nodes.Nodes[0].RPCURL})
				if err == nil && strings.Contains(status, "peers=3") && !strings.Contains(status, "head=0 ") {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("MCP did not observe native production: %v %s", err, status)
				}
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-time.After(time.Second):
				}
			}
			sources := map[string]string{}
			files, err := filepath.Glob("*.go")
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range files {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(content)
				sources[path] = hex.EncodeToString(sum[:])
			}
			record := map[string]any{"chain": sample.chain, "binary": binary, "binarySHA256": hex.EncodeToString(binarySHA[:]), "sourceFiles": sources, "setup": setup, "observations": json.RawMessage(observed), "status": status, "nodes": state.Nodes}
			raw, err := json.MarshalIndent(record, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			proof := filepath.Join(fixture, "proof.json")
			if err := os.WriteFile(proof, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Logf("native MCP registry proof: %s", proof)
		})
	}
}
