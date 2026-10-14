package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestNativeNonceOrderAndReplacement runs the public cases and independently
// reads their actual mined positions and the evicted transaction on all families.
func TestNativeNonceOrderAndReplacement(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native nonce proof requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			inputs := []string{"CT-TX-012-nonce-ordering.json", "CT-TX-013-same-nonce-replacement.json"}
			raws := [][]byte{}
			for _, name := range inputs {
				p := filepath.Join(repoRoot(t), "tests/tc/common/tx", name)
				data, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
					t.Fatal(err)
				}
				specs, err := dsl.ReadFilesWithChainPreset([]string{p}, filepath.Join(repoRoot(t), "presets/chain", f.chain+"-bp4.json"))
				if err != nil {
					t.Fatal(err)
				}
				raws = append(raws, specs...)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary}
			if runErr != nil {
				proof["runError"] = runErr.Error()
			}
			defer func() {
				data, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err := os.WriteFile(filepath.Join(root, "reports/nonce-semantics-proof.json"), data, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 2 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("public cases failed: %v %+v", runErr, out.Summary)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(status.Nodes.Nodes) < 2 {
				t.Fatal("missing actual node records", err)
			}
			c := rpc.Dial(status.Nodes.Nodes[1].RPCURL)
			for i, id := range []string{"nonce-ordering", "replacement-tx"} {
				dir := filepath.Join(out.SessionRoot, "tests", fmt.Sprintf("%03d_%s", i+1, id))
				data, err := os.ReadFile(filepath.Join(dir, "steps.json"))
				if err != nil {
					t.Fatal(err)
				}
				var steps []session.StepResult
				if err := json.Unmarshal(data, &steps); err != nil {
					t.Fatal(err)
				}
				var observed []map[string]any
				for _, step := range steps {
					if step.Type != "sendTx" || step.Hash == "" {
						continue
					}
					var tx, receipt map[string]any
					if err := c.Call(ctx, "eth_getTransactionByHash", &tx, step.Hash); err != nil {
						t.Fatal(err)
					}
					if err := c.Call(ctx, "eth_getTransactionReceipt", &receipt, step.Hash); err != nil {
						t.Fatal(err)
					}
					observed = append(observed, map[string]any{"hash": step.Hash, "transaction": tx, "receipt": receipt})
				}
				proof[id] = map[string]any{"steps": filepath.Join(dir, "steps.json"), "assertions": filepath.Join(dir, "assert.json"), "transactions": observed}
				quantity := func(m map[string]any, k string) uint64 {
					s, _ := m[k].(string)
					v, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
					if err != nil || !strings.HasPrefix(s, "0x") {
						t.Fatalf("missing or invalid %s: %v", k, m)
					}
					return v
				}
				check := func(p map[string]any, nonce uint64) (uint64, uint64) {
					tx, _ := p["transaction"].(map[string]any)
					r, _ := p["receipt"].(map[string]any)
					if tx["hash"] != p["hash"] || r["transactionHash"] != p["hash"] || tx["blockHash"] != r["blockHash"] || r["status"] != "0x1" || quantity(tx, "nonce") != nonce || tx["blockNumber"] != r["blockNumber"] || tx["transactionIndex"] != r["transactionIndex"] {
						t.Fatal("actual mined transaction mismatch", p)
					}
					return quantity(r, "blockNumber"), quantity(r, "transactionIndex")
				}
				if id == "nonce-ordering" {
					if len(observed) != 4 {
						t.Fatal("not funding + three reversed submissions", len(observed))
					}
					var block, index uint64
					for n, p := range []map[string]any{observed[3], observed[2], observed[1]} {
						b, j := check(p, uint64(n))
						if n > 0 && (b < block || b == block && j <= index) {
							t.Fatal("actual mined order reversed or duplicated", observed)
						}
						block, index = b, j
					}
				} else {
					if len(observed) != 5 {
						t.Fatal("missing replacement or gap transactions", observed)
					}
					oldTx, _ := observed[1]["transaction"].(map[string]any)
					oldReceipt, _ := observed[1]["receipt"].(map[string]any)
					if oldTx != nil || oldReceipt != nil {
						t.Fatal("evicted hash remains or missing gap transaction", observed)
					}
					check(observed[2], 2)
					check(observed[3], 0)
					check(observed[4], 1)
				}
			}
		})
	}
}
