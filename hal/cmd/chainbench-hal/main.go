package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.0.0-dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "chainbench-hal",
		Short:         "Network abstraction layer for chainbench",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the binary version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "chainbench-hal %s\n", version)
			return err
		},
	}
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
