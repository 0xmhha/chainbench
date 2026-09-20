package upgradecmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

func newRunCmd() *cobra.Command {
	var profilePath, presetDir, fromBinary, toBinary, template, dataDir, genesisOverlay string
	var waitFor int
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Launch and bootstrap the full concurrent handoff from a golden profile",
		Long: "Composes the handoff end to end: build the producer's base genesis, " +
			"merge the successor fork section, launch the mixed binaries " +
			"concurrently, wire a full peer mesh, bootstrap governance + etcd on " +
			"the producer, and confirm the cluster formed. Requires the built " +
			"binaries, etcd, and a preset key set.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profilePath == "" || template == "" || dataDir == "" {
				return fmt.Errorf("--profile, --template, and --data-dir are required")
			}
			out := cmd.OutOrStdout()
			res, err := app.UpgradeRun(cmd.Context(), deps(cmd), app.UpgradeRunIn{
				ProfilePath: profilePath, PresetDir: presetDir,
				FromBinary: fromBinary, ToBinary: toBinary,
				Template: template, GenesisOverlay: genesisOverlay,
				DataDir:   dataDir,
				AwaitFork: time.Duration(waitFor) * time.Second,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "handoff %s -> %s at %s block %d; %d nodes\n",
				res.Plan.From, res.Plan.To, res.Plan.AtFork, res.Plan.ForkBlock, res.Plan.Nodes)
			for _, n := range res.Nodes.Nodes {
				fmt.Fprintf(out, "  node%d  %s  pid=%d\n", n.Index+1, n.RPCURL, n.PID)
			}
			fmt.Fprintf(out, "governance deployed at %s, etcd cluster %q, mesh wired.\n", res.Governance, res.Cluster)
			if res.Confirmed != "" {
				fmt.Fprintf(out, "handoff confirmed: %s\n", res.Confirmed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profilePath, "profile", "", "golden upgrade profile (profiles/*.yaml)")
	cmd.Flags().StringVar(&presetDir, "preset", "keys/preset", "preset key set directory")
	cmd.Flags().StringVar(&fromBinary, "from-binary", "", "from-chain (producer) binary path")
	cmd.Flags().StringVar(&toBinary, "to-binary", "", "to-chain (validator) binary path")
	cmd.Flags().StringVar(&template, "template", "", "wemix genesis template path")
	cmd.Flags().StringVar(&genesisOverlay, "genesis-overlay", "", "optional genesis overlay file ({\"genesis\":{...}}) deep-merged into the handoff genesis")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "node data root")
	cmd.Flags().IntVar(&waitFor, "wait", 0, "seconds to poll for the post-fork handoff (0=don't wait)")
	return cmd
}

// deps is what every upgrade verb hands the app layer.
func deps(cmd *cobra.Command) app.Deps {
	errOut := cmd.ErrOrStderr()
	return app.Deps{Logf: func(format string, args ...any) {
		fmt.Fprintf(errOut, format+"\n", args...)
	}}
}
