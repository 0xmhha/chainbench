package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/resourcecmd"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/home"
	"github.com/0xmhha/chainbench/internal/dashboard"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// runReport is the --json shape for a run: the session path plus the verdict
// (testengine.Summary is embedded so its tests/summary fields flatten in).
type runReport struct {
	Session string `json:"session"`
	testengine.Summary
}

// newRunCmd runs DSL test specs through the test engine. With --workspace-dir
// it composes the network the specs declare through the workspace steps (a
// handoff env composes the handoff) and runs against that; with --rpc it
// attaches to a running network. The engine's self-assembly build path is
// gone (R4): composition belongs to chainsetup alone.
func newRunCmd() *cobra.Command {
	var (
		chain        string
		rpcURLs      []string
		binary       string
		keysDir      string
		keysSource   string
		artifactRoot string
		validators   int
		chainID      int64
		networkID    int64
		launchOpts   []string
		dashboardURL string
		jsonOut      bool
		workspaceDir string
		keepUp       bool
		waitBlocks   uint64
		docker       bool
		sf           resourcecmd.ServerFlags
	)
	cmd := &cobra.Command{
		Use:   "run [spec.json ...]",
		Short: "Run DSL test specs (compose the declared network, or attach)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case len(rpcURLs) > 0 && workspaceDir != "":
				return fmt.Errorf("run: --workspace-dir composes a network; it does not combine with --rpc")
			case len(rpcURLs) > 0:
				return runAttach(cmd, args, chain, rpcURLs, artifactRoot, keysDir, dashboardURL, jsonOut)
			case workspaceDir == "":
				return fmt.Errorf("run: provide --workspace-dir <dir> (compose the network the specs declare) or --rpc <url> (attach to a running one)")
			}
			in := app.RunSuiteIn{
				SpecPaths: args, DataDir: workspaceDir, Chain: chain,
				Binary: binary, Server: sf.Ref(), Docker: docker, KeepUp: keepUp, WaitBlocks: waitBlocks,
				ChainID: chainID, NetworkID: networkID, LaunchOpts: launchOpts,
			}
			if cmd.Flags().Changed("keys") {
				in.KeysDir = keysDir
			}
			if cmd.Flags().Changed("keys-source") {
				in.KeysSource = keysSource
			}
			if cmd.Flags().Changed("validators") {
				in.Validators = validators
			}
			if cmd.Flags().Changed("artifact-root") {
				in.ArtifactRoot = artifactRoot
			}
			return runComposed(cmd, in, jsonOut)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (e.g. stablenet); required to attach, with --workspace-dir it must agree with what the specs declare and may be omitted")
	cmd.Flags().StringVar(&workspaceDir, "workspace-dir", "", "compose: workspace where the network the specs declare is set up, then run against it")
	cmd.Flags().BoolVar(&keepUp, "keep-up", false, "compose: leave the network running after the run")
	cmd.Flags().Uint64Var(&waitBlocks, "wait-blocks", 0, "compose: wait until the head reaches this height before running")
	cmd.Flags().StringArrayVar(&rpcURLs, "rpc", nil, "attach: node RPC URL (repeatable) — runs against a live network")
	cmd.Flags().StringVar(&binary, "binary", "", "compose: node binary path, overriding what the specs declare")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "compose: key set directory, overriding what the specs declare")
	cmd.Flags().StringVar(&keysSource, "keys-source", "preset",
		"compose: where node identities come from — preset (use --keys as-is) | generate (create a fresh set in --keys)")
	cmd.Flags().StringVar(&artifactRoot, "artifact-root", defaultArtifactRoot(),
		"session artifact base directory (compose default: the workspace's sessions directory)")
	cmd.Flags().IntVar(&validators, "validators", 4, "compose: validator node count, overriding what the specs declare")
	cmd.Flags().Int64Var(&chainID, "chain-id", 0, "compose: override the chain id in the built genesis (0 = declared/manifest)")
	cmd.Flags().Int64Var(&networkID, "network-id", 0, "compose: pin the devp2p network id on every node (0 = binary default)")
	cmd.Flags().StringArrayVar(&launchOpts, "launch-opt", nil,
		"compose: high-precedence launch knob key=value (repeatable; bare key for boolean flags, e.g. nodiscover)")
	sf.Bind(cmd)
	cmd.Flags().BoolVar(&docker, "docker", false,
		"compose: the server set's hosts are local docker containers — translate this tool's dials via the localmap next to the server set (addresses only; docker itself is untouched)")
	cmd.Flags().StringVar(&dashboardURL, "dashboard", "", "attach: chainbench-dashboard URL to stream run events to (e.g. http://127.0.0.1:8787)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the session summary as JSON instead of a table")
	return cmd
}

// defaultArtifactRoot is where a run's session lands when no root is named.
//
// A composed run keeps its sessions beside its workspace; this is the answer
// for attaching to a network somebody else is running, which has no workspace
// to keep them beside. It used to be the relative "chainbench-out", which
// scattered sessions across the filesystem one working directory at a time.
func defaultArtifactRoot() string {
	d, err := home.Sessions()
	if err != nil {
		return "chainbench-out"
	}
	return d
}

// runAttach runs the specs against a running network over its RPC endpoints,
// optionally streaming orchestration events to a dashboard. Emission never
// blocks the run; the bus is closed and the forwarder drained before exiting
// so buffered events are flushed.
func runAttach(cmd *cobra.Command, args []string, chain string, rpcURLs []string, artifactRoot, keysDir, dashboardURL string, jsonOut bool) error {
	if chain == "" {
		return fmt.Errorf("run: --chain is required to attach")
	}
	specs, err := dsl.ReadFiles(args)
	if err != nil {
		return err
	}
	var bus *collector.Bus
	var forwardDone <-chan struct{}
	if dashboardURL != "" {
		bus = collector.NewBus()
		forwardDone = dashboard.Forward(bus, dashboardURL, nil)
	}
	flush := func() {
		if bus != nil {
			bus.Close()
			<-forwardDone
		}
	}
	eng, err := testengine.NewAttachEngine(testengine.AttachConfig{
		Chain: chain, RPCURLs: rpcURLs, ArtifactRoot: artifactRoot, Bus: bus,
		// The key set is what turns "node1" in a spec into an address. An
		// operator attaching to a network they composed has it; passing it here
		// is the difference between labels working and a spec having to paste
		// hex it would have to update whenever the keys change.
		KeysDir: keysDir,
	})
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
}

// runComposed composes the network the specs declare and runs them against
// it, printing the setup steps before the session.
func runComposed(cmd *cobra.Command, in app.RunSuiteIn, jsonOut bool) error {
	out := cmd.OutOrStdout()
	res, err := app.RunSuite(cmd.Context(), app.Deps{}, in)
	for _, step := range res.SetupSteps {
		fmt.Fprintln(out, step)
	}
	if res.Preflight != "" {
		fmt.Fprintf(out, "preflight: %s\n", res.Preflight)
	}
	if err != nil {
		return err
	}
	return printSession(out, res.SessionRoot, jsonOut)
}

// printSession reads the saved session and prints a table plus a summary,
// returning a non-nil error when any test failed or was blocked.
func printSession(out io.Writer, root string, jsonOut bool) error {
	doc, err := testengine.ReadSessionSummary(root)
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
