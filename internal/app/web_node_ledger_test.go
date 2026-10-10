package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/process"
)

func TestWebNodeExecutionRefusesLedgerPIDSubstitution(t *testing.T) {
	owned := exec.Command("/bin/sleep", "120")
	foreign := exec.Command("/bin/sleep", "121")
	for _, child := range []*exec.Cmd{owned, foreign} {
		child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
		defer func(child *exec.Cmd) { _ = child.Process.Kill(); _ = child.Wait() }(child)
	}
	state, p := webOwnedControlFixture(t)
	state.Binary = "/bin/sleep"
	state.Nodes[0].PID = owned.Process.Pid
	state.Nodes[0].Args = []string{"120"}
	root := t.TempDir()
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewWebChainEngine(root, "", store, nil, nil, nil)
	p.ControlDir = filepath.Join(root, "network")
	if err = os.MkdirAll(p.ControlDir, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(p.ControlDir, "chain-record.json")
	if err = os.WriteFile(record, raw, 0600); err != nil {
		t.Fatal(err)
	}
	ledger, err := process.OpenLedger(p.ControlDir)
	if err != nil {
		t.Fatal(err)
	}
	ns := state.Nodes[0]
	if err = ledger.Record(process.Proc{Label: string(ns.NodeLabel()), PID: foreign.Process.Pid, Host: ns.Host, DataDir: ns.DataDir}); err != nil {
		t.Fatal(err)
	}
	if err = ledger.Save(); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(p.ControlDir, process.LedgerFile)
	ledgerBefore, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	checksum, err := (filestore.Local{}).Checksum(context.Background(), state.Binary)
	if err != nil {
		t.Fatal(err)
	}
	p.ExecutionBinary = state.Binary
	p.RecordDigest = manifestHash(raw)
	p.Binary = ManifestBinaryEvidence{Chain: "wbft", SHA256: strings.TrimPrefix(checksum, "sha256:")}
	p.Input = WebPlanInput{Operation: "node.stop", NodeIDs: []string{"node1"}}
	_, err = engine.execute(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, p, func(WebJobPhase) error { return nil })
	if !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("core substituted an unchecked ledger PID: %v", err)
	}
	after, err := os.ReadFile(record)
	if err != nil || string(after) != string(raw) {
		t.Fatal("refused control changed record", err)
	}
	ledgerAfter, err := os.ReadFile(ledgerPath)
	if err != nil || string(ledgerBefore) != string(ledgerAfter) {
		t.Fatal("refused control changed ledger", err)
	}
	if !process.Alive(owned.Process.Pid) || !process.Alive(foreign.Process.Pid) {
		t.Fatal("refused control stopped a process")
	}
}
