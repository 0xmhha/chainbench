package upgradecmd_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/upgradecmd"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// hardfork swaps a binary at a fork block on a composed chain. It is
// destructive enough that its refusals matter more than its happy path, and it
// could not be reached from a test while it lived in package main (worklist
// §1l, U1).

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(upgradecmd.NewHardfork())
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

// TestHardfork_RefusesWithoutAWorkspace: the swap restarts the nodes a
// workspace records, so without one there is nothing to fork.
func TestHardfork_RefusesWithoutAWorkspace(t *testing.T) {
	out, err := run(t, "hardfork")
	if err == nil {
		t.Fatalf("hardfork with no workspace was accepted:\n%s", out)
	}
}

// TestUpgradeRun_AllServersSatisfiesTheDataDir: --data-dir names a LOCAL data
// root, and on a target the workspace-config owns it instead. A target is named
// by either --server or --all-servers, but the guard knew only the first, so the
// whole-set form — the one a 15+15 handoff uses — was rejected before it reached
