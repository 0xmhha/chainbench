package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func TestNativePublicCoreCases(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("all native binaries required for public core cases")
		}
	}
	paths := []string{"rpc/CT-RPC-001-block-number-advances.json", "rpc/CT-RPC-006-transaction-count-increments.json", "rpc/CT-RPC-007-balance-query.json", "fee/CT-FEE-010-suggested-gas-price.json", "contract/CT-CONTRACT-001-deploy-and-call.json", "contract/CT-CONTRACT-002-storage-write-and-read.json", "contract/CT-CONTRACT-003-view-call-leaves-state.json"}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			preset := f.chain + "-bp4-en1"
			raws := [][]byte{}
			for i, p := range paths {
				path := filepath.Join(repoRoot(t), "tests/tc/common", p)
				raw, e := dsl.ReadFilesWithChainPreset([]string{path}, filepath.Join(repoRoot(t), "presets/chain", preset+".json"))
				if e != nil {
					t.Fatal(e)
				}
				if e = os.WriteFile(filepath.Join(root, fmt.Sprintf("case%d.json", i+1)), raw[0], 0600); e != nil {
					t.Fatal(e)
				}
				raws = append(raws, raw...)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "scope": "Fresh seven public core cases on managed BP4 EN1 native families. Direct WBFT Croissant0; not full87/mainnet/Linux/legacy/staged handoff/final Web."}
			defer func() {
				b, e := json.MarshalIndent(proof, "", "  ")
				if e != nil {
					t.Error(e)
					return
				}
				if e = os.WriteFile(filepath.Join(root, "reports/public-core-proof.json"), b, 0600); e != nil {
					t.Error(e)
				}
			}()
			if runErr != nil {
				t.Fatal(runErr)
			}
			if e := verifyPublicCoreWitness(preset, out.Summary); e != nil {
				t.Fatal(e, out.Summary)
			}
			state, e := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if e != nil || len(state.Nodes.Nodes) != 5 {
				t.Fatal("missing actual BP4 EN1", e)
			}
			c := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
			var head, price string
			if e = c.Call(ctx, "eth_blockNumber", &head); e != nil {
				t.Fatal(e)
			}
			if e = c.Call(ctx, "eth_gasPrice", &price); e != nil {
				t.Fatal(e)
			}
			proof["head"] = head
			proof["gasPrice"] = price
			observed := map[string]any{}
			for i, id := range publicCoreIDs {
				dir := filepath.Join(out.SessionRoot, "tests", fmt.Sprintf("%03d_%s", i+1, id))
				row := map[string]any{"assertions": filepath.Join(dir, "assert.json"), "spec": filepath.Join(dir, "spec.json")}
				var steps []session.StepResult
				b, e := os.ReadFile(filepath.Join(dir, "steps.json"))
				if e == nil {
					if e = json.Unmarshal(b, &steps); e != nil {
						t.Fatal(e)
					}
					row["steps"] = filepath.Join(dir, "steps.json")
				} else if !os.IsNotExist(e) {
					t.Fatal(e)
				}
				receipts := []map[string]any{}
				for _, st := range steps {
					if st.Type != "sendTx" && st.Type != "deployContract" {
						continue
					}
					var tx, r map[string]any
					if e = c.Call(ctx, "eth_getTransactionByHash", &tx, st.Hash); e != nil {
						t.Fatal(e)
					}
					if e = c.Call(ctx, "eth_getTransactionReceipt", &r, st.Hash); e != nil {
						t.Fatal(e)
					}
					if st.Hash == "" || tx["hash"] != st.Hash || r["transactionHash"] != st.Hash || r["status"] != "0x1" || tx["blockHash"] != r["blockHash"] || tx["blockNumber"] != r["blockNumber"] {
						t.Fatal("actual core transaction/receipt mismatch", id, tx, r)
					}
					receipts = append(receipts, map[string]any{"transaction": tx, "receipt": r})
				}
				row["transactions"] = receipts
				if strings.HasPrefix(id, "contract-") || id == "storage-write-and-read" || id == "view-call-leaves-state" {
					if len(receipts) == 0 {
						t.Fatal("missing actual deployment", id)
					}
					deployment := receipts[0]["receipt"].(map[string]any)
					addr, ok := deployment["contractAddress"].(string)
					if !ok || len(addr) != 42 {
						t.Fatal("missing deployed address", id)
					}
					var code, output string
					if e = c.Call(ctx, "eth_getCode", &code, addr, "latest"); e != nil || code == "0x" {
						t.Fatal("missing actual deployed code", e, id)
					}
					data := "0x00"
					if id == "contract-roundtrip" {
						data = "0x"
					}
					if e = c.Call(ctx, "eth_call", &output, map[string]string{"to": addr, "data": data}, "latest"); e != nil || output != "0x"+strings.Repeat("0", 62)+"2a" {
						t.Fatal("actual core contract not42", e, id, output)
					}
					row["contract"] = addr
					row["code"] = code
					row["viewOutput"] = output
					if id != "contract-roundtrip" {
						var stored string
						if e = c.Call(ctx, "eth_getStorageAt", &stored, addr, "0x0", "latest"); e != nil || stored != output {
							t.Fatal("actual storage/view differs", e, id)
						}
						row["slot0"] = stored
					}
				}
				observed[id] = row
			}
			proof["cases"] = observed
		})
	}
}
