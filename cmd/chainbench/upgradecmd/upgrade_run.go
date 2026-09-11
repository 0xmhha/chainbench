package upgradecmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

func newRunCmd() *cobra.Command {
	var profilePath, presetDir, fromBinary, toBinary, template, dataDir, genesisOverlay string
	var server, serverSet, workspaceConfig string
	var docker bool
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
			// --data-dir names the local data root. With --server the target's
			// workspace-config owns it instead, so it is not required there.
			if profilePath == "" || template == "" {
				return fmt.Errorf("--profile and --template are required")
			}
			if dataDir == "" && server == "" {
				return fmt.Errorf("--data-dir is required (or --server, whose workspace-config owns the target data root)")
			}
			out := cmd.OutOrStdout()
			res, err := app.UpgradeRun(cmd.Context(), surface.Deps(cmd), app.UpgradeRunIn{
				ProfilePath: profilePath, PresetDir: presetDir,
				FromBinary: fromBinary, ToBinary: toBinary,
				Template: template, GenesisOverlay: genesisOverlay,
				DataDir:             dataDir,
				AwaitFork:           time.Duration(waitFor) * time.Second,
				Server:              app.ServerRef{Name: server, SetPath: serverSet},
				WorkspaceConfigPath: workspaceConfig,
				Docker:              docker,
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
	cmd.Flags().StringVar(&server, "server", "", "run the handoff's data plane on this server, by name from the server set (default: this machine)")
	cmd.Flags().StringVar(&serverSet, "server-set", "", "server-set file: which servers exist and how to reach them (default: server-set.yaml when present)")
	cmd.Flags().StringVar(&workspaceConfig, "workspace-config", "", "environment file owning the target dataRoot and its purpose directories; required with --server")
	cmd.Flags().BoolVar(&docker, "docker", false, "the server set's hosts are local docker containers — translate this tool's dials via the localmap next to the server set")
	return cmd
}
