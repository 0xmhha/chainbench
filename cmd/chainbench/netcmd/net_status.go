package netcmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// newNetStatusCmd shows the workspace composition state and which steps have
// run. Rendering only — the read goes through chainsetup.NetStatus, shared with the
// MCP tool.
func newNetStatusCmd() *cobra.Command {
	var dataDir string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the workspace composition state and which steps have run",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			res, err := chainsetup.NetStatus(cmd.Context(), deps(cmd), chainsetup.NetStatusIn{DataDir: dataDir})
			if err != nil {
				return err
			}
			st := res.State
			out := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(st)
			}

			// Where the workspace points is the machine module's question;
			// the status line just prints its answer.
			target := st.Target.Describe()
			fmt.Fprintf(out, "workspace: %s\nchain: %s  binary: %s  keys: %s  validators: %d\ntarget: %s\n",
				res.Dir, st.Chain, orDash(st.Binary), orDash(st.KeysDir), st.Validators, orDash(target))

			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "STEP\tDONE\tDETAIL")
			for _, name := range sortedSteps(st) {
				s := st.Steps[name]
				fmt.Fprintf(w, "%s\t%v\t%s\n", name, s.Done, s.Detail)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory (where the composition is set up)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the workspace state as JSON")
	return cmd
}
