package suitecmd_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/mcp"
)

type blacklistObserver struct {
	base       http.RoundTripper
	mu         sync.Mutex
	methods    map[string]int
	rejections []map[string]any
}

func (o *blacklistObserver) snapshot() (map[string]int, []map[string]any) {
	o.mu.Lock()
	defer o.mu.Unlock()
	methods := make(map[string]int, len(o.methods))
	for method, count := range o.methods {
		methods[method] = count
	}
	return methods, append([]map[string]any(nil), o.rejections...)
}

func (o *blacklistObserver) RoundTrip(request *http.Request) (*http.Response, error) {
	method := ""
	forward := request
	if request.Method == http.MethodPost && request.Body != nil {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		if err := request.Body.Close(); err != nil {
			return nil, err
		}
		forward = request.Clone(request.Context())
		forward.Body = io.NopCloser(bytes.NewReader(body))
		var call struct {
			Method string `json:"method"`
		}
		if json.Unmarshal(body, &call) == nil {
			method = call.Method
		}
	}
	response, err := o.base.RoundTrip(forward)
	if method != "" {
		o.mu.Lock()
		o.methods[method]++
		o.mu.Unlock()
	}
	if err != nil {
		return nil, err
	}
	if method == "eth_sendRawTransaction" || method == "eth_sendTransaction" {
		var reply map[string]any
		if err := inspectNativeResponse(response, &reply); err != nil {
			_ = response.Body.Close()
			return nil, err
		}
		if rejected, ok := reply["error"].(map[string]any); ok {
			o.mu.Lock()
			o.rejections = append(o.rejections, map[string]any{"method": method, "statusCode": response.StatusCode, "result": reply["result"], "error": rejected})
			o.mu.Unlock()
		}
	}
	return response, nil
}

func TestNativeBlacklistRejectionsCLIAndMCP(t *testing.T) {
	for _, name := range []string{"01-sender-blacklisted-rejected.json", "02-recipient-blacklisted-rejected.json", "03-feepayer-blacklisted-rejected.json"} {
		t.Run(name, func(t *testing.T) { nativeBlacklistRejection(t, name) })
	}
}

func nativeBlacklistRejection(t *testing.T, filename string) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("native blacklist proof requires GSTABLE_BIN")
	}
	root := os.Getenv("WEBUI_RUNTIME_ROOT")
	if root == "" {
		root = t.TempDir()
	}
	fixture, err := os.MkdirTemp(root, "blacklist-native-")
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(fixture, "w")
	t.Cleanup(func() {
		if _, err := app.ChainStop(context.Background(), app.Deps{}, app.ChainStopIn{DataDir: workspace}); err != nil {
			t.Error(err)
			return
		}
		diagnostics := filepath.Join(fixture, "diagnostics")
		if err := os.MkdirAll(diagnostics, 0700); err != nil {
			t.Error(err)
			return
		}
		for _, pattern := range []string{"chain-record.json", "config*.toml", "genesis*.json", "*.log"} {
			files, _ := filepath.Glob(filepath.Join(workspace, pattern))
			for _, p := range files {
				content, err := os.ReadFile(p)
				if err != nil {
					t.Error(err)
					continue
				}
				if err := os.WriteFile(filepath.Join(diagnostics, filepath.Base(p)), content, 0600); err != nil {
					t.Error(err)
				}
			}
		}
		if _, err := app.ChainRm(context.Background(), app.Deps{}, app.ChainRmIn{DataDir: workspace}); err != nil {
			t.Error(err)
		}
	})
	originalTransport := http.DefaultTransport
	observer := &blacklistObserver{base: originalTransport, methods: map[string]int{}}
	http.DefaultTransport = observer
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	path := filepath.Join("../../../tests/tc/go-stablenet/regression/blacklist-authorized", filename)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	output, progress, runErr := runSplit(t, "run", "--workspace-dir", workspace, "--artifact-root", filepath.Join(fixture, "cli"), "--keys", repoKeys(t), "--read-only-keys", "--binary", binary, "--wait-blocks", "1", "--json", "--no-skips", path)
	if err := os.WriteFile(filepath.Join(fixture, "cli.json"), []byte(output), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "cli.log"), []byte(progress), 0600); err != nil {
		t.Fatal(err)
	}
	var cli struct {
		Session string `json:"session"`
	}
	if err := json.Unmarshal([]byte(output), &cli); err != nil {
		t.Fatal(err, output)
	}
	methodsAfterCLI, rejectionsAfterCLI := observer.snapshot()
	observation, err := json.MarshalIndent(map[string]any{"methods": methodsAfterCLI, "submissionRejections": rejectionsAfterCLI}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "cli-requests.json"), observation, 0600); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatalf("original blacklist case failed: %v; session=%s", runErr, cli.Session)
	}
	requireProfilePass(t, cli.Session)
	artifacts := filepath.Join(fixture, "mcp")
	request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"dataDir": workspace, "binary": binary, "keysDir": repoKeys(t), "readOnlyKeys": true, "artifactRoot": artifacts, "spec": string(raw), "waitBlocks": 1}}})
	response := mcp.Default("blacklist", "native").Handle(context.Background(), request)
	if err := os.WriteFile(filepath.Join(fixture, "mcp.json"), response, 0600); err != nil {
		t.Fatal(err)
	}
	var reply struct {
		Result struct {
			IsError bool `json:"isError"`
		}
		Error any
	}
	if err := json.Unmarshal(response, &reply); err != nil || reply.Error != nil || reply.Result.IsError {
		t.Fatal(err, string(response))
	}
	ids, err := session.List(artifacts)
	if err != nil || len(ids) != 1 {
		t.Fatal(err, ids)
	}
	requireProfilePass(t, filepath.Join(artifacts, ids[0]))
	methods, rejections := observer.snapshot()
	if len(rejections) != 2 {
		t.Fatalf("expected two genuine node blacklist rejections, got %+v; methods=%+v", rejections, methods)
	}
	method := "eth_sendRawTransaction"
	if strings.HasPrefix(filename, "02-") {
		method = "eth_sendTransaction"
	}
	for _, r := range rejections {
		e := r["error"].(map[string]any)
		if r["method"] != method || r["statusCode"] != http.StatusOK || r["result"] != nil || e["code"] != float64(-32000) || !strings.Contains(strings.ToLower(fmt.Sprint(e["message"])), "blacklist") {
			t.Fatalf("wrong blacklist rejection provenance: %+v", r)
		}
	}
	sources := map[string]string{}
	for _, name := range []string{path, "blacklist_native_test.go", "endpoint_profile_proxy_test.go", "../../../internal/testhelper/sendtx.go", "../../../internal/testhelper/outcome.go", "../../../internal/testengine/attach.go", "../../../internal/accounts/wallet.go", "../../../internal/accounts/wallet_rpc_send.go", "../../../internal/accounts/wallet_rpc.go", "../../../internal/core/rpc/client.go", "../../../internal/mcp/run_tool.go", "../../../internal/mcp/validate_tool.go", "../../../internal/mcp/spec_presets_test.go", "../../../internal/app/workflow.go", "../../../internal/dsl/files.go", "../../../internal/dsl/parse_v2.go"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		sources[name] = hex.EncodeToString(sum[:])
	}
	content, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	proof, _ := json.MarshalIndent(map[string]any{"case": filename, "binary": binary, "binarySHA256": hex.EncodeToString(sum[:]), "sources": sources, "cliSession": cli.Session, "mcpSession": filepath.Join(artifacts, ids[0]), "methods": methods, "submissionRejections": rejections, "transport": "observes and forwards the original native node bytes; no fabricated chain responses"}, "", "  ")
	if err := os.WriteFile(filepath.Join(fixture, "proof.json"), proof, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("native blacklist proof: %s", filepath.Join(fixture, "proof.json"))
}
