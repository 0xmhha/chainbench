package testengine_test

import (
	"bytes"
	"context"
	"encoding/hex"
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

func TestNativeBurnRefundPublicCases(t *testing.T) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("native burn/refund proof requires GSTABLE_BIN")
	}
	names := []string{"03-burn-cancel-refundable.json", "04-burn-reject-refundable.json", "05-burn-expire-refundable.json", "06-burn-execute-no-refundable.json", "07-claim-burn-refund-succeeds.json", "08-claim-zero-refund-reverts.json", "09-claim-burn-refund-double-reverts.json"}
	var originals map[string]string
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "internal/testhelper/testdata/burn-proof-payloads.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &originals); err != nil {
		t.Fatal(err)
	}
	defaultAccounts := map[string]bool{}
	for _, data := range originals {
		if len(data) != 906 {
			t.Fatal("invalid original proof fixture")
		}
		defaultAccounts["0x"+strings.ToLower(data[10+2*64+24:10+3*64])] = true
	}
	for _, name := range names {
		if !t.Run(name, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			source := filepath.Join(repoRoot(t), "tests/tc/go-stablenet/post-v1.0.0-change/common-all", name)
			original, e := os.ReadFile(source)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(root, "public-source.json"), original, 0600); e != nil {
				t.Fatal(e)
			}
			var doc struct {
				ID    string
				Steps []map[string]any
			}
			if e = json.Unmarshal(original, &doc); e != nil {
				t.Fatal(e)
			}
			actions := []map[string]any{}
			wantAsserts := 0
			for _, s := range doc.Steps {
				if s["do"] != nil {
					actions = append(actions, s)
				} else if s["expect"] != nil {
					wantAsserts++
				}
			}
			raws, e := dsl.ReadFiles([]string{source})
			if e != nil {
				t.Fatal(e)
			}
			workspace := filepath.Join(root, "w")
			input := testengine.RunSuiteIn{SpecContent: raws, DataDir: workspace, ArtifactRoot: filepath.Join(root, "reports"), Binary: binary, KeysSource: "generate", KeysDir: filepath.Join(workspace, "keys"), ReadOnlyKeys: true, KeepUp: true, LaunchOpts: []string{"cache=64"}, NodeMonitorTimeout: 5 * time.Minute}
			up, plan, e := testengine.PlanSuiteLayout(ctx, input)
			if e != nil || up.KeysSource != "generate" || plan.Keys.Source != "generate" || plan.Keys.Dir != input.KeysDir {
				t.Fatal("refuse keys outside owned workspace", e)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, input)
			proof := map[string]any{"scope": "One unmodified public v2 burn/refund case on owned StableNet BP4 with generated private identities. Records exact assertions, real transaction/receipt inputs and actual proof caller. Original7-member quorum/Boho100 versus BP4/Boho0, short-expiry30/40s fixture remain explicit differences. Not whole legacy/L1/Web acceptance.", "case": name, "binary": binary, "session": out.SessionRoot, "summary": out.Summary, "expectedAssertions": wantAsserts, "expectedActions": len(actions), "keysSource": "generate", "fullLegacyAccepted": false, "fullL1Accepted": false, "all14Accepted": false}
			defer func() {
				body, e := json.MarshalIndent(proof, "", "  ")
				if e != nil {
					t.Error(e)
					return
				}
				if e = os.WriteFile(filepath.Join(root, "reports/burn-refund-proof.json"), body, 0600); e != nil {
					t.Error(e)
				}
			}()
			if runErr != nil {
				proof["runError"] = runErr.Error()
				t.Fatal(runErr)
			}
			if out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 || len(out.Summary.Tests) != 1 || out.Summary.Tests[0].ID != doc.ID {
				t.Fatal("expected one exact public pass", out.Summary)
			}
			directory := filepath.Join(out.SessionRoot, "tests", "001_"+doc.ID)
			var assertions []session.AssertResult
			raw, e := os.ReadFile(filepath.Join(directory, "assert.json"))
			if e != nil {
				t.Fatal(e)
			}
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if e = decoder.Decode(&assertions); e != nil || len(assertions) != wantAsserts {
				t.Fatal("missing actual public assertions", e, len(assertions), wantAsserts)
			}
			exactErrors := 0
			for _, a := range assertions {
				if !a.Pass {
					t.Fatal("failed public assertion", a)
				}
				if a.Assert == "rpcError" {
					actual, ok := a.Actual.(map[string]any)
					if !ok || actual["data"] != "0xe49c14cb" || fmt.Sprint(actual["code"]) != "3" || a.Provenance["errorData"] != "0xe49c14cb" {
						t.Fatal("specific NoRefundAvailable error not recorded", a)
					}
					exactErrors++
				}
			}
			wantErrors := 0
			if strings.HasPrefix(name, "08-") || strings.HasPrefix(name, "09-") {
				wantErrors = 1
			}
			if exactErrors != wantErrors {
				t.Fatal("missing custom error check", exactErrors, wantErrors)
			}
			var steps []session.StepResult
			raw, e = os.ReadFile(filepath.Join(directory, "steps.json"))
			if e != nil {
				t.Fatal(e)
			}
			if e = json.Unmarshal(raw, &steps); e != nil || len(steps) != len(actions) {
				t.Fatal("missing public action trace", e, len(steps), len(actions))
			}
			var keys struct {
				Nodes []struct {
					Index   int
					Address string
				}
			}
			raw, e = os.ReadFile(filepath.Join(workspace, "keys/metadata.json"))
			if e != nil {
				t.Fatal(e)
			}
			if e = json.Unmarshal(raw, &keys); e != nil || len(keys.Nodes) != 4 {
				t.Fatal("missing generated BP4 identities", e, len(keys.Nodes))
			}
			accounts := map[string]string{}
			for _, n := range keys.Nodes {
				if defaultAccounts[strings.ToLower(n.Address)] {
					t.Fatal("actual owned key is a default fixture identity")
				}
				accounts[fmt.Sprintf("node%d", n.Index)] = n.Address
			}
			if len(accounts) != 4 {
				t.Fatal("duplicate generated identities")
			}
			transactions := []map[string]any{}
			proposalCount := 0
			state, e := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
			if e != nil || len(state.Nodes.Nodes) != 4 {
				t.Fatal("recorded BP4 network missing", e)
			}
			var c *rpc.Client
			for _, n := range state.Nodes.Nodes {
				if n.Index == 1 {
					c = rpc.Dial(n.RPCURL)
				}
			}
			if c == nil {
				t.Fatal("recorded primary RPC missing")
			}
			proof["nodes"] = state.Nodes
			for i, s := range steps {
				if s.Error != "" {
					t.Fatal("public action failed", s.Error)
				}
				if s.Type != "sendTx" {
					continue
				}
				expectedRevert := actions[i]["expect"] == "revert"
				if s.Hash == "" {
					if !expectedRevert {
						t.Fatal("successful submission has no hash")
					}
					transactions = append(transactions, map[string]any{"action": i, "hashAbsent": true, "actualSubmissionRefusal": true, "noMinedFailureClaim": true})
					continue
				}
				var tx, receipt map[string]any
				if e = c.Call(ctx, "eth_getTransactionByHash", &tx, s.Hash); e != nil {
					t.Fatal(e)
				}
				if e = c.Call(ctx, "eth_getTransactionReceipt", &receipt, s.Hash); e != nil {
					t.Fatal(e)
				}
				wantedStatus := "0x1"
				if expectedRevert {
					wantedStatus = "0x0"
				}
				if tx["hash"] != s.Hash || receipt["transactionHash"] != s.Hash || tx["blockNumber"] != receipt["blockNumber"] || tx["blockHash"] != receipt["blockHash"] || receipt["status"] != wantedStatus {
					t.Fatal("actual mined transaction/receipt differs", tx, receipt, wantedStatus)
				}
				if label, ok := actions[i]["from"].(string); ok {
					if !strings.EqualFold(fmt.Sprint(tx["from"]), accounts[label]) {
						t.Fatal("sender not actual generated identity", label, tx["from"])
					}
				}
				data, _ := tx["input"].(string)
				if literal, ok := actions[i]["data"].(string); ok && !strings.HasPrefix(literal, "$") && data != literal {
					t.Fatal("actual literal calldata differs", i, data, literal)
				}
				var failureTrace any
				if expectedRevert {
					var trace struct {
						Failed      bool   `json:"failed"`
						ReturnValue string `json:"returnValue"`
					}
					if e = c.Call(ctx, "debug_traceTransaction", &trace, s.Hash, map[string]bool{"disableMemory": true, "disableStack": true, "disableStorage": true}); e != nil || !trace.Failed || strings.TrimPrefix(strings.ToLower(trace.ReturnValue), "0x") != "e49c14cb" {
						t.Fatal("actual failed transaction is not NoRefundAvailable", e, trace)
					}
					failureTrace = trace
				}
				if strings.HasPrefix(data, "0x64e2a8fc") {
					expected, e := hex.DecodeString(strings.TrimPrefix(originals[name], "0x"))
					if e != nil || len(expected) != 452 {
						t.Fatal("original fixture unavailable", e)
					}
					sender, e := hex.DecodeString(strings.TrimPrefix(fmt.Sprint(tx["from"]), "0x"))
					if e != nil || len(sender) != 20 {
						t.Fatal("actual transaction sender invalid", e)
					}
					copy(expected[4+2*32+12:4+3*32], sender)
					actual, e := hex.DecodeString(strings.TrimPrefix(data, "0x"))
					if e != nil || !bytes.Equal(actual, expected) {
						t.Fatal("actual proof differs beyond generated caller", e)
					}
					value, ok := new(big.Int).SetString(strings.TrimPrefix(fmt.Sprint(tx["value"]), "0x"), 16)
					if !ok || value.String() != "1000000000000000000" {
						t.Fatal("actual deposit differs", tx["value"])
					}
					proposalCount++
				}
				transactions = append(transactions, map[string]any{"action": i, "transaction": tx, "receipt": receipt, "failureTrace": failureTrace})
			}
			wantProposals := 1
			if strings.HasPrefix(name, "08-") {
				wantProposals = 0
			}
			if proposalCount != wantProposals {
				t.Fatal("actual proposals missing", proposalCount, wantProposals)
			}
			proof["assertions"] = assertions
			proof["transactions"] = transactions
			proof["generatedAccounts"] = accounts
			proof["nonDefaultKeysVerified"] = true
			proof["actualProofCallerVerified"] = true
			proof["customErrorAssertions"] = exactErrors
		}) {
			t.Fatal("stop after preceding case or cleanup failure")
		}
	}
}
