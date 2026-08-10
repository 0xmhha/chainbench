package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/netcompose"
)

// newNetCmd is the composable step surface: it composes a chain network for
// testing one customizable step at a time over a shared --data-dir workspace,
// so each step can be run, customized, and verified independently. Each
// subcommand mirrors an MCP tool (net_*) driving the same netcompose core.
//
// The workspace (control state) is always local; a step's files/processes live
// on the target — this machine or a remote SSH host — selected once at `net new`
// (see targetFlags). Subcommands live in the net_*.go files.
func newNetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "net",
		Short: "Compose a chain network step by step (keys, genesis, config, start, ...)",
		Long: "Compose a chain network for testing one customizable step at a time over a\n" +
			"shared --data-dir workspace. Each step runs independently, can be re-run,\n" +
			"and is inspectable with `net status`. The workspace state is local; a step's\n" +
			"data plane lives on the target (local, or a remote SSH host set at `net new`).\n" +
			"The same steps are exposed as MCP tools.",
	}
	cmd.AddCommand(newNetNewCmd(), newNetStatusCmd())
	return cmd
}

// openWorkspace opens the local data-dir workspace for a step command.
func openWorkspace(dataDir string) (*netcompose.Workspace, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("--data-dir is required")
	}
	return netcompose.Open(dataDir, nil)
}

// targetFlags holds the compose-target selection shared by commands that set it.
type targetFlags struct {
	remoteHost string
	remoteUser string
	remotePort int
	targetDir  string
}

// bind attaches the target flags to a command.
func (f *targetFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.remoteHost, "remote-host", "", "run the data plane on this SSH host (empty = local)")
	cmd.Flags().StringVar(&f.remoteUser, "remote-user", "", "SSH user (or CHAINBENCH_REMOTE_USER)")
	cmd.Flags().IntVar(&f.remotePort, "remote-port", 0, "SSH port (default 22)")
	cmd.Flags().StringVar(&f.targetDir, "target-dir", "", "data-root path ON the target (required for remote; defaults to the workspace dir for local)")
}

// spec builds a TargetSpec from the flags: remote when --remote-host is set,
// else local. Secrets are never captured here — they come from the environment
// when the target is resolved.
func (f *targetFlags) spec() netcompose.TargetSpec {
	if f.remoteHost == "" {
		return netcompose.TargetSpec{Kind: netcompose.TargetLocal, DataRoot: f.targetDir}
	}
	return netcompose.TargetSpec{
		Kind: netcompose.TargetRemote, Host: f.remoteHost, User: f.remoteUser,
		Port: f.remotePort, DataRoot: f.targetDir,
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func sortedSteps(st netcompose.State) []string {
	names := make([]string, 0, len(st.Steps))
	for n := range st.Steps {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
