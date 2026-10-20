package testengine_test

import (
	"context"
	"encoding/json"
	"errors"
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

func TestNativeRPCMethodPresence(t *testing.T) {
	families := []struct {
		chain, variable string
		id              uint64
	}{{"stablenet", "GSTABLE_BIN", 8283}, {"wbft", "GWBFT_BIN", 8284}, {"wemix", "GWEMIX_BIN", 8285}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native method presence proof requires " + f.variable)
		}
	}
	for _, f := range families {
		if !t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			name := "CT-TX-009-fee-delegate-sign-rpc-present.json"
			path := filepath.Join(repoRoot(t), "tests/tc/common/tx", name)
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(root, name), b, 0600); err != nil {
				t.Fatal(err)
			}
			raw, err := dsl.ReadFilesWithChainPreset([]string{path}, f.chain+"-bp4")
			if err != nil {
				t.Fatal(err)
			}
			input := testengine.RunSuiteIn{SpecContent: raw, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysSource: "generate", KeysDir: filepath.Join(root, "w/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2, LaunchOpts: []string{"cache=64"}}
			up, plan, err := testengine.PlanSuiteLayout(ctx, input)
			if err != nil || up.KeysSource != "generate" || plan.Keys.Source != "generate" || plan.Keys.Dir != input.KeysDir {
				t.Fatal("refuse keys outside owned workspace", plan.Keys, err)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, input)
			observations := []map[string]any{}
			proof := map[string]any{"chain": f.chain, "binary": input.Binary, "plannedKeysDir": plan.Keys.Dir, "session": out.SessionRoot, "summary": out.Summary, "scope": "One unchanged public method-presence case on each owned BP4 family, plus four direct typed argument-refusal probes per family; not complete signing behavior, whole R3/R5/L1/legacy/87 or final Web acceptance."}
			defer func() {
				proof["observations"] = observations
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/rpc-method-presence-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil {
				proof["runError"] = runErr.Error()
				t.Fatal(runErr)
			}
			if out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 || len(out.Summary.Tests) != 1 || out.Summary.Tests[0].ID != "fee-delegate-sign-rpc-present" {
				t.Fatal("expected exactly one public pass", out.Summary)
			}
			state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: input.DataDir})
			if err != nil || len(state.Nodes.Nodes) != 4 {
				t.Fatal("missing actual BP4", err)
			}
			for _, node := range state.Nodes.Nodes {
				client := rpc.Dial(node.RPCURL)
				id, err := client.ChainID(ctx)
				if err != nil || id != f.id {
					t.Fatal("wrong actual chain ID", id, err)
				}
				var result any
				method := "eth_signRawFeeDelegateTransaction"
				callErr := client.Call(ctx, method, &result, map[string]any{"from": "0x00000000000000000000000000000000C0FFEE08"}, "0x00")
				var rejection *rpc.ServerError
				if !errors.As(callErr, &rejection) || result != nil || rejection.Method != method || (rejection.Code != -32000 && rejection.Code != -32602) || rejection.Message == "" {
					t.Fatal("probe did not reach a registered method", result, callErr)
				}
				observations = append(observations, map[string]any{"rpcURL": node.RPCURL, "chainID": id, "result": result, "method": rejection.Method, "code": rejection.Code, "message": rejection.Message})
			}
		}) {
			t.Fatal("stop after preceding proof or cleanup failed")
		}
	}
}
