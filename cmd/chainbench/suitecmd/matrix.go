package suitecmd

import (
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/cmd/chainbench/exitcode"
	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/spf13/cobra"
)

func runMatrix(cmd *cobra.Command, in app.RunMatrixIn, logins string, jsonOut, noSkips bool) error {
	out, err := app.RunMatrix(cmd.Context(), surface.Deps(cmd).WithLogins(logins), in)
	if err != nil {
		return err
	}
	if jsonOut {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(out); err != nil {
			return err
		}
	} else {
		for _, row := range out.Runs {
			fmt.Fprintf(cmd.OutOrStdout(), "preset %d: %s (%s), session %s\n", row.Index, row.PresetID, row.Chain, row.Session)
			if row.Error != "" {
				fmt.Fprintln(cmd.OutOrStdout(), row.Error)
			}
			if row.Plan != nil {
				fmt.Fprintln(cmd.OutOrStdout(), row.Plan)
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "pass=%d fail=%d blocked=%d skip=%d setupErrors=%d\n", out.Summary.Pass, out.Summary.Fail, out.Summary.Blocked, out.Summary.Skip, out.SetupErrors)
	}
	if code := out.ExitCode(noSkips); code != 0 {
		return &exitcode.Error{Code: code, Err: fmt.Errorf("run: matrix has %d setup error(s), %d failed, %d blocked, %d skipped", out.SetupErrors, out.Summary.Fail, out.Summary.Blocked, out.Summary.Skip)}
	}
	return nil
}
