package main

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/accountcmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/chaincmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/keyringcmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/resourcecmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/txcmd"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/dashboard"
)

// dashboardURL is set by the persistent --dashboard flag; when non-empty, a
// command's obs events are forwarded to a running chainbench-dashboard.
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
		newChainsCmd(),
		newCapabilitiesCmd(),
		newStopCmd(),
		newStatusCmd(),
		newCleanCmd(),
		newVerifyCmd(),
		newRunCmd(),
		chaincmd.New(),
		keyringcmd.NewValidator(),
		newValidateCmd(),
		newMigrateSpecCmd(),
		newNodeCmd(),
		newConsensusCmd(),
		newHardforkCmd(),
		newUpgradeCmd(),
		newReportCmd(),
		accountcmd.NewFaucet(),
		newLogCmd(),
		accountcmd.New(),
		txcmd.NewTx(),
		txcmd.NewContract(),
		keyringcmd.New(),
		resourcecmd.New(),
	)
	return root
}

// obsBus returns an event bus and a cleanup func. When --dashboard is set, bus
// events are forwarded to that chainbench-dashboard; cleanup closes the bus and waits
// for the forwarder to flush.
func obsBus() (*collector.Bus, func()) {
	bus := collector.NewBus()
	if dashboardURL == "" {
		return bus, bus.Close
	}
	done := dashboard.Forward(bus, dashboardURL, http.DefaultClient)
	return bus, func() {
		bus.Close()
		<-done
	}
}
