package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/core/state"
)

func newTestCmd() *cobra.Command {
	var (
		dataDir    string
		chain      string
		rpcURLs    []string
		names      []string
		categories []string
	)
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run test cases against a network (requirement #10)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ns, err := resolveNodeSet(dataDir, chain, rpcURLs)
			if err != nil {
				return err
			}
			rep, err := testrun.Run(cmd.Context(), ns, testrun.Options{Names: names, Categories: categories})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "CASE\tCATEGORY\tSTATUS\tMESSAGE")
			for _, r := range rep.Results {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Name, r.Category, r.Status, r.Message)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			pass, fail, skip := rep.Counts()
			fmt.Fprintf(out, "\npass=%d fail=%d skip=%d\n", pass, fail, skip)
			if rep.Failed() {
				return fmt.Errorf("%d test(s) failed", fail)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "load NodeSet from <dir>/nodeset.json (from setup)")
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (with --rpc)")
	cmd.Flags().StringArrayVar(&rpcURLs, "rpc", nil, "node RPC URL to attach (repeatable)")
	cmd.Flags().StringArrayVar(&names, "name", nil, "run only these case names")
	cmd.Flags().StringArrayVar(&categories, "category", nil, "run only these categories")
	return cmd
}

// resolveNodeSet builds a NodeSet from explicit RPC endpoints (attach) or from a
// setup's saved nodeset.json.
func resolveNodeSet(dataDir, chain string, rpcURLs []string) (node.NodeSet, error) {
	if len(rpcURLs) > 0 {
		eps := make([]attach.Endpoint, len(rpcURLs))
		for i, u := range rpcURLs {
			eps[i] = attach.Endpoint{RPCURL: u}
		}
		return attach.Build(chain, "attached", eps)
	}
	if dataDir != "" {
		return state.LoadNodeSet(dataDir)
	}
	return node.NodeSet{}, fmt.Errorf("provide --rpc <url> or --data-dir <dir> (from a setup)")
}
