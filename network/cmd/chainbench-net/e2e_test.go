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

// TestE2E_NodeBlockNumber_WithAuth drives cobra twice in-process: first attach
// a mock RPC as "auth-e2e" with an api-key auth block pointing at env var
// AUTH_E2E_KEY, then issue node.block_number against the attached network. The
// mock RPC fails (401-shaped JSON-RPC error) on eth_blockNumber when the
// X-Api-Key header is missing, so a regression where auth is not actually wired
// into the dial path would surface as an UPSTREAM error rather than a false
// pass. Asserts that (a) the handler result is ok, (b) block_number matches
// the mock's 0x99 (=153), and (c) the mock observed the exact configured
// header value.
func TestE2E_NodeBlockNumber_WithAuth(t *testing.T) {
	var gotAuth string
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		key := r.Header.Get("X-Api-Key")
		// The attach probe (eth_chainId / istanbul_getValidators /
		// wemix_getReward) runs before Node.Auth is persisted and takes an
		// unauthenticated path by design — let those through unconditionally.
		// The post-attach node.block_number call is the one that exercises
		// auth, so we fail loudly there if the header is missing to guard
		// against a silent regression in auth wiring.
		switch req.Method {
		case "eth_chainId":
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "istanbul_getValidators", "wemix_getReward":
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		case "eth_blockNumber":
			if key == "" {
				_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32001,"message":"missing X-Api-Key header"}}`, req.ID)
				return
			}
			gotAuth = key
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x99"}`, req.ID)
		default:
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("AUTH_E2E_KEY", "e2e-secret")

	// First drive: attach the mock under "auth-e2e" with api-key auth. The
	// handler persists only the env var NAME, not the resolved value.
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(
		`{"command":"network.attach","args":{"rpc_url":%q,"name":"auth-e2e","auth":{"type":"api-key","header":"X-Api-Key","env":"AUTH_E2E_KEY"}}}`,
		rpcSrv.URL,
	)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v\nstderr: %s", err, stderr.String())
	}

	// Verify the state file persisted the env var NAME but NOT the value.
	stateFile := filepath.Join(stateDir, "networks", "auth-e2e.json")
	raw, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	if !strings.Contains(string(raw), "AUTH_E2E_KEY") {
		t.Errorf("state file missing env var name: %s", raw)
	}
	if strings.Contains(string(raw), "e2e-secret") {
		t.Errorf("state file leaked secret value: %s", raw)
	}

	// Second drive: node.block_number against the attached network.
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	bnCmd := `{"command":"node.block_number","args":{"network":"auth-e2e","node_id":"node1"}}`
	root2.SetIn(strings.NewReader(bnCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("block_number: %v\nstderr: %s", err, stderr.String())
	}

	// Scan NDJSON output; validate each line and find the result terminator.
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
	if got, want := res.Data["network"], "auth-e2e"; got != want {
		t.Errorf("network = %v, want %q", got, want)
	}
	if got, want := res.Data["node_id"], "node1"; got != want {
		t.Errorf("node_id = %v, want %q", got, want)
	}
	bn, ok := res.Data["block_number"].(float64)
	if !ok {
		t.Fatalf("block_number is not a number: %T %v", res.Data["block_number"], res.Data["block_number"])
	}
	if bn != 153 {
		t.Errorf("block_number = %v, want 153 (0x99)", bn)
	}
	// The mock only records the header on eth_blockNumber specifically; this
	// proves the auth RoundTripper was injected into the dial path the
	// handler takes, not just the attach probe.
	if gotAuth != "e2e-secret" {
		t.Errorf("mock did not see configured header on eth_blockNumber: got %q, want %q", gotAuth, "e2e-secret")
	}
}

// TestE2E_NodeRemoteReads_WithAuth exercises the three new remote read
// commands (node.chain_id, node.balance, node.gas_price) end-to-end against a
// single attached remote network configured with api-key auth. The mock RPC
// tracks whether at least one JSON-RPC request arrived carrying the expected
// X-Api-Key header value — so a regression where dialNode stops threading
// auth through to any of these three paths would fail the final assertion
// rather than silently pass. Each command is driven through a fresh
// newRootCmd() execution, matching the single-command-per-invocation shape
// the binary has in production.
func TestE2E_NodeRemoteReads_WithAuth(t *testing.T) {
	var authSeen bool
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") == "e2e-reads" {
			authSeen = true
		}
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x2a"}`, req.ID)
		case "eth_getBalance":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x500"}`, req.ID)
		case "eth_gasPrice":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x64"}`, req.ID)
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("READS_KEY", "e2e-reads")

	// runRoot drives a fresh cobra root command for a single envelope on stdin,
	// scans every NDJSON line for schema validity + terminator, and returns the
	// decoded result.data map. Failures abort the test with full stderr for
	// diagnostics.
	runRoot := func(t *testing.T, stdinJSON string) map[string]any {
		t.Helper()
		var stdout, stderr bytes.Buffer
		root := newRootCmd()
		root.SetIn(strings.NewReader(stdinJSON))
		root.SetOut(&stdout)
		root.SetErr(&stderr)
		root.SetArgs([]string{"run"})
		if err := root.Execute(); err != nil {
			t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
		}
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
			Ok   bool           `json:"ok"`
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(resultLine, &res); err != nil {
			t.Fatalf("unmarshal terminator: %v", err)
		}
		if !res.Ok {
			t.Fatalf("not ok: %s", resultLine)
		}
		return res.Data
	}

	// Attach the mock under "reads-e2e" with X-Api-Key auth sourced from
	// env(READS_KEY). ValidateAuth is exercised here implicitly — passing
	// means the structurally-correct api-key config was accepted.
	attachCmd := fmt.Sprintf(
		`{"command":"network.attach","args":{"rpc_url":%q,"name":"reads-e2e","auth":{"type":"api-key","header":"X-Api-Key","env":"READS_KEY"}}}`,
		rpcSrv.URL,
	)
	runRoot(t, attachCmd)

	// node.chain_id — mock returns 0x2a (=42). JSON decode of result.data
	// surfaces uint64 as float64 through map[string]any.
	chainData := runRoot(t, `{"command":"node.chain_id","args":{"network":"reads-e2e","node_id":"node1"}}`)
	if cid, ok := chainData["chain_id"].(float64); !ok || cid != 42 {
		t.Errorf("chain_id = %v (%T), want 42", chainData["chain_id"], chainData["chain_id"])
	}
	if got, want := chainData["network"], "reads-e2e"; got != want {
		t.Errorf("chain_id.network = %v, want %q", got, want)
	}
	if got, want := chainData["node_id"], "node1"; got != want {
		t.Errorf("chain_id.node_id = %v, want %q", got, want)
	}

	// node.balance — mock returns 0x500; handler preserves the hex prefix.
	balData := runRoot(t, `{"command":"node.balance","args":{"network":"reads-e2e","node_id":"node1","address":"0x0000000000000000000000000000000000000001"}}`)
	if got, want := balData["balance"], "0x500"; got != want {
		t.Errorf("balance = %v, want %q", got, want)
	}
	if got, want := balData["address"], "0x0000000000000000000000000000000000000001"; got != want {
		t.Errorf("balance.address = %v, want %q", got, want)
	}
	if got, want := balData["block"], "latest"; got != want {
		t.Errorf("balance.block = %v, want %q", got, want)
	}

	// node.gas_price — mock returns 0x64.
	gpData := runRoot(t, `{"command":"node.gas_price","args":{"network":"reads-e2e","node_id":"node1"}}`)
	if got, want := gpData["gas_price"], "0x64"; got != want {
		t.Errorf("gas_price = %v, want %q", got, want)
	}

	// The mock flips authSeen when X-Api-Key arrives with the expected value.
	// If dialNode ever regresses and stops wiring auth for any of the three
	// commands, this assertion fails — guarding against a silent auth gap.
	if !authSeen {
		t.Errorf("mock never saw the X-Api-Key header — dialNode may not be threading auth uniformly")
	}
}

// TestE2E_NodeTxSend_AgainstAttachedRemote drives the cobra root command twice
// in-process: first attaches a mock JSON-RPC endpoint, then issues
// node.tx_send with env-sourced signer key + fully pinned nonce/gas/gas_price
// so the mock only has to answer eth_chainId + eth_sendRawTransaction. Verifies
// (a) the broadcast actually arrived at the mock, (b) the handler returned a
// 0x-prefixed 66-character tx_hash, and (c) neither stdout nor stderr leaked
// the raw private key hex — this is the Go-side half of the S4 key-material
// boundary from VISION §5.17.5. The bash counterpart spawns the binary as a
// subprocess; this in-process test covers the cobra wiring path.
func TestE2E_NodeTxSend_AgainstAttachedRemote(t *testing.T) {
	var sawSendRaw bool
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			sawSendRaw = true
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x2222222222222222222222222222222222222222222222222222222222222222"}`, req.ID)
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	// Synthetic test-only key — not associated with any real funds. Tests
	// treat it as secret and assert it never appears in any observable
	// output stream.
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// First drive: attach the mock as "txs-e2e".
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"txs-e2e"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v\nstderr: %s", err, stderr.String())
	}

	// Second drive: node.tx_send against the attached network. Explicit
	// nonce/gas/gas_price short-circuit the auto-fill paths so the mock
	// doesn't need to answer eth_getTransactionCount / eth_estimateGas /
	// eth_gasPrice — keeps the test hermetic to the sign + broadcast slice.
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	sendCmd := `{"command":"node.tx_send","args":{"network":"txs-e2e","node_id":"node1","signer":"alice","to":"0x0000000000000000000000000000000000000002","value":"0x0","gas":21000,"gas_price":"0x1","nonce":0}}`
	root2.SetIn(strings.NewReader(sendCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("tx_send: %v\nstderr: %s", err, stderr.String())
	}

	// Scan NDJSON output; validate each line against the event schema and
	// find the result terminator.
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
	if !sawSendRaw {
		t.Error("mock did not see eth_sendRawTransaction")
	}
	if tx, _ := res.Data["tx_hash"].(string); !strings.HasPrefix(tx, "0x") || len(tx) != 66 {
		t.Errorf("tx_hash shape wrong: %v", res.Data["tx_hash"])
	}

	// Key-material leak guard at the E2E layer. stderr carries structured
	// logs from the run path; stdout carries the wire terminator. Neither
	// may contain the raw key hex (case-sensitive — hex is lower).
	const keyHex = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
	for _, label := range []string{"stdout", "stderr"} {
		var buf *bytes.Buffer
		if label == "stdout" {
			buf = &stdout
		} else {
			buf = &stderr
		}
		if strings.Contains(buf.String(), keyHex) {
			t.Errorf("%s leaks raw key material", label)
		}
	}
}

// TestE2E_NodeTxSend_DynamicFee_AgainstAttachedRemote drives the full
// cobra root command twice in-process to exercise the EIP-1559 selection
// path of node.tx_send: attach a mock RPC, then issue a tx_send carrying
// max_fee_per_gas + max_priority_fee_per_gas (and no gas_price). The mock
// captures the broadcast raw-hex; we decode the leading byte to confirm
// the handler emitted a DynamicFeeTx (type 2) rather than legacy. Pinned
// nonce/gas keep the mock surface to chainId + sendRawTransaction.
//
// decodeBroadcastTxType is defined in handlers_test.go and is in scope
// here because both files share the `package main` namespace.
func TestE2E_NodeTxSend_DynamicFee_AgainstAttachedRemote(t *testing.T) {
	var sentRaw string
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			if len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], &sentRaw)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x%s"}`, req.ID, strings.Repeat("c", 64))
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// attach
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"dyn"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v stderr=%s", err, stderr.String())
	}

	// tx_send (1559)
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	sendCmd := `{"command":"node.tx_send","args":{"network":"dyn","node_id":"node1","signer":"alice","to":"0x0000000000000000000000000000000000000002","value":"0x0","gas":21000,"max_fee_per_gas":"0x59682f00","max_priority_fee_per_gas":"0x3b9aca00","nonce":0}}`
	root2.SetIn(strings.NewReader(sendCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("tx_send: %v stderr=%s", err, stderr.String())
	}

	if sentRaw == "" {
		t.Fatal("mock did not see eth_sendRawTransaction")
	}
	if got := decodeBroadcastTxType(t, sentRaw); got != 2 {
		t.Errorf("tx type = %d, want 2 (DynamicFee)", got)
	}
}

// TestE2E_NodeTxWait_SuccessImmediate drives cobra twice in-process: first
// attach a mock RPC, then issue node.tx_wait against a tx hash that the
// mock immediately resolves to a successful receipt. Asserts the wire
// terminator carries status=success — exercising the full
// run.go → handler → remote.Client.TransactionReceipt → wire emit slice.
//
// The receipt JSON includes cumulativeGasUsed + logsBloom because
// go-ethereum's Client.TransactionReceipt parser rejects receipts
// missing those fields.
func TestE2E_NodeTxWait_SuccessImmediate(t *testing.T) {
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_getTransactionReceipt":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":"0x%s","blockHash":"0x%s","blockNumber":"0x20",
                "cumulativeGasUsed":"0x5208","gasUsed":"0x5208","effectiveGasPrice":"0x1",
                "status":"0x1","contractAddress":null,"logsBloom":"0x%s","logs":[]}}`,
				req.ID, strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("0", 512))
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"wt"}}`, rpcSrv.URL)))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	waitCmd := fmt.Sprintf(`{"command":"node.tx_wait","args":{"network":"wt","node_id":"node1","tx_hash":"0x%s","timeout_ms":3000}}`, strings.Repeat("a", 64))
	root2.SetIn(strings.NewReader(waitCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("tx_wait: %v", err)
	}

	// Last NDJSON line is the result terminator. Just assert success status.
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	last := lines[len(lines)-1]
	if !strings.Contains(last, `"status":"success"`) {
		t.Errorf("last line missing success: %q", last)
	}
}

// TestE2E_NodeTxSend_SetCode_AgainstAttachedRemote drives cobra twice
// in-process to exercise the EIP-7702 SetCode path of node.tx_send: attach
// a mock RPC, then issue tx_send with one authorization_list entry. The
// mock captures the broadcast raw-hex; we decode the leading byte to
// confirm the handler emitted a SetCodeTx (type 4) rather than 1559.
//
// decodeBroadcastTxType is defined in handlers_test.go and is in scope
// here because both files share the `package main` namespace.
func TestE2E_NodeTxSend_SetCode_AgainstAttachedRemote(t *testing.T) {
	var sentRaw string
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			if len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], &sentRaw)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x%s"}`, req.ID, strings.Repeat("d", 64))
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	// Sender (alice) signs the outer SetCodeTx; authorizer (bob) signs the
	// inner tuple. Distinct synthetic keys → distinct recovered addresses.
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	t.Setenv("CHAINBENCH_SIGNER_BOB_KEY", "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d")

	// attach
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"sc-e2e"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v stderr=%s", err, stderr.String())
	}

	// tx_send with authorization_list
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	sendCmd := `{"command":"node.tx_send","args":{"network":"sc-e2e","node_id":"node1","signer":"alice","to":"0x0000000000000000000000000000000000000002","value":"0x0","gas":100000,"max_fee_per_gas":"0x59682f00","max_priority_fee_per_gas":"0x3b9aca00","nonce":0,"authorization_list":[{"chain_id":"0x1","address":"0x000000000000000000000000000000000000beef","nonce":"0x0","signer":"bob"}]}}`
	root2.SetIn(strings.NewReader(sendCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("tx_send: %v stderr=%s", err, stderr.String())
	}

	if sentRaw == "" {
		t.Fatal("mock did not see eth_sendRawTransaction")
	}
	if got := decodeBroadcastTxType(t, sentRaw); got != 4 {
		t.Errorf("tx type = %d, want 4 (SetCode)", got)
	}
}

// TestE2E_NodeTxFeeDelegationSend_AgainstAttachedRemote drives cobra
// twice in-process to exercise the go-stablenet 0x16 fee-delegation
// path: attach with --override stablenet (so the adapter accepts the
// fee-delegation envelope), then issue tx_fee_delegation_send with two
// distinct signer aliases. The mock captures the broadcast raw-hex; we
// assert the leading byte == 0x16 confirming the outer wrap.
func TestE2E_NodeTxFeeDelegationSend_AgainstAttachedRemote(t *testing.T) {
	var sentRaw string
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			if len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], &sentRaw)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x%s"}`, req.ID, strings.Repeat("e", 64))
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	t.Setenv("CHAINBENCH_SIGNER_FPAYER_KEY", "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cde8b0c1")

	// attach with --override stablenet so the chain_type adapter accepts
	// fee-delegation. Without the override the probe path would still
	// classify the mock as ethereum (no istanbul_* response).
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"fd-e2e","override":"stablenet"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v stderr=%s", err, stderr.String())
	}

	// tx_fee_delegation_send
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	sendCmd := `{"command":"node.tx_fee_delegation_send","args":{"network":"fd-e2e","node_id":"node1","signer":"alice","fee_payer":"fpayer","to":"0x0000000000000000000000000000000000000003","value":"0x0","max_fee_per_gas":"0x59682f00","max_priority_fee_per_gas":"0x3b9aca00","gas":21000,"nonce":7}}`
	root2.SetIn(strings.NewReader(sendCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("tx_fee_delegation_send: %v stderr=%s", err, stderr.String())
	}

	if sentRaw == "" {
		t.Fatal("mock did not see eth_sendRawTransaction")
	}
	// Fee-delegation envelope leading byte is 0x16 (decimal 22). The shared
	// decodeBroadcastTxType helper returns the int form of that byte.
	if got := decodeBroadcastTxType(t, sentRaw); got != 0x16 {
		t.Errorf("tx type = 0x%x, want 0x16 (fee-delegation)", got)
	}
}

// TestE2E_NodeContractDeploy_AgainstAttachedRemote drives cobra twice
// in-process: first attaches a mock JSON-RPC endpoint, then issues
// node.contract_deploy with a tiny bytecode stub + 1559 fee fields. The
// mock captures the broadcast raw-hex; we decode the leading byte to
// confirm the handler emitted a DynamicFeeTx (type 2 — contract creation
// with 1559 fields uses the same envelope as a normal 1559 call). Asserts
// (a) the broadcast arrived, (b) the wire byte is 0x02, and (c) the
// terminator data carries the locally-computed contract_address (0x +
// 40 hex chars, distinct from tx_hash's 66-char shape).
func TestE2E_NodeContractDeploy_AgainstAttachedRemote(t *testing.T) {
	var sentRaw string
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_sendRawTransaction":
			if len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], &sentRaw)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x%s"}`, req.ID, strings.Repeat("a", 64))
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// attach
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"deploy-e2e"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v stderr=%s", err, stderr.String())
	}

	// contract_deploy with 1559 fee fields + pinned nonce/gas so the mock
	// only has to answer chainId + sendRawTransaction.
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	deployCmd := `{"command":"node.contract_deploy","args":{"network":"deploy-e2e","node_id":"node1","signer":"alice","bytecode":"0x6080604052348015600f57600080fd5b50","gas":100000,"max_fee_per_gas":"0x59682f00","max_priority_fee_per_gas":"0x3b9aca00","nonce":0}}`
	root2.SetIn(strings.NewReader(deployCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("contract_deploy: %v stderr=%s", err, stderr.String())
	}

	if sentRaw == "" {
		t.Fatal("mock did not see eth_sendRawTransaction")
	}
	if got := decodeBroadcastTxType(t, sentRaw); got != 2 {
		t.Errorf("tx type = %d, want 2 (DynamicFee with contract creation)", got)
	}

	// Find result terminator and assert contract_address + tx_hash shapes.
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
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resultLine, &res); err != nil {
		t.Fatalf("unmarshal terminator: %v", err)
	}
	if !res.Ok {
		t.Fatalf("not ok: %s", resultLine)
	}
	if tx, _ := res.Data["tx_hash"].(string); !strings.HasPrefix(tx, "0x") || len(tx) != 66 {
		t.Errorf("tx_hash shape wrong: %v", res.Data["tx_hash"])
	}
	if addr, _ := res.Data["contract_address"].(string); !strings.HasPrefix(addr, "0x") || len(addr) != 42 {
		t.Errorf("contract_address shape wrong: %v", res.Data["contract_address"])
	}
}

// TestE2E_NodeContractCall_AgainstAttachedRemote drives cobra twice
// in-process: attach a mock JSON-RPC endpoint, then issue node.contract_call
// with raw calldata. The mock returns a fixed 32-byte uint256(66) response;
// we assert result_raw round-trips that response verbatim through the wire
// terminator. The handler does not need a signer (read-only path).
func TestE2E_NodeContractCall_AgainstAttachedRemote(t *testing.T) {
	const mockResult = "0x0000000000000000000000000000000000000000000000000000000000000042"
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_call":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, mockResult)
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

	// attach
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"call-e2e"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v stderr=%s", err, stderr.String())
	}

	// contract_call with raw calldata (balanceOf(address) selector + zero arg).
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	callCmd := `{"command":"node.contract_call","args":{"network":"call-e2e","node_id":"node1","contract_address":"0x000000000000000000000000000000000000abcd","calldata":"0x70a082310000000000000000000000000000000000000000000000000000000000000001"}}`
	root2.SetIn(strings.NewReader(callCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("contract_call: %v stderr=%s", err, stderr.String())
	}

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
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resultLine, &res); err != nil {
		t.Fatalf("unmarshal terminator: %v", err)
	}
	if !res.Ok {
		t.Fatalf("not ok: %s", resultLine)
	}
	if got, want := res.Data["result_raw"], mockResult; got != want {
		t.Errorf("result_raw = %v, want %q", got, want)
	}
	if got, want := res.Data["block"], "latest"; got != want {
		t.Errorf("block = %v, want %q", got, want)
	}
}

// TestE2E_NodeEventsGet_AgainstAttachedRemote drives cobra twice in-process:
// attach a mock JSON-RPC endpoint that returns a single Transfer-like log
// from eth_getLogs, then issue node.events_get with an address+topic[0]
// filter. Asserts the wire terminator carries logs[] with exactly one
// entry whose required fields surface unchanged.
func TestE2E_NodeEventsGet_AgainstAttachedRemote(t *testing.T) {
	const transferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_getLogs":
			// Single log with 3 topics + 32-byte data. blockNumber/blockHash/
			// transactionHash/transactionIndex/logIndex/removed are all
			// required by go-ethereum's Log JSON unmarshaller.
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":[{
				"address":"0x000000000000000000000000000000000000abcd",
				"topics":[%q,
					"0x000000000000000000000000000000000000000000000000000000000000beef",
					"0x000000000000000000000000000000000000000000000000000000000000cafe"],
				"data":"0x0000000000000000000000000000000000000000000000000000000000000064",
				"blockNumber":"0x10",
				"blockHash":"0x%s",
				"transactionHash":"0x%s",
				"transactionIndex":"0x0",
				"logIndex":"0x0",
				"removed":false
			}]}`, req.ID, transferTopic, strings.Repeat("b", 64), strings.Repeat("a", 64))
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

	// attach
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"events-e2e"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v stderr=%s", err, stderr.String())
	}

	// events_get with address + topic[0] filter.
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	eventsCmd := fmt.Sprintf(
		`{"command":"node.events_get","args":{"network":"events-e2e","node_id":"node1","address":"0x000000000000000000000000000000000000abcd","topics":[%q]}}`,
		transferTopic,
	)
	root2.SetIn(strings.NewReader(eventsCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("events_get: %v stderr=%s", err, stderr.String())
	}

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
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resultLine, &res); err != nil {
		t.Fatalf("unmarshal terminator: %v", err)
	}
	if !res.Ok {
		t.Fatalf("not ok: %s", resultLine)
	}
	logs, ok := res.Data["logs"].([]any)
	if !ok {
		t.Fatalf("logs is not an array: %T %v", res.Data["logs"], res.Data["logs"])
	}
	if len(logs) != 1 {
		t.Fatalf("logs len = %d, want 1", len(logs))
	}
	entry, ok := logs[0].(map[string]any)
	if !ok {
		t.Fatalf("logs[0] is not a map: %T", logs[0])
	}
	// block_number is uint64 → JSON number → float64 through map[string]any.
	if bn, _ := entry["block_number"].(float64); bn != 16 {
		t.Errorf("logs[0].block_number = %v, want 16", entry["block_number"])
	}
	topics, _ := entry["topics"].([]any)
	if len(topics) != 3 {
		t.Errorf("logs[0].topics len = %d, want 3", len(topics))
	}
	if removed, _ := entry["removed"].(bool); removed {
		t.Errorf("logs[0].removed = true, want false")
	}
}

// TestE2E_NodeAccountState_AgainstAttachedRemote drives cobra twice
// in-process: attach a mock JSON-RPC endpoint, then issue node.account_state
// with default fields (balance + nonce + code, no storage). Asserts all
// three default fields surface in the wire terminator with the mock-supplied
// values, and that storage is absent (opt-in only).
func TestE2E_NodeAccountState_AgainstAttachedRemote(t *testing.T) {
	rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
		case "eth_getBalance":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0xde0b6b3a7640000"}`, req.ID)
		case "eth_getTransactionCount":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x2a"}`, req.ID)
		case "eth_getCode":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x6080604052348015600f57600080fd5b50"}`, req.ID)
		case "istanbul_getValidators", "wemix_getReward":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
		}
	}))
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

	// attach
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"state-e2e"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(attachCmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("attach: %v stderr=%s", err, stderr.String())
	}

	// account_state with default fields (balance + nonce + code).
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	stateCmd := `{"command":"node.account_state","args":{"network":"state-e2e","node_id":"node1","address":"0x0000000000000000000000000000000000000001"}}`
	root2.SetIn(strings.NewReader(stateCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("account_state: %v stderr=%s", err, stderr.String())
	}

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
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resultLine, &res); err != nil {
		t.Fatalf("unmarshal terminator: %v", err)
	}
	if !res.Ok {
		t.Fatalf("not ok: %s", resultLine)
	}
	if got, want := res.Data["balance"], "0xde0b6b3a7640000"; got != want {
		t.Errorf("balance = %v, want %q", got, want)
	}
	// nonce is uint64 → JSON number → float64.
	if nonce, _ := res.Data["nonce"].(float64); nonce != 42 {
		t.Errorf("nonce = %v, want 42", res.Data["nonce"])
	}
	if code, _ := res.Data["code"].(string); !strings.HasPrefix(code, "0x") || len(code) <= 2 {
		t.Errorf("code shape wrong: %v", res.Data["code"])
	}
	if _, ok := res.Data["storage"]; ok {
		t.Errorf("storage should be absent in default fields, got %v", res.Data["storage"])
	}
}
