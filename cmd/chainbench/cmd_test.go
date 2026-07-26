package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// run executes the root command with args and returns combined output.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

func TestChainsCmd(t *testing.T) {
	out, err := run(t, "chains")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"stablenet", "wbft", "wemix", "istanbul", "wemix"} {
		if !strings.Contains(out, want) {
			t.Errorf("chains output missing %q:\n%s", want, out)
		}
	}
}

func TestSetupCmd_DryRunPlan(t *testing.T) {
	out, err := run(t, "setup", "--chain", "wbft", "--validators", "3", "--endpoints", "1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "chain_id 8284") {
		t.Errorf("expected wbft chain_id 8284:\n%s", out)
	}
	// 3 validators + 1 endpoint = 4 node rows; check node 4 present.
	if !strings.Contains(out, "\n4\t") && !strings.Contains(out, "4  ") {
		t.Errorf("expected 4 nodes in plan:\n%s", out)
	}
	if !strings.Contains(out, "genesis:  template=true") {
		t.Errorf("wbft should report a genesis template:\n%s", out)
	}
}

func TestSetupCmd_DryRunFalseWithoutLaunch(t *testing.T) {
	_, err := run(t, "setup", "--chain", "stablenet", "--dry-run=false")
	if err == nil || !strings.Contains(err.Error(), "needs --launch") {
		t.Errorf("expected guidance to use --launch, got %v", err)
	}
}

func TestSetupCmd_ProvisionWritesRealGenesis(t *testing.T) {
	dir := t.TempDir()
	out, err := run(t, "setup",
		"--chain", "stablenet",
		"--validators", "4", "--endpoints", "1",
		"--data-dir", dir,
		"--keys-dir", filepath.Join("..", "..", "keys", "preset"),
		"--provision",
	)
	if err != nil {
		t.Fatalf("provision: %v\n%s", err, out)
	}
	if !strings.Contains(out, "provisioned: genesis") {
		t.Errorf("expected provisioned genesis:\n%s", out)
	}
	b, err := os.ReadFile(filepath.Join(dir, "genesis.json"))
	if err != nil {
		t.Fatalf("read genesis: %v", err)
	}
	var g struct {
		Config struct {
			ChainID int64 `json:"chainId"`
			Anzeon  struct {
				Init struct {
					Validators []string `json:"validators"`
				} `json:"init"`
			} `json:"anzeon"`
		} `json:"config"`
	}
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatalf("genesis not valid JSON: %v", err)
	}
	if g.Config.ChainID != 8283 {
		t.Errorf("chainId: %d", g.Config.ChainID)
	}
	// Real preset validators land in the genesis.
	if len(g.Config.Anzeon.Init.Validators) != 4 {
		t.Errorf("validators in genesis: %d, want 4", len(g.Config.Anzeon.Init.Validators))
	}
	if g.Config.Anzeon.Init.Validators[0] != "0xc17d493883eaa3b4cceb0f214b273392d562f9d8" {
		t.Errorf("first validator: %s", g.Config.Anzeon.Init.Validators[0])
	}

	// Per-node TOML configs are written too.
	if !strings.Contains(out, "node config(s)") {
		t.Errorf("expected node configs written:\n%s", out)
	}
	cfg1, err := os.ReadFile(filepath.Join(dir, "config_node1.toml"))
	if err != nil {
		t.Fatalf("read config_node1.toml: %v", err)
	}
	s := string(cfg1)
	if !strings.Contains(s, "[Eth.Miner]") { // node1 is a validator
		t.Errorf("node1 config should have miner section:\n%s", s)
	}
	if !strings.Contains(s, "HTTPPort = 8501") || !strings.Contains(s, `"istanbul"`) {
		t.Errorf("node1 config ports/namespace wrong:\n%s", s)
	}
	if !strings.Contains(s, "enode://8d2153cc") { // static node from preset pubkey
		t.Errorf("node1 config missing static-node enode:\n%s", s)
	}
	// node5 is an endpoint: no miner section.
	cfg5, err := os.ReadFile(filepath.Join(dir, "config_node5.toml"))
	if err != nil {
		t.Fatalf("read config_node5.toml: %v", err)
	}
	if strings.Contains(string(cfg5), "[Eth.Miner]") {
		t.Errorf("node5 (endpoint) should not have miner section")
	}
}

func TestSetupCmd_LaunchWithFakeBinary(t *testing.T) {
	dir := t.TempDir()
	// Fake node binary: `init` exits 0; run sleeps briefly then exits.
	bin := filepath.Join(dir, "fakegeth")
	script := "#!/bin/sh\nif [ \"$1\" = \"init\" ]; then exit 0; fi\nsleep 0.3\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, "setup",
		"--chain", "stablenet",
		"--validators", "2", "--endpoints", "0",
		"--data-dir", filepath.Join(dir, "data"),
		"--keys-dir", filepath.Join("..", "..", "keys", "preset"),
		"--binary", bin,
		"--launch",
	)
	if err != nil {
		t.Fatalf("launch: %v\n%s", err, out)
	}
	if !strings.Contains(out, "launched 2 node(s)") {
		t.Errorf("expected launch confirmation:\n%s", out)
	}
	// NodeSet state persisted.
	b, err := os.ReadFile(filepath.Join(dir, "data", "nodeset.json"))
	if err != nil {
		t.Fatalf("read nodeset.json: %v", err)
	}
	var ns struct {
		Chain string `json:"chain"`
		Nodes []struct {
			Index  int    `json:"index"`
			RPCURL string `json:"rpc_url"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(b, &ns); err != nil {
		t.Fatalf("nodeset.json invalid: %v", err)
	}
	if ns.Chain != "stablenet" || len(ns.Nodes) != 2 {
		t.Errorf("nodeset: %+v", ns)
	}
	if ns.Nodes[0].RPCURL != "http://127.0.0.1:8501" {
		t.Errorf("node1 rpc: %s", ns.Nodes[0].RPCURL)
	}
	// Datadirs were initialized.
	if _, err := os.Stat(filepath.Join(dir, "data", "node1")); err != nil {
		t.Errorf("node1 datadir not created: %v", err)
	}
}

func TestStopCmd_KillsNodes(t *testing.T) {
	// Spawn a real long-lived process to act as a launched node.
	proc := exec.Command("sleep", "30")
	if err := proc.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	pid := proc.Process.Pid

	dir := t.TempDir()
	nsJSON := fmt.Sprintf(`{"chain":"stablenet","network":"local","nodes":[{"index":1,"pid":%d}]}`, pid)
	if err := os.WriteFile(filepath.Join(dir, "nodeset.json"), []byte(nsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, "stop", "--data-dir", dir)
	if err != nil {
		t.Fatalf("stop: %v\n%s", err, out)
	}
	if !strings.Contains(out, "stopped 1 node") {
		t.Errorf("expected stop confirmation:\n%s", out)
	}
	// Reap; Wait should report the process was killed by a signal.
	if werr := proc.Wait(); werr == nil {
		t.Error("process should have been killed by stop")
	}
}

func TestVerifyCmd_ForwardsToDashboard(t *testing.T) {
	// A node returning a stable chain (not producing) so verify completes fast.
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch req.Method {
		case "eth_chainId", "net_peerCount":
			res["result"] = "0x1"
		case "eth_blockNumber":
			res["result"] = "0x5"
		case "eth_syncing":
			res["result"] = false
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer node.Close()

	// A dashboard sink capturing forwarded obs events.
	var mu sync.Mutex
	var count int
	dash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" {
			mu.Lock()
			count++
			mu.Unlock()
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer dash.Close()

	_, err := run(t, "verify", "--rpc", node.URL, "--progress-delay", "1ms", "--dashboard", dash.URL)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if count == 0 {
		t.Error("expected obs events forwarded to the dashboard")
	}
}

func TestVerifyCmd_FromState(t *testing.T) {
	var block int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch req.Method {
		case "eth_blockNumber":
			block++
			res["result"] = "0x" + itoaHex(block)
		case "eth_chainId":
			res["result"] = "0x205b"
		case "net_peerCount":
			res["result"] = "0x1"
		case "eth_syncing":
			res["result"] = false
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	dir := t.TempDir()
	nsJSON := `{"chain":"wbft","network":"local","nodes":[{"index":1,"role":"validator","rpc_url":"` + srv.URL + `"}]}`
	if err := os.WriteFile(filepath.Join(dir, "nodeset.json"), []byte(nsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "verify", "--data-dir", dir, "--progress-delay", "1ms")
	if err != nil {
		t.Fatalf("verify --data-dir: %v\n%s", err, out)
	}
	if !strings.Contains(out, "producing: true") || !strings.Contains(out, "8283") {
		t.Errorf("expected producing + chain id from saved state:\n%s", out)
	}
}

func TestVerifyCmd_HTTPMock(t *testing.T) {
	var block int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch req.Method {
		case "eth_blockNumber":
			block++
			res["result"] = "0x" + strings.TrimPrefix(itoaHex(block), "0x")
		case "eth_chainId":
			res["result"] = "0x205b"
		case "net_peerCount":
			res["result"] = "0x2"
		case "eth_syncing":
			res["result"] = false
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	out, err := run(t, "verify", "--rpc", srv.URL, "--progress-delay", "1ms")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "producing: true") {
		t.Errorf("expected producing:true:\n%s", out)
	}
	if !strings.Contains(out, "8283") {
		t.Errorf("expected chain id 8283:\n%s", out)
	}
}

func TestTestCmd_RunsCasesViaAttach(t *testing.T) {
	// Mock node returning a stable non-zero chain id (chain-id case passes).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x205b"})
	}))
	defer srv.Close()

	out, err := run(t, "test", "--chain", "wbft", "--rpc", srv.URL, "--name", "chain-id")
	if err != nil {
		t.Fatalf("test cmd: %v\n%s", err, out)
	}
	if !strings.Contains(out, "chain-id") || !strings.Contains(out, "pass") {
		t.Errorf("expected chain-id pass in output:\n%s", out)
	}
	if !strings.Contains(out, "pass=1") {
		t.Errorf("expected pass=1:\n%s", out)
	}
}

func TestTestCmd_PersistsAndReport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x205b"})
	}))
	defer srv.Close()

	dir := t.TempDir()
	nsJSON := `{"chain":"wbft","network":"local","capabilities":["rpc"],"nodes":[{"index":1,"role":"endpoint","rpc_url":"` + srv.URL + `"}]}`
	if err := os.WriteFile(filepath.Join(dir, "nodeset.json"), []byte(nsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run test against the saved network; results persist to runs.json.
	if _, err := run(t, "test", "--data-dir", dir, "--name", "chain-id"); err != nil {
		t.Fatalf("test: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "runs.json")); err != nil {
		t.Fatalf("runs.json not written: %v", err)
	}

	// report reads them back.
	out, err := run(t, "report", "--data-dir", dir)
	if err != nil {
		t.Fatalf("report: %v\n%s", err, out)
	}
	if !strings.Contains(out, "test/chain-id") || !strings.Contains(out, "total=1 ok=1") {
		t.Errorf("report output:\n%s", out)
	}
}

func TestReportCmd_Empty(t *testing.T) {
	out, err := run(t, "report", "--data-dir", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no runs recorded") {
		t.Errorf("expected empty report:\n%s", out)
	}
}

func TestTestCmd_FromState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x205b"})
	}))
	defer srv.Close()

	// Write a nodeset.json pointing at the mock, then run test --data-dir.
	dir := t.TempDir()
	nsJSON := `{"chain":"wbft","network":"local","capabilities":["rpc"],"nodes":[{"index":1,"role":"endpoint","rpc_url":"` + srv.URL + `"}]}`
	if err := os.WriteFile(filepath.Join(dir, "nodeset.json"), []byte(nsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "test", "--data-dir", dir, "--name", "chain-id")
	if err != nil {
		t.Fatalf("test --data-dir: %v\n%s", err, out)
	}
	if !strings.Contains(out, "pass=1") {
		t.Errorf("expected pass=1 from state:\n%s", out)
	}
}

func TestHardforkCmd_DryRunPlan(t *testing.T) {
	dir := t.TempDir()
	// A running wemix network's saved state.
	nsJSON := `{"chain":"wemix","network":"local","nodes":[{"index":1,"role":"validator"},{"index":2,"role":"validator"}]}`
	if err := os.WriteFile(filepath.Join(dir, "nodeset.json"), []byte(nsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "hardfork", "--data-dir", dir, "--to-chain", "wbft", "--block", "100")
	if err != nil {
		t.Fatalf("hardfork: %v\n%s", err, out)
	}
	if !strings.Contains(out, "wemix -> wbft") || !strings.Contains(out, "gwemix -> gwbft") {
		t.Errorf("expected wemix->wbft binary swap:\n%s", out)
	}
	if !strings.Contains(out, "block 100") {
		t.Errorf("expected fork block 100:\n%s", out)
	}
}

func TestHardforkCmd_ExecuteNeedsBinary(t *testing.T) {
	dir := t.TempDir()
	nsJSON := `{"chain":"wemix","network":"local","nodes":[{"index":1,"role":"validator"}]}`
	_ = os.WriteFile(filepath.Join(dir, "nodeset.json"), []byte(nsJSON), 0o644)
	// gwbft is not on PATH → resolveBinary fails.
	_, err := run(t, "hardfork", "--data-dir", dir, "--to-chain", "wbft", "--block", "100", "--dry-run=false")
	if err == nil || !strings.Contains(err.Error(), "cannot find node binary") {
		t.Errorf("expected binary-not-found error, got %v", err)
	}
}

func TestHardforkCmd_Execute(t *testing.T) {
	dir := t.TempDir()
	// A running from-node to be swapped.
	old := exec.Command("sleep", "30")
	if err := old.Start(); err != nil {
		t.Fatal(err)
	}
	oldPID := old.Process.Pid
	nsJSON := fmt.Sprintf(`{"chain":"wemix","network":"local","nodes":[{"index":1,"role":"validator","ports":{"http":8501,"p2p":30301},"pid":%d}]}`, oldPID)
	if err := os.WriteFile(filepath.Join(dir, "nodeset.json"), []byte(nsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "faketo")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nif [ \"$1\" = \"init\" ]; then exit 0; fi\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, "hardfork", "--data-dir", dir, "--to-chain", "wbft", "--block", "100",
		"--to-binary", bin, "--dry-run=false")
	if err != nil {
		t.Fatalf("hardfork execute: %v\n%s", err, out)
	}
	if !strings.Contains(out, "upgraded 1 node(s) to wbft") {
		t.Errorf("expected upgrade confirmation:\n%s", out)
	}
	if werr := old.Wait(); werr == nil {
		t.Error("old node should have been stopped")
	}
	// nodeset.json now records the new chain + a new pid; clean up the process.
	b, _ := os.ReadFile(filepath.Join(dir, "nodeset.json"))
	var ns struct {
		Chain string `json:"chain"`
		Nodes []struct {
			PID int `json:"pid"`
		} `json:"nodes"`
	}
	_ = json.Unmarshal(b, &ns)
	if ns.Chain != "wbft" || len(ns.Nodes) != 1 || ns.Nodes[0].PID == 0 {
		t.Errorf("state after upgrade: %+v", ns)
	}
	if p, err := os.FindProcess(ns.Nodes[0].PID); err == nil {
		_ = p.Kill()
	}
}

func TestNodeRPCCmd_Passthrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int           `json:"id"`
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if req.Method == "eth_getBlockByNumber" && len(req.Params) == 2 {
			res["result"] = map[string]any{"number": "0x2a"}
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	out, err := run(t, "node", "rpc", "--rpc", srv.URL, "--method", "eth_getBlockByNumber", "--params", `["latest",false]`)
	if err != nil {
		t.Fatalf("node rpc: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"number":"0x2a"`) {
		t.Errorf("expected block result:\n%s", out)
	}
}

func TestConsensusCmd_HTTPMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if req.Method == "istanbul_getValidators" {
			res["result"] = []string{"0xaaa", "0xbbb", "0xccc"}
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	out, err := run(t, "consensus", "--chain", "stablenet", "--rpc", srv.URL)
	if err != nil {
		t.Fatalf("consensus: %v\n%s", err, out)
	}
	if !strings.Contains(out, "istanbul_getValidators") || !strings.Contains(out, "3") {
		t.Errorf("expected 3 validators via istanbul method:\n%s", out)
	}
	if !strings.Contains(out, "0xaaa") {
		t.Errorf("validator addresses missing:\n%s", out)
	}
}

func TestFaucetCmd_HTTPMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch req.Method {
		case "eth_chainId":
			res["result"] = "0x205b"
		case "eth_getProof":
			res["result"] = map[string]any{"extra": "0x0"}
		case "eth_getTransactionCount":
			res["result"] = "0x0"
		case "eth_maxPriorityFeePerGas", "eth_gasPrice":
			res["result"] = "0x3b9aca00"
		case "eth_sendRawTransaction":
			res["result"] = "0x" + strings.Repeat("ab", 32)
		default:
			res["result"] = nil
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	out, err := run(t, "faucet",
		"--chain", "stablenet",
		"--rpc", srv.URL,
		"--from-key", "0000000000000000000000000000000000000000000000000000000000000001",
		"--to", "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf",
		"--amount", "1000000",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "0x") {
		t.Errorf("expected tx hash, got %q", out)
	}
}

// itoaHex renders a small int as a hex quantity string.
func itoaHex(n int) string {
	const digits = "0123456789abcdef"
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{digits[n%16]}, b...)
		n /= 16
	}
	return string(b)
}
