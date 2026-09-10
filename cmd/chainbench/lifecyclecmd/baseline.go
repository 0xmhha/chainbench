package lifecyclecmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

// NewBaseline builds the baseline command group: the approved fingerprint of a
// regression environment's inputs.
//
// Approving is its own command, and deliberately not something a run does. A
// run that refreshed the baseline would re-approve whatever it found, and then
// the record could never catch a prepared genesis edited on the server — which
// is the whole reason to keep one.
func NewBaseline() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "The approved fingerprint of an environment's prepared inputs",
		Long: "A regression environment is only fixed if what it runs against is fixed.\n" +
			"`show` reads the genesis and node configs as they are on the target right\n" +
			"now and reports whether they still match what was approved;\n" +
			"`approve` records what is on the target now as the approved baseline.\n" +
			"Runs only ever read it — updating is this command's job, so a drifted\n" +
			"input is caught rather than silently adopted.\n" +
			"It records content hashes and validator addresses only: no key material.",
	}
	cmd.AddCommand(newBaselineShowCmd(), newBaselineApproveCmd())
	return cmd
}

// newBaselineShowCmd reports the check without changing anything.
func newBaselineShowCmd() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Read the target's inputs now and report whether they match the approved baseline",
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.BaselineCheck(cmd.Context(), surface.Deps(cmd), app.NetBaselineIn{DataDir: dataDir})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			c := res.Check
			if !c.Approved {
				fmt.Fprintf(out, "no baseline approved yet (%s)\n", c.Path)
				return nil
			}
			fmt.Fprintf(out, "baseline: %s\nmatch: %v\n", c.Path, c.Match)
			for _, d := range c.Diffs {
				fmt.Fprintf(out, "  %s\n", d)
			}
			if !c.Match {
				return fmt.Errorf("composition drifted from the approved baseline")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "the composed workspace to check")
	return surface.ReadOnly(cmd)
}

// newBaselineApproveCmd records the current composition as approved.
func newBaselineApproveCmd() *cobra.Command {
	var dataDir, note string
	cmd := &cobra.Command{
		Use:   "approve",
		Short: "Record the current composition as the environment's approved baseline",
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.BaselineApprove(cmd.Context(), surface.Deps(cmd), app.NetBaselineIn{
				DataDir: dataDir, Note: note,
				// Stamped by the surface so the record does not depend on a
				// clock read inside the module.
				ApprovedAt: time.Now().UTC().Format(time.RFC3339),
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "approved: %s\n", res.Check.Path)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "the composed workspace whose inputs are approved")
	cmd.Flags().StringVar(&note, "note", "", "why this baseline is approved, in your words")
	return cmd
}
