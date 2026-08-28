package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
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

func TestStopCmd_KillsNodes(t *testing.T) {
	// Spawn a real long-lived process to act as a launched node.
	proc := exec.Command("sleep", "30")
	if err := proc.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	pid := proc.Process.Pid

	dir := t.TempDir()
	writeWorkspace(t, dir, "stablenet", wsNode(dir, 1, "validator", 8501, pid))

	out, err := run(t, "stop", "--workspace-dir", dir)
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
	writeWorkspace(t, dir, "wbft", wsNode(dir, 1, "validator", portOf(t, srv.URL), 0))
	out, err := run(t, "verify", "--workspace-dir", dir, "--progress-delay", "1ms")
	if err != nil {
		t.Fatalf("verify --workspace-dir: %v\n%s", err, out)
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
	// Mock node answering every method with a value (the RPC-presence case passes).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x205b"})
	}))
	defer srv.Close()

	out, err := run(t, "test", "--chain", "wbft", "--rpc", srv.URL, "--name", "fee-delegate-sign-rpc-present")
	if err != nil {
		t.Fatalf("test cmd: %v\n%s", err, out)
	}
	if !strings.Contains(out, "fee-delegate-sign-rpc-present") || !strings.Contains(out, "pass") {
		t.Errorf("expected fee-delegate-sign-rpc-present pass in output:\n%s", out)
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
	writeWorkspace(t, dir, "wbft", wsNode(dir, 1, "endpoint", portOf(t, srv.URL), 0))

	// Run test against the recorded network; results persist to runs.json.
	if _, err := run(t, "test", "--workspace-dir", dir, "--name", "fee-delegate-sign-rpc-present"); err != nil {
		t.Fatalf("test: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "runs.json")); err != nil {
		t.Fatalf("runs.json not written: %v", err)
	}

	// report reads them back.
	out, err := run(t, "report", "--workspace-dir", dir)
	if err != nil {
		t.Fatalf("report: %v\n%s", err, out)
	}
	if !strings.Contains(out, "test/fee-delegate-sign-rpc-present") || !strings.Contains(out, "total=1 ok=1") {
		t.Errorf("report output:\n%s", out)
	}
}

func TestReportCmd_Empty(t *testing.T) {
	out, err := run(t, "report", "--workspace-dir", t.TempDir())
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

	// Record a workspace pointing at the mock, then run test --workspace-dir.
	dir := t.TempDir()
	writeWorkspace(t, dir, "wbft", wsNode(dir, 1, "endpoint", portOf(t, srv.URL), 0))
	out, err := run(t, "test", "--workspace-dir", dir, "--name", "fee-delegate-sign-rpc-present")
	if err != nil {
		t.Fatalf("test --workspace-dir: %v\n%s", err, out)
	}
	if !strings.Contains(out, "pass=1") {
		t.Errorf("expected pass=1 from state:\n%s", out)
	}
}

func TestHardforkCmd_DryRunPlan(t *testing.T) {
	dir := t.TempDir()
	// A running wemix network's record.
	writeWorkspace(t, dir, "wemix", wsNode(dir, 1, "validator", 8501, 0), wsNode(dir, 2, "validator", 8511, 0))
	out, err := run(t, "hardfork", "--workspace-dir", dir, "--to-chain", "wbft", "--block", "100")
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
	writeWorkspace(t, dir, "wemix", wsNode(dir, 1, "validator", 8501, 0))
	// gwbft is not on PATH → resolveBinary fails.
	_, err := run(t, "hardfork", "--workspace-dir", dir, "--to-chain", "wbft", "--block", "100", "--dry-run=false")
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
	// hardfork reuses the node's recorded argv (identity args) and swaps only
	// the binary, so the record must carry the arming a start leaves behind.
	n1 := wsNode(dir, 1, "validator", 8501, oldPID)
	n1.Args = []string{"--datadir", dir}
	writeWorkspace(t, dir, "wemix", n1)
	bin := filepath.Join(dir, "faketo")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nif [ \"$1\" = \"init\" ]; then exit 0; fi\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, "hardfork", "--workspace-dir", dir, "--to-chain", "wbft", "--block", "100",
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
	// The workspace now records the new chain + a new pid; clean up the process.
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatalf("reopen workspace: %v", err)
	}
	st := ws.State()
	if st.Chain != "wbft" || len(st.Nodes) != 1 || st.Nodes[0].PID == 0 {
		t.Errorf("state after upgrade: chain=%s nodes=%+v", st.Chain, st.Nodes)
	}
	if p, err := os.FindProcess(st.Nodes[0].PID); err == nil {
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

// writeWorkspace records a composed network the way the steps leave one, so a
// command can be driven against it without composing anything.
func writeWorkspace(t *testing.T, dir, chain string, nodes ...node.Record) {
	t.Helper()
	comp, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	st := chainsetup.State{
		Chain: chain, Binary: "/opt/fakebin", Validators: len(nodes),
		Target: machine.Spec{DataRoot: dir}, Nodes: nodes,
		Capabilities: []string{"rpc"},
		Steps:        map[string]chainsetup.Step{},
	}
	if err := comp.Save(st); err != nil {
		t.Fatalf("write workspace: %v", err)
	}
}

// wsNode is one recorded node on this machine: its HTTP port decides the RPC
// URL the commands read back, and pid whether it counts as running.
func wsNode(root string, index int, role string, http, pid int) node.Record {
	label := node.LabelFor(index)
	layout := node.Layout{Root: root}
	return node.Record{
		Index: index, Label: string(label), Role: role, Host: "127.0.0.1",
		DataDir: layout.DataDir(label), ConfigPath: layout.ConfigPath(label), LogPath: layout.LogPath(label),
		Endpoints: node.Endpoints{P2P: 30300 + index, HTTP: http},
		PID:       pid,
	}
}

// portOf is the port of an httptest server URL.
func portOf(t *testing.T, rawURL string) int {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
