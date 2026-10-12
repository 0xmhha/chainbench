package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/process"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebNodeExecutionRechecksProcessBeforeStopping(t *testing.T) {
	child := exec.Command("/bin/sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	root := t.TempDir()
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewWebChainEngine(root, "", store, nil, nil, nil)
	control := filepath.Join(root, "network")
	if err = os.Mkdir(control, 0o700); err != nil {
		t.Fatal(err)
	}
	state, payload := webOwnedControlFixture(t)
	state.Binary = "/bin/sleep"
	state.Nodes[0].PID = child.Process.Pid
	state.Nodes[0].Args = []string{"--config", state.Nodes[0].ConfigPath}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(control, "chain-record.json")
	if err = os.WriteFile(record, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	checksum, err := (filestore.Local{}).Checksum(context.Background(), "/bin/sleep")
	if err != nil {
		t.Fatal(err)
	}
	payload.ExecutionBinary, payload.ControlDir, payload.RecordDigest = "/bin/sleep", control, manifestHash(raw)
	payload.Binary = ManifestBinaryEvidence{Chain: "wbft", SHA256: strings.TrimPrefix(checksum, "sha256:")}
	payload.Input = WebPlanInput{Operation: "node.stop", NodeIDs: []string{"node1"}}
	_, err = engine.execute(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, payload, func(WebJobPhase) error { return nil })
	if !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("unverified process reached the node stop verb: %v", err)
	}
	if !process.Alive(child.Process.Pid) {
		t.Fatal("foreign fixture process was stopped")
	}
	after, err := os.ReadFile(record)
	if err != nil || string(after) != string(raw) {
		t.Fatal("rejected control mutated the record", err)
	}
}

func TestWebControlRefusesUnrecordedLiveProcess(t *testing.T) {
	child := exec.Command("/bin/sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	state, _ := webOwnedControlFixture(t)
	state.Binary = "/bin/sleep"
	state.Nodes[0].PID = 0
	state.Nodes[0].Args = []string{"30"}
	if err := verifyWebNodeProcesses(context.Background(), state, []string{"node1"}, nil); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("PID-zero record allowed control over a live process: %v", err)
	}
	if state.Nodes[0].PID != 0 || !process.Alive(child.Process.Pid) {
		t.Fatal("read-only discovery adopted or stopped a process")
	}
}
