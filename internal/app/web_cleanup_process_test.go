package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/process"
)

func TestWebCleanupRefusesUnrecordedLiveProcess(t *testing.T) {
	child := exec.Command("/bin/sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	state, p := webOwnedControlFixture(t)
	state.Binary = "/bin/sleep"
	state.Nodes[0].PID = 0
	state.Nodes[0].Args = []string{"30"}
	state.Nodes[1].PID = 0
	state.Nodes[1].Args = []string{"31"}
	state.StatePath = "CHAIN/CHAIN_INITIALIZED"
	root := t.TempDir()
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewWebChainEngine(root, "", store, nil, nil, nil)
	p.Input.WorkspaceID = "owned"
	p.ControlDir = filepath.Join(root, "networks", "owned")
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
	marker := filepath.Join(state.Nodes[0].DataDir, "preserve-data")
	if err = os.WriteFile(marker, []byte("owned data"), 0600); err != nil {
		t.Fatal(err)
	}
	p.RecordDigest = manifestHash(raw)
	if err = recheckWebNetworkRecord(context.Background(), p, nil, true); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("execution recheck accepted residual process: %v", err)
	}
	payload, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Cleanup(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, WebPreparedJob{Payload: payload})
	if !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("cleanup did not reject live unrecorded process: result=%+v err=%v", result, err)
	}
	if result.NodeDisposition != "cleanup_failed" || len(result.UnresolvedResources) == 0 {
		t.Fatal("refused cleanup lost unresolved ownership", result)
	}
	after, err := os.ReadFile(record)
	if err != nil || string(after) != string(raw) {
		t.Fatal("refused cleanup changed record", err)
	}
	if _, err = os.Stat(marker); err != nil || !process.Alive(child.Process.Pid) {
		t.Fatal("refused cleanup destroyed data or process", err)
	}
}

func TestWebNetworkRecordRefusesConflictingLedgerPID(t *testing.T) {
	state, p := webOwnedControlFixture(t)
	state.Binary = "/bin/sleep"
	for i := range state.Nodes {
		state.Nodes[i].PID = 0
		state.Nodes[i].Args = []string{"29"}
	}
	p.ControlDir = t.TempDir()
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(p.ControlDir, "chain-record.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	p.RecordDigest = manifestHash(raw)
	ledger, err := process.OpenLedger(p.ControlDir)
	if err != nil {
		t.Fatal(err)
	}
	if err = ledger.Record(process.Proc{Label: "node1", PID: 123456, DataDir: state.Nodes[0].DataDir, Host: state.Nodes[0].Host}); err != nil {
		t.Fatal(err)
	}
	if err = ledger.Save(); err != nil {
		t.Fatal(err)
	}
	for _, vacant := range []bool{true, false} {
		if err = recheckWebNetworkRecord(context.Background(), p, nil, vacant); !errors.Is(err, ErrDeploymentConflict) {
			t.Fatalf("conflicting ledger PID was accepted, vacant=%t err=%v", vacant, err)
		}
	}
}
