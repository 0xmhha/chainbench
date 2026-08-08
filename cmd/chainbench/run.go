package main

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/engine"
)

// newRunCmd runs DSL test specs through the redesign engine. With --rpc it
// attaches to a running network; with --binary it builds a local one.
func newRunCmd() *cobra.Command {
	var (
		chain        string
		rpcURLs      []string
		binary       string
		keysDir      string
		artifactRoot string
		validators   int
	)
	cmd := &cobra.Command{
		Use:   "run [spec.json ...]",
		Short: "Run DSL test specs through the engine (attach or local)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			specs, err := readSpecFiles(args)
			if err != nil {
				return err
			}
			eng, err := buildRunEngine(chain, rpcURLs, binary, keysDir, artifactRoot, validators)
			if err != nil {
				return err
			}
			root, err := eng.Run(cmd.Context(), specs)
			if err != nil {
				return err
			}
			return printSession(cmd.OutOrStdout(), root)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (e.g. stablenet)")
	cmd.Flags().StringArrayVar(&rpcURLs, "rpc", nil, "attach: node RPC URL (repeatable) — runs against a live network")
	cmd.Flags().StringVar(&binary, "binary", "", "local: node binary path — builds a network")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "local: preset keys directory")
	cmd.Flags().StringVar(&artifactRoot, "artifact-root", "chainbench-out", "session artifact base directory")
	cmd.Flags().IntVar(&validators, "validators", 4, "local: validator node count")
	return cmd
}

// buildRunEngine selects attach mode (when --rpc endpoints are given) or local
// mode (when --binary is given).
func buildRunEngine(chain string, rpcURLs []string, binary, keysDir, artifactRoot string, validators int) (engine.Engine, error) {
	if chain == "" {
		return nil, fmt.Errorf("run: --chain is required")
	}
	switch {
	case len(rpcURLs) > 0:
		return engine.NewAttachEngine(engine.AttachConfig{
			Chain: chain, RPCURLs: rpcURLs, ArtifactRoot: artifactRoot,
		})
	case binary != "":
		return engine.NewLocalEngine(engine.LocalConfig{
			Chain: chain, Binary: binary, KeysDir: keysDir,
			ArtifactRoot: artifactRoot, Validators: validators,
		})
	default:
		return nil, fmt.Errorf("run: provide --rpc <url> (attach) or --binary <path> (local)")
	}
}

// readSpecFiles reads each spec file into raw JSON bytes.
func readSpecFiles(paths []string) ([][]byte, error) {
	specs := make([][]byte, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("run: read spec %s: %w", p, err)
		}
		specs = append(specs, b)
	}
	return specs, nil
}

// printSession reads the saved session and prints a table plus a summary,
// returning a non-nil error when any test failed or was blocked.
func printSession(out io.Writer, root string) error {
	doc, err := engine.ReadSessionSummary(root)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SEQ\tID\tSTATUS")
	for _, tst := range doc.Tests {
		fmt.Fprintf(w, "%d\t%s\t%s\n", tst.Seq, tst.ID, tst.Status)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(out, "\npass=%d fail=%d blocked=%d skip=%d\nsession: %s\n",
		doc.Summary.Pass, doc.Summary.Fail, doc.Summary.Blocked, doc.Summary.Skip, root)
	if doc.Failed() {
		return fmt.Errorf("run: %d failed, %d blocked", doc.Summary.Fail, doc.Summary.Blocked)
	}
	return nil
}
