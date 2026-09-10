package chaincmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newNetCmd is the composable step surface: it composes a chain network for
// testing one customizable step at a time over a shared --workspace-dir workspace,
// so each step can be run, customized, and verified independently. Each
// subcommand mirrors an MCP tool (net_*) driving the same netcompose core.
//
// The workspace (control state) is always local; a step's files/processes live
// on the target — this machine or a remote SSH host — selected once at `chain new`
// (see targetFlags). Subcommands live in the net_*.go files.
// New builds the net command group.
func New() *cobra.Command { return newNetCmd() }

func newNetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chain",
		Short: "Compose a chain network step by step (keys, place, genesis, config, build, deploy, run, ...)",
		Long: "Compose a chain network for testing one customizable step at a time over a\n" +
			"shared --workspace-dir workspace. Each step runs independently, can be re-run,\n" +
			"and is inspectable with `chain status`. The workspace state is local; a step's\n" +
			"data plane lives on the target (local, or a remote SSH host set at `chain new`).\n" +
			"The same steps are exposed as MCP tools.",
	}
	cmd.AddCommand(
		newNetUpCmd(),
		newNetNewCmd(), newNetStatusCmd(),
		newNetKeysCmd(), newNetAllocateCmd(), newNetEnodeCmd(), newNetGenesisCmd(), newNetConfigCmd(),
		newNetLaunchOptsCmd(), newNetProvisionCmd(),
		newNetInitCmd(), newNetStartCmd(), newNetStopCmd(), newNetRestartCmd(), newNetResumeCmd(),
		newNetRmCmd(), newNetLogsCmd(), newNetHealthCmd(),
		newNetShowCmd(),
		newBlueprintCmd(),
	)
	return cmd
}

// targetFlags holds the compose-target selection shared by commands that set
// it: the single-path --target syntax (preferred), or the legacy four-flag
// form.
type targetFlags struct {
	target     string
	remoteHost string
	remoteUser string
	remotePort int
	targetDir  string
}

// bind attaches the target flags to a command.
func (f *targetFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.target, "target", "",
		"where the data plane lives, as one path: /local/path | user@host:/path | ssh://user@host:port/path (folds the four flags below)")
	cmd.Flags().StringVar(&f.remoteHost, "remote-host", "", "legacy: run the data plane on this SSH host (prefer --target)")
	cmd.Flags().StringVar(&f.remoteUser, "remote-user", "", "legacy: SSH user (prefer --target)")
	cmd.Flags().IntVar(&f.remotePort, "remote-port", 0, "legacy: SSH port (prefer --target)")
	cmd.Flags().StringVar(&f.targetDir, "target-dir", "", "legacy: data-root path ON the target (prefer --target)")
}

// spec builds a TargetSpec from the flags. --target wins; mixing it with the
// legacy flags is ambiguous and refused. Secrets are never captured here —
// they come from the environment when the target is resolved.
func (f *targetFlags) spec() (app.TargetSpec, error) {
	if f.target != "" {
		if f.remoteHost != "" || f.remoteUser != "" || f.remotePort != 0 || f.targetDir != "" {
			return app.TargetSpec{}, fmt.Errorf(
				"--target and the legacy --remote-host/--remote-user/--remote-port/--target-dir flags cannot be mixed")
		}
		return app.ParseTarget(f.target)
	}
	return app.TargetSpec{
		Host: f.remoteHost, User: f.remoteUser,
		Port: f.remotePort, DataRoot: f.targetDir,
	}, nil
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func sortedSteps(st app.State) []string {
	names := make([]string, 0, len(st.Steps))
	for n := range st.Steps {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
