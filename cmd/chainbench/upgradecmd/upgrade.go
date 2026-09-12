// Package upgradecmd owns moving a chain from one binary or one fork to another:
// the wemix-to-wbft handoff (upgrade run) and a fork applied to a composed chain
// (hardfork), with the genesis derivation both need.
//
// It is its own group because a handoff runs TWO binaries against one chain,
// which no other command does — the assumption that a network has one binary is
// what six defects in this path had in common.
package upgradecmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

// newUpgradeCmd drives the concurrent consensus-family handoff (go-wemix+etcd ->
// go-wbft) framework in pkg/consensus/upgrade from a golden profile. Unlike the
// `hardfork` command (an in-place binary swap for a homogeneous fork), this
// composes a plan where producers and validators run concurrently.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Plan a concurrent consensus-family handoff from a golden profile",
	}
	cmd.AddCommand(newGenesisCmd(), newRunCmd())
	return cmd
}

func newGenesisCmd() *cobra.Command {
	var profilePath, fromGenesis, out string
	cmd := &cobra.Command{
		Use:   "genesis",
		Short: "Build the merged handoff genesis (from-chain base + successor fork section)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profilePath == "" || fromGenesis == "" {
				return fmt.Errorf("--profile and --from-genesis are required")
			}
			res, err := app.UpgradeGenesis(surface.Deps(cmd), profilePath, fromGenesis)
			if err != nil {
				return err
			}
			if out != "" {
				if err := os.WriteFile(out, res.Genesis, 0o644); err != nil {
					return err
				}
			}
			o := cmd.OutOrStdout()
			fmt.Fprintf(o, "handoff: %s -> %s at %s block; %d node(s)\n",
				res.Plan.From, res.Plan.To, res.Plan.AtFork, res.Plan.Nodes)
			w := tabwriter.NewWriter(o, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tCHAIN\tROLE\tNETID\tP2P\tHTTP\tETCD")
			for _, n := range res.Nodes {
				role := "validator"
				if n.Producer {
					role = "producer"
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\t%d\n",
					n.Index+1, n.Chain, role, n.NetworkID, n.P2P, n.HTTP, n.Etcd)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if out != "" {
				fmt.Fprintf(o, "merged genesis written to %s (%d bytes)\n", out, len(res.Genesis))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profilePath, "profile", "", "golden upgrade profile (profiles/*.yaml)")
	cmd.Flags().StringVar(&fromGenesis, "from-genesis", "", "from-chain base genesis (e.g. gwemix wemix genesis output)")
	cmd.Flags().StringVar(&out, "out", "", "write the merged genesis to this path")
	return cmd
}
