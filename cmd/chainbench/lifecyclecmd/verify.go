package lifecyclecmd

import (
	"fmt"
	"github.com/0xmhha/chainbench/internal/dashboard"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewVerify() *cobra.Command {
	var (
		chain        string
		dataDir      string
		rpcURLs      []string
		delay        time.Duration
		readyTimeout time.Duration
		validators   bool
	)
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a network is producing blocks (from --rpc or a --workspace-dir)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			url, _ := cmd.Flags().GetString("dashboard")
			bus, closeBus := dashboard.Stream(url)
			defer closeBus()
			res, err := app.VerifyNetwork(cmd.Context(), deps(cmd), app.VerifyNetworkIn{
				DataDir: dataDir, Chain: chain, RPCURLs: rpcURLs,
				ProgressDelay: delay,
				ReadyTimeout:  readyTimeout,
				Bus:           bus,
			})
			if err != nil {
				return err
			}
			rep := res.Report

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "producing: %v\n", rep.Producing)
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tRPC\tCHAIN_ID\tBLOCK\tPEERS\tSYNCING\tOK")
			for _, n := range rep.Nodes {
				fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%d\t%v\t%v\n",
					n.Index, n.RPCURL, n.ChainID, n.BlockNumber, n.PeerCount, n.Syncing, n.OK)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if validators {
				if dataDir == "" {
					return fmt.Errorf("--validators needs --workspace-dir (it reads the composed keys to compare against)")
				}
				vres, verr := app.VerifyValidators(cmd.Context(), deps(cmd), app.NetVerifyValidatorsIn{DataDir: dataDir})
				if verr != nil {
					return verr
				}
				c := vres.Check
				fmt.Fprintf(out, "\nvalidators (%s) match: %v\n", c.Method, c.Match)
				if !c.Match {
					fmt.Fprintf(out, "  %s\n", c.Mismatch)
					return fmt.Errorf("validator set does not match the composed keys")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (optional metadata, used with --rpc)")
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "load the network from a workspace")
	cmd.Flags().StringArrayVar(&rpcURLs, "rpc", nil, "node RPC URL (repeatable)")
	cmd.Flags().DurationVar(&delay, "progress-delay", 2*time.Second, "wait between block-height samples")
	cmd.Flags().DurationVar(&readyTimeout, "ready-timeout", 45*time.Second, "how long to wait for the network to start producing blocks (0 = single check, no wait)")
	cmd.Flags().BoolVar(&validators, "validators", false, "also check the running chain recognizes exactly the composed keys as its validators (needs --workspace-dir)")
	return surface.ReadOnly(cmd)
}

// deps is what every lifecycle verb hands the app layer.
func deps(cmd *cobra.Command) app.Deps {
	errOut := cmd.ErrOrStderr()
	return app.Deps{
		Env: os.Getenv,
		Logf: func(format string, args ...any) {
			fmt.Fprintf(errOut, format+"\n", args...)
		},
	}
}
