package suitecmd_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/mcp"
)

// Native nodes supply every RPC, WS notification, and metric. The gateway only
// enforces authentication and forwards bytes; it never fabricates chain data.
func TestNativeExternalEndpointProfilesCLIAndMCP(t *testing.T) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("native profile proof requires GSTABLE_BIN")
	}
	root := os.Getenv("WEBUI_RUNTIME_ROOT")
	if root == "" {
		root = t.TempDir()
	}
	fixture, err := os.MkdirTemp(root, "endpoint-native-")
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(fixture, "w")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if _, err := app.ChainStop(ctx, app.Deps{}, app.ChainStopIn{DataDir: workspace}); err != nil {
			t.Error(err)
			return
		}
		if _, err := app.ChainRm(ctx, app.Deps{}, app.ChainRmIn{DataDir: workspace}); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	up, err := app.ChainUp(ctx, app.Deps{}, app.ChainUpIn{DataDir: workspace, Chain: "stablenet", Binary: binary, KeysDir: repoKeys(t), BPCount: 4, ENCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(up.Nodes.Nodes.Nodes) != 5 {
		t.Fatal("native profile requires actual bp4/en1")
	}
	status, err := app.ChainStatus(ctx, app.Deps{}, app.ChainStatusIn{DataDir: workspace})
	if err != nil {
		t.Fatal(err)
	}
	// Preserve public inputs and node logs before removal; never copy private keys.
	t.Cleanup(func() {
		diagnostics := filepath.Join(fixture, "diagnostics")
		if err := os.MkdirAll(diagnostics, 0o700); err != nil {
			t.Error(err)
			return
		}
		files := []string{filepath.Join(workspace, "chain-record.json")}
		for _, n := range status.State.Nodes {
			files = append(files, n.ConfigPath, n.LogPath)
		}
		files = append(files, status.State.GenesisPath)
		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				t.Errorf("preserve diagnostic %s: %v", file, err)
				continue
			}
			if err := os.WriteFile(filepath.Join(diagnostics, filepath.Base(file)), content, 0o600); err != nil {
				t.Error(err)
			}
		}
	})
	endpoint := up.Nodes.Nodes.Nodes[4]
	if endpoint.Role != node.RoleEN || endpoint.WSURL == "" || endpoint.MetricsURL == "" {
		t.Fatal("native endpoint lacks recorded transports")
	}
	for {
		head, err := rpc.Dial(endpoint.RPCURL).BlockNumber(ctx)
		if err == nil && head >= 2 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("native endpoint did not advance")
		case <-time.After(time.Second):
		}
	}
	var calls, rawTransactions atomic.Int32
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer native-test-token-only" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		calls.Add(1)
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				http.Error(w, "unreadable request", 400)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			var request struct {
				Method string `json:"method"`
			}
			if err := json.Unmarshal(body, &request); err != nil {
				t.Error(err)
				return
			}
			if request.Method == "eth_sendRawTransaction" {
				rawTransactions.Add(1)
			}
			if request.Method == "eth_sendTransaction" {
				t.Error("external signing fell back to a node's unlocked account")
			}
		}
		target := endpoint.RPCURL
		if r.URL.Path == "/ws" {
			target = strings.Replace(endpoint.WSURL, "ws://", "http://", 1)
		}
		if r.URL.Path == "/metrics" {
			target = endpoint.MetricsURL
		}
		parsed, err := url.Parse(target)
		if err != nil {
			t.Error(err)
			http.Error(w, "bad target", 500)
			return
		}
		proxy := &httputil.ReverseProxy{Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(parsed)
			request.Out.URL.Path = parsed.Path
			request.Out.Host = parsed.Host
			request.Out.Header.Del("Authorization")
		}}
		proxy.ServeHTTP(w, r)
	}))
	defer gateway.Close()
	t.Setenv("CHAINBENCH_NATIVE_PROFILE_RPC", gateway.URL)
	t.Setenv("CHAINBENCH_NATIVE_PROFILE_TOKEN", "native-test-token-only")
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"native-profile","requires":["rpc"],"chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"external","chain":"stablenet","attach":{"endpoints":[{"rpc":"${CHAINBENCH_NATIVE_PROFILE_RPC}","role":"en","ws":"` + strings.Replace(gateway.URL, "http://", "ws://", 1) + `/ws","metrics":"` + gateway.URL + `/metrics","auth":{"type":"jwt","env":"CHAINBENCH_NATIVE_PROFILE_TOKEN"},"apis":["eth","net","istanbul"]}]}},"steps":[{"expect":"blockNumber","on":"en1","compare":"GreaterOrEqual","is":"2"},{"expect":"metric","on":"en1","name":"chain_head_block","compare":"GreaterOrEqual","is":2},{"expect":"wsSubscribe","on":"en1","event":"newHeads","count":1,"timeout":"15s"}]}`)
	var signedSpec map[string]any
	if err := json.Unmarshal(raw, &signedSpec); err != nil {
		t.Fatal(err)
	}
	signedSpec["chainPreset"].(map[string]any)["accounts"] = map[string]any{"payer": map[string]any{"keyFile": filepath.Join(repoKeys(t), "node1/private")}}
	signedSpec["steps"] = append(signedSpec["steps"].([]any),
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "1"},
		map[string]any{"expect": "balanceAt", "on": "en1", "address": "0x000000000000000000000000000000000000dead", "compare": "GreaterOrEqual", "is": "1"})
	raw, err = json.Marshal(signedSpec)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(fixture, "case.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	output, progress, err := runSplit(t, "run", "--json", "--no-skips", "--artifact-root", filepath.Join(fixture, "cli-reports"), path)
	if err != nil {
		t.Fatal(err, output, progress)
	}
	var cli struct {
		Session string `json:"session"`
	}
	if err := json.Unmarshal([]byte(output), &cli); err != nil {
		t.Fatal(err)
	}
	requireProfilePass(t, cli.Session)
	mcpArtifacts := filepath.Join(fixture, "mcp-reports")
	request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"spec": string(raw), "artifactRoot": mcpArtifacts}}})
	response := mcp.Default("profile", "native").Handle(ctx, request)
	var reply struct {
		Result struct {
			IsError bool `json:"isError"`
		}
		Error any
	}
	if err := json.Unmarshal(response, &reply); err != nil || reply.Error != nil || reply.Result.IsError {
		t.Fatal(err, string(response))
	}
	ids, err := session.List(mcpArtifacts)
	if err != nil || len(ids) != 1 {
		t.Fatal(err, ids)
	}
	requireProfilePass(t, filepath.Join(mcpArtifacts, ids[0]))
	if rawTransactions.Load() != 2 {
		t.Fatalf("CLI and MCP must each submit one local signature: %d", rawTransactions.Load())
	}
	sources := map[string]string{}
	for _, pattern := range []string{"../../../internal/app/workflow.go", "../../../internal/accounts/provider.go", "../../../internal/accounts/wallet*.go", "../../../internal/core/node/*.go", "../../../internal/core/remote/auth.go", "../../../internal/core/rpc/subscribe.go", "../../../internal/core/collector/metrics.go", "../../../internal/dsl/endpoint_profile_test.go", "../../../internal/dsl/spec_v2.go", "../../../internal/dsl/lower_v1.go", "../../../internal/dsl/schema/v2.schema.json", "../../../internal/dsl/interp/interpreter.go", "../../../internal/testengine/attach*.go", "../../../internal/testhelper/metric.go", "../../../internal/testhelper/websocket.go", "run.go", "endpoint_profile*_test.go"} {
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range files {
			bytes, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(bytes)
			sources[name] = hex.EncodeToString(sum[:])
		}
	}
	content, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	binarySHA := sha256.Sum256(content)
	proof := map[string]any{"binary": binary, "binarySHA256": hex.EncodeToString(binarySHA[:]), "sources": sources, "actualEndpoint": endpoint, "cliSession": cli.Session, "mcpSession": filepath.Join(mcpArtifacts, ids[0]), "authenticatedGatewayRequests": calls.Load(), "authenticatedRawTransactions": rawTransactions.Load(), "gateway": "auth and byte forwarding only"}
	bytes, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "proof.json"), bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("native endpoint proof: %s", filepath.Join(fixture, "proof.json"))
}
