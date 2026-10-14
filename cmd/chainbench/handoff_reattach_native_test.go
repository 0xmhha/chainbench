//go:build e2e

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
)

func TestNativeHandoffWorkspaceReattachmentRetainsChainAndProduction(t *testing.T) {
	if os.Getenv("CHAINBENCH_HANDOFF_REATTACH_PROOF") != "1" {
		t.Skip("set CHAINBENCH_HANDOFF_REATTACH_PROOF=1 for the real workspace reattachment proof")
	}
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Fatal("both handoff binaries are required")
	}
	w, _, _ := runHandoffKeepDatadir(t, from, to)
	root := filepath.Dir(w)
	fork, err := verb.ChainFork(t.Context(), chainsetup.Deps{}, verb.ChainForkIn{DataDir: w})
	if err != nil || !fork.HandedOver {
		t.Fatalf("handover not retained: %+v %v", fork, err)
	}
	before, err := verb.NetworkStatus(t.Context(), chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: w})
	if err != nil {
		t.Fatal(err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"run", filepath.Join(root, "case.json"), "--attach", "--workspace-dir", w, "--artifact-root", filepath.Join(root, "reports", "reattach"), "--keys", presetKeysDir(t), "--read-only-keys", "--no-skips", "--no-node-control", "--json"})
	var output, stderr bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&stderr)
	runErr := cmd.Execute()
	if err := os.WriteFile(filepath.Join(root, "reattach.log"), append(output.Bytes(), stderr.Bytes()...), 0600); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatalf("reattachment: %v %s %s", runErr, &output, &stderr)
	}
	var result struct {
		Session string
		Summary struct{ Pass, Fail, Blocked, Skip int }
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Summary.Pass != 1 || result.Summary.Fail != 0 || result.Summary.Blocked != 0 || result.Summary.Skip != 0 {
		t.Fatalf("incomplete reattachment: %s", &output)
	}
	after, err := verb.NetworkStatus(t.Context(), chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: w})
	if err != nil || len(before.Nodes.Nodes) != len(after.Nodes.Nodes) {
		t.Fatal("node table changed", err)
	}
	for i, n := range before.Nodes.Nodes {
		if n.PID != after.Nodes.Nodes[i].PID || n.Role != after.Nodes.Nodes[i].Role || n.Chain != after.Nodes.Nodes[i].Chain {
			t.Fatal("read-only reattachment changed a node", n, after.Nodes.Nodes[i])
		}
	}
	probePath := filepath.Join(result.Session, "tests", "001_gov-handoff", "observations", "capability-probes.json")
	data, err := os.ReadFile(probePath)
	if err != nil {
		t.Fatal(err)
	}
	var probes struct {
		Status string
		Nodes  []struct{ Chain, Status string }
	}
	if err := json.Unmarshal(data, &probes); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, n := range probes.Nodes {
		if n.Status != "supported" {
			t.Fatal("unconfirmed node", n)
		}
		counts[n.Chain]++
	}
	if probes.Status != "supported" || counts["wbft"] != 4 || counts["wemix"] != 1 {
		t.Fatal("mixed chain provenance missing", counts, probes.Status)
	}
	sources := map[string]string{}
	for _, p := range []string{"handoff_reattach_native_test.go", "handoff_runtime_test.go", "upgrade_data_migration_e2e_test.go", "../../internal/testengine/fork_record.go", "../../internal/testengine/attach_workspace.go", "../../internal/chainsetup/verb/verbs_network.go"} {
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		sources[abs] = hex.EncodeToString(sum[:])
	}
	proof := map[string]any{"runtime": root, "session": result.Session, "handedOver": true, "passed": 1, "skipped": 0, "unchangedNodes": len(after.Nodes.Nodes), "consensusChains": counts, "sources": sources}
	data, err = json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "reports", "reattach-proof.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("handoff reattachment proof:", path)
}
