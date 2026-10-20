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
	"github.com/0xmhha/chainbench/internal/mcp"
)

func requireNativeInvalidNumericRuns(t *testing.T, fixture string, preset any, rawCount *atomic.Int32) []map[string]any {
	t.Helper()
	cases := []struct {
		action, field string
		bad           any
	}{
		{"sendTx", "gas", "not-a-number"}, {"sendTx", "value", 1.5}, {"sendTx", "nonce", "18446744073709551616"},
		{"faucet", "amount", 1.5}, {"deployContract", "gas", float64(9007199254740992)},
		{"registerContract", "maxPriorityFeePerGas", "0xgg"}, {"load", "nonce", nil},
		{"load", "blocks", 1.5}, {"load", "blocks", 0}, {"load", "blocks", "18446744073709551615"}, {"load", "fillPercent", 1.5}, {"load", "fillPercent", 101},
	}
	observations := []map[string]any{}
	for _, mode := range []string{"local", "node"} {
		for index, c := range cases {
			id := fmt.Sprintf("invalid-%s-%d", mode, index)
			from := "payer"
			if mode == "node" {
				from = "0x000000000000000000000000000000000000dead"
			}
			step := map[string]any{"do": c.action, "on": "en1", "from": from, "gas": "300000"}
			switch c.action {
			case "sendTx":
				step["to"] = "0x000000000000000000000000000000000000dead"
				step["value"] = "1"
				step["wait"] = false
			case "faucet":
				step["to"] = "0x000000000000000000000000000000000000dead"
				step["amount"] = "1"
			case "deployContract":
				step["bytecode"] = "0x60006000f3"
			case "registerContract":
				step["to"] = "0x000000000000000000000000000000000000dead"
				step["data"] = "0x01"
			case "load":
				step["blocks"] = 1
			}
			if c.field == "fillPercent" {
				// The real DSL rejects fillPercent when gas overrides it.
				// Exercise this field as the chosen gas-sizing input.
				delete(step, "gas")
			}
			step[c.field] = c.bad
			raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": id, "requires": []string{"rpc"}, "chainPreset": preset, "steps": []any{step, map[string]any{"expect": "blockNumber", "on": "en1", "compare": "GreaterOrEqual", "is": "0"}}})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(fixture, id+".json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			before := rawCount.Load()
			output, progress, runErr := runSplit(t, "run", "--json", "--no-skips", "--artifact-root", filepath.Join(fixture, "invalid-cli", id), path)
			if codeOf(t, runErr) != 1 {
				t.Fatalf("invalid numeric CLI did not fail: %v %s %s", runErr, output, progress)
			}
			var cli struct {
				Session string `json:"session"`
			}
			if err := json.Unmarshal([]byte(output), &cli); err != nil {
				t.Fatalf("CLI report unavailable: %v; run error=%v stdout=%q progress=%q", err, runErr, output, progress)
			}
			reason := requireInvalidNumericSession(t, cli.Session, c.field)
			observations = append(observations, map[string]any{"case": id, "action": c.action, "field": c.field, "mode": mode, "surface": "CLI", "session": cli.Session, "reason": reason})
			artifacts := filepath.Join(fixture, "invalid-mcp", id)
			request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"spec": string(raw), "artifactRoot": artifacts}}})
			response := mcp.Default("numbers", "native").Handle(t.Context(), request)
			var reply struct {
				Result struct {
					IsError bool `json:"isError"`
				}
				Error any
			}
			if err := json.Unmarshal(response, &reply); err != nil || reply.Error != nil || !reply.Result.IsError {
				t.Fatalf("invalid numeric MCP did not report failure: %v %s", err, response)
			}
			ids, err := session.List(artifacts)
			if err != nil || len(ids) != 1 {
				t.Fatal(err, ids)
			}
			root := filepath.Join(artifacts, ids[0])
			reason = requireInvalidNumericSession(t, root, c.field)
			observations = append(observations, map[string]any{"case": id, "action": c.action, "field": c.field, "mode": mode, "surface": "MCP", "session": root, "reason": reason})
			if rawCount.Load() != before {
				t.Fatalf("invalid numeric input submitted a raw transaction: %s", id)
			}
		}
	}
	return observations
}

func requireInvalidNumericSession(t *testing.T, root, field string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Summary struct{ Pass, Fail, Blocked, Skip int }
	}
	if err := json.Unmarshal(raw, &record); err != nil || record.Summary.Pass != 0 || record.Summary.Fail != 1 || record.Summary.Blocked != 0 || record.Summary.Skip != 0 {
		t.Fatalf("invalid input was not an explicit failure: %v %s", err, raw)
	}
	files, err := filepath.Glob(filepath.Join(root, "tests", "*", "status.json"))
	if err != nil || len(files) != 1 {
		t.Fatal(err, files)
	}
	raw, err = os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var status struct{ Reason string }
	if err := json.Unmarshal(raw, &status); err != nil || !strings.Contains(status.Reason, "bad "+field) {
		t.Fatalf("failure did not identify the bad numeric field: %v %s", err, raw)
	}
	return status.Reason
}
