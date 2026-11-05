package resourcecmd

import (
	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/spf13/cobra"
)

// newProbeCmd runs one inspection of a binary in a target's data root. It runs a
// program on that target, so it is no part of `resource inspect`.
func newProbeCmd() *cobra.Command {
	var request, logins string
	var jsonOut bool
	cmd := &cobra.Command{Use: "probe", Short: "Run one inspection of a binary in a remote target's data root", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		var in app.ResourceProbeIn
		if err := readRequestFile(request, &in); err != nil {
			return err
		}
		out, err := app.ResourceProbe(cmd.Context(), surface.Deps(cmd).WithLogins(logins), in)
		if err != nil {
			return err
		}
		return writeResult(cmd, out, jsonOut)
	}}
	cmd.Flags().StringVar(&request, "request", "", "JSON file naming the target, binary and arguments")
	cmd.Flags().StringVar(&logins, "server-logins", "", "server set supplying this call's SSH logins")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the result as JSON")
	_ = cmd.MarkFlagRequired("request")
	return cmd
}
