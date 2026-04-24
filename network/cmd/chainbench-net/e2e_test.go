package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/wire"
	"github.com/0xmhha/chainbench/network/schema"
)

// setupE2EDirs builds two tempdirs: stateDir populated with pids.json +
// current-profile.yaml fixtures, and chainbenchDir with the stub script copied
// in as chainbench.sh (so drivers that exec "${dir}/chainbench.sh" find it).
//
// Tests that need to write custom log files or mutate fixtures should do so
// after calling this helper. Tests that don't use the stub (network.load,
// tail_log) can pass a nil chainbenchDir override by ignoring the second
// return value.
func setupE2EDirs(t *testing.T) (stateDir, chainbenchDir string) {
	t.Helper()
	stateDir = t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stateDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	chainbenchDir = t.TempDir()
	stub, err := os.ReadFile(filepath.Join("testdata", "chainbench-stub.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), stub, 0o755); err != nil {
		t.Fatal(err)
	}
	return stateDir, chainbenchDir
}

func TestE2E_NetworkLoad_ViaRootCommand(t *testing.T) {
	stateDir, _ := setupE2EDirs(t)
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

	// Drive cobra root via in-memory IO.
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"network.load","args":{"name":"local"}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Collect lines; find the result terminator.
	var resultLine []byte
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		msg, derr := wire.DecodeMessage(line)
		if derr != nil {
			t.Fatalf("decode %q: %v", line, derr)
		}
		if _, ok := msg.(wire.ResultMessage); ok {
			resultLine = line
		}
	}
	if resultLine == nil {
		t.Fatal("no result line in output")
	}

	// Parse terminator and cross-validate its data against the network schema.
	var res struct {
		Type string         `json:"type"`
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resultLine, &res); err != nil {
		t.Fatalf("unmarshal terminator: %v", err)
	}
	if !res.Ok {
		t.Fatalf("not ok: %s", resultLine)
	}
	raw, err := json.Marshal(res.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	if err := schema.ValidateBytes("network", raw); err != nil {
		t.Fatalf("network schema validation failed: %v\nraw: %s", err, raw)
	}

	// Schema validity also on the full terminator line against event schema.
	if err := schema.ValidateBytes("event", resultLine); err != nil {
		t.Fatalf("event schema validation failed: %v\nline: %s", err, resultLine)
	}
}

func TestE2E_NodeStop_ViaRootCommand(t *testing.T) {
	stateDir, chainbenchDir := setupE2EDirs(t)
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_DIR", chainbenchDir)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"node.stop","args":{"node_id":"node1"}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Collect and classify lines.
	var sawEvent, sawResultOK bool
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		msg, derr := wire.DecodeMessage(line)
		if derr != nil {
			t.Fatalf("decode %q: %v", line, derr)
		}
		switch m := msg.(type) {
		case wire.EventMessage:
			if string(m.Name) == "node.stopped" {
				sawEvent = true
			}
		case wire.ResultMessage:
			if m.Ok {
				sawResultOK = true
			}
		}
		// Also validate each line against the event schema.
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
	}
	if !sawEvent {
		t.Error("expected a node.stopped event")
	}
	if !sawResultOK {
		t.Error("expected a successful result terminator")
	}
}

func TestE2E_NodeStart_ViaRootCommand(t *testing.T) {
	stateDir, chainbenchDir := setupE2EDirs(t)
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_DIR", chainbenchDir)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"node.start","args":{"node_id":"node1"}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	var sawEvent, sawResultOK bool
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
		msg, _ := wire.DecodeMessage(line)
		switch m := msg.(type) {
		case wire.EventMessage:
			if string(m.Name) == "node.started" {
				sawEvent = true
			}
		case wire.ResultMessage:
			if m.Ok {
				sawResultOK = true
			}
		}
	}
	if !sawEvent {
		t.Error("expected node.started event")
	}
	if !sawResultOK {
		t.Error("expected successful result")
	}
}

func TestE2E_NodeRestart_ViaRootCommand_EventOrder(t *testing.T) {
	stateDir, chainbenchDir := setupE2EDirs(t)
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_DIR", chainbenchDir)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"node.restart","args":{"node_id":"node1"}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Parse all lines, collect event names in order.
	var eventNames []string
	var sawResultOK bool
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
		msg, _ := wire.DecodeMessage(line)
		switch m := msg.(type) {
		case wire.EventMessage:
			eventNames = append(eventNames, string(m.Name))
		case wire.ResultMessage:
			if m.Ok {
				sawResultOK = true
			}
		}
	}
	wantOrder := []string{"node.stopped", "node.started"}
	if len(eventNames) != 2 {
		t.Fatalf("event count: got %d (%v), want 2", len(eventNames), eventNames)
	}
	for i := range wantOrder {
		if eventNames[i] != wantOrder[i] {
			t.Errorf("event[%d]: got %q, want %q", i, eventNames[i], wantOrder[i])
		}
	}
	if !sawResultOK {
		t.Error("expected successful result")
	}
}

func TestE2E_ExitCodeViaAPIError(t *testing.T) {
	// Verify that an intentional handler error propagates through Execute()
	// and that main's exitCode() maps it correctly. Since we can't os.Exit
	// in a test, we call exitCode directly on the returned error.
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`not json`)) // triggers PROTOCOL_ERROR
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if code := exitCode(err); code != 3 {
		t.Errorf("exit code: got %d, want 3 (PROTOCOL_ERROR)", code)
	}
}

func TestE2E_NodeTailLog_ViaRootCommand(t *testing.T) {
	stateDir := t.TempDir()
	// Load pids.json and rewrite log_file to a tempdir path, then write the log.
	pidsBytes, err := os.ReadFile(filepath.Join("testdata", "pids.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(pidsBytes, &raw); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(stateDir, "node1.log")
	raw["nodes"].(map[string]any)["1"].(map[string]any)["log_file"] = logPath
	patched, _ := json.Marshal(raw)
	if err := os.WriteFile(filepath.Join(stateDir, "pids.json"), patched, 0o644); err != nil {
		t.Fatal(err)
	}
	profBytes, _ := os.ReadFile(filepath.Join("testdata", "current-profile.yaml"))
	if err := os.WriteFile(filepath.Join(stateDir, "current-profile.yaml"), profBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("line1\nline2\nline3\nline4\nline5\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	// Handler doesn't need CHAINBENCH_DIR, but set something harmless.
	t.Setenv("CHAINBENCH_DIR", t.TempDir())

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"node.tail_log","args":{"node_id":"node1","lines":3}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Parse all lines — expect zero events and one OK result with data.lines.
	var gotLines []any
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
		msg, _ := wire.DecodeMessage(line)
		if rm, ok := msg.(wire.ResultMessage); ok {
			if !rm.Ok {
				t.Fatalf("result not ok: %s", line)
			}
			if ls, ok := rm.Data["lines"].([]any); ok {
				gotLines = ls
			}
		}
	}
	if len(gotLines) != 3 {
		t.Fatalf("lines: got %v (len %d)", gotLines, len(gotLines))
	}
	want := []string{"line3", "line4", "line5"}
	for i, w := range want {
		if gotLines[i] != w {
			t.Errorf("line %d: got %v, want %q", i, gotLines[i], w)
		}
	}
}

func TestE2E_NetworkProbe_ViaRootCommand(t *testing.T) {
	// Mock RPC endpoint returning stablenet-shaped responses:
	// eth_chainId -> 0x205b (8283); istanbul_getValidators -> []; other -> -32601.
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			ID     int    `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":"0x205b"}`, req.ID)
		case "istanbul_getValidators":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":[]}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"error":{"code":-32601,"message":"method not found"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	// Drive cobra root via in-memory IO, matching the pattern used by the
	// other E2E cases in this file (network.load, node.*). The command
	// envelope on stdin is parsed by run.go and dispatched to the probe
	// handler; the handler writes an NDJSON result terminator to stdout.
	cmdLine := fmt.Sprintf(`{"command":"network.probe","args":{"rpc_url":%q}}`, rpcSrv.URL)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(cmdLine))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Scan NDJSON lines; probe issues no events, so we expect exactly one
	// result terminator.
	var resultLine []byte
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
		msg, derr := wire.DecodeMessage(line)
		if derr != nil {
			t.Fatalf("decode %q: %v", line, derr)
		}
		if _, ok := msg.(wire.ResultMessage); ok {
			resultLine = line
		}
	}
	if resultLine == nil {
		t.Fatal("no result line in output")
	}

	var res struct {
		Type string         `json:"type"`
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resultLine, &res); err != nil {
		t.Fatalf("unmarshal terminator: %v", err)
	}
	if !res.Ok {
		t.Fatalf("result.ok = false: %s", resultLine)
	}
	if got, want := res.Data["chain_type"], "stablenet"; got != want {
		t.Errorf("chain_type = %v, want %q", got, want)
	}
	// JSON numbers decode as float64; probe.Result.ChainID is int64 (8283).
	chainID, ok := res.Data["chain_id"].(float64)
	if !ok {
		t.Fatalf("chain_id is not a number: %T %v", res.Data["chain_id"], res.Data["chain_id"])
	}
	if int64(chainID) != 8283 {
		t.Errorf("chain_id = %v, want 8283", chainID)
	}
}

// TestE2E_NetworkAttach_ViaRootCommand drives the cobra root command twice
// in-process: first to attach a remote network (probed via an httptest mock
// RPC), then to load the same name back. Verifies that the persisted state
// file round-trips through state.LoadActive's non-local routing.
func TestE2E_NetworkAttach_ViaRootCommand(t *testing.T) {
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			ID     int    `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":"0x205b"}`, req.ID)
		case "istanbul_getValidators":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":[]}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"error":{"code":-32601,"message":"method not found"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	cmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"integration"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(cmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Scan NDJSON output; validate every line against the event schema and
	// find the result terminator.
	var resultLine []byte
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
		msg, derr := wire.DecodeMessage(line)
		if derr != nil {
			t.Fatalf("decode %q: %v", line, derr)
		}
		if _, ok := msg.(wire.ResultMessage); ok {
			resultLine = line
		}
	}
	if resultLine == nil {
		t.Fatal("no result line in attach output")
	}

	var res struct {
		Type string         `json:"type"`
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resultLine, &res); err != nil {
		t.Fatalf("unmarshal terminator: %v", err)
	}
	if !res.Ok {
		t.Fatalf("attach result.ok = false: %s", resultLine)
	}
	if got, want := res.Data["chain_type"], "stablenet"; got != want {
		t.Errorf("chain_type = %v, want %q", got, want)
	}
	if got, want := res.Data["name"], "integration"; got != want {
		t.Errorf("name = %v, want %q", got, want)
	}

	// Verify the state file landed on disk.
	if _, err := os.Stat(filepath.Join(stateDir, "networks", "integration.json")); err != nil {
		t.Errorf("state file missing: %v", err)
	}

	// Second drive: network.load the same name and assert equivalence.
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	loadCmd := `{"command":"network.load","args":{"name":"integration"}}`
	root2.SetIn(strings.NewReader(loadCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("load execute: %v\nstderr: %s", err, stderr.String())
	}

	var loadResult []byte
	scanner = bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("load schema validation: %v\nline: %s", err, line)
		}
		msg, derr := wire.DecodeMessage(line)
		if derr != nil {
			t.Fatalf("load decode %q: %v", line, derr)
		}
		if _, ok := msg.(wire.ResultMessage); ok {
			loadResult = line
		}
	}
	if loadResult == nil {
		t.Fatal("no result line in load output")
	}
	var loadRes struct {
		Type string         `json:"type"`
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(loadResult, &loadRes); err != nil {
		t.Fatalf("unmarshal load terminator: %v", err)
	}
	if !loadRes.Ok {
		t.Fatalf("load result.ok = false: %s", loadResult)
	}
	if got, want := loadRes.Data["name"], "integration"; got != want {
		t.Errorf("load name = %v, want %q", got, want)
	}
}

// TestE2E_NodeBlockNumber_AgainstAttachedRemote drives cobra twice in-process:
// first attach a mock RPC as "remote-e2e", then issue node.block_number against
// the attached network. Exercises the ethclient remote-driver path end-to-end
// and asserts the block number surfaces through the wire result terminator.
func TestE2E_NodeBlockNumber_AgainstAttachedRemote(t *testing.T) {
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"0x1"}`))
		case "istanbul_getValidators", "wemix_getReward":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"nf"}}`))
		case "eth_blockNumber":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"0x2a"}`))
		default:
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"nf"}}`))
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

	// First drive: attach the mock as "remote-e2e".
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"remote-e2e"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v\nstderr: %s", err, stderr.String())
	}

	// Second drive: node.block_number against the attached network.
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	bnCmd := `{"command":"node.block_number","args":{"network":"remote-e2e","node_id":"node1"}}`
	root2.SetIn(strings.NewReader(bnCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("block_number: %v\nstderr: %s", err, stderr.String())
	}

	// Scan NDJSON output; validate each line against the event schema and find
	// the result terminator.
	var resultLine []byte
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema: %v\nline: %s", err, line)
		}
		msg, derr := wire.DecodeMessage(line)
		if derr != nil {
			t.Fatalf("decode %q: %v", line, derr)
		}
		if _, ok := msg.(wire.ResultMessage); ok {
			resultLine = line
		}
	}
	if resultLine == nil {
		t.Fatal("no result terminator")
	}

	var res struct {
		Type string         `json:"type"`
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resultLine, &res); err != nil {
		t.Fatalf("unmarshal terminator: %v", err)
	}
	if !res.Ok {
		t.Fatalf("not ok: %s", resultLine)
	}
	if got, want := res.Data["network"], "remote-e2e"; got != want {
		t.Errorf("network = %v, want %q", got, want)
	}
	if got, want := res.Data["node_id"], "node1"; got != want {
		t.Errorf("node_id = %v, want %q", got, want)
	}
	// block_number round-trips as float64 through JSON decode into map[string]any.
	bn, ok := res.Data["block_number"].(float64)
	if !ok {
		t.Fatalf("block_number is not a number: %T %v", res.Data["block_number"], res.Data["block_number"])
	}
	if bn != 42 {
		t.Errorf("block_number = %v, want 42", bn)
	}
}
