package reportcmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewReport() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Show a run's report (from a session directory)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			rep, err := app.Report(cmd.Context(), surface.Deps(cmd), app.ReportIn{Dir: dataDir})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(rep.Tests) == 0 {
				fmt.Fprintln(out, "no runs recorded")
				return nil
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "SEQ\tID\tENV\tSTATUS")
			for _, t := range rep.Tests {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", t.Seq, t.ID, t.Env, t.Status)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			fmt.Fprintf(out, "\nsession=%s pass=%d fail=%d blocked=%d skip=%d\n",
				rep.Session, rep.Summary.Pass, rep.Summary.Fail, rep.Summary.Blocked, rep.Summary.Skip)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "session directory, or a root holding sessions")
	return surface.ReadOnly(cmd)
}
