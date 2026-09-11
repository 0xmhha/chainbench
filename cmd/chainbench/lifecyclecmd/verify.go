package lifecyclecmd

import (
	"fmt"
	"github.com/0xmhha/chainbench/internal/dashboard"
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
		baseline     bool
	)
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a network is producing blocks (from --rpc or a --workspace-dir)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			url, _ := cmd.Flags().GetString("dashboard")
			bus, closeBus := dashboard.Stream(url)
			defer closeBus()
			res, err := app.VerifyNetwork(cmd.Context(), surface.Deps(cmd), app.VerifyNetworkIn{
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
			// Agreement is reported next to producing because the two together are
			// the verdict: a split network produces on every side of the split, so
			// "producing: true" alone has answered a question nobody asked. An
			// unchecked comparison says so rather than reading as agreement.
			fmt.Fprintf(out, "producing: %v\n", rep.Producing)
			switch {
			case !rep.Agreement.Checked:
				fmt.Fprintf(out, "agreement: not checked (%s)\n", rep.Agreement.Detail)
			case rep.Agreement.Agreed:
				fmt.Fprintf(out, "agreement: yes, at block %d\n", rep.Agreement.Height)
			default:
				fmt.Fprintf(out, "agreement: NO — %s\n", rep.Agreement.Detail)
			}
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
				vres, verr := app.VerifyValidators(cmd.Context(), surface.Deps(cmd), app.NetVerifyValidatorsIn{DataDir: dataDir})
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
			if baseline {
				if dataDir == "" {
					return fmt.Errorf("--baseline needs --workspace-dir (the composition to check against the approved record)")
				}
				bres, berr := app.BaselineCheck(cmd.Context(), surface.Deps(cmd), app.NetBaselineIn{DataDir: dataDir})
				if berr != nil {
					return berr
				}
				c := bres.Check
				if !c.Approved {
					fmt.Fprintf(out, "\nbaseline: none approved yet (%s) — approve one with `chainbench baseline approve`\n", c.Path)
					return nil
				}
				fmt.Fprintf(out, "\nbaseline (%s) match: %v\n", c.Path, c.Match)
				if !c.Match {
					for _, d := range c.Diffs {
						fmt.Fprintf(out, "  %s\n", d)
					}
					return fmt.Errorf("composition drifted from the approved baseline")
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
	cmd.Flags().BoolVar(&baseline, "baseline", false, "also read the target's genesis and node configs now and check they match the environment's approved baseline (needs --workspace-dir); never updates it")
	return surface.ReadOnly(cmd)
}
