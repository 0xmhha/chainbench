package main

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "chainbench",
		Short:         "Multi-chain local blockchain test bench",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newChainsCmd(),
		newSetupCmd(),
		newVerifyCmd(),
		newTestCmd(),
		newConsensusCmd(),
		newHardforkCmd(),
		newFaucetCmd(),
	)
	return root
}
