package main

import (
	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/accountcmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/catalogcmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/chaincmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/keyringcmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/lifecyclecmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/nodecmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/reportcmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/resourcecmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/suitecmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/txcmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/upgradecmd"
)

// dashboardURL binds the persistent --dashboard flag. It is declared on the
// root so every subcommand inherits it, and each command that streams events
// reads the value off its own cobra.Command rather than from here.
var dashboardURL string

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "chainbench",
		Short:         "Multi-chain local blockchain test bench",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&dashboardURL, "dashboard", "",
		"chainbench-dashboard URL to stream run events to (e.g. http://127.0.0.1:8787)")
	root.AddCommand(
		catalogcmd.NewChains(),
		catalogcmd.NewCapabilities(),
		lifecyclecmd.NewStop(),
		lifecyclecmd.NewStatus(),
		lifecyclecmd.NewClean(),
		lifecyclecmd.NewVerify(),
		suitecmd.NewRun(),
		chaincmd.New(),
		keyringcmd.NewValidator(),
		suitecmd.NewValidate(),
		suitecmd.NewMigrateSpec(),
		nodecmd.New(),
		lifecyclecmd.NewConsensus(),
		upgradecmd.NewHardfork(),
		upgradecmd.New(),
		reportcmd.NewReport(),
		accountcmd.NewFaucet(),
		reportcmd.NewLog(),
		accountcmd.New(),
		txcmd.NewTx(),
		txcmd.NewContract(),
		keyringcmd.New(),
		resourcecmd.New(),
	)
	return root
}
