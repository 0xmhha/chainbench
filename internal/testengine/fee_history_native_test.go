package testengine_test

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativeFeeHistoryArrays(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native history requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			p := filepath.Join(repoRoot(t), "tests/tc/common/fee/CT-FEE-012-fee-history.json")
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(root, "case.json"), b, 0600); err != nil {
				t.Fatal(err)
			}
			raws, err := dsl.ReadFilesWithChainPreset([]string{p}, filepath.Join(repoRoot(t), "presets/chain", f.chain+"-bp4.json"))
			if err != nil {
				t.Fatal(err)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "scope": "Managed local public RPC-017 fixture, not complete87/mainnet/Linux/final Web acceptance."}
			defer func() {
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/fee-history-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("public history failed: %v %+v", runErr, out.Summary)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(status.Nodes.Nodes) < 4 {
				t.Fatal("missing actual nodes", err)
			}
			var history struct {
				OldestBlock   string     `json:"oldestBlock"`
				BaseFeePerGas []string   `json:"baseFeePerGas"`
				GasUsedRatio  []float64  `json:"gasUsedRatio"`
				Reward        [][]string `json:"reward"`
			}
			params := []any{"0xa", "latest", []int{25, 75}}
			if err = rpc.Dial(status.Nodes.Nodes[0].RPCURL).Call(ctx, "eth_feeHistory", &history, params...); err != nil {
				t.Fatal(err)
			}
			quantity := func(s string) *big.Int {
				n, ok := new(big.Int).SetString(s, 0)
				if !ok {
					t.Fatalf("bad quantity %q", s)
				}
				return n
			}
			if quantity(history.OldestBlock).Sign() < 0 || len(history.BaseFeePerGas) != 11 || len(history.GasUsedRatio) != 10 || len(history.Reward) != 10 {
				t.Fatal("wrong history shape", history)
			}
			for _, base := range history.BaseFeePerGas {
				if quantity(base).Sign() <= 0 {
					t.Fatal("nonpositive baseFee", base)
				}
			}
			for i, ratio := range history.GasUsedRatio {
				if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 0 || ratio > 1 || len(history.Reward[i]) != 2 {
					t.Fatal("wrong ratio/reward", i, history)
				}
				for _, reward := range history.Reward[i] {
					if quantity(reward).Sign() < 0 {
						t.Fatal("negative reward")
					}
				}
			}
			proof["params"] = params
			proof["history"] = history
			proof["assertions"] = filepath.Join(out.SessionRoot, "tests/001_fee-history-well-formed/assert.json")
			proof["steps"] = filepath.Join(out.SessionRoot, "tests/001_fee-history-well-formed/steps.json")
		})
	}
}
