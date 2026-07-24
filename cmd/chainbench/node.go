package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

func newNodeCmd() *cobra.Command {
	node := &cobra.Command{
		Use:   "node",
		Short: "Inspect a node over RPC",
	}
	node.AddCommand(newNodeRPCCmd())
	return node
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
