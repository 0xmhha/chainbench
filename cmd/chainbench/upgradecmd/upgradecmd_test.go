package upgradecmd_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/upgradecmd"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// The upgrade group plans and runs a consensus-family handoff, and hardfork
// swaps a binary at a fork block. Both are destructive enough that their
// refusals matter more than their happy paths, and neither could be reached
// from a test while they lived in package main (worklist §1l, U1).

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(upgradecmd.New(), upgradecmd.NewHardfork())
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

// TestUpgradeGenesis_RefusesWithoutAProfileAndBase: the merged genesis is the
// from-chain's genesis plus the successor's fork section, so neither input has
// a sensible default.
func TestUpgradeGenesis_RefusesWithoutAProfileAndBase(t *testing.T) {
	out, err := run(t, "upgrade", "genesis")
	if err == nil {
		t.Fatalf("upgrade genesis with no inputs was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--profile and --from-genesis are required") {
		t.Errorf("the error does not name the missing inputs: %v", err)
	}
}

// TestUpgradeGenesis_RefusesAMissingProfile: a path that is not there has to
// fail as a missing file, not as an empty plan.
func TestUpgradeGenesis_RefusesAMissingProfile(t *testing.T) {
	out, err := run(t, "upgrade", "genesis",
		"--profile", "/no/such/profile.yaml", "--from-genesis", "/no/such/genesis.json")
	if err == nil {
		t.Fatalf("a missing profile was accepted:\n%s", out)
	}
}

// TestUpgrade_MountsItsSubcommands: the group is the only way an operator
// reaches genesis and run, so losing one to a rename would be silent.
func TestUpgrade_MountsItsSubcommands(t *testing.T) {
	out, err := run(t, "upgrade", "--help")
	if err != nil {
		t.Fatalf("upgrade --help: %v\n%s", err, out)
	}
	for _, want := range []string{"genesis", "run"} {
		if !strings.Contains(out, want) {
			t.Errorf("the upgrade group no longer offers %q:\n%s", want, out)
		}
	}
}

// TestHardfork_RefusesWithoutAWorkspace: the swap restarts the nodes a
// workspace records, so without one there is nothing to fork.
func TestHardfork_RefusesWithoutAWorkspace(t *testing.T) {
	out, err := run(t, "hardfork")
	if err == nil {
		t.Fatalf("hardfork with no workspace was accepted:\n%s", out)
	}
}
