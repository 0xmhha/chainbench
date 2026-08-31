package resourcecmd

import (
	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newPlanCmd runs the allocator as a question: the placement a network of this
// shape would get, computed from the server set (or the built-in pool) with
// nothing written anywhere. It exists so a placement change can be inspected —
// and tested — without composing a network.
func newPlanCmd() *cobra.Command {
	var chain string
	var validators, endpoints int
	var asJSON bool
	var sf ServerFlags
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Compute the placement a network of this shape would get, without composing it",
		Long: "Run the deterministic allocator over the requested shape and print the map it\n" +
			"would record: node i takes host i mod hosts, hosts are consumed before port\n" +
			"slots, and the same inputs always place the same. The chain matters because a\n" +
			"family reserves a different number of ports per node.\n\n" +
			"Nothing is written: no workspace, no files on any server.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := app.NetPlan(cmd.Context(), deps(cmd), app.NetPlanIn{
				Chain: chain, Validators: validators, Endpoints: endpoints, Server: sf.Ref(),
			})
			if err != nil {
				return err
			}
			if asJSON {
				return MapJSON(cmd.OutOrStdout(), out)
			}
			PrintMap(cmd.OutOrStdout(), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "chain id (stablenet|wbft|wemix) — sets the family's per-node port reservation")
	cmd.Flags().IntVar(&validators, "validators", 4, "validator node count")
	cmd.Flags().IntVar(&endpoints, "endpoints", 0, "endpoint (non-validator) node count")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the plan as JSON")
	sf.Bind(cmd)
	return cmd
}
