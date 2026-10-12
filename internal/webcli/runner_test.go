package webcli_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/webcli"
)

// cli is the real chainbench binary built once for this package, so every test
// drives the command an operator runs rather than a stand-in.
var cli string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "webcli-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cli = filepath.Join(dir, "chainbench")
	build := exec.Command("go", "build", "-o", cli, "github.com/0xmhha/chainbench/cmd/chainbench")
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err = build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build chainbench:", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// workspace writes a one-node composition whose record names recordPID and
// whose process ledger names ledgerPID.
func workspace(t *testing.T, recordPID, ledgerPID int) string {
	t.Helper()
	dir := t.TempDir()
	comp, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	label := node.LabelFor(1)
	layout := node.Layout{Root: dir}
	if err = comp.Save(chainsetup.State{
		FormatVersion: chainsetup.StateFormatVersion, Chain: "stablenet", BPCount: 1,
		Target: resource.Spec{DataRoot: dir}, Steps: map[string]chainsetup.Step{},
		Nodes: []node.Record{{Index: 1, Label: string(label), Role: "bp", Host: "127.0.0.1",
			DataDir: layout.DataDir(label), PID: recordPID}},
	}); err != nil {
		t.Fatal(err)
	}
	ledger, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = ledger.Record(process.Proc{PID: ledgerPID, Label: string(label), Host: "127.0.0.1", DataDir: layout.DataDir(label)}); err != nil {
		t.Fatal(err)
	}
	if err = ledger.Save(); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestChainStatusReportsWhatTheCLIReports: the state comes from `chainbench
// chain status`, which takes each pid from the process ledger, not from a
// direct read of the record.
func TestChainStatusReportsWhatTheCLIReports(t *testing.T) {
	dir := workspace(t, 111, 222)
	st, err := webcli.Runner{Binary: cli}.ChainStatus(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Nodes) != 1 || st.Nodes[0].PID != 222 || st.Chain != "stablenet" {
		t.Fatalf("status = chain %q nodes %+v, want stablenet with the ledger pid 222", st.Chain, st.Nodes)
	}
}

// TestRunCarriesTheCLIErrorMessage: a failed command is reported with the
// message the CLI printed, so a job shows the engine's reason.
func TestRunCarriesTheCLIErrorMessage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "chain-record.json"), []byte(`{"formatVersion":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := webcli.Runner{Binary: cli}.ChainStatus(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), "format 1") || !strings.Contains(err.Error(), "chain status") {
		t.Fatalf("status of an old record gave %v, want the CLI's format refusal naming the command", err)
	}
}

// TestRunStopsTheCommandWhenCancelled: cancelling the context interrupts a
// command that is still waiting, the way an operator's Ctrl-C does, instead of
// leaving the job hanging until the command gives up by itself.
func TestRunStopsTheCommandWhenCancelled(t *testing.T) {
	release := make(chan struct{})
	stall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(stall.Close)
	t.Cleanup(func() { close(release) })
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	began := time.Now()
	_, err := webcli.Runner{Binary: cli}.Run(ctx, "node", "rpc", "--rpc", stall.URL, "--method", "eth_blockNumber")
	if err == nil {
		t.Fatal("an interrupted command reported success")
	}
	if took := time.Since(began); took > 10*time.Second {
		t.Fatalf("the command ran %v after its job was cancelled", took)
	}
}

// TestFindBinaryFindsTheCLI: an explicit path wins; otherwise the binary beside the
// dashboard is used; a missing binary is refused at startup rather than at the
// first job.
func TestFindBinaryFindsTheCLI(t *testing.T) {
	if got, err := webcli.FindBinary(cli, ""); err != nil || got != cli {
		t.Fatalf("explicit path: %q %v", got, err)
	}
	beside := t.TempDir()
	sibling := filepath.Join(beside, "chainbench")
	if err := os.WriteFile(sibling, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := webcli.FindBinary("", filepath.Join(beside, "chainbench-dashboard")); err != nil || got != sibling {
		t.Fatalf("sibling: %q %v", got, err)
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := webcli.FindBinary("", filepath.Join(t.TempDir(), "chainbench-dashboard")); err == nil {
		t.Fatal("a missing chainbench binary was accepted")
	}
	if _, err := webcli.FindBinary(filepath.Join(beside, "absent"), ""); err == nil {
		t.Fatal("an explicit path that does not exist was accepted")
	}
}
