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

// TestWebRestartRefusesChangedInputsBeforeStopping verifies the execution
// boundary, after acceptance, against a real owned PID.
func TestWebRestartRefusesChangedInputsBeforeStopping(t *testing.T) {
	for _, input := range []string{"config", "genesis", "workspace", "server-set"} {
		t.Run(input, func(t *testing.T) {
			child := exec.Command("/bin/sleep", "120")
			child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
			state, p := webOwnedControlFixture(t)
			state.Binary = "/bin/sleep"
			state.Nodes[0].PID, state.Nodes[0].Args = child.Process.Pid, []string{"120"}
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
			checksum, err := (filestore.Local{}).Checksum(context.Background(), state.Binary)
			if err != nil {
				t.Fatal(err)
			}
			p.ExecutionBinary, p.RecordDigest = state.Binary, manifestHash(raw)
			p.Binary = ManifestBinaryEvidence{Chain: "wbft", SHA256: strings.TrimPrefix(checksum, "sha256:")}
			p.Input = WebPlanInput{Operation: "node.restart", NodeIDs: []string{"node1"}}
			path := map[string]string{"config": state.Nodes[0].ConfigPath, "genesis": state.GenesisPath, "workspace": state.WorkspaceConfig, "server-set": state.ServerSet}[input]
			if err = os.WriteFile(path, []byte("changed after acceptance"), 0600); err != nil {
				t.Fatal(err)
			}
			_, err = engine.execute(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, p, func(WebJobPhase) error { return nil })
			if !errors.Is(err, ErrDeploymentConflict) {
				t.Fatalf("changed accepted input reached restart: %v", err)
			}
			if !process.Alive(child.Process.Pid) {
				t.Fatal("input rejection stopped the existing process")
			}
			after, err := os.ReadFile(record)
			if err != nil || string(after) != string(raw) {
				t.Fatal("input rejection mutated record", err)
			}
		})
	}
}

func TestWebRestartInputsPermitProducerAndRefuseUnboundBinary(t *testing.T) {
	state, p := webResetFixture(t)
	state.Nodes[0].Role = "bp"
	p.Input.Operation = "node.restart"
	if err := verifyWebNodeInputs(context.Background(), state, p, nil); err != nil {
		t.Fatal("producer cannot preserve and restart its data", err)
	}
	state.Nodes[0].Binary = "unreviewed"
	if err := verifyWebNodeInputs(context.Background(), state, p, nil); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("unbound per-node binary accepted", err)
	}
}
