//go:build e2e

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestWemixPostforkPublicCasesE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	workspace, url := runGovHandoffWorkspace(t, from, to, writeHandoffCase(t, nil))
	root := filepath.Dir(workspace)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil {
		t.Fatal(err)
	}
	successors := node.NodeSet{Chain: "wbft", Network: "post-handoff-successors"}
	for _, n := range state.Nodes.Nodes {
		if n.Chain == "wbft" {
			successors.Nodes = append(successors.Nodes, n)
		}
	}
	if len(state.Nodes.Nodes) != 5 || len(successors.Nodes) != 4 {
		t.Fatal("not four recorded successors and one retired producer", state.Nodes)
	}
	for i, n := range successors.Nodes {
		if n.Index != i+1 || n.Role != node.RoleBP || n.RPCURL == "" {
			t.Fatal("successor identity/role missing", n)
		}
	}
	plugin, err := registry.Get("wbft")
	if err != nil {
		t.Fatal(err)
	}
	caps := plugin.Manifest().Capabilities
	successors.Capabilities = caps
	paths := []string{
		"common/tx/CT-TX-005-fee-delegated-transfer.json",
		"go-wbft/accounts/01-secp256r1-precompile-valid.json",
		"common/tx/CT-TX-015-insufficient-funds-rejected.json",
		"common/tx/CT-TX-007-fd-feepayer-sig-tampered-rejected.json",
		"go-wbft/accounts/03-secp256r1-precompile-short-input.json",
	}
	raws := [][]byte{}
	originals := []map[string]any{}
	for i, p := range paths {
		path := filepath.Join("../../tests/tc", p)
		original := readPostforkPublicJSON[map[string]any](t, path)
		originalBytes, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		resolved, e := dsl.ReadFilesWithChainPreset([]string{path}, "../../presets/chain/wbft-bp4.json")
		if e != nil || len(resolved) != 1 {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, fmt.Sprintf("public-source-%d.json", i+1)), originalBytes, 0600); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, fmt.Sprintf("public-resolved-%d.json", i+1)), resolved[0], 0600); e != nil {
			t.Fatal(e)
		}
		var doc map[string]any
		if e = json.Unmarshal(resolved[0], &doc); e != nil || doc["id"] != postforkPublicIDs[i] || !reflect.DeepEqual(doc["steps"], original["steps"]) {
			t.Fatal("public steps changed", e, p)
		}
		raws = append(raws, resolved...)
		originals = append(originals, original)
	}
	engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "wbft", Nodes: successors, Caps: caps, KeysDir: presetKeysDir(t), ReadOnlyKeys: true, ArtifactRoot: filepath.Join(root, "reports/postfork")})
	if err != nil {
		t.Fatal(err)
	}
	proof := map[string]any{"workspace": workspace, "allNodes": state.Nodes, "successors": successors, "declaredCapabilities": caps, "sources": paths, "scope": "Five unmodified public cases on four recorded WBFT successors after WEMIX-to-WBFT Croissant20. Retired WEMIX producer excluded. Not mixed-network capability support, invalid-signature TX019, full87, mainnet, Linux, legacy or final Web."}
	defer func() {
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/postfork-public-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	sessionRoot, err := engine.Run(ctx, raws)
	proof["session"] = sessionRoot
	if err != nil {
		t.Fatal(err)
	}
	summary, err := testengine.ReadSessionSummary(sessionRoot)
	proof["summary"] = summary
	if err != nil {
		t.Fatal(err)
	}
	if err = verifyPostforkPublicSummary(summary); err != nil {
		t.Fatal(err, summary, sessionRoot)
	}
	c := rpc.Dial(url)
	cases := []map[string]any{}
	for i, id := range postforkPublicIDs {
		dir := filepath.Join(sessionRoot, "tests", fmt.Sprintf("%03d_%s", i+1, id))
		recorded, err := os.ReadFile(filepath.Join(dir, "spec.json"))
		if err != nil {
			t.Fatal(err)
		}
		original, err := json.Marshal(originals[i])
		if err != nil {
			t.Fatal(err)
		}
		if err := verifyPostforkRecordedSpec(original, raws[i], recorded); err != nil {
			t.Fatal("recorded public case differs", id, err)
		}
		expectedAsserts, expectedActions := 0, 0
		for _, raw := range originals[i]["steps"].([]any) {
			s := raw.(map[string]any)
			if _, ok := s["do"]; ok {
				expectedActions++
			} else if _, ok = s["expect"]; ok {
				expectedAsserts++
			}
		}
		assertions := readPostforkPublicJSON[[]session.AssertResult](t, filepath.Join(dir, "assert.json"))
		if len(assertions) != expectedAsserts {
			t.Fatal("missing individual public assertions", id)
		}
		for _, a := range assertions {
			if !a.Pass {
				t.Fatal("failed public assertion", id, a)
			}
		}
		var steps []session.StepResult
		if expectedActions > 0 {
			steps = readPostforkPublicJSON[[]session.StepResult](t, filepath.Join(dir, "steps.json"))
		}
		if len(steps) != expectedActions {
			t.Fatal("missing public action trace", id)
		}
		transactions := []map[string]any{}
		for _, s := range steps {
			if s.Error != "" {
				t.Fatal("public action failed", id, s)
			}
			if s.Type != "sendTx" || s.Hash == "" {
				continue
			}
			var tx, receipt map[string]any
			if err = c.Call(ctx, "eth_getTransactionByHash", &tx, s.Hash); err != nil {
				t.Fatal(err)
			}
			if err = c.Call(ctx, "eth_getTransactionReceipt", &receipt, s.Hash); err != nil {
				t.Fatal(err)
			}
			if tx["hash"] != s.Hash || receipt["transactionHash"] != s.Hash || receipt["status"] != "0x1" || tx["blockHash"] != receipt["blockHash"] || tx["blockNumber"] != receipt["blockNumber"] {
				t.Fatal("public transaction not actually mined", id, tx, receipt)
			}
			transactions = append(transactions, map[string]any{"transaction": tx, "receipt": receipt})
		}
		cases = append(cases, map[string]any{"id": id, "directory": dir, "assertions": len(assertions), "actions": len(steps), "transactions": transactions})
	}
	proof["cases"] = cases
}

func readPostforkPublicJSON[T any](t *testing.T, path string) T {
	t.Helper()
	var value T
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
