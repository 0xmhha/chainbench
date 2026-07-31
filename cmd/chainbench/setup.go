package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/chains/external"
	"github.com/0xmhha/chainbench/pkg/core/config"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/setup"
	"github.com/0xmhha/chainbench/pkg/core/registry"
	"github.com/0xmhha/chainbench/pkg/core/state"
	"github.com/0xmhha/chainbench/pkg/core/topology"
)

// saveTopology copies the resolved topology file into the data root as
// topology.yaml, so the running network's layout is inspectable from its datadir
// (which node plays which role). A no-op when no topology file was used.
func saveTopology(root, topologyPath string) error {
	if topologyPath == "" {
		return nil
	}
	b, err := os.ReadFile(topologyPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "topology.yaml"), b, 0o644)
}

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
			// An optional topology file gives an explicit per-node layout (role,
			// sync mode, bootnode) instead of the positional validator/endpoint
			// counts. Its `chain` selects the chain unless --chain is given.
			var topo *topology.Topology
			if topologyPath != "" {
				loaded, err := topology.Load(topologyPath)
				if err != nil {
					return err
				}
				topo = &loaded
				if !cmd.Flags().Changed("chain") && loaded.Chain != "" {
					chain = loaded.Chain
				}
			}
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
			// --set key=value overrides an arbitrary flat config key, e.g.
			// --set genesis.overrides.bohoBlock=10 for a delayed-fork network.
			for _, kv := range setValues {
				k, v, ok := strings.Cut(kv, "=")
				if !ok || k == "" {
					return fmt.Errorf("--set expects key=value, got %q", kv)
				}
				override[k] = v
			}
			// --genesis-overlay <path> supplies a JSON overlay file
			// {capabilities:[...], genesis:{...}}: the genesis fragment is
			// deep-merged into the built genesis and the capabilities are advertised
			// on the NodeSet so overlay-gated cases (e.g. account-extra) run.
			if genesisOverlay != "" {
				raw, err := os.ReadFile(genesisOverlay)
				if err != nil {
					return err
				}
				var ov struct {
					Capabilities []string        `json:"capabilities"`
					Genesis      json.RawMessage `json:"genesis"`
				}
				if err := json.Unmarshal(raw, &ov); err != nil {
					return fmt.Errorf("bad --genesis-overlay %q: %w", genesisOverlay, err)
				}
				if len(ov.Genesis) > 0 {
					override["genesis.overlay"] = string(ov.Genesis)
				}
				if len(ov.Capabilities) > 0 {
					override["genesis.capabilities"] = strings.Join(ov.Capabilities, ",")
				}
			}
			cfg := config.Resolve(nil, override)

			root := dataDir
			if !filepath.IsAbs(root) {
				root = filepath.Clean(root)
			}
			plan, err := setup.BuildPlanWithTopology(cfg, p, root, topo)
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
			fmt.Fprintln(w, "NODE\tROLE\tSYNC\tHOST\tP2P\tHTTP\tWS")
			for _, n := range plan.Nodes {
				sync := n.SyncMode
				if sync == "" {
					sync = "-"
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\t%d\n",
					n.Index, n.Role, sync, n.Host, n.Ports.P2P, n.Ports.HTTP, n.Ports.WS)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if topo != nil && topo.BootnodeIndex() > 0 {
				fmt.Fprintf(out, "bootnode: node %d\n", topo.BootnodeIndex())
			}

			if launch {
				// A remote host routes launch through the SSH RemoteDriver; nil
				// keeps the default local driver.
				remoteDrv, err := remoteDriver(remoteHost, remoteUser, remotePort)
				if err != nil {
					return err
				}
				// For a remote launch the binary path lives on the remote host,
				// so it is used as-is; only a local launch resolves/stats it.
				bin := binaryPath
				if remoteDrv == nil {
					if bin, err = resolveBinary(binaryPath, p.Manifest().Binary); err != nil {
						return err
					}
				} else if bin == "" {
					return fmt.Errorf("--binary <remote path> is required with --remote-host")
				}
				bus, closeBus := obsBus()
				defer closeBus()
				ns, specs, err := setup.LaunchWithSpecs(cmd.Context(), setup.LaunchOptions{
					Plugin: p, Config: cfg, DataRoot: root, Binary: bin, KeysDir: keysDir, Bus: bus,
					Driver: remoteDrv,
				})
				if err != nil {
					return err
				}
				if err := state.SaveNodeSet(root, ns); err != nil {
					return err
				}
				// Persist the armed specs so `node start --index` can relaunch a
				// single node after `node stop --index`.
				if err := state.SaveNodeSpecs(root, specs); err != nil {
					return err
				}
				if err := saveTopology(root, topologyPath); err != nil {
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
				if err := saveTopology(root, topologyPath); err != nil {
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
	cmd.Flags().StringVar(&topologyPath, "topology", "", "per-node topology YAML (role/sync-mode/bootnode per node); overrides --validators/--endpoints")
	cmd.Flags().StringArrayVar(&setValues, "set", nil, "override a flat config key (repeatable), e.g. --set genesis.overrides.bohoBlock=10")
	cmd.Flags().StringVar(&genesisOverlay, "genesis-overlay", "", "JSON overlay file {capabilities,genesis} deep-merged into the genesis (e.g. pkg/chains/stablenet/overlays/account-extra.json)")
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
