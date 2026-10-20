package suitecmd_test

import (
	"context"
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

func requireNativePreparedSharing(t *testing.T, ctx context.Context, fixture string, preset any, rawCount *atomic.Int32) []map[string]any {
	t.Helper()
	observations := []map[string]any{}
	for _, failed := range []bool{false, true} {
		producer := map[string]any{"schemaVersion": "2", "kind": "case", "id": "prepare-assets", "requires": []string{"rpc"}, "chainPreset": preset, "prepared": map[string]any{"exports": map[string]string{"contract": "$contract", "hash": "$sent", "receipt": "$receipt"}}, "steps": []any{
			map[string]any{"do": "deployContract", "on": "en1", "from": "payer", "bytecode": "0x600a600c600039600a6000f3602a60005260206000f3", "save": "contract"},
			map[string]any{"do": "sendTx", "on": "en1", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "1", "save": "sent"},
			map[string]any{"do": "read", "on": "en1", "source": "rpcCall", "method": "eth_getTransactionReceipt", "params": []any{"$sent"}, "save": "receipt"},
			map[string]any{"expect": "codeAt", "on": "en1", "address": "$contract", "compare": "Equal", "is": "0x602a60005260206000f3"},
		}}
		if failed {
			producer["steps"].([]any)[3].(map[string]any)["is"] = "0x00"
		}
		consumer := map[string]any{"schemaVersion": "2", "kind": "case", "id": "consume-assets", "requires": []string{"rpc"}, "chainPreset": preset, "prepared": map[string]any{"imports": map[string]any{"contract": map[string]string{"case": "prepare-assets", "name": "contract"}, "hash": map[string]string{"case": "prepare-assets", "name": "hash"}, "receipt": map[string]string{"case": "prepare-assets", "name": "receipt"}}}, "steps": []any{
			map[string]any{"expect": "codeAt", "on": "en1", "address": "$contract", "compare": "Equal", "is": "0x602a60005260206000f3"},
			map[string]any{"expect": "rpcCall", "on": "en1", "method": "eth_getTransactionReceipt", "params": []any{"$hash"}, "compare": "Equal", "is": "$receipt"},
		}}
		raws := []string{}
		paths := []string{}
		for index, doc := range []map[string]any{producer, consumer} {
			raw, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			raws = append(raws, string(raw))
			path := filepath.Join(fixture, fmt.Sprintf("prepared-%t-%d.json", failed, index))
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			paths = append(paths, path)
		}
		for _, surface := range []string{"CLI", "MCP"} {
			before := rawCount.Load()
			artifacts := filepath.Join(fixture, fmt.Sprintf("prepared-%s-%t", strings.ToLower(surface), failed))
			root := ""
			if surface == "CLI" {
				args := append([]string{"run", "--json", "--no-skips", "--artifact-root", artifacts}, paths...)
				out, progress, err := runSplit(t, args...)
				want := 0
				if failed {
					want = 2
				}
				if codeOf(t, err) != want {
					t.Fatal(err, out, progress)
				}
				var report struct{ Session string }
				if err := json.Unmarshal([]byte(out), &report); err != nil {
					t.Fatal(err, out)
				}
				root = report.Session
			} else {
				request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"specs": raws, "artifactRoot": artifacts}}})
				response := mcp.Default("prepared", "native").Handle(ctx, request)
				var report struct {
					Result struct{ IsError bool }
					Error  any
				}
				if err := json.Unmarshal(response, &report); err != nil || report.Error != nil || report.Result.IsError != failed {
					t.Fatal(err, string(response))
				}
				ids, err := session.List(artifacts)
				if err != nil || len(ids) != 1 {
					t.Fatal(err, ids)
				}
				root = filepath.Join(artifacts, ids[0])
			}
			if rawCount.Load() != before+2 {
				t.Fatal("consumer resubmitted preparation or preparation did not run", before, rawCount.Load())
			}
			statuses := []string{}
			for index, id := range []string{"prepare-assets", "consume-assets"} {
				dir := filepath.Join(root, "tests", fmt.Sprintf("%03d_%s", index+1, id))
				data, err := os.ReadFile(filepath.Join(dir, "status.json"))
				if err != nil {
					t.Fatal(err)
				}
				var status struct{ Result string }
				if err := json.Unmarshal(data, &status); err != nil {
					t.Fatal(err)
				}
				want := "pass"
				if failed {
					want = []string{"fail", "blocked"}[index]
				}
				if status.Result != want {
					t.Fatal("wrong prepared result", string(data))
				}
				statuses = append(statuses, status.Result)
				if failed && index == 1 {
					if _, err := os.Stat(filepath.Join(dir, "steps.json")); err == nil {
						t.Fatal("unavailable preparation executed a consumer")
					}
				}
			}
			observations = append(observations, map[string]any{"surface": surface, "producerFailed": failed, "session": root, "statuses": statuses, "rawSubmissions": 2})
		}
	}
	return observations
}
