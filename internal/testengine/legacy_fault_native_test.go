package testengine_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// Each public case owns a new BP4 network. The recorded progress values and
// independent live RPC observations establish more than a passing summary.
func TestNativeLegacyFaultRecovery(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, family := range families {
		if os.Getenv(family.variable) == "" {
			t.Skip("native fault recovery requires " + family.variable)
		}
	}
	cases := []struct {
		file, id             string
		progress, assertions int
	}{
		{"CT-FAULT-001-producer-crash-and-restart.json", "fault-node-crash", 10, 8},
		{"CT-FAULT-002-node-recover-and-sync.json", "fault-node-recover", 20, 3},
	}
	for _, family := range families {
		for _, testcase := range cases {
			if !t.Run(family.chain+"/"+testcase.id, func(t *testing.T) {
				root := failureRuntime(t)
				ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
				defer cancel()
				path := filepath.Join(repoRoot(t), "tests/tc/common/fault", testcase.file)
				original, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(root, testcase.file), original, 0600); err != nil {
					t.Fatal(err)
				}
				raw, err := dsl.ReadFilesWithChainPreset([]string{path}, family.chain+"-bp4")
				if err != nil {
					t.Fatal(err)
				}
				input := testengine.RunSuiteIn{SpecContent: raw, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(family.variable), KeysSource: "generate", KeysDir: filepath.Join(root, "w/keys"), ReadOnlyKeys: true, KeepUp: true, LaunchOpts: []string{"cache=64"}, NodeMonitorTimeout: 5 * time.Minute}
				up, plan, err := testengine.PlanSuiteLayout(ctx, input)
				if err != nil || up.KeysSource != "generate" || plan.Keys.Source != "generate" || plan.Keys.Dir != input.KeysDir {
					t.Fatal("refuse identities outside owned workspace", plan.Keys, err)
				}
				out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, input)
				proof := map[string]any{"chain": family.chain, "case": testcase.file, "binary": input.Binary, "session": out.SessionRoot, "summary": out.Summary, "scope": "Two public recovery cases per family, relative10/20-block progress, current recovery checkpoint, common block/stateRoot and generated private keys. Not full temporal recovery equivalence, whole legacy/L1, W3/W4 resolution or final Web acceptance.", "fullL1Accepted": false, "all14Accepted": false}
				defer func() {
					b, e := json.MarshalIndent(proof, "", "  ")
					if e != nil {
						t.Error(e)
						return
					}
					if e = os.WriteFile(filepath.Join(root, "reports/legacy-fault-proof.json"), b, 0600); e != nil {
						t.Error(e)
					}
				}()
				if runErr != nil {
					proof["runError"] = runErr.Error()
					t.Fatal(runErr)
				}
				if out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 || len(out.Summary.Tests) != 1 || out.Summary.Tests[0].ID != testcase.id {
					t.Fatal("expected exactly one public pass", out.Summary)
				}
				assertPath := filepath.Join(out.SessionRoot, "tests", "001_"+testcase.id, "assert.json")
				body, err := os.ReadFile(assertPath)
				if err != nil {
					t.Fatal(err)
				}
				var assertions []session.AssertResult
				decoder := json.NewDecoder(bytes.NewReader(body))
				decoder.UseNumber()
				if err = decoder.Decode(&assertions); err != nil {
					t.Fatal(err)
				}
				if len(assertions) != testcase.assertions {
					t.Fatal("missing public checkpoint assertion", len(assertions), testcase.assertions)
				}
				var progressedTo uint64
				progressChecks, lagChecks := 0, 0
				for _, assertion := range assertions {
					if !assertion.Pass {
						t.Fatal("failed public assertion", assertion)
					}
					if assertion.Assert != "derive" || assertion.Provenance["op"] != "diff" {
						continue
					}
					actual, ok := new(big.Int).SetString(fmt.Sprint(assertion.Actual), 10)
					if !ok {
						t.Fatal("unusable numeric observation", assertion)
					}
					switch assertion.Provenance["compare"] {
					case "GreaterOrEqual":
						values, ok := assertion.Provenance["of"].([]any)
						if !ok || len(values) != 2 {
							t.Fatal("missing actual progress endpoints", assertion)
						}
						a, aok := new(big.Int).SetString(fmt.Sprint(values[0]), 10)
						b, bok := new(big.Int).SetString(fmt.Sprint(values[1]), 10)
						if !aok || !bok || !a.IsUint64() || a.Cmp(b) < 0 || new(big.Int).Sub(a, b).Cmp(actual) != 0 || actual.Cmp(big.NewInt(int64(testcase.progress))) < 0 {
							t.Fatal("relative progress not established", assertion)
						}
						progressedTo = a.Uint64()
						progressChecks++
						proof["relativeProgress"] = assertion
					case "LessOrEqual":
						if actual.Cmp(big.NewInt(2)) > 0 {
							t.Fatal("public recovery head lag too large", assertion)
						}
						lagChecks++
					}
				}
				wantLags := 1
				if testcase.progress == 10 {
					wantLags = 4
				}
				if progressChecks != 1 || lagChecks != wantLags || progressedTo < uint64(testcase.progress) {
					t.Fatal("incomplete public progress/lag predicates", progressChecks, lagChecks, progressedTo)
				}
				state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: input.DataDir})
				if err != nil || len(state.Nodes.Nodes) != 4 {
					t.Fatal("missing actual BP4", err)
				}
				heads := map[int]uint64{}
				observations := []map[string]any{}
				minimum, maximum := ^uint64(0), uint64(0)
				for _, n := range state.Nodes.Nodes {
					head, e := rpc.Dial(n.RPCURL).BlockNumber(ctx)
					if e != nil || head < progressedTo && progressedTo-head > 2 {
						t.Fatal("live node lost observed progress", n.Index, head, progressedTo, e)
					}
					if _, duplicate := heads[n.Index]; duplicate {
						t.Fatal("duplicate observed node index", n.Index)
					}
					heads[n.Index] = head
					minimum = min(minimum, head)
					maximum = max(maximum, head)
					observations = append(observations, map[string]any{"node": n.Index, "rpcURL": n.RPCURL, "head": head})
				}
				proof["liveNodes"] = observations
				primary, hasPrimary := heads[1]
				recovered, hasRecovered := heads[3]
				if !hasPrimary || !hasRecovered {
					t.Fatal("independent observations omitted primary or recovered node", heads)
				}
				if testcase.progress == 10 && maximum-minimum > 2 || testcase.progress == 20 && primary > recovered && primary-recovered > 2 {
					t.Fatal("independent live recovery lag exceeded 2", heads)
				}
				var wantHash, wantRoot string
				for i, n := range state.Nodes.Nodes {
					var block struct{ Number, Hash, StateRoot string }
					tag := fmt.Sprintf("0x%x", minimum)
					if e := rpc.Dial(n.RPCURL).Call(ctx, "eth_getBlockByNumber", &block, tag, false); e != nil || block.Number != tag || !regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`).MatchString(block.Hash) || !regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`).MatchString(block.StateRoot) {
						t.Fatal("missing independent common block", block, e)
					}
					if wantHash != "" && (block.Hash != wantHash || block.StateRoot != wantRoot) {
						t.Fatal("independent common block/stateRoot differs", block)
					}
					wantHash, wantRoot = block.Hash, block.StateRoot
					observations[i]["commonBlock"] = block
				}
				proof["independentRecoveryVerified"] = true
				proof["publicAssertions"] = assertPath
			}) {
				t.Fatal("stop after preceding proof or cleanup failed")
			}
		}
	}
}
