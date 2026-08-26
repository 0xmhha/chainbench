package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
)

// envFundedKey names the funded-account key variable once, so the reader and
// its error message cannot disagree about what to set.
const envFundedKey = "CHAINBENCH_FUNDED_KEY"

// fundedKeyFromEnv reads an optional funded-account private key from
// envFundedKey (0x-optional hex). It is env-only — never a flag or a
// committed literal — so a secret never lands in shell history or the repo. It
// lets chain-agnostic write cases act on a project-supplied chain (e.g. an L2).
func fundedKeyFromEnv() ([]byte, error) {
	v := strings.TrimSpace(os.Getenv(envFundedKey))
	if v == "" {
		return nil, nil
	}
	key, err := hex.DecodeString(strings.TrimPrefix(v, "0x"))
	if err != nil {
		return nil, fmt.Errorf("%s is not valid hex: %w", envFundedKey, err)
	}
	return key, nil
}

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
			bus, closeBus := obsBus()
			defer closeBus()
			fundedKey, err := fundedKeyFromEnv()
			if err != nil {
				return err
			}
			opts := testrun.Options{Names: names, Categories: categories, Bus: bus, FundedKey: fundedKey}
			// Persist results when a data dir is given, so `report` can read them.
			if dataDir != "" {
				store, err := obs.NewFileStore(filepath.Join(dataDir, "runs.json"))
				if err != nil {
					return err
				}
				opts.Store = store
			}
			rep, err := testrun.Run(cmd.Context(), ns, opts)
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
			fmt.Fprintf(out, "\npass=%d fail=%d skip=%d coverage=%d%% (ran %d of %d applicable)\n",
				pass, fail, skip, rep.Coverage(), pass+fail, rep.Applicable)
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
		eps := make([]node.RPCEndpoint, len(rpcURLs))
		for i, u := range rpcURLs {
			eps[i] = node.RPCEndpoint{RPCURL: u}
		}
		return node.AttachedSet(chain, "attached", eps)
	}
	if dataDir != "" {
		res, err := app.NetworkStatus(context.Background(), app.Deps{}, app.NetworkStatusIn{DataDir: dataDir})
		return res.Nodes, err
	}
	return node.NodeSet{}, fmt.Errorf("provide --rpc <url> or --data-dir <dir> (from a setup)")
}
