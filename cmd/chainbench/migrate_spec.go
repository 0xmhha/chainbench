package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/testspec"
)

// newMigrateSpecCmd mechanically converts a v1 spec to the v2 grammar
// (dsl-v2-proposal §3.6): chain/topology/... fold into an inline env, steps
// become do statements, assertions become expect statements appended after
// them (v1's fixed order), and pre/post actions become hooks.
func newMigrateSpecCmd() *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "migrate-spec <v1-spec.json>",
		Short: "Convert a v1 test spec to the v2 grammar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			if testspec.IsV2(raw) {
				return fmt.Errorf("%s already declares schemaVersion 2", args[0])
			}
			out, err := testspec.MigrateV1(raw)
			if err != nil {
				return err
			}
			if outPath != "" {
				return os.WriteFile(outPath, out, 0o644)
			}
			_, err = cmd.OutOrStdout().Write(append(out, '\n'))
			return err
		},
	}
	cmd.Flags().StringVar(&outPath, "out", "", "write the v2 spec here (default stdout)")
	return cmd
}
