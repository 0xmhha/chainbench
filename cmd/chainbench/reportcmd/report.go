package reportcmd

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/report"
	"github.com/0xmhha/chainbench/internal/core/session"
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
			// The argument may be a session directory itself or a root holding
			// several sessions; in the latter case show the most recent.
			sessionDir := dataDir
			if ids, _ := session.List(dataDir); len(ids) > 0 {
				sessionDir = session.SessionDir(dataDir, ids[len(ids)-1])
			}
			// Prefer the persisted report.json; fall back to building it from
			// session.json so a session written before report.json still shows.
			rep, err := report.Read(sessionDir)
			if err != nil {
				rep, err = report.Build(sessionDir)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						fmt.Fprintln(cmd.OutOrStdout(), "no runs recorded")
						return nil
					}
					return err
				}
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
	return cmd
}
