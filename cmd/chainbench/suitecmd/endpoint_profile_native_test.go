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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/mcp"
)

// Native nodes supply every RPC, WS notification, and metric. The gateway only
// enforces authentication and forwards bytes; it never fabricates chain data.
func TestNativeExternalEndpointProfilesCLIAndMCP(t *testing.T) {
	nativeExternalEndpointProfiles(t, "stablenet", "GSTABLE_BIN")
}

func TestNativeExternalEndpointProfilesWBFTCLIAndMCP(t *testing.T) {
	nativeExternalEndpointProfiles(t, "wbft", "GWBFT_BIN")
}

func TestNativeExternalEndpointProfilesWEMIXCLIAndMCP(t *testing.T) {
	nativeExternalEndpointProfiles(t, "wemix", "GWEMIX_BIN")
}

func nativeExternalEndpointProfiles(t *testing.T, chain, binaryEnv string) {
	binary := os.Getenv(binaryEnv)
	if binary == "" {
		t.Skip("native profile proof requires " + binaryEnv)
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
	up, err := app.ChainUp(ctx, app.Deps{}, app.ChainUpIn{DataDir: workspace, Chain: chain, Binary: binary, KeysDir: repoKeys(t), BPCount: 4, ENCount: 1, GenesisSet: []string{"applepieBlock=0"}})
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
	genesisBytes, err := os.ReadFile(status.State.GenesisPath)
	if err != nil {
		t.Fatal(err)
	}
	var genesis struct {
		Config struct {
			ApplepieBlock *int64 `json:"applepieBlock"`
		} `json:"config"`
	}
	if err := json.Unmarshal(genesisBytes, &genesis); err != nil || genesis.Config.ApplepieBlock == nil || *genesis.Config.ApplepieBlock != 0 {
		t.Fatal("native fee delegation requires declared Applepie activation at block zero", err)
	}
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
	var estimateMu sync.Mutex
	estimates := []map[string]any{}
	rejections := []map[string]any{}
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer native-test-token-only" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		calls.Add(1)
		var creationEstimate map[string]any
		observeSubmission := false
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				http.Error(w, "unreadable request", 400)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			var request struct {
				Method string            `json:"method"`
				Params []json.RawMessage `json:"params"`
			}
			if err := json.Unmarshal(body, &request); err != nil {
				t.Error(err)
				return
			}
			if request.Method == "eth_sendRawTransaction" {
				rawTransactions.Add(1)
				observeSubmission = true
			}
			if request.Method == "eth_sendTransaction" {
				t.Error("external signing fell back to a node's unlocked account")
			}
			if request.Method == "eth_estimateGas" && len(request.Params) == 1 {
				var args map[string]any
				if err := json.Unmarshal(request.Params[0], &args); err != nil {
					t.Error(err)
				}
				if args["data"] == "0x600a600c600039600a6000f3602a60005260206000f3" || args["type"] == "0x16" || (args["data"] == "0xab" && args["accessList"] == nil) {
					creationEstimate = args
				}
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
		if creationEstimate != nil {
			proxy.ModifyResponse = func(response *http.Response) error {
				gas, err := inspectNativeEstimate(response)
				if err != nil {
					return err
				}
				estimateMu.Lock()
				estimates = append(estimates, map[string]any{"request": creationEstimate, "gas": gas})
				estimateMu.Unlock()
				return nil
			}
		}
		if observeSubmission {
			proxy.ModifyResponse = func(response *http.Response) error {
				var reply map[string]any
				if err := inspectNativeResponse(response, &reply); err != nil {
					return err
				}
				if rejection, ok := reply["error"].(map[string]any); ok {
					estimateMu.Lock()
					rejections = append(rejections, map[string]any{"method": "eth_sendRawTransaction", "statusCode": response.StatusCode, "error": rejection, "result": reply["result"]})
					estimateMu.Unlock()
				}
				return nil
			}
		}
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
	signedSpec["chainPreset"].(map[string]any)["chain"] = chain
	signedSpec["chainPreset"].(map[string]any)["accounts"] = map[string]any{"payer": map[string]any{"keyFile": filepath.Join(repoKeys(t), "node1/private")}}
	signedSpec["chainPreset"].(map[string]any)["accounts"].(map[string]any)["sponsor"] = map[string]any{"keyFile": filepath.Join(repoKeys(t), "node2/private")}
	signedSpec["steps"] = append(signedSpec["steps"].([]any),
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "1"},
		map[string]any{"do": "faucet", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "amount": "1"},
		map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "deployNonce"},
		map[string]any{"do": "deployContract", "on": "en1", "from": "payer", "bytecode": "0x600a600c600039600a6000f3602a60005260206000f3", "nonce": "$deployNonce", "save": "contract"},
		map[string]any{"expect": "codeAt", "on": "en1", "address": "$contract", "compare": "NotEqual", "is": "0x"},
		map[string]any{"do": "registerContract", "on": "en1", "from": "payer", "to": "$contract", "data": "0x00", "gas": "200000"},
		map[string]any{"do": "load", "on": "en1", "from": "payer", "gas": "300000", "blocks": 2},
		map[string]any{"expect": "balanceAt", "on": "en1", "address": "0x000000000000000000000000000000000000dead", "compare": "GreaterOrEqual", "is": "1"},
		map[string]any{"do": "read", "on": "en1", "source": "gasPrice", "save": "legacyPrice"},
		map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "legacyNonce"},
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "2", "data": "0xab", "gas": "70000", "nonce": "$legacyNonce", "gasPrice": "$legacyPrice"},
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "2", "data": "0xab", "gas": "70000", "gasPrice": "$legacyPrice", "accessList": []any{map[string]any{"address": "payer", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}},
		map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "legacyDeployNonce"},
		map[string]any{"do": "deployContract", "on": "en1", "from": "payer", "bytecode": "0x600a600c600039600a6000f3602a60005260206000f3", "nonce": "$legacyDeployNonce", "gasPrice": "$legacyPrice", "save": "legacyContract"},
		map[string]any{"expect": "codeAt", "on": "en1", "address": "$legacyContract", "compare": "NotEqual", "is": "0x"},
		map[string]any{"do": "registerContract", "on": "en1", "from": "payer", "to": "$legacyContract", "data": "0x00", "gas": "200000", "gasPrice": "$legacyPrice"},
		map[string]any{"do": "load", "on": "en1", "from": "payer", "gas": "300000", "blocks": 2, "gasPrice": "$legacyPrice"},
		map[string]any{"do": "read", "on": "en1", "source": "gasPrice", "save": "listPrice"},
		map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "listNonce"},
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "2", "data": "0xab", "gas": "70000", "nonce": "$listNonce", "maxFeePerGas": "$listPrice", "maxPriorityFeePerGas": "$listPrice", "accessList": []any{map[string]any{"address": "payer", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}})
	signedSpec["steps"] = append(signedSpec["steps"].([]any),
		map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "feeNonce"},
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "feePayer": "sponsor", "to": "0x000000000000000000000000000000000000dead", "value": "2", "data": "0xab", "gas": "70000", "nonce": "$feeNonce", "maxFeePerGas": "$listPrice", "maxPriorityFeePerGas": "$listPrice", "accessList": []any{map[string]any{"address": "payer", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}},
		map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "feeAutoNonce"},
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "feePayer": "sponsor", "to": "0x000000000000000000000000000000000000dead", "value": "2", "data": "0xab", "nonce": "$feeAutoNonce", "maxFeePerGas": "$listPrice", "maxPriorityFeePerGas": "$listPrice", "accessList": []any{map[string]any{"address": "payer", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}})
	signedSpec["steps"] = append(signedSpec["steps"].([]any),
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "2", "gas": "70000", "nonce": "0", "maxFeePerGas": "$listPrice", "maxPriorityFeePerGas": "$listPrice", "expect": "reject", "reason": "nonce"},
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "2", "gas": "10000", "maxFeePerGas": "$listPrice", "maxPriorityFeePerGas": "$listPrice", "expect": "reject", "reason": "intrinsic gas"})
	signedSpec["steps"] = append(signedSpec["steps"].([]any),
		map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "queuedBase"},
		map[string]any{"do": "read", "source": "derive", "op": "sum", "of": []any{"$queuedBase", "100"}, "save": "queuedNonce"},
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "2", "data": "0xab", "gas": "70000", "nonce": "$queuedNonce", "maxFeePerGas": "$listPrice", "maxPriorityFeePerGas": "$listPrice", "accessList": []any{map[string]any{"address": "payer", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}, "expect": "keptOut", "blocks": 2, "timeout": "15s", "pollInterval": "100ms"})
	signedSpec["steps"] = append(signedSpec["steps"].([]any),
		map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "calldataNonce"},
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "2", "data": "0xab", "nonce": "$calldataNonce", "maxFeePerGas": "$listPrice", "maxPriorityFeePerGas": "$listPrice"},
		map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "legacyCalldataNonce"},
		map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "2", "data": "0xab", "nonce": "$legacyCalldataNonce", "gasPrice": "$listPrice"})
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
	wantRaw := int32(40)
	wantSend := 9
	if rawTransactions.Load() != wantRaw {
		t.Fatalf("wrong number of local signatures: %d, want %d", rawTransactions.Load(), wantRaw)
	}
	observedTransactions := []map[string]any{}
	observedDeployments := []map[string]any{}
	estimateMu.Lock()
	recordedEstimates := append([]map[string]any(nil), estimates...)
	estimateMu.Unlock()
	frozenRejections := append([]map[string]any(nil), rejections...)
	if len(frozenRejections) != 4 {
		t.Fatalf("expected four genuine native submit rejections: %+v", frozenRejections)
	}
	for index, rejection := range frozenRejections {
		err := rejection["error"].(map[string]any)
		if rejection["statusCode"] != http.StatusOK || rejection["result"] != nil || err["code"] != float64(-32000) || !strings.Contains(strings.ToLower(err["message"].(string)), []string{"nonce", "intrinsic gas"}[index%2]) {
			t.Fatalf("wrong native rejection provenance: %+v", rejection)
		}
	}
	frozenEstimates := []map[string]any{}
	feeEstimates := []map[string]any{}
	transferEstimates := []map[string]any{}
	for _, estimate := range recordedEstimates {
		if estimate["request"].(map[string]any)["type"] == "0x16" {
			feeEstimates = append(feeEstimates, estimate)
		} else if args := estimate["request"].(map[string]any); args["data"] == "0xab" && args["accessList"] == nil {
			transferEstimates = append(transferEstimates, estimate)
		} else {
			frozenEstimates = append(frozenEstimates, estimate)
		}
	}
	if len(transferEstimates) != 4 {
		t.Fatalf("expected four actual calldata estimates: %+v", transferEstimates)
	}
	if len(feeEstimates) != 2 {
		t.Fatalf("expected CLI and MCP fee-delegated estimates: %+v", feeEstimates)
	}
	if len(frozenEstimates) != 4 {
		t.Fatalf("expected four actual deployment estimates: %+v", frozenEstimates)
	}
	for sessionIndex, root := range []string{cli.Session, filepath.Join(mcpArtifacts, ids[0])} {
		files, err := filepath.Glob(filepath.Join(root, "tests", "*", "steps.json"))
		if err != nil || len(files) != 1 {
			t.Fatal(err, files)
		}
		content, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatal(err)
		}
		var steps []struct{ Type, Hash, Error string }
		if err := json.Unmarshal(content, &steps); err != nil {
			t.Fatal(err)
		}
		index := 0
		rejectedSteps := 0
		for _, step := range steps {
			if step.Type == "deployContract" {
				var tx map[string]any
				if err := rpc.Dial(endpoint.RPCURL).Call(ctx, "eth_getTransactionByHash", &tx, step.Hash); err != nil {
					t.Fatal(err)
				}
				index := len(observedDeployments)
				if index >= len(frozenEstimates) {
					t.Fatal("unexpected extra deployment")
				}
				args := frozenEstimates[index]["request"].(map[string]any)
				if tx["gas"] != frozenEstimates[index]["gas"] || tx["nonce"] != args["nonce"] || tx["input"] != args["data"] || tx["from"] != args["from"] {
					t.Fatalf("deployment changed estimated transaction: estimate=%+v tx=%+v", frozenEstimates[index], tx)
				}
				if index%2 == 1 && (tx["type"] != "0x0" || tx["gasPrice"] != args["gasPrice"]) {
					t.Fatalf("deployment lost legacy price: %+v", tx)
				}
				if index%2 == 0 && tx["type"] != "0x2" {
					t.Fatalf("nonce-only deployment changed fee model: %+v", tx)
				}
				observedDeployments = append(observedDeployments, tx)
			}
			if step.Type != "sendTx" {
				continue
			}
			if step.Hash == "" {
				if step.Error != "" {
					t.Fatalf("rejection step failed: %+v", step)
				}
				rejectedSteps++
				continue
			}
			var tx map[string]any
			if err := rpc.Dial(endpoint.RPCURL).Call(ctx, "eth_getTransactionByHash", &tx, step.Hash); err != nil {
				t.Fatal(err)
			}
			if index >= wantSend || tx["type"] != []string{"0x2", "0x0", "0x1", "0x2", "0x16", "0x16", "0x2", "0x2", "0x0"}[index] {
				t.Fatalf("wrong native transaction type: %+v", tx)
			}
			if index > 0 && (tx["input"] != "0xab" || (index != 5 && index < 7 && tx["gas"] != "0x11170") || tx["value"] != "0x2") {
				t.Fatalf("legacy payload/gas/value changed: %+v", tx)
			}
			if index >= 2 && index <= 6 {
				list, ok := tx["accessList"].([]any)
				if !ok || len(list) != 1 {
					t.Fatalf("access list lost: %+v", tx)
				}
				tuple, ok := list[0].(map[string]any)
				if !ok {
					t.Fatal(list)
				}
				keys, ok := tuple["storageKeys"].([]any)
				if !ok || len(keys) != 1 || keys[0] != "0x"+strings.Repeat("0", 63)+"1" || tuple["address"] != tx["from"] {
					t.Fatalf("access tuple changed: %+v", tuple)
				}
			}
			if index >= 4 && index <= 5 {
				key, err := os.ReadFile(filepath.Join(repoKeys(t), "node2/private"))
				if err != nil {
					t.Fatal(err)
				}
				private, err := hex.DecodeString(strings.TrimSpace(string(key)))
				if err != nil {
					t.Fatal(err)
				}
				address, err := accounts.AddressForKey(private)
				if err != nil {
					t.Fatal(err)
				}
				if tx["feePayer"] != address {
					t.Fatalf("wrong signed fee payer: %+v", tx)
				}
				if index == 5 {
					estimate := feeEstimates[sessionIndex]
					args := estimate["request"].(map[string]any)
					for _, field := range []string{"from", "to", "value", "nonce", "feePayer", "maxFeePerGas", "maxPriorityFeePerGas"} {
						if tx[field] != args[field] {
							t.Fatalf("fee-delegated estimate changed %s: estimate=%+v tx=%+v", field, estimate, tx)
						}
					}
					if tx["gas"] != estimate["gas"] || tx["input"] != args["data"] {
						t.Fatalf("fee-delegated estimate changed gas or data: estimate=%+v tx=%+v", estimate, tx)
					}
				}
			}
			if index == 6 {
				var receipt map[string]any
				if err := rpc.Dial(endpoint.RPCURL).Call(ctx, "eth_getTransactionReceipt", &receipt, step.Hash); err != nil || receipt != nil || tx["blockHash"] != nil || tx["blockNumber"] != nil {
					t.Fatalf("queued transaction did not remain authoritatively absent: %+v receipt=%+v error=%v", tx, receipt, err)
				}
			}
			if index >= 7 {
				estimate := transferEstimates[sessionIndex*2+index-7]
				args := estimate["request"].(map[string]any)
				for _, field := range []string{"from", "to", "value", "nonce", "gasPrice", "maxFeePerGas", "maxPriorityFeePerGas"} {
					if expected, supplied := args[field]; supplied && tx[field] != expected {
						t.Fatalf("calldata estimate changed %s: estimate=%+v tx=%+v", field, estimate, tx)
					}
				}
				if tx["gas"] != estimate["gas"] || tx["input"] != args["data"] {
					t.Fatalf("calldata estimate changed gas/data: estimate=%+v tx=%+v", estimate, tx)
				}
			}
			observedTransactions = append(observedTransactions, tx)
			index++
		}
		if rejectedSteps != 2 {
			t.Fatalf("missing successful rejection steps: %d", rejectedSteps)
		}
		if index != wantSend {
			t.Fatalf("missing native transaction types: %d", index)
		}
	}
	if len(observedDeployments) != 4 {
		t.Fatalf("missing observed deployments: %d", len(observedDeployments))
	}
	sources := map[string]string{}
	invalidNumericRuns := requireNativeInvalidNumericRuns(t, fixture, signedSpec["chainPreset"], &rawTransactions)
	delegatedCreations := requireNativeDelegatedCreations(t, ctx, fixture, signedSpec["chainPreset"], endpoint.RPCURL, &rawTransactions, func() []map[string]any {
		estimateMu.Lock()
		defer estimateMu.Unlock()
		return append([]map[string]any(nil), estimates...)
	})
	presetValueRuns := requireNativePresetValues(t, ctx, fixture, chain, endpoint.RPCURL, signedSpec["chainPreset"], &rawTransactions)
	identityRuns := requireNativeChainIdentity(t, ctx, fixture, chain, signedSpec["chainPreset"], up.Nodes.Nodes, &rawTransactions)
	capabilityRuns := requireNativeDeclaredCapabilities(t, ctx, fixture, signedSpec["chainPreset"], &rawTransactions)
	assetSources := []string{"../../../internal/testengine/capability.go", "../../../internal/testengine/engine_impl.go", "../../../internal/testengine/capability_reason_test.go", "../../../internal/testengine/chain_identity.go", "../../../internal/testengine/chain_identity_test.go", "../../../internal/app/chain_identity_schema_test.go", "../../../internal/dsl/preset_values.go", "../../../internal/dsl/preset_values_test.go", "../../../internal/dsl/spec.go", "../../../internal/dsl/parse_v2.go", "../../../internal/dsl/interp/binding.go", "../../../internal/dsl/interp/run.go", "../../../internal/dsl/interp/resolve.go", "../../../internal/dsl/interp/preset_values_test.go", "../../../internal/app/preset_values_test.go", "../../../internal/app/web_document_secrets.go", "../../../internal/testhelper/assets.go", "../../../internal/testhelper/load.go", "../../../internal/testhelper/local_assets.go", "../../../internal/testhelper/sendtx.go", "../../../internal/testhelper/arguments.go", "../../../internal/testhelper/transaction_numbers.go", "../../../internal/testhelper/transaction_numbers_test.go", "../../../internal/testhelper/editor_contract.go", "../../../internal/app/testcase_transaction_numbers_test.go", "../../../internal/app/testcase_delegation_test.go", "../../../internal/testhelper/delegated_creation_test.go", "../../../internal/testhelper/delegated.go", "../../../internal/testhelper/outcome.go", "../../../internal/core/rpc/client.go"}
	for _, name := range assetSources {
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(content)
		sources[name] = hex.EncodeToString(sum[:])
	}
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
	proof["chain"] = chain
	proof["delegatedCreations"] = delegatedCreations
	proof["presetValueRuns"] = presetValueRuns
	proof["identityRuns"] = identityRuns
	proof["capabilityRuns"] = capabilityRuns
	proof["invalidNumericRuns"] = invalidNumericRuns
	proof["submissionRejections"] = frozenRejections
	proof["applepieBlock"] = *genesis.Config.ApplepieBlock
	proof["observedTransactions"] = observedTransactions
	proof["deploymentEstimates"] = frozenEstimates
	proof["feeDelegatedEstimates"] = feeEstimates
	proof["calldataEstimates"] = transferEstimates
	proof["observedDeployments"] = observedDeployments
	bytes, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "proof.json"), bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("native endpoint proof: %s", filepath.Join(fixture, "proof.json"))
}
