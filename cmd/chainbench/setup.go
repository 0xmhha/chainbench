package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/driver"
)

func newSetupCmd() *cobra.Command {
	var (
		chain          string
		manifestPath   string
		templatePath   string
		validators     int
		endpoints      int
		dataDir        string
		keysDir        string
		binaryPath     string
		remoteHost     string
		remoteUser     string
		remotePort     int
		provision      bool
		launch         bool
		dryRun         bool
		setValues      []string
		genesisOverlay string
		topologyPath   string
	)
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Plan (and, when wired, launch) a local chain network",
		RunE: func(cmd *cobra.Command, _ []string) error {
			spec := app.NetworkSpecIn{
				Chain:              chain,
				ChainExplicit:      cmd.Flags().Changed("chain"),
				ManifestPath:       manifestPath,
				TemplatePath:       templatePath,
				TopologyPath:       topologyPath,
				DataDir:            dataDir,
				Set:                setValues,
				GenesisOverlayPath: genesisOverlay,
			}
			// A zero count is a meaningful request, so only a flag the user
			// actually set overrides the configured value.
			if cmd.Flags().Changed("validators") {
				spec.Validators = &validators
			}
			if cmd.Flags().Changed("endpoints") {
				spec.Endpoints = &endpoints
			}

			deps := app.Deps{}
			if remoteHost != "" {
				deps.Driver = func() (driver.Driver, error) {
					return remoteDriver(remoteHost, remoteUser, remotePort)
				}
			}

			planned, err := app.NetworkPlan(cmd.Context(), deps, spec)
			if err != nil {
				return err
			}
			printPlan(cmd, planned)

			out := cmd.OutOrStdout()
			switch {
			case launch:
				// For a remote launch the binary path lives on the remote host,
				// so it is used as-is; only a local launch resolves it on PATH.
				bin := binaryPath
				if remoteHost == "" {
					if bin, err = resolveBinary(binaryPath, planned.Plugin.Manifest().Binary); err != nil {
						return err
					}
				} else if bin == "" {
					return fmt.Errorf("--binary <remote path> is required with --remote-host")
				}
				bus, closeBus := obsBus()
				defer closeBus()
				res, err := app.NetworkLaunch(cmd.Context(), deps, app.NetworkLaunchIn{
					Spec: spec, KeysDir: keysDir, Binary: bin, Bus: bus,
				})
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "launched %d node(s); state: %s\n",
					len(res.Nodes.Nodes), filepath.Join(res.Plan.DataRoot, "nodeset.json"))
				return nil

			case provision:
				res, err := app.NetworkProvision(cmd.Context(), deps, app.NetworkProvisionIn{
					Spec: spec, KeysDir: keysDir,
				})
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "provisioned: genesis + %d node config(s) in %s\n",
					len(res.Plan.Plan.Nodes), res.Plan.DataRoot)
				return nil

			case !dryRun:
				return fmt.Errorf("live launch needs --launch (with --binary or a %s on PATH). Use --provision to write artifacts, --dry-run to plan",
					planned.Plugin.Manifest().Binary)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "embedded chain id (stablenet|wbft|wemix); ignored with --manifest")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to an external chain manifest JSON (project-supplied chain, on a built-in family)")
	cmd.Flags().StringVar(&templatePath, "genesis-template", "", "path to the genesis template for --manifest")
	cmd.Flags().IntVar(&validators, "validators", 0, "override validator count")
	cmd.Flags().IntVar(&endpoints, "endpoints", 0, "override endpoint count")
	cmd.Flags().StringVar(&topologyPath, "topology", "", "per-node topology YAML (role/sync-mode/bootnode per node); overrides --validators/--endpoints")
	cmd.Flags().StringArrayVar(&setValues, "set", nil, "override a flat config key (repeatable), e.g. --set genesis.overrides.bohoBlock=10")
	cmd.Flags().StringVar(&genesisOverlay, "genesis-overlay", "", "JSON overlay file {capabilities,genesis} deep-merged into the genesis (e.g. internal/chains/stablenet/overlays/account-extra.json)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "data", "data root directory")
	cmd.Flags().StringVar(&keysDir, "keys-dir", "keys/preset", "preset keys directory (for --provision)")
	cmd.Flags().StringVar(&binaryPath, "binary", "", "node binary path (for --launch); default: chain binary on PATH")
	cmd.Flags().StringVar(&remoteHost, "remote-host", "", "launch nodes on this SSH host (RemoteDriver); password from CHAINBENCH_REMOTE_PASS env")
	cmd.Flags().StringVar(&remoteUser, "remote-user", "", "SSH user for --remote-host")
	cmd.Flags().IntVar(&remotePort, "remote-port", 22, "SSH port for --remote-host")
	cmd.Flags().BoolVar(&provision, "provision", false, "write genesis.json + node configs from preset keys")
	cmd.Flags().BoolVar(&launch, "launch", false, "init datadirs and launch the nodes (implies --provision)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "plan only; do not launch")
	return cmd
}

// printPlan renders the resolved plan: the chain identity, then one row per
// planned node.
func printPlan(cmd *cobra.Command, planned app.NetworkPlanOut) {
	out := cmd.OutOrStdout()
	m := planned.Plugin.Manifest()
	fmt.Fprintf(out, "chain:    %s (family %s, binary %s, chain_id %d)\n",
		m.ID, m.ConsensusFamily, m.Binary, m.ChainID)
	fmt.Fprintf(out, "network:  %s\n", planned.Plan.Network)
	fmt.Fprintf(out, "dataRoot: %s\n", planned.Plan.DataRoot)
	fmt.Fprintf(out, "genesis:  template=%v (engine=%q)\n",
		len(planned.Plugin.GenesisTemplate()) > 0, m.Genesis.EngineField)

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NODE\tROLE\tSYNC\tHOST\tP2P\tHTTP\tWS")
	for _, n := range planned.Plan.Nodes {
		sync := n.SyncMode
		if sync == "" {
			sync = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\t%d\n",
			n.Index, n.Role, sync, n.Host, n.Ports.P2P, n.Ports.HTTP, n.Ports.WS)
	}
	// The plan is advisory output; a rendering failure must not fail the setup.
	_ = w.Flush()
	if planned.BootnodeIndex > 0 {
		fmt.Fprintf(out, "bootnode: node %d\n", planned.BootnodeIndex)
	}
}

// resolveBinary returns the executable path for launch: the explicit path if
// given, otherwise the chain's binary looked up on PATH.
func resolveBinary(explicit, chainBinary string) (string, error) {
	name := explicit
	if name == "" {
		name = chainBinary
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("cannot find node binary %q: %w (build it or pass --binary)", name, err)
	}
	return path, nil
}
