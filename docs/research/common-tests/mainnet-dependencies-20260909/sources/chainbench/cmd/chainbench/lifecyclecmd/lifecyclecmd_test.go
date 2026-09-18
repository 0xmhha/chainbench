package lifecyclecmd_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/lifecyclecmd"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// These verbs act on a composed network: what it is doing, stopping it, and
// removing what it left. They could not be tested at all while they lived in
// package main (worklist §1l, U1).

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(lifecyclecmd.NewStatus(), lifecyclecmd.NewStop(),
		lifecyclecmd.NewClean(), lifecyclecmd.NewVerify(), lifecyclecmd.NewConsensus())
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

// TestClean_RefusesWithoutATarget: clean removes data, so it has to be told
// what. Defaulting to a guess is how a command deletes the wrong tree.
func TestClean_RefusesWithoutATarget(t *testing.T) {
	out, err := run(t, "clean")
	if err == nil {
		t.Fatalf("clean with no target was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("the error does not say what is missing: %v", err)
	}
}

// TestStatus_SaysSoWhenThereIsNoNetwork: an empty workspace is a fact to
// report, not a crash and not a blank success.
func TestStatus_SaysSoWhenThereIsNoNetwork(t *testing.T) {
	out, err := run(t, "status", "--workspace-dir", t.TempDir())
	if err == nil && strings.TrimSpace(out) == "" {
		t.Fatal("an empty workspace produced neither an error nor a word of output")
	}
}

// TestStop_OnAnEmptyWorkspaceIsNotSilentSuccess: stop reads pids from a
// workspace. With none recorded it must not report that it stopped anything.
func TestStop_OnAnEmptyWorkspaceIsNotSilentSuccess(t *testing.T) {
	out, err := run(t, "stop", "--workspace-dir", t.TempDir())
	if err == nil && strings.Contains(out, "stopped") && !strings.Contains(out, "0") {
		t.Fatalf("stopping nothing reported a stop:\n%s", out)
	}
}

// TestVerify_RefusesWithoutSomethingToAsk: verify needs either an endpoint or a
// workspace to find one in.
func TestVerify_RefusesWithoutSomethingToAsk(t *testing.T) {
	out, err := run(t, "verify")
	if err == nil {
		t.Fatalf("verify with neither --rpc nor --workspace-dir was accepted:\n%s", out)
	}
}
