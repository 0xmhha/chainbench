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

func TestWebUnboundExecutableRefusalPreservesRealProcessAndRecord(t *testing.T) {
	child := exec.Command("/bin/sleep", "120")
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	state, p := webOwnedControlFixture(t)
	state.Binary = "/bin/sleep"
	state.Nodes[0].PID, state.Nodes[0].Args = child.Process.Pid, []string{"120"}
	state.Nodes[0].Binary = "unreviewed"
	state.Binaries = map[string]string{"unreviewed": "/bin/sleep"}
	state.Nodes[1].PID = 0
	marker := filepath.Join(state.Nodes[0].DataDir, "retained-data")
	if err := os.WriteFile(marker, []byte("retain selected data"), 0600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewWebChainEngine(root, "", store, nil, nil, nil)
	p.ControlDir = filepath.Join(root, "network")
	if err = os.Mkdir(p.ControlDir, 0700); err != nil {
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
	checksum, err := (filestore.Local{}).Checksum(context.Background(), state.Binary)
	if err != nil {
		t.Fatal(err)
	}
	p.ExecutionBinary, p.RecordDigest = state.Binary, manifestHash(raw)
	p.Binary = ManifestBinaryEvidence{Chain: "wbft", SHA256: strings.TrimPrefix(checksum, "sha256:")}
	p.Input = WebPlanInput{Operation: "node.stop", NodeIDs: []string{"node1"}}
	_, err = engine.execute(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, p, func(WebJobPhase) error { return nil })
	if err == nil && !process.Alive(child.Process.Pid) {
		t.Fatal("unbound named executable was authorized and stopped the selected real process")
	}
	if !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("unbound executable reached control", err)
	}
	if !process.Alive(child.Process.Pid) {
		t.Fatal("refusal stopped the existing process")
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "retain selected data" {
		t.Fatal("refusal changed selected node data", err)
	}
	after, err := os.ReadFile(record)
	if err != nil || string(after) != string(raw) {
		t.Fatal("refusal changed owned record", err)
	}
}
