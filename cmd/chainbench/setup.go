package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/chains/external"
	"github.com/0xmhha/chainbench/pkg/core/config"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/setup"
	"github.com/0xmhha/chainbench/pkg/core/registry"
	"github.com/0xmhha/chainbench/pkg/core/state"
)

// resolveChain returns the chain plugin for a run: an external, project-supplied
// manifest when --manifest is given (the hybrid model), otherwise the embedded
// chain registered for the --chain id.
func resolveChain(chain, manifestPath, templatePath string) (registry.ChainPlugin, error) {
	if manifestPath != "" {
		return external.Load(manifestPath, templatePath)
	}
	return registry.Get(chain)
}

func newSetupCmd() *cobra.Command {
	var (
		chain        string
		manifestPath string
		templatePath string
		validators   int
		endpoints    int
		dataDir      string
		keysDir      string
		binaryPath   string
		provision    bool
		launch       bool
		dryRun       bool
	)
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Plan (and, when wired, launch) a local chain network",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := resolveChain(chain, manifestPath, templatePath)
			if err != nil {
				return err
			}
			override := config.Values{}
			if cmd.Flags().Changed("validators") {
				override["nodes.validators"] = strconv.Itoa(validators)
			}
			if cmd.Flags().Changed("endpoints") {
				override["nodes.endpoints"] = strconv.Itoa(endpoints)
			}
			cfg := config.Resolve(nil, override)

			root := dataDir
			if !filepath.IsAbs(root) {
				root = filepath.Clean(root)
			}
			plan, err := setup.BuildPlan(cfg, p, root)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "chain:    %s (family %s, binary %s, chain_id %d)\n",
				p.Manifest().ID, p.Manifest().ConsensusFamily, p.Manifest().Binary, p.Manifest().ChainID)
			fmt.Fprintf(out, "network:  %s\n", plan.Network)
			fmt.Fprintf(out, "dataRoot: %s\n", plan.DataRoot)
			hasTmpl := len(p.GenesisTemplate()) > 0
			fmt.Fprintf(out, "genesis:  template=%v (engine=%q)\n", hasTmpl, p.Manifest().Genesis.EngineField)

			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tROLE\tHOST\tP2P\tHTTP\tWS")
			for _, n := range plan.Nodes {
				fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\n",
					n.Index, n.Role, n.Host, n.Ports.P2P, n.Ports.HTTP, n.Ports.WS)
			}
			if err := w.Flush(); err != nil {
				return err
			}

			if launch {
				bin, err := resolveBinary(binaryPath, p.Manifest().Binary)
				if err != nil {
					return err
				}
				bus, closeBus := obsBus()
				defer closeBus()
				ns, err := setup.Launch(cmd.Context(), setup.LaunchOptions{
					Plugin: p, Config: cfg, DataRoot: root, Binary: bin, KeysDir: keysDir, Bus: bus,
				})
				if err != nil {
					return err
				}
				if err := state.SaveNodeSet(root, ns); err != nil {
					return err
				}
				fmt.Fprintf(out, "launched %d node(s); state: %s\n",
					len(ns.Nodes), filepath.Join(root, "nodeset.json"))
				return nil
			}

			if provision {
				if err := setup.Provision(cmd.Context(), plan, p, cfg, keysDir); err != nil {
					return err
				}
				fmt.Fprintf(out, "provisioned: genesis + %d node config(s) in %s\n", len(plan.Nodes), plan.DataRoot)
				return nil
			}

			if !dryRun {
				return fmt.Errorf("live launch needs --launch (with --binary or a %s on PATH). Use --provision to write artifacts, --dry-run to plan", p.Manifest().Binary)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "embedded chain id (stablenet|wbft|wemix); ignored with --manifest")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to an external chain manifest JSON (project-supplied chain, on a built-in family)")
	cmd.Flags().StringVar(&templatePath, "genesis-template", "", "path to the genesis template for --manifest")
	cmd.Flags().IntVar(&validators, "validators", 0, "override validator count")
	cmd.Flags().IntVar(&endpoints, "endpoints", 0, "override endpoint count")
	cmd.Flags().StringVar(&dataDir, "data-dir", "data", "data root directory")
	cmd.Flags().StringVar(&keysDir, "keys-dir", "keys/preset", "preset keys directory (for --provision)")
	cmd.Flags().StringVar(&binaryPath, "binary", "", "node binary path (for --launch); default: chain binary on PATH")
	cmd.Flags().BoolVar(&provision, "provision", false, "write genesis.json + node configs from preset keys")
	cmd.Flags().BoolVar(&launch, "launch", false, "init datadirs and launch the nodes (implies --provision)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "plan only; do not launch")
	return cmd
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
