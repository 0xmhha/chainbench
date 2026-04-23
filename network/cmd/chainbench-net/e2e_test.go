package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/wire"
	"github.com/0xmhha/chainbench/network/schema"
)

func TestE2E_NetworkLoad_ViaRootCommand(t *testing.T) {
	// Prepare state dir.
	dir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("CHAINBENCH_STATE_DIR", dir)

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
		Type string                 `json:"type"`
		Ok   bool                   `json:"ok"`
		Data map[string]interface{} `json:"data"`
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
	// Lay out state dir + chainbench dir (with our stub renamed to chainbench.sh).
	stateDir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stateDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	chainbenchDir := t.TempDir()
	stub, err := os.ReadFile(filepath.Join("testdata", "chainbench-stub.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), stub, 0o755); err != nil {
		t.Fatal(err)
	}

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
	stateDir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stateDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	chainbenchDir := t.TempDir()
	stub, _ := os.ReadFile(filepath.Join("testdata", "chainbench-stub.sh"))
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), stub, 0o755); err != nil {
		t.Fatal(err)
	}
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
	stateDir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stateDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	chainbenchDir := t.TempDir()
	stub, _ := os.ReadFile(filepath.Join("testdata", "chainbench-stub.sh"))
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), stub, 0o755); err != nil {
		t.Fatal(err)
	}
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
