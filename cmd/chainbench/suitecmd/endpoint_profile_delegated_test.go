package suitecmd_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/mcp"
)

func nativeFixtureAddress(t *testing.T, label string) string {
	t.Helper()
	key, err := os.ReadFile(filepath.Join(repoKeys(t), label, "private"))
	if err != nil {
		t.Fatal(err)
	}
	private, err := hex.DecodeString(strings.TrimSpace(string(key)))
	if err != nil {
		t.Fatal(err)
	}
	address, err := accounts.AddressForKey(private)
	if err != nil {
		t.Fatal(err)
	}
	return address
}

// Each surface creates the same three real contracts on the authenticated EN:
// explicit gas, exact estimation, and default fees/nonce without overrides.
func requireNativeDelegatedCreations(t *testing.T, ctx context.Context, fixture string, preset any, endpoint string, rawCount *atomic.Int32, estimates func() []map[string]any) []map[string]any {
	t.Helper()
	code := "0x600a600c600039600a6000f3602a60005260206000f3"
	deployedCode := "0x602a60005260206000f3"
	steps := []any{map[string]any{"do": "read", "on": "en1", "source": "gasPrice", "save": "price"}}
	for _, mode := range []string{"full", "auto", "default"} {
		if mode != "default" {
			steps = append(steps, map[string]any{"do": "read", "on": "en1", "source": "nonceAt", "address": "payer", "save": "nonce"})
		}
		step := map[string]any{"do": "deployContract", "on": "en1", "from": "payer", "feePayer": "sponsor", "bytecode": code, "value": "5", "save": "contract"}
		if mode != "default" {
			step["nonce"] = "$nonce"
			step["maxFeePerGas"] = "$price"
			step["maxPriorityFeePerGas"] = "$price"
			step["accessList"] = []any{map[string]any{"address": "payer", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}
		}
		if mode == "full" {
			step["gas"] = "70000"
		}
		steps = append(steps, step, map[string]any{"expect": "codeAt", "on": "en1", "address": "$contract", "compare": "Equal", "is": deployedCode}, map[string]any{"expect": "balanceAt", "on": "en1", "address": "$contract", "compare": "Equal", "is": "5"})
	}
	raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "native-delegated-creations", "requires": []string{"rpc"}, "chainPreset": preset, "steps": steps})
	if err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(fixture, "delegated-creations.json")
	if err := os.WriteFile(spec, raw, 0600); err != nil {
		t.Fatal(err)
	}
	client := rpc.Dial(endpoint)
	payer, sponsor := nativeFixtureAddress(t, "node1"), nativeFixtureAddress(t, "node2")
	observations := []map[string]any{}
	for _, surface := range []string{"CLI", "MCP"} {
		before := rawCount.Load()
		firstNonce, err := client.NonceAt(ctx, payer)
		if err != nil {
			t.Fatal(err)
		}
		estimateStart := len(estimates())
		artifacts := filepath.Join(fixture, "delegated-"+strings.ToLower(surface))
		root := ""
		if surface == "CLI" {
			output, progress, err := runSplit(t, "run", "--json", "--no-skips", "--artifact-root", artifacts, spec)
			if err != nil {
				t.Fatal(err, output, progress)
			}
			var result struct{ Session string }
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatal(err)
			}
			root = result.Session
		} else {
			request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"spec": string(raw), "artifactRoot": artifacts}}})
			response := mcp.Default("delegated", "native").Handle(ctx, request)
			var result struct {
				Result struct{ IsError bool }
				Error  any
			}
			if err := json.Unmarshal(response, &result); err != nil || result.Error != nil || result.Result.IsError {
				t.Fatal(err, string(response))
			}
			ids, err := session.List(artifacts)
			if err != nil || len(ids) != 1 {
				t.Fatal(err, ids)
			}
			root = filepath.Join(artifacts, ids[0])
		}
		requireProfilePass(t, root)
		if rawCount.Load() != before+3 {
			t.Fatalf("%s must submit exactly three local creations: before=%d after=%d", surface, before, rawCount.Load())
		}
		observedEstimates := estimates()[estimateStart:]
		if len(observedEstimates) != 2 {
			t.Fatalf("%s needs two actual creation estimates: %+v", surface, observedEstimates)
		}
		files, err := filepath.Glob(filepath.Join(root, "tests", "*", "steps.json"))
		if err != nil || len(files) != 1 {
			t.Fatal(err, files)
		}
		content, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatal(err)
		}
		var records []struct{ Type, Hash string }
		if err := json.Unmarshal(content, &records); err != nil {
			t.Fatal(err)
		}
		index := 0
		for _, step := range records {
			if step.Type != "deployContract" {
				continue
			}
			if index >= 3 {
				t.Fatal("unexpected extra contract creation")
			}
			var tx, receipt map[string]any
			if err := client.Call(ctx, "eth_getTransactionByHash", &tx, step.Hash); err != nil {
				t.Fatal(err)
			}
			if err := client.Call(ctx, "eth_getTransactionReceipt", &receipt, step.Hash); err != nil {
				t.Fatal(err)
			}
			if tx["type"] != "0x16" || tx["to"] != nil || tx["feePayer"] != sponsor || tx["from"] != payer || tx["input"] != code || tx["value"] != "0x5" || tx["nonce"] != fmt.Sprintf("0x%x", firstNonce+uint64(index)) {
				t.Fatalf("delegated creation lost transaction fields: %+v", tx)
			}
			for _, signature := range []string{"fr", "fs", "fv"} {
				if tx[signature] == nil {
					t.Fatalf("missing fee-payer signature %s: %+v", signature, tx)
				}
			}
			address, ok := receipt["contractAddress"].(string)
			if !ok || address == "" || receipt["status"] != "0x1" {
				t.Fatalf("creation was not mined successfully: %+v", receipt)
			}
			actualCode, err := client.CodeAt(ctx, address)
			if err != nil || actualCode != deployedCode {
				t.Fatal("deployed code differs", err, actualCode)
			}
			if index == 0 && tx["gas"] != "0x11170" {
				t.Fatalf("explicit gas changed: %+v", tx)
			}
			if index < 2 {
				tuples, ok := tx["accessList"].([]any)
				if !ok || len(tuples) != 1 {
					t.Fatalf("creation lost access list: %+v", tx)
				}
				want := map[string]any{"address": payer, "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}
				if !reflect.DeepEqual(tuples[0], want) {
					t.Fatalf("creation changed access tuple: %+v", tuples)
				}
			}
			item := map[string]any{"surface": surface, "mode": []string{"full", "auto", "default"}[index], "session": root, "transaction": tx, "receipt": receipt, "deployedCode": actualCode}
			if index > 0 {
				estimate := observedEstimates[index-1]
				args := estimate["request"].(map[string]any)
				if args["to"] != nil || args["type"] != "0x16" || args["feePayer"] != sponsor || args["data"] != code || tx["gas"] != estimate["gas"] {
					t.Fatalf("creation estimate lost payer, gas or init code: %+v tx=%+v", estimate, tx)
				}
				for _, field := range []string{"from", "value", "nonce", "maxFeePerGas", "maxPriorityFeePerGas", "accessList"} {
					if expected, has := args[field]; has && expected != nil && !reflect.DeepEqual(tx[field], expected) {
						t.Fatalf("creation estimate changed %s: estimate=%+v tx=%+v", field, estimate, tx)
					}
				}
				item["estimate"] = estimate
			}
			observations = append(observations, item)
			index++
		}
		if index != 3 {
			t.Fatalf("%s missing native creations: %d", surface, index)
		}
	}
	return observations
}
