package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/dashboard"
	"github.com/0xmhha/chainbench/internal/engine"
)

// runReport is the --json shape for a run: the session path plus the verdict
// (engine.Summary is embedded so its tests/summary fields flatten in).
type runReport struct {
	Session string `json:"session"`
	engine.Summary
}

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
		dashboardURL string
		jsonOut      bool
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

			// When a dashboard is given, publish orchestration events to a local
			// bus and forward them to the running chainbenchd. Emission never
			// blocks the run; we close the bus and drain the forwarder before
			// exiting so buffered events are flushed.
			var bus *obs.Bus
			var forwardDone <-chan struct{}
			if dashboardURL != "" {
				bus = obs.NewBus()
				forwardDone = dashboard.Forward(bus, dashboardURL, nil)
			}
			flush := func() {
				if bus != nil {
					bus.Close()
					<-forwardDone
				}
			}

			eng, err := buildRunEngine(chain, rpcURLs, binary, keysDir, artifactRoot, validators, bus)
			if err != nil {
				flush()
				return err
			}
			root, err := eng.Run(cmd.Context(), specs)
			flush()
			if err != nil {
				return err
			}
			return printSession(cmd.OutOrStdout(), root, jsonOut)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (e.g. stablenet)")
	cmd.Flags().StringArrayVar(&rpcURLs, "rpc", nil, "attach: node RPC URL (repeatable) — runs against a live network")
	cmd.Flags().StringVar(&binary, "binary", "", "local: node binary path — builds a network")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "local: preset keys directory")
	cmd.Flags().StringVar(&artifactRoot, "artifact-root", "chainbench-out", "session artifact base directory")
	cmd.Flags().IntVar(&validators, "validators", 4, "local: validator node count")
	cmd.Flags().StringVar(&dashboardURL, "dashboard", "", "chainbenchd URL to stream run events to (e.g. http://127.0.0.1:8787)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the session summary as JSON instead of a table")
	return cmd
}

// buildRunEngine selects attach mode (when --rpc endpoints are given) or local
// mode (when --binary is given). A non-nil bus streams orchestration events.
func buildRunEngine(chain string, rpcURLs []string, binary, keysDir, artifactRoot string, validators int, bus *obs.Bus) (engine.Engine, error) {
	if chain == "" {
		return nil, fmt.Errorf("run: --chain is required")
	}
	switch {
	case len(rpcURLs) > 0:
		return engine.NewAttachEngine(engine.AttachConfig{
			Chain: chain, RPCURLs: rpcURLs, ArtifactRoot: artifactRoot, Bus: bus,
		})
	case binary != "":
		return engine.NewLocalEngine(engine.LocalConfig{
			Chain: chain, Binary: binary, KeysDir: keysDir,
			ArtifactRoot: artifactRoot, Validators: validators, Bus: bus,
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
func printSession(out io.Writer, root string, jsonOut bool) error {
	doc, err := engine.ReadSessionSummary(root)
	if err != nil {
		return err
	}

	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(runReport{Session: root, Summary: doc}); err != nil {
			return err
		}
	} else {
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
	}
	if doc.Failed() {
		// Blocked/infrastructure errors are more severe than a plain test
		// failure, so they map to exit code 2 (F16-O5).
		code := 1
		if doc.Summary.Blocked > 0 {
			code = 2
		}
		return &exitError{code: code, err: fmt.Errorf("run: %d failed, %d blocked", doc.Summary.Fail, doc.Summary.Blocked)}
	}
	return nil
}
