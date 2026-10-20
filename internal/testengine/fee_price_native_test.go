package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
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

// TestNativeFeePriceFields binds public legacy/dynamic fee assertions to the
// actual transaction and the exact block where it was included on each family.
func TestNativeFeePriceFields(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native price proof requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			raws := [][]byte{}
			for _, name := range []string{"CT-TX-002-legacy-transfer.json", "CT-TX-003-dynamic-fee-transfer.json"} {
				p := filepath.Join(repoRoot(t), "tests/tc/common/tx", name)
				b, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, name), b, 0600); err != nil {
					t.Fatal(err)
				}
				raw, err := dsl.ReadFilesWithChainPreset([]string{p}, filepath.Join(repoRoot(t), "presets/chain", f.chain+"-bp4.json"))
				if err != nil {
					t.Fatal(err)
				}
				raws = append(raws, raw...)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "scope": "Controlled public legacy/dynamic fee cases with node-suggested price/tip. Not arbitrary-price equivalence across families or all87/mainnet/Web acceptance."}
			if runErr != nil {
				proof["runError"] = runErr.Error()
			}
			defer func() {
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err := os.WriteFile(filepath.Join(root, "reports/fee-price-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 2 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("public fee cases failed: %v %+v", runErr, out.Summary)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(status.Nodes.Nodes) < 4 {
				t.Fatal("missing actual nodes", err)
			}
			c := rpc.Dial(status.Nodes.Nodes[0].RPCURL)
			for i, id := range []string{"legacy-value-transfer", "dynamic-fee-transfer"} {
				dir := filepath.Join(out.SessionRoot, "tests", fmt.Sprintf("%03d_%s", i+1, id))
				b, err := os.ReadFile(filepath.Join(dir, "steps.json"))
				if err != nil {
					t.Fatal(err)
				}
				var steps []session.StepResult
				if err := json.Unmarshal(b, &steps); err != nil {
					t.Fatal(err)
				}
				hash := ""
				for _, st := range steps {
					if st.Type == "sendTx" {
						hash = st.Hash
					}
				}
				var tx, receipt map[string]any
				if err := c.Call(ctx, "eth_getTransactionByHash", &tx, hash); err != nil {
					t.Fatal(err)
				}
				if err := c.Call(ctx, "eth_getTransactionReceipt", &receipt, hash); err != nil {
					t.Fatal(err)
				}
				if hash == "" || tx["hash"] != hash || receipt["transactionHash"] != hash || tx["blockHash"] != receipt["blockHash"] || receipt["status"] != "0x1" || tx["type"] != receipt["type"] {
					t.Fatal("actual fee transaction mismatch", tx, receipt)
				}
				quantity := func(m map[string]any, k string) *big.Int {
					s, _ := m[k].(string)
					n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
					if !ok || !strings.HasPrefix(s, "0x") {
						t.Fatalf("missing actual fee field %s", k)
					}
					return n
				}
				price := quantity(receipt, "effectiveGasPrice")
				fields := map[string]any{"steps": filepath.Join(dir, "steps.json"), "assertions": filepath.Join(dir, "assert.json"), "transaction": tx, "receipt": receipt}
				if i == 0 {
					if tx["type"] != "0x0" || price.Cmp(quantity(tx, "gasPrice")) != 0 {
						t.Fatal("legacy actual prices differ")
					}
				} else {
					block, err := c.BlockByNumber(ctx, fmt.Sprint(receipt["blockNumber"]))
					if err != nil || block.BaseFeePerGas == nil || block.Hash != receipt["blockHash"] {
						t.Fatal("missing exact fee block", err)
					}
					cap, tip := quantity(tx, "maxFeePerGas"), quantity(tx, "maxPriorityFeePerGas")
					headroom := new(big.Int).Sub(cap, block.BaseFeePerGas)
					if headroom.Sign() < 0 {
						t.Fatal("mined cap below block baseFee")
					}
					effectiveTip := new(big.Int).Set(tip)
					if headroom.Cmp(effectiveTip) < 0 {
						effectiveTip.Set(headroom)
					}
					expected := new(big.Int).Add(block.BaseFeePerGas, effectiveTip)
					if tx["type"] != "0x2" || expected.Cmp(price) != 0 {
						t.Fatal("actual EIP1559 capped price differs", expected, price)
					}
					fields["minedBlock"] = block
					fields["expectedPrice"] = expected.String()
					fields["capLimited"] = headroom.Cmp(tip) < 0
				}
				proof[id] = fields
			}
		})
	}
}
