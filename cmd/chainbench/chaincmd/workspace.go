package chaincmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

// defaultWorkspaceDir is what a composition gets when --workspace-dir is
// omitted. The chosen path is printed BEFORE anything uses it: every later step
// needs it, so it must never be a guess.
//
// The default itself comes from app rather than from this surface's own
// arithmetic. A CLI run and an MCP call that both omit the workspace have to
// land in the same directory, or one composes somewhere the other cannot find.
func defaultWorkspaceDir(cmd *cobra.Command) (string, error) {
	dir, err := app.DefaultWorkspaceDir(surface.Deps(cmd))
	if err != nil {
		return "", err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "workspace: %s (--workspace-dir omitted)\n", dir)
	return dir, nil
}
