package suitecmd_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/mcp"
	"github.com/0xmhha/chainbench/internal/testhelper"
)

func TestNativeUnsignedControlFixturesUseOnlyConsumedArguments(t *testing.T) {
	preset := map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "fixture", "chain": "stablenet", "attach": map[string]any{"rpc": []string{"http://127.0.0.1:1"}}}
	var count atomic.Int32
	requireNativeUnsignedControls(t, t.TempDir(), preset, &count, true)
}

func requireNativeUnsignedControls(t *testing.T, fixture string, preset any, rawCount *atomic.Int32, fixtureOnly bool) []map[string]any {
	t.Helper()
	observations := []map[string]any{}
	for index, c := range []struct {
		kind, name, field string
		bad               any
	}{
		{"action", "waitBlock", "target", 1.5},
		{"action", "sendTx", "blocks", 0},
		{"action", "load", "blocks", 1.5},
		{"action", "load", "fillPercent", 101},
		{"action", "readNodeLog", "maxBytes", 1.5},
		{"assertion", "blockHalt", "maxAdvance", 1.5},
		{"assertion", "blockInterval", "blocks", 0},
		{"assertion", "blockInterval", "maxSeconds", 1.5},
		{"assertion", "blockInterval", "minSeconds", 1.5},
		{"assertion", "blockInterval", "maxMillis", 1.5},
		{"assertion", "blockInterval", "minMillis", 1.5},
		{"assertion", "wsSubscribe", "count", "18446744073709551615"},
		{"assertion", "wsCollected", "count", 0},
		{"reader", "derive", "index", "18446744073709551615"},
		{"reader", "logs", "index", 1.5},
		{"reader", "receiptLog", "index", 1.5},
		{"reader", "receiptLog", "topic", "18446744073709551615"},
		{"reader", "createAddress", "nonce", 1.5},
	} {
		id := fmt.Sprintf("unsigned-%d", index)
		step := map[string]any{"on": "en1"}
		steps := []any{}
		switch c.kind {
		case "action":
			step["do"] = c.name
			if c.name == "sendTx" || c.name == "load" {
				step["from"] = "payer"
				step["gas"] = "300000"
			}
			if c.name == "sendTx" {
				step["to"] = "0x000000000000000000000000000000000000dead"
				step["value"] = "1"
				step["expect"] = "keptOut"
			}
			if c.name == "load" {
				step["blocks"] = 1
				if c.field == "fillPercent" {
					delete(step, "gas")
				}
			}
		case "reader":
			step["do"] = "read"
			step["source"] = c.name
			step["save"] = "value"
			switch c.name {
			case "derive":
				step["op"] = "word"
				step["of"] = []any{"0x0000000000000000000000000000000000000000000000000000000000000001"}
			case "logs":
				step["select"] = "data"
			case "receiptLog":
				step["hash"] = "0x0000000000000000000000000000000000000000000000000000000000000001"
			case "createAddress":
				step["deployer"] = "payer"
			}
		case "assertion":
			step["expect"] = c.name
			if c.name == "blockHalt" {
				step["within"] = "1ms"
			}
			if c.name == "wsCollected" {
				delete(step, "on")
				steps = append(steps, map[string]any{"do": "wsOpen", "on": "en1", "event": "newHeads", "save": "captured"})
				step["sub"] = "$captured"
			}
		}
		step[c.field] = c.bad
		steps = append(steps, step)
		steps = append(steps, map[string]any{"expect": "blockNumber", "on": "en1", "compare": "GreaterOrEqual", "is": "0"})
		raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": id, "requires": []string{"rpc"}, "chainPreset": preset, "steps": steps})
		if err != nil {
			t.Fatal(err)
		}
		spec, err := dsl.Parse(raw)
		if err != nil {
			t.Fatal("invalid unsigned control proof fixture", c.name, c.field, err)
		}
		if ignored := testhelper.IgnoredArguments(spec); len(ignored) != 0 {
			t.Fatal("unsigned control fixture contains ignored arguments", c.name, c.field, ignored)
		}
		if fixtureOnly {
			continue
		}
		path := filepath.Join(fixture, id+".json")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		for _, surface := range []string{"CLI", "MCP"} {
			before := rawCount.Load()
			artifacts := filepath.Join(fixture, "unsigned-"+surface, id)
			root := ""
			if surface == "CLI" {
				out, progress, runErr := runSplit(t, "run", "--json", "--no-skips", "--artifact-root", artifacts, path)
				if codeOf(t, runErr) != 1 {
					t.Fatal("unsigned CLI verdict", runErr, out, progress)
				}
				var report struct{ Session string }
				if err := json.Unmarshal([]byte(out), &report); err != nil {
					t.Fatal(c.name, c.field, "invalid CLI JSON", err, "execution error", runErr, out, progress)
				}
				root = report.Session
			} else {
				request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"spec": string(raw), "artifactRoot": artifacts}}})
				response := mcp.Default("controls", "native").Handle(t.Context(), request)
				var report struct {
					Result struct{ IsError bool }
					Error  any
				}
				if err := json.Unmarshal(response, &report); err != nil || report.Error != nil || !report.Result.IsError {
					t.Fatal(err, string(response))
				}
				ids, err := session.List(artifacts)
				if err != nil || len(ids) != 1 {
					t.Fatal(err, ids)
				}
				root = filepath.Join(artifacts, ids[0])
			}
			reason := ""
			if c.kind == "assertion" {
				reason = requireInvalidUnsignedAssertion(t, root, c.name, c.field)
			} else {
				reason = requireInvalidNumericSession(t, root, c.field)
			}
			if rawCount.Load() != before {
				t.Fatal("invalid control submitted a transaction", c.name, c.field)
			}
			observations = append(observations, map[string]any{"surface": surface, "kind": c.kind, "operation": c.name, "field": c.field, "session": root, "reason": reason, "rawSubmissions": 0})
		}
	}
	return observations
}

func requireInvalidUnsignedAssertion(t *testing.T, root, name, field string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Summary struct{ Pass, Fail, Blocked, Skip int }
	}
	if err := json.Unmarshal(raw, &record); err != nil || record.Summary.Pass != 0 || record.Summary.Fail != 1 || record.Summary.Blocked != 0 || record.Summary.Skip != 0 {
		t.Fatalf("invalid assertion was not an explicit failure: %v %s", err, raw)
	}
	files, err := filepath.Glob(filepath.Join(root, "tests", "*", "status.json"))
	if err != nil || len(files) != 1 {
		t.Fatal(err, files)
	}
	raw, err = os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var status struct{ Result, Reason string }
	if err := json.Unmarshal(raw, &status); err != nil || status.Result != "fail" || status.Reason != "1 assertion(s) failed" {
		t.Fatalf("unexpected assertion failure status: %v %s", err, raw)
	}
	raw, err = os.ReadFile(filepath.Join(filepath.Dir(files[0]), "assert.json"))
	if err != nil {
		t.Fatal(err)
	}
	var results []session.AssertResult
	if err := json.Unmarshal(raw, &results); err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Assert == name && !result.Pass {
			if reason, ok := result.Actual.(string); ok && strings.Contains(reason, "bad "+field) {
				return reason
			}
		}
	}
	t.Fatalf("assertion evidence did not identify the bad numeric field: %s", raw)
	return ""
}
