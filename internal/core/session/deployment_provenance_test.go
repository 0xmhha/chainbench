package session_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func TestDeploymentProvenanceSurvivesDiskRecording(t *testing.T) {
	sess := newSession(t)
	record := sess.Test(1, "deployment")
	nonce := uint64(0)
	wanted := session.ContractDeployment{Deployer: "0xpublic", Nonce: &nonce, ActualAddress: "0xactual", BytecodeSHA256: strings.Repeat("ab", 32), Artifact: "sha256:checksum"}
	record.Step(1, session.StepResult{Type: "deployContract", Hash: "0xtx", Receipt: map[string]any{"blockNumber": "0x3"}, Deployment: &wanted})
	record.Status(session.StatusPass)
	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(filepath.Join(record.Dir(), "steps.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rows []session.StepResult
	if err := json.Unmarshal(bytes, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Deployment == nil || rows[0].Deployment.Nonce == nil || *rows[0].Deployment.Nonce != 0 || rows[0].Deployment.BytecodeSHA256 != wanted.BytecodeSHA256 {
		t.Fatal("public metadata was dropped or scrubbed")
	}
}
