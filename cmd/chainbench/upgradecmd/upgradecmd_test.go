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

// TestUpgradeRun_AllServersSatisfiesTheDataDir: --data-dir names a LOCAL data
// root, and on a target the workspace-config owns it instead. A target is named
// by either --server or --all-servers, but the guard knew only the first, so the
// whole-set form — the one a 15+15 handoff uses — was rejected before it reached
// the code that handles it.
func TestUpgradeRun_AllServersSatisfiesTheDataDir(t *testing.T) {
	_, err := run(t, "upgrade", "run",
		"--profile", "/no/such/profile.yaml", "--template", "/no/such/template.json",
		"--all-servers", "--workspace-config", "/no/such/workspace.yaml")
	if err == nil {
		t.Fatal("a missing profile should still fail")
	}
	if strings.Contains(err.Error(), "--data-dir is required") {
		t.Errorf("--all-servers names the target, so --data-dir must not be demanded: %v", err)
	}
}

// TestUpgradeRun_AllServersNeedsAWorkspaceConfig is the other half: the target's
// data root comes from the workspace-config, so --all-servers without one has no
// data root at all. It used to get past the CLI and fail deep inside the handoff
// as "a data dir is required", which does not say which flag is missing.
func TestUpgradeRun_AllServersNeedsAWorkspaceConfig(t *testing.T) {
	_, err := run(t, "upgrade", "run",
		"--profile", "/no/such/profile.yaml", "--template", "/no/such/template.json",
		"--all-servers")
	if err == nil {
		t.Fatal("--all-servers with no workspace-config was accepted")
	}
	if !strings.Contains(err.Error(), "--workspace-config is required") {
		t.Errorf("the refusal should name the missing flag: %v", err)
	}
}
