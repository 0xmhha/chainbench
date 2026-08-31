package netcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/resourcecmd"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// Step subcommands of `net`. Every RunE is flag binding + one app call +
// rendering; the step logic lives in the app/netcompose layers, shared with
// the MCP tools.

// stepCmd builds a subcommand whose RunE returns a single detail line.
func stepCmd(use, short string, run func(cmd *cobra.Command, dataDir string) (string, error)) (*cobra.Command, *string) {
	var dataDir string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			detail, err := run(cmd, dataDir)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), detail)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory (where the composition is set up)")
	return cmd, &dataDir
}

func newNetKeysCmd() *cobra.Command {
	var source, bootnode string
	var nodes, validators int
	cmd, _ := stepCmd("keys", "Ensure the key set exists and covers the node count (preset or generate)",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetKeys(cmd.Context(), deps(cmd), chainsetup.NetKeysIn{
				DataDir: dataDir, Source: source, Nodes: nodes, Validators: validators,
			})
			return out.Detail, err
		})
	cmd.Flags().StringVar(&source, "keys-source", "preset", "preset (use the recorded key set) | generate (create a fresh set)")
	cmd.Flags().StringVar(&bootnode, "bootnode", "", "deprecated: ignored, BLS material is derived in process")
	_ = cmd.Flags().MarkDeprecated("bootnode", "no longer needed — BLS material is derived in process")
	cmd.Flags().IntVar(&nodes, "nodes", 0, "identities the set must cover (default: the allocated node count)")
	cmd.Flags().IntVar(&validators, "validators", 0, "identities joining the validator set (generate; 0 = all)")
	return cmd
}

func newNetAllocateCmd() *cobra.Command {
	var validators, endpoints int
	var endpointSyncMode, topologyPath, peering string
	var sf resourcecmd.ServerFlags
	cmd, _ := stepCmd("allocate", "Build the node table: roles, paths, deterministic ports",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetAllocate(cmd.Context(), deps(cmd), chainsetup.NetAllocateIn{
				DataDir: dataDir, Validators: validators, Endpoints: endpoints,
				EndpointSyncMode: endpointSyncMode, TopologyPath: topologyPath, Peering: peering,
				Server: sf.Ref(),
			})
			return out.Detail, err
		})
	cmd.Flags().IntVar(&validators, "validators", 4, "validator node count")
	cmd.Flags().IntVar(&endpoints, "endpoints", 0, "endpoint (non-validator) node count")
	cmd.Flags().StringVar(&endpointSyncMode, "endpoint-syncmode", "", "sync mode for endpoints (snap|archive); default full")
	cmd.Flags().StringVar(&topologyPath, "topology", "", "per-node layout YAML (role/sync-mode/bootnode); overrides --validators/--endpoints")
	cmd.Flags().StringVar(&peering, "peering", "", "peer graph: mesh (default, every node dials every other) | proxied (bp <-> pn <-> en; endpoints never dial a producer)")
	sf.Bind(cmd)
	return cmd
}

func newNetGenesisCmd() *cobra.Command {
	var chainID int64
	var sets []string
	var overlay string
	cmd, _ := stepCmd("genesis", "Build the genesis from the key set and write it to the target",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetGenesis(cmd.Context(), deps(cmd), chainsetup.NetGenesisIn{
				DataDir: dataDir, ChainID: chainID, Set: sets, OverlayPath: overlay,
			})
			return out.Detail, err
		})
	cmd.Flags().Int64Var(&chainID, "chain-id", 0, "override the manifest chain id (0 = manifest)")
	cmd.Flags().StringArrayVar(&sets, "set", nil, "override a genesis config key (repeatable), e.g. --set bohoBlock=10")
	cmd.Flags().StringVar(&overlay, "overlay", "", "JSON overlay file {capabilities,genesis} deep-merged into the genesis")
	return cmd
}

func newNetConfigCmd() *cobra.Command {
	cmd, _ := stepCmd("config", "Render and write each node's TOML config",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetConfig(cmd.Context(), deps(cmd), chainsetup.NetConfigIn{DataDir: dataDir})
			return out.Detail, err
		})
	return cmd
}

func newNetLaunchOptsCmd() *cobra.Command {
	var sets []string
	var dataDir string
	cmd := &cobra.Command{
		Use:   "launchopts",
		Short: "Assemble (and show) each node's launch command without running it",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			out, err := chainsetup.NetLaunchOpts(cmd.Context(), deps(cmd), chainsetup.NetLaunchOptsIn{
				DataDir: dataDir, Set: sets,
			})
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fmt.Fprintln(w, out.Detail)
			for _, ns := range out.Nodes {
				fmt.Fprintf(w, "node%d: %s\n", ns.Index, strings.Join(ns.Args, " "))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory (where the composition is set up)")
	cmd.Flags().StringArrayVar(&sets, "set", nil, "high-precedence launch knob key=value (repeatable; bare key for booleans)")
	return cmd
}

func newNetProvisionCmd() *cobra.Command {
	cmd, _ := stepCmd("provision", "Verify the launch inputs are present on the target (skip-if-exists)",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetProvision(cmd.Context(), deps(cmd), chainsetup.NetProvisionIn{DataDir: dataDir})
			return out.Detail, err
		})
	return cmd
}

func newNetInitCmd() *cobra.Command {
	var binary string
	cmd, _ := stepCmd("init", "Initialize each node's datadir from the built genesis",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetInit(cmd.Context(), deps(cmd), chainsetup.NetInitIn{DataDir: dataDir, Binary: binary})
			return out.Detail, err
		})
	cmd.Flags().StringVar(&binary, "binary", "", "node binary path (default: the workspace's)")
	return cmd
}

func newNetStartCmd() *cobra.Command {
	var binary string
	cmd, _ := stepCmd("start", "Launch every stopped node and record its PID",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetStart(cmd.Context(), deps(cmd), chainsetup.NetStartIn{DataDir: dataDir, Binary: binary})
			return out.Detail, err
		})
	cmd.Flags().StringVar(&binary, "binary", "", "node binary path (default: the workspace's)")
	return cmd
}

func newNetStopCmd() *cobra.Command {
	cmd, _ := stepCmd("stop", "Stop every running node by its recorded PID",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetStop(cmd.Context(), deps(cmd), chainsetup.NetStopIn{DataDir: dataDir})
			return out.Detail, err
		})
	return cmd
}

func newNetRestartCmd() *cobra.Command {
	var nodeIdx int
	cmd, _ := stepCmd("restart", "Stop and relaunch one node with its recorded arming",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetRestart(cmd.Context(), deps(cmd), chainsetup.NetRestartIn{DataDir: dataDir, Node: nodeIdx})
			return out.Detail, err
		})
	cmd.Flags().IntVar(&nodeIdx, "node", 0, "node index (1-based)")
	return cmd
}

func newNetRmCmd() *cobra.Command {
	cmd, _ := stepCmd("rm", "Remove the composed data plane (stopped nodes only)",
		func(cmd *cobra.Command, dataDir string) (string, error) {
			out, err := chainsetup.NetRm(cmd.Context(), deps(cmd), chainsetup.NetRmIn{DataDir: dataDir})
			return out.Detail, err
		})
	return cmd
}

func newNetLogsCmd() *cobra.Command {
	var nodeIdx, lines int
	var dataDir string
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show the last lines of one node's log",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			out, err := chainsetup.NetLogs(cmd.Context(), deps(cmd), chainsetup.NetLogsIn{
				DataDir: dataDir, Node: nodeIdx, Lines: lines,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out.Text)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory (where the composition is set up)")
	cmd.Flags().IntVar(&nodeIdx, "node", 0, "node index (1-based)")
	cmd.Flags().IntVar(&lines, "lines", 50, "lines from the end")
	return cmd
}

func newNetHealthCmd() *cobra.Command {
	var dataDir string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Probe every node's HTTP RPC for its latest block",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			out, err := chainsetup.NetHealth(cmd.Context(), deps(cmd), chainsetup.NetHealthIn{DataDir: dataDir})
			if err != nil {
				return err
			}
			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out.Nodes)
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tPID\tBLOCK\tERROR")
			for _, n := range out.Nodes {
				fmt.Fprintf(w, "node%d\t%d\t%d\t%s\n", n.Index, n.PID, n.Block, n.Err)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory (where the composition is set up)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the probe table as JSON")
	return cmd
}

func newNetResumeCmd() *cobra.Command {
	var binary string
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Recover a workspace whose run died: reconcile pids, continue from the first unfinished step, bring nodes back",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dataDir, _ := cmd.Flags().GetString("workspace-dir")
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			out, err := chainsetup.NetResume(cmd.Context(), deps(cmd), chainsetup.NetResumeIn{DataDir: dataDir, Binary: binary})
			w := cmd.OutOrStdout()
			for _, line := range out.Reconciled {
				fmt.Fprintln(w, "reconcile:", line)
			}
			if out.Resumed != "" {
				fmt.Fprintf(w, "resumed from: %s\n", out.Resumed)
			}
			for _, step := range out.Steps {
				fmt.Fprintln(w, step)
			}
			for _, s := range out.Started {
				fmt.Fprintln(w, "started:", s)
			}
			if err != nil {
				return err
			}
			if out.Resumed == "" && len(out.Started) == 0 {
				fmt.Fprintln(w, "nothing to resume: every step is done and every node is running")
			}
			printNodeSet(w, out.Nodes.Nodes)
			return nil
		},
	}
	cmd.Flags().String("workspace-dir", "", "workspace directory (where the composition is set up)")
	cmd.Flags().StringVar(&binary, "binary", "", "node binary path (default: the one the workspace recorded)")
	return cmd
}

// printNodeSet renders a node table.
func printNodeSet(out io.Writer, ns node.NodeSet) {
	if len(ns.Nodes) == 0 {
		return
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NODE\tROLE\tRPC\tPID")
	for _, n := range ns.Nodes {
		fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", n.Index, n.Role, n.RPCURL, n.PID)
	}
	_ = w.Flush()
}
