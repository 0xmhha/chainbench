package suitecmd_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/mcp"
)

func requireNativePresetValues(t *testing.T, ctx context.Context, fixture, chain, endpoint string, preset any, rawCount *atomic.Int32) []map[string]any {
	t.Helper()
	to := "0x000000000000000000000000000000000000dead"
	values := map[string]any{"recipient": to, "amount": "1", "gas": "70000", "expectedChainID": map[string]string{"stablenet": "8283", "wbft": "8284", "wemix": "8285"}[chain], "balanceParams": []any{to, "latest"}}
	source, _ := json.Marshal(preset)
	var prepared map[string]any
	if err := json.Unmarshal(source, &prepared); err != nil {
		t.Fatal(err)
	}
	prepared["values"] = values
	specs := [][]byte{}
	files := []string{}
	for _, id := range []string{"initial-first", "initial-second"} {
		raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": id, "requires": []string{"rpc"}, "chainPreset": prepared,
			"hooks": map[string]any{"pre": []any{map[string]any{"do": "read", "on": "en1", "source": "rpcCall", "method": "eth_getBalance", "params": "$balanceParams", "save": "before"}}},
			"steps": []any{map[string]any{"expect": "chainId", "on": "en1", "compare": "Equal", "is": "${expectedChainID}"},
				map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "$recipient", "value": "$amount", "gas": "$gas"},
				map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "amount"},
				map[string]any{"expect": "blockNumber", "on": "en1", "compare": "GreaterOrEqual", "is": "2"}}})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(fixture, id+".json")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		specs = append(specs, raw)
		files = append(files, path)
	}
	observations := []map[string]any{}
	for _, surface := range []string{"CLI", "MCP"} {
		artifacts := filepath.Join(fixture, "values-"+strings.ToLower(surface))
		root := ""
		before := rawCount.Load()
		if surface == "CLI" {
			args := append([]string{"run", "--suite", "--json", "--no-skips", "--artifact-root", artifacts}, files...)
			out, progress, err := runSplit(t, args...)
			if err != nil {
				t.Fatal(err, out, progress)
			}
			var report struct{ Session string }
			if err := json.Unmarshal([]byte(out), &report); err != nil {
				t.Fatal(err)
			}
			root = report.Session
		} else {
			request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"specs": []string{string(specs[0]), string(specs[1])}, "suite": true, "artifactRoot": artifacts}}})
			response := mcp.Default("values", "native").Handle(ctx, request)
			var report struct {
				Result struct{ IsError bool }
				Error  any
			}
			if err := json.Unmarshal(response, &report); err != nil || report.Error != nil || report.Result.IsError {
				t.Fatal(err, string(response))
			}
			ids, err := session.List(artifacts)
			if err != nil || len(ids) != 1 {
				t.Fatal(err, ids)
			}
			root = filepath.Join(artifacts, ids[0])
		}
		data, err := os.ReadFile(filepath.Join(root, "session.json"))
		if err != nil {
			t.Fatal(err)
		}
		var report struct {
			Summary struct{ Pass, Fail, Skip, Blocked int }
		}
		if err := json.Unmarshal(data, &report); err != nil || report.Summary.Pass != 2 || report.Summary.Fail != 0 || report.Summary.Skip != 0 || report.Summary.Blocked != 0 {
			t.Fatal("initial values suite did not pass twice", err, string(data))
		}
		if rawCount.Load() != before+2 {
			t.Fatalf("initial scope suite signed wrong number of transfers: %d -> %d", before, rawCount.Load())
		}
		paths, err := filepath.Glob(filepath.Join(root, "tests", "*", "steps.json"))
		if err != nil || len(paths) != 2 {
			t.Fatal(err, paths)
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var steps []struct{ Type, Hash string }
			if err := json.Unmarshal(data, &steps); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, step := range steps {
				if step.Type != "sendTx" {
					continue
				}
				found = true
				var tx map[string]any
				if err := rpc.Dial(endpoint).Call(ctx, "eth_getTransactionByHash", &tx, step.Hash); err != nil {
					t.Fatal(err)
				}
				if tx["value"] != "0x1" || tx["to"] != to || tx["gas"] != "0x11170" {
					t.Fatalf("initial values changed or leaked into next case: %+v", tx)
				}
				observations = append(observations, map[string]any{"surface": surface, "session": root, "caseDirectory": filepath.Dir(path), "transaction": tx, "values": values})
			}
			if !found {
				t.Fatal("initial values case did not submit")
			}
		}
	}
	return observations
}
