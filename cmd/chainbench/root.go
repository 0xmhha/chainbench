package main

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/obs"
	"github.com/0xmhha/chainbench/pkg/dashboard"
)

// dashboardURL is set by the persistent --dashboard flag; when non-empty, a
// command's obs events are forwarded to a running chainbenchd.
var dashboardURL string

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "chainbench",
		Short:         "Multi-chain local blockchain test bench",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&dashboardURL, "dashboard", "",
		"chainbenchd URL to stream run events to (e.g. http://127.0.0.1:8787)")
	root.AddCommand(
		newChainsCmd(),
		newSetupCmd(),
		newStopCmd(),
		newVerifyCmd(),
		newTestCmd(),
		newNodeCmd(),
		newConsensusCmd(),
		newHardforkCmd(),
		newUpgradeCmd(),
		newReportCmd(),
		newFaucetCmd(),
		newLogCmd(),
		newAccountCmd(),
		newTxCmd(),
		newContractCmd(),
	)
	return root
}

// obsBus returns an event bus and a cleanup func. When --dashboard is set, bus
// events are forwarded to that chainbenchd; cleanup closes the bus and waits
// for the forwarder to flush.
func obsBus() (*obs.Bus, func()) {
	bus := obs.NewBus()
	if dashboardURL == "" {
		return bus, bus.Close
	}
	done := dashboard.Forward(bus, dashboardURL, http.DefaultClient)
	return bus, func() {
		bus.Close()
		<-done
	}
}
