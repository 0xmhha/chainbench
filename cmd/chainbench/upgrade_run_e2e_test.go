//go:build e2e

// This E2E is gated behind the `e2e` build tag and skips unless the chain
// binaries are provided, so `go test ./...` never runs it. It drives the real
// `chainbench upgrade run` framework path against the built go-wemix and go-wbft
// binaries and asserts the live croissant handoff completes.
//
// Run it with:
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	CHAINBENCH_E2E_TEMPLATE=/path/go-wemix/wemix/scripts/genesis-template.json \
//	go test -tags e2e -run TestUpgradeRunE2E -timeout 6m ./cmd/chainbench
package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestUpgradeRunE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	template := os.Getenv("CHAINBENCH_E2E_TEMPLATE")
	if fromBin == "" || toBin == "" || template == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN, CHAINBENCH_E2E_TO_BIN, CHAINBENCH_E2E_TEMPLATE to run")
	}
	dataDir := t.TempDir()
	// The command leaves the node processes running; stop anything bound to this
	// run's datadir when the test ends so ports are freed for the next run.
	t.Cleanup(func() { _ = exec.Command("pkill", "-9", "-f", dataDir).Run() })

	cmd := newUpgradeRunCmd()
	cmd.SetArgs([]string{
		"--profile", "../../profiles/wemix-upgrade.yaml",
		"--preset", "../../keys/preset",
		"--from-binary", fromBin,
		"--to-binary", toBin,
		"--template", template,
		"--data-dir", dataDir,
		"--wait", "150",
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	t.Logf("upgrade run output:\n%s", out.String())
	if err != nil {
		t.Fatalf("upgrade run failed: %v", err)
	}
	if !strings.Contains(out.String(), "handoff confirmed") {
		t.Fatalf("handoff not confirmed in output:\n%s", out.String())
	}
}
