package suitecmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
)

func TestRunSuitePlanReportsResolvedLayoutWithoutComposing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(t.TempDir(), "case.json")
	raw := `{"schemaVersion":"2","kind":"case","id":"layout","chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet","topology":{"bp":2,"en":1}},"steps":[{"expect":"blockNumber","is":0}]}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	out, _, err := runSplit(t, "run", path, path, "--suite", "--plan", "--json", "--workspace-dir", dir, "--keys", repoKeys(t), "--binary", "/selected/gstable")
	if err != nil {
		t.Fatal(err)
	}
	var plans []struct {
		Plan   app.ComposePlan `json:"plan"`
		Layout app.ChainUpIn   `json:"layout"`
	}
	if err := json.Unmarshal([]byte(out), &plans); err != nil {
		t.Fatal(err, out)
	}
	if len(plans) != 1 || plans[0].Layout.BPCount != 2 || plans[0].Layout.ENCount != 1 || plans[0].Plan.Binary != "/selected/gstable" {
		t.Fatalf("missing suite layout: %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "chain-record.json")); !os.IsNotExist(err) {
		t.Fatal("planning composed a network")
	}
}

func TestRunPlanRejectsAttachmentInsteadOfRunningIt(t *testing.T) {
	_, _, err := runSplit(t, "run", tcPath("08-attached-chain-produces.json"), "--attach", "--workspace-dir", t.TempDir(), "--plan", "--json")
	if err == nil || !strings.Contains(err.Error(), "plan is only available for composition") {
		t.Fatalf("attach plan reached execution: %v", err)
	}
}
