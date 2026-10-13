package resourcecmd

import (
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/spf13/cobra"
)

func newAccessCmd() *cobra.Command {
	var in app.ResourceAccessIn
	var logins string
	var jsonOut bool
	cmd := &cobra.Command{Use: "access", Short: "Check SSH authentication and data-root permission without changing files", RunE: func(cmd *cobra.Command, _ []string) error {
		out, err := app.ResourceCheckAccess(cmd.Context(), surface.Deps(cmd).WithLogins(logins), in)
		if err != nil {
			return err
		}
		if jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "authenticated=%t reachable=%t %s\n", out.Authenticated, out.Reachable, out.Reason)
		return err
	}}
	cmd.Flags().StringVar(&in.ServerSet, "server-set", "", "server-set file to check")
	cmd.Flags().StringVar(&in.Server, "server", "", "server name to check")
	cmd.Flags().StringVar(&in.DataRoot, "data-root", "", "absolute target data root whose permission is checked")
	cmd.Flags().StringVar(&logins, "server-logins", "", "private logins separate from the declared server set")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit authentication and permission observations as JSON")
	_ = cmd.MarkFlagRequired("server-set")
	_ = cmd.MarkFlagRequired("server")
	_ = cmd.MarkFlagRequired("data-root")
	return cmd
}
