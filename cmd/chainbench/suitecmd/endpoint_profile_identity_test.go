package suitecmd_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/mcp"
)

func requireNativeChainIdentity(t *testing.T, ctx context.Context, fixture, chain string, preset any, nodes node.NodeSet, rawCount *atomic.Int32) []map[string]any {
	t.Helper()
	source, _ := json.Marshal(preset)
	var configured map[string]any
	_ = json.Unmarshal(source, &configured)
	original := configured["attach"].(map[string]any)["endpoints"].([]any)[0]
	endpoints := []any{}
	for _, n := range nodes.Nodes {
		if n.RPCURL == "" {
			t.Fatalf("native node%d has no RPC for identity proof", n.Index)
		}
		ep := map[string]any{"rpc": n.RPCURL, "role": string(n.Role)}
		if n.Role == node.RoleEN {
			ep = original.(map[string]any)
		}
		endpoints = append(endpoints, ep)
	}
	configured["attach"] = map[string]any{"endpoints": endpoints}
	wanted := map[string]uint64{"stablenet": 8283, "wbft": 8284, "wemix": 8285}[chain]
	// Hold and close an exclusively selected port. The unknown target has no
	// RPC server; no chain reply is fabricated by this fixture.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	unavailable := "http://" + listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	observations := []map[string]any{}
	for _, scenario := range []struct {
		name, status string
		id           uint64
	}{{"matching", "pass", wanted}, {"wrong", "fail", wanted + 1}, {"unavailable", "blocked", wanted}} {
		clone, _ := json.Marshal(configured)
		var env map[string]any
		_ = json.Unmarshal(clone, &env)
		env["values"] = map[string]any{"chainId": strconv.FormatUint(scenario.id, 10)}
		if scenario.name == "unavailable" {
			env["attach"].(map[string]any)["endpoints"].([]any)[4].(map[string]any)["rpc"] = unavailable
		}
		raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "identity-" + scenario.name, "requires": []string{"rpc"}, "chainPreset": env,
			"hooks": map[string]any{"pre": []any{map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "1", "gas": "70000"}}},
			"steps": []any{map[string]any{"expect": "blockNumber", "on": "en1", "compare": "GreaterOrEqual", "is": "2"}}})
		if err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(fixture, "identity-"+scenario.name+".json")
		if err := os.WriteFile(file, raw, 0600); err != nil {
			t.Fatal(err)
		}
		for _, surface := range []string{"CLI", "MCP"} {
			before := rawCount.Load()
			artifacts := filepath.Join(fixture, "identity-"+strings.ToLower(surface), scenario.name)
			root := ""
			if surface == "CLI" {
				output, progress, err := runSplit(t, "run", "--json", "--no-skips", "--artifact-root", artifacts, file)
				code := codeOf(t, err)
				expectedCode := 0
				if scenario.status != "pass" {
					expectedCode = 1
				}
				if scenario.status == "blocked" {
					expectedCode = 2
				}
				if code != expectedCode {
					t.Fatal("wrong identity verdict", err, output, progress)
				}
				var report struct{ Session string }
				if err := json.Unmarshal([]byte(output), &report); err != nil {
					t.Fatal(err, output, progress)
				}
				root = report.Session
			} else {
				request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"spec": string(raw), "artifactRoot": artifacts}}})
				response := mcp.Default("identity", "native").Handle(ctx, request)
				var report struct {
					Result struct{ IsError bool }
					Error  any
				}
				if err := json.Unmarshal(response, &report); err != nil || report.Error != nil || report.Result.IsError != (scenario.status != "pass") {
					t.Fatal(err, string(response))
				}
				ids, err := session.List(artifacts)
				if err != nil || len(ids) != 1 {
					t.Fatal(err, ids)
				}
				root = filepath.Join(artifacts, ids[0])
			}
			records, err := filepath.Glob(filepath.Join(root, "tests", "*", "status.json"))
			if err != nil || len(records) != 1 {
				t.Fatal(err, records)
			}
			content, err := os.ReadFile(records[0])
			if err != nil {
				t.Fatal(err)
			}
			var result struct{ Result, Reason string }
			_ = json.Unmarshal(content, &result)
			if result.Result != scenario.status {
				t.Fatalf("identity verdict %s: %s", scenario.name, content)
			}
			delta := rawCount.Load() - before
			wantDelta := int32(0)
			if scenario.status == "pass" {
				wantDelta = 1
			}
			if delta != wantDelta {
				t.Fatalf("identity gate leaked pre-hook: %s %d", scenario.name, delta)
			}
			probePath := filepath.Join(filepath.Dir(records[0]), "observations", "chain-identity.json")
			data, err := os.ReadFile(probePath)
			if err != nil {
				t.Fatal(err)
			}
			var probe struct {
				Source, Method, Expected, Status string
				Nodes                            []map[string]any
			}
			if err := json.Unmarshal(data, &probe); err != nil {
				t.Fatal(err)
			}
			if probe.Source != "preset.values.chainId" || probe.Method != "eth_chainId" || probe.Expected != fmt.Sprint(scenario.id) || len(probe.Nodes) != 5 {
				t.Fatalf("identity observation lacks declared source or five nodes: %s", data)
			}
			for i, n := range probe.Nodes {
				if scenario.name == "unavailable" && i == 4 {
					if n["status"] != "unknown" {
						t.Fatal(n)
					}
				} else if n["actual"] != fmt.Sprint(wanted) {
					t.Fatal("node identity was not read from real RPC", n)
				}
			}
			observations = append(observations, map[string]any{"surface": surface, "scenario": scenario.name, "status": scenario.status, "reason": result.Reason, "session": root, "probePath": probePath, "probe": json.RawMessage(data), "preHookSubmissions": delta})
		}
	}
	return observations
}
