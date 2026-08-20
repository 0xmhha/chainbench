package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func newNodeCmd() *cobra.Command {
	node := &cobra.Command{
		Use:   "node",
		Short: "Inspect and control individual nodes of a launched network",
	}
	node.AddCommand(newNodeRPCCmd(), newNodeStopCmd(), newNodeStartCmd())
	return node
}

// newNodeStopCmd stops a single node of a launched network by index, so a sync
// gap can be created while the rest keep producing blocks.
func newNodeStopCmd() *cobra.Command {
	var (
		dataDir string
		index   int
	)
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a single launched node by index (--data-dir from setup)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.NodeStop(cmd.Context(), app.Deps{}, app.NodeStopIn{DataDir: dataDir, Index: index}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "stopped node%d\n", index)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root with nodeset.json (from setup)")
	cmd.Flags().IntVar(&index, "index", 0, "1-based node index to stop")
	return cmd
}

// newNodeStartCmd relaunches a single previously-stopped node from its saved
// spec, so it rejoins its peers and re-syncs the blocks it missed.
func newNodeStartCmd() *cobra.Command {
	var (
		dataDir string
		index   int
	)
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Relaunch a single stopped node by index (--data-dir from setup)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.NodeStart(cmd.Context(), app.Deps{}, app.NodeStartIn{DataDir: dataDir, Index: index})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "started node%d (pid %d)\n", index, res.Node.PID)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root with nodeset.json + nodespecs.json (from setup)")
	cmd.Flags().IntVar(&index, "index", 0, "1-based node index to relaunch")
	return cmd
}

func newNodeRPCCmd() *cobra.Command {
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
			var raw json.RawMessage
			if err := rpc.Dial(rpcURL).Call(cmd.Context(), method, &raw, params...); err != nil {
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
