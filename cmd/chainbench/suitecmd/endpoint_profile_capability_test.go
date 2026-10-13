package suitecmd_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/mcp"
)

func requireNativeDeclaredCapabilities(t *testing.T, ctx context.Context, fixture string, preset any, nodes node.NodeSet, rawCount *atomic.Int32) []map[string]any {
	t.Helper()
	observations := []map[string]any{}
	for _, c := range []struct {
		name, status      string
		requires, missing []string
		unexpected        bool
	}{
		{"provided", "pass", []string{"rpc", "account-extra"}, nil, false},
		{"missing", "skip", []string{"rpc", "ws", "process-control", "metrics"}, []string{"ws", "process-control", "metrics"}, false},
		{"unexpected", "fail", []string{"rpc", "ws"}, []string{"ws"}, true},
		{"consensus", "pass", []string{"rpc", "consensus"}, nil, false},
		{"consensus-unavailable", "blocked", []string{"rpc", "consensus"}, nil, false},
	} {
		source, err := json.Marshal(preset)
		if err != nil {
			t.Fatal(err)
		}
		var configured map[string]any
		if err := json.Unmarshal(source, &configured); err != nil {
			t.Fatal(err)
		}
		if c.name == "provided" {
			configured["attach"].(map[string]any)["provides"] = []string{"account-extra"}
		}
		if strings.HasPrefix(c.name, "consensus") {
			original := configured["attach"].(map[string]any)["endpoints"].([]any)[0]
			endpoints := []any{}
			for _, n := range nodes.Nodes {
				ep := map[string]any{"rpc": n.RPCURL, "role": string(n.Role)}
				if n.Role == node.RoleEN {
					ep = original.(map[string]any)
				}
				endpoints = append(endpoints, ep)
			}
			if len(endpoints) != 5 {
				t.Fatal("consensus proof requires actual BP4/EN1")
			}
			if c.status == "blocked" {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				unavailable := "http://" + listener.Addr().String()
				if err := listener.Close(); err != nil {
					t.Fatal(err)
				}
				endpoints[4].(map[string]any)["rpc"] = unavailable
			}
			configured["attach"] = map[string]any{"endpoints": endpoints, "provides": []string{"consensus"}}
		}
		doc := map[string]any{"schemaVersion": "2", "kind": "case", "id": "capability-" + c.name, "chainPreset": configured, "requires": c.requires, "steps": []any{map[string]any{"expect": "blockNumber", "on": "en1", "compare": "GreaterOrEqual", "is": 2}}}
		if c.name == "provided" {
			doc["steps"] = append(doc["steps"].([]any), map[string]any{"expect": "balanceAt", "on": "en1", "address": "sponsor", "compare": "GreaterOrEqual", "is": "1"})
		}
		if c.unexpected {
			doc["skipsOn"] = []string{}
		}
		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(fixture, "capability-"+c.name+".json")
		if err := os.WriteFile(file, raw, 0600); err != nil {
			t.Fatal(err)
		}
		for _, surface := range []string{"CLI", "MCP"} {
			artifacts := filepath.Join(fixture, "capability-"+strings.ToLower(surface), c.name)
			before := rawCount.Load()
			root := ""
			if surface == "CLI" {
				out, progress, err := runSplit(t, "run", "--json", "--artifact-root", artifacts, file)
				wantCode := 0
				if c.status == "fail" {
					wantCode = 1
				}
				if c.status == "blocked" {
					wantCode = 2
				}
				if codeOf(t, err) != wantCode {
					t.Fatal(err, out, progress)
				}
				var report struct{ Session string }
				if err := json.Unmarshal([]byte(out), &report); err != nil {
					t.Fatal(err, out)
				}
				root = report.Session
			} else {
				request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"spec": string(raw), "artifactRoot": artifacts}}})
				response := mcp.Default("capabilities", "native").Handle(ctx, request)
				var report struct {
					Result struct{ IsError bool }
					Error  any
				}
				if err := json.Unmarshal(response, &report); err != nil || report.Error != nil || report.Result.IsError != (c.status == "fail" || c.status == "blocked") {
					t.Fatal(err, string(response))
				}
				ids, err := session.List(artifacts)
				if err != nil || len(ids) != 1 {
					t.Fatal(err, ids)
				}
				root = filepath.Join(artifacts, ids[0])
			}
			if rawCount.Load() != before {
				t.Fatal("read-only capability campaign submitted a transaction")
			}
			dir := filepath.Join(root, "tests", "001_capability-"+c.name)
			data, err := os.ReadFile(filepath.Join(dir, "status.json"))
			if err != nil {
				t.Fatal(err)
			}
			var verdict struct{ Result, Reason string }
			if err := json.Unmarshal(data, &verdict); err != nil {
				t.Fatal(err)
			}
			if verdict.Result != c.status {
				t.Fatalf("wrong capability verdict: %s", data)
			}
			if len(c.missing) > 0 && !strings.Contains(verdict.Reason, "missing required capabilities: "+strings.Join(c.missing, ",")) {
				t.Fatalf("missing precise skip reason: %s", data)
			}
			probePath := filepath.Join(dir, "observations", "capability-gate.json")
			data, err = os.ReadFile(probePath)
			if err != nil {
				t.Fatal(err)
			}
			var probe struct {
				Source                      string
				Required, Provided, Missing []string
				Runs                        bool
			}
			if err := json.Unmarshal(data, &probe); err != nil {
				t.Fatal(err)
			}
			if probe.Source != "declared_capabilities" || strings.Join(probe.Missing, ",") != strings.Join(c.missing, ",") || strings.Join(probe.Required, ",") != strings.Join(c.requires, ",") || probe.Runs != (len(c.missing) == 0) {
				t.Fatalf("incorrect gate observation: %s", data)
			}
			wantProvided := "rpc"
			if c.name == "provided" {
				wantProvided += ",account-extra"
			}
			if strings.HasPrefix(c.name, "consensus") {
				wantProvided += ",consensus"
			}
			if strings.Join(probe.Provided, ",") != wantProvided {
				t.Fatalf("preset provides did not reach gate: %s", data)
			}
			if c.status != "pass" {
				if _, err := os.Stat(filepath.Join(dir, "steps.json")); err == nil {
					t.Fatal("unmet requirement executed steps")
				}
			}
			observations = append(observations, map[string]any{"surface": surface, "scenario": c.name, "status": verdict.Result, "reason": verdict.Reason, "session": root, "probePath": probePath, "probe": json.RawMessage(data), "rawSubmissions": 0})
			if strings.HasPrefix(c.name, "consensus") {
				path := filepath.Join(dir, "observations", "capability-probes.json")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var actual struct {
					Status string
					Nodes  []struct {
						Node           int
						Status, Source string
						ValidatorCount int
					}
				}
				if err := json.Unmarshal(data, &actual); err != nil || len(actual.Nodes) != 5 {
					t.Fatal("missing actual consensus nodes", err, string(data))
				}
				for i, n := range actual.Nodes {
					wanted := "supported"
					if c.status == "blocked" && i == 4 {
						wanted = "unknown"
					}
					if n.Node != i+1 || n.Status != wanted || (wanted == "supported" && (n.Source == "" || n.ValidatorCount < 4)) {
						t.Fatalf("incorrect actual consensus node: %s", data)
					}
				}
				wanted := "supported"
				if c.status == "blocked" {
					wanted = "unknown"
				}
				if actual.Status != wanted {
					t.Fatalf("incorrect actual consensus aggregate: %s", data)
				}
				observations[len(observations)-1]["consensusProbePath"] = path
				observations[len(observations)-1]["consensusProbe"] = json.RawMessage(data)
			}
		}
	}
	return observations
}
