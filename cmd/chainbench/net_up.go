package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newNetUpCmd composes and brings up a whole network in one command — the nine
// `net` steps run in order. Flag binding + app.NetUp + output; the logic lives
// in the app layer, shared with the MCP tool.
func newNetUpCmd() *cobra.Command {
	var (
		dataDir, chain, binary, keysDir       string
		manifestPath, templatePath            string
		validators, endpoints                 int
		endpointSyncMode, topologyPath, stage string
		keysSource, bootnode                  string
		chainID                               int64
		genesisSet, launchSet                 []string
		overlayPath                           string
		tf                                    targetFlags
		sf                                    serverFlags
	)
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Compose and launch a network in one command (runs every net step in order)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--data-dir is required")
			}
			target, err := tf.spec()
			if err != nil {
				return err
			}
			out, err := app.NetUp(cmd.Context(), app.Deps{}, app.NetUpIn{
				DataDir: dataDir, Stage: app.UpStage(stage),
				Chain: chain, ManifestPath: manifestPath, TemplatePath: templatePath,
				KeysDir: keysDir, Target: target, Binary: binary,
				Validators: validators, Endpoints: endpoints,
				EndpointSyncMode: endpointSyncMode, TopologyPath: topologyPath,
				Server:     sf.ref(),
				KeysSource: keysSource,
				ChainID:    chainID, GenesisSet: genesisSet, OverlayPath: overlayPath,
				LaunchSet: launchSet,
			})
			// The steps that did run are worth printing even when a later one
			// failed: they say how far the composition got.
			printUpSteps(cmd, out)
			if err != nil {
				return err
			}
			printUpNodes(cmd, out)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "local workspace directory (keep it short: node IPC sockets have a 104-char limit)")
	cmd.Flags().StringVar(&stage, "stage", string(app.UpStart), "how far to go: provision (write artifacts only) or start")
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix); ignored with --manifest")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to an external chain manifest JSON (project-supplied chain, on a built-in family)")
	cmd.Flags().StringVar(&templatePath, "genesis-template", "", "path to the genesis template for --manifest")
	cmd.Flags().StringVar(&binary, "binary", "", "node binary path (required for --stage=start)")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "key set the network composes from")
	cmd.Flags().IntVar(&validators, "validators", 4, "validator node count")
	cmd.Flags().IntVar(&endpoints, "endpoints", 0, "endpoint (non-validator) node count")
	cmd.Flags().StringVar(&endpointSyncMode, "endpoint-syncmode", "", "sync mode for endpoints (snap|archive); default full")
	cmd.Flags().StringVar(&topologyPath, "topology", "", "per-node layout YAML (role/sync-mode/bootnode); overrides --validators/--endpoints")
	cmd.Flags().StringVar(&keysSource, "keys-source", "", "preset (default) or generate")
	cmd.Flags().StringVar(&bootnode, "bootnode", "", "deprecated: ignored, BLS material is derived in process")
	_ = cmd.Flags().MarkDeprecated("bootnode", "no longer needed — BLS material is derived in process")
	cmd.Flags().Int64Var(&chainID, "chain-id", 0, "override the manifest chain id (0 = manifest)")
	cmd.Flags().StringArrayVar(&genesisSet, "set", nil, "override a genesis config key (repeatable), e.g. --set bohoBlock=10")
	cmd.Flags().StringVar(&overlayPath, "overlay", "", "JSON overlay file {capabilities,genesis} deep-merged into the genesis")
	cmd.Flags().StringArrayVar(&launchSet, "launch-opt", nil, "override a launch option (repeatable), e.g. --launch-opt networkid=4242")
	tf.bind(cmd)
	sf.bind(cmd)
	return cmd
}

// printUpSteps lists what each step recorded, in order.
func printUpSteps(cmd *cobra.Command, out app.NetUpOut) {
	w := cmd.OutOrStdout()
	for _, s := range out.Steps {
		fmt.Fprintln(w, s)
	}
}

// printUpNodes renders the composed node table.
func printUpNodes(cmd *cobra.Command, out app.NetUpOut) {
	nodes := out.Nodes.Nodes.Nodes
	if len(nodes) == 0 {
		return
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NODE\tROLE\tRPC\tPID")
	for _, n := range nodes {
		fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", n.Index, n.Role, n.RPCURL, n.PID)
	}
	_ = w.Flush()
}
