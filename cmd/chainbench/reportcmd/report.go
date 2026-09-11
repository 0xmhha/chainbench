package reportcmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewReport() *cobra.Command {
	var (
		dataDir string
		all     bool
	)
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Show a run's report (from a session directory)",
		Long: "Shows one run's verdicts and the evidence behind them. With --all, " +
			"combines every session under the directory into one tally — a run per " +
			"spec file is the usual shape, and the newest session alone does not " +
			"answer whether the batch passed.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			rep, err := app.Report(cmd.Context(), surface.Deps(cmd), app.ReportIn{Dir: dataDir, All: all})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(rep.Tests) == 0 {
				fmt.Fprintln(out, "no runs recorded")
				return nil
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			// A combined report needs the run named per row: seq is unique within a
			// run, so several runs each have a seq 1 and the number alone misleads.
			combined := rep.Session == app.CombinedSession
			if combined {
				fmt.Fprintln(w, "SESSION\tSEQ\tID\tENV\tSTATUS")
			} else {
				fmt.Fprintln(w, "SEQ\tID\tENV\tSTATUS")
			}
			for _, t := range rep.Tests {
				if combined {
					fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", t.Session, t.Seq, t.ID, t.Env, t.Status)
					continue
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", t.Seq, t.ID, t.Env, t.Status)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			label := "session=" + rep.Session
			if combined {
				label = fmt.Sprintf("%d session(s) combined", app.ReportSessions(rep))
			}
			fmt.Fprintf(out, "\n%s pass=%d fail=%d blocked=%d skip=%d\n",
				label, rep.Summary.Pass, rep.Summary.Fail, rep.Summary.Blocked, rep.Summary.Skip)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "session directory, or a root holding sessions")
	cmd.Flags().BoolVar(&all, "all", false,
		"combine every session under the directory into one tally, instead of reading the most recent")
	return surface.ReadOnly(cmd)
}
