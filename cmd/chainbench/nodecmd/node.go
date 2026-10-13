// Package nodecmd owns the commands that act on ONE node of a composed network:
// starting and stopping it individually, and speaking RPC to it. It is the
// surface fault injection uses, so it deliberately offers no whole-network verb
// — that is lifecyclecmd.
package nodecmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

func New() *cobra.Command {
	node := &cobra.Command{
		Use:   "node",
		Short: "Inspect and control individual nodes of a launched network",
	}
	node.AddCommand(newRPCCmd(), newStopCmd(), newStartCmd(), newSwapCmd(), newResetCmd())
	return node
}

// newStopCmd stops a single node of a launched network by index, so a sync
// gap can be created while the rest keep producing blocks.
func newStopCmd() *cobra.Command {
	var (
		dataDir string
		index   int
	)
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a single node by index (--workspace-dir)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.NodeStop(cmd.Context(), surface.Deps(cmd), app.NodeStopIn{DataDir: dataDir, Index: index}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "stopped node%d\n", index)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory")
	cmd.Flags().IntVar(&index, "index", 0, "1-based node index to stop")
	surface.BindLogins(cmd)
	return cmd
}

// newStartCmd relaunches a single previously-stopped node from its saved
// spec, so it rejoins its peers and re-syncs the blocks it missed.
func newStartCmd() *cobra.Command {
	var (
		dataDir string
		index   int
	)
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Relaunch a single stopped node by index (--workspace-dir)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.NodeStart(cmd.Context(), surface.Deps(cmd), app.NodeStartIn{DataDir: dataDir, Index: index})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "started node%d (pid %d)\n", index, res.Node.PID)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory")
	cmd.Flags().IntVar(&index, "index", 0, "1-based node index to relaunch")
	surface.BindLogins(cmd)
	return cmd
}

// newSwapCmd relaunches one node on a different binary and/or configuration,
// keeping its datadir and genesis, so a network can run mixed versions.
func newSwapCmd() *cobra.Command {
	var (
		in      app.NodeSwapIn
		overlay string
	)
	cmd := &cobra.Command{
		Use:   "swap",
		Short: "Relaunch one node on a different binary or configuration (--workspace-dir, --index)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if overlay != "" {
				b, err := os.ReadFile(overlay)
				if err != nil {
					return fmt.Errorf("read --genesis-overlay: %w", err)
				}
				in.GenesisOverlay = b
			}
			res, err := app.NodeSwap(cmd.Context(), surface.Deps(cmd), in)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "swapped node%d (pid %d)\n", in.Index, res.Node.PID)
			return nil
		},
	}
	cmd.Flags().StringVar(&in.DataDir, "workspace-dir", "", "workspace directory")
	cmd.Flags().IntVar(&in.Index, "index", 0, "1-based node index to swap")
	cmd.Flags().StringVar(&in.Binary, "binary", "", "binary to relaunch the node on (empty keeps the current one)")
	cmd.Flags().StringArrayVar(&in.Config, "config", nil, "key=value config override applied before relaunch (repeatable)")
	cmd.Flags().StringVar(&overlay, "genesis-overlay", "", "genesis JSON fragment file merged into this node's genesis")
	cmd.Flags().StringArrayVar(&in.Args, "arg", nil, "flag appended to the node's command line (repeatable)")
	cmd.Flags().StringVar(&in.Purpose, "purpose", "", "name recorded for the config fixture (config-<purpose>)")
	surface.BindLogins(cmd)
	return cmd
}

// newResetCmd stops one non-producing node and initialises its datadir again,
// leaving it down at genesis for a later `node start`.
func newResetCmd() *cobra.Command {
	var in app.NodeResetIn
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset one non-producing node to its genesis and leave it stopped (--workspace-dir, --index)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.NodeReset(cmd.Context(), surface.Deps(cmd), in); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "node%d reset to its genesis and left stopped\n", in.Index)
			return nil
		},
	}
	cmd.Flags().StringVar(&in.DataDir, "workspace-dir", "", "workspace directory")
	cmd.Flags().IntVar(&in.Index, "index", 0, "1-based node index to reset")
	surface.BindLogins(cmd)
	return cmd
}

func newRPCCmd() *cobra.Command {
	var (
		rpcURL     string
		method     string
		paramsJSON string
	)
	cmd := &cobra.Command{
		Use:   "rpc",
		Short: "Call an arbitrary JSON-RPC method and print the raw result",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rpcURL == "" || method == "" {
				return fmt.Errorf("--rpc and --method are required")
			}
			params, err := decodeParams(paramsJSON)
			if err != nil {
				return err
			}
			raw, err := app.NodeCall(cmd.Context(), surface.Deps(cmd), app.NodeCallIn{
				RPC: rpcURL, Method: method, Params: params,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return nil
		},
	}
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	cmd.Flags().StringVar(&method, "method", "", "JSON-RPC method (e.g. eth_blockNumber)")
	cmd.Flags().StringVar(&paramsJSON, "params", "", "JSON array of params, e.g. '[\"latest\",false]'")
	return cmd
}

// decodeParams parses a JSON array of params, or returns nil for an empty
// string.
func decodeParams(s string) ([]any, error) {
	if s == "" {
		return nil, nil
	}
	var params []any
	if err := json.Unmarshal([]byte(s), &params); err != nil {
		return nil, fmt.Errorf("bad --params (JSON array expected): %w", err)
	}
	return params, nil
}
