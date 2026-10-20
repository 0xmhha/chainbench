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

const regularEGPFeeCap = "300000000000000"

// This is a distinguishing observation, not acceptance of the legacy formula.
// A requested tx tip above header tip prevents their accidental equality.
func TestNativeRegularEGPDistinguishingWitness(t *testing.T) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("requires GSTABLE_BIN")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Minute)
	defer cancel()
	source := filepath.Join(repoRoot(t), "tests/tc/common/fee/CT-FEE-009-snap-receipt-gas-price.json")
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal(original, &doc); err != nil {
		t.Fatal(err)
	}
	const id = "regular-egp-distinguishing-witness"
	doc["id"] = id
	doc["description"] = "Observation derived from public CT-FEE-009 with requested priority tip5x+1 and fixed high fee cap. Retains reset/genesis,150-block pivot and actual receipt-download metric. Does not change or accept legacy regular EGP formula."
	steps := []any{}
	for _, raw := range doc["steps"].([]any) {
		s := raw.(map[string]any)
		if s["save"] == "feeCap" {
			s["of"] = []any{regularEGPFeeCap}
		}
		if s["do"] == "sendTx" && s["save"] == "h" {
			s["maxPriorityFeePerGas"] = "$highTip"
		}
		steps = append(steps, s)
		if s["save"] == "tip" {
			steps = append(steps, map[string]any{"do": "read", "source": "derive", "op": "sum", "of": []any{"$tip", "$tip", "$tip", "$tip", "$tip", "1"}, "save": "highTip"})
		}
	}
	doc["steps"] = steps
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	for name, b := range map[string][]byte{"public-source.json": original, "witness-input.json": raw} {
		if err = os.WriteFile(filepath.Join(root, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	resolved, err := dsl.ReadContent(raw, filepath.Dir(source))
	if err != nil {
		t.Fatal(err)
	}
	wdir := filepath.Join(root, "w")
	in := testengine.RunSuiteIn{SpecContent: [][]byte{resolved}, DataDir: wdir, ArtifactRoot: filepath.Join(root, "reports"), Binary: binary, KeysSource: "generate", KeysDir: filepath.Join(wdir, "keys"), ReadOnlyKeys: true, KeepUp: true, LaunchOpts: []string{"cache=64"}, NodeMonitorTimeout: 5 * time.Minute}
	up, plan, err := testengine.PlanSuiteLayout(ctx, in)
	if err != nil || up.KeysSource != "generate" || plan.Keys.Source != "generate" || plan.Keys.Dir != in.KeysDir {
		t.Fatal("unowned keys", err)
	}
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, in)
	proof := map[string]any{"scope": "Distinguishing regular-account EGP observation on owned BP4+snap EN1. Original formula acceptance requires user decision. Not complete legacy/L1/Web acceptance.", "session": out.SessionRoot, "summary": out.Summary, "fullLegacyAccepted": false, "fullL1Accepted": false, "all14Accepted": false}
	defer func() {
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/regular-egp-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	if runErr != nil {
		proof["runError"] = runErr.Error()
		t.Fatal(runErr)
	}
	if out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 || len(out.Summary.Tests) != 1 || out.Summary.Tests[0].ID != id {
		t.Fatal("missing exact witness pass", out.Summary)
	}
	dir := filepath.Join(out.SessionRoot, "tests/001_"+id)
	var actions []session.StepResult
	var assertions []session.AssertResult
	for name, dst := range map[string]any{"steps.json": &actions, "assert.json": &assertions} {
		b, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(b, dst); e != nil {
			t.Fatal(e)
		}
	}
	if len(assertions) != 4 {
		t.Fatal("expected unchanged4 snap assertions", len(assertions))
	}
	metric := uint64(0)
	for _, a := range assertions {
		if !a.Pass {
			t.Fatal(a)
		}
		if a.Assert == "metric" {
			v, ok := a.Actual.(float64)
			if !ok || v < 1 || v != float64(uint64(v)) {
				t.Fatal("no actual receipt metric", a)
			}
			metric = uint64(v)
		}
	}
	hashes := []string{}
	for _, a := range actions {
		if a.Type == "sendTx" {
			if a.Hash == "" || a.Receipt["status"] != "0x1" {
				t.Fatal("missing real transaction", a)
			}
			hashes = append(hashes, a.Hash)
		}
	}
	if len(hashes) != 2 {
		t.Fatal("expected funding+subject", hashes)
	}
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: wdir})
	if err != nil || len(state.Nodes.Nodes) != 5 {
		t.Fatal("missing owned BP4+EN", err)
	}
	var bp, en *rpc.Client
	for _, n := range state.Nodes.Nodes {
		if n.Index == 1 {
			bp = rpc.Dial(n.RPCURL)
		}
		if string(n.Role) == "en" {
			en = rpc.Dial(n.RPCURL)
		}
	}
	if bp == nil || en == nil {
		t.Fatal("missing endpoints")
	}
	query := func(c *rpc.Client, method string, args ...any) map[string]any {
		var m map[string]any
		if e := c.Call(ctx, method, &m, args...); e != nil || m == nil {
			t.Fatal(method, e)
		}
		return m
	}
	tx := query(bp, "eth_getTransactionByHash", hashes[1])
	receipt := query(bp, "eth_getTransactionReceipt", hashes[1])
	block := fmt.Sprint(receipt["blockNumber"])
	funding := query(bp, "eth_getTransactionByHash", hashes[0])
	if !strings.EqualFold(fmt.Sprint(funding["to"]), fmt.Sprint(tx["from"])) {
		t.Fatal("generated funded sender not bound")
	}
	authData := "0xfe9fbb80" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(fmt.Sprint(tx["from"])), "0x")
	var auth string
	if err = bp.Call(ctx, "eth_call", &auth, map[string]any{"to": "0x0000000000000000000000000000000000b00003", "data": authData}, block); err != nil {
		t.Fatal(err)
	}
	w := regularEGPWitness{Sender: fmt.Sprint(funding["to"]), Authorization: auth, Transaction: tx, BPReceipt: receipt, SnapReceipt: query(en, "eth_getTransactionReceipt", hashes[1]), Block: query(bp, "eth_getBlockByNumber", block, false), SnapBlock: query(en, "eth_getBlockByNumber", block, false), Extra: query(bp, "istanbul_getWbftExtraInfo", block), ReceiptMetric: metric}
	proof["witness"] = w
	proof["fundingTransaction"] = funding
	proof["authorizationCall"] = map[string]any{"to": "0x0000000000000000000000000000000000b00003", "data": authData, "block": block, "result": auth}
	proof["assertions"] = assertions
	proof["nodes"] = state.Nodes
	prices, err := verifyRegularEGPWitness(w)
	if err != nil {
		t.Fatal("distinction not proved", err)
	}
	proof["prices"] = prices
	proof["originalFormulaConflictObserved"] = true
	t.Logf("actual EGP %s, original formula %s, txTip %s > headerTip %s", prices["observed"], prices["original"], prices["txTip"], prices["headerTip"])
}
