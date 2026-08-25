package main

import (
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/cmd/chainbench/internal/serverflag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/dashboard"
	"github.com/0xmhha/chainbench/internal/engine"
	"github.com/0xmhha/chainbench/internal/testspec"
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
		keysSource   string
		bootnode     string // deprecated, ignored
		artifactRoot string
		validators   int
		chainID      int64
		networkID    int64
		launchOpts   []string
		dashboardURL string
		jsonOut      bool
		sf           serverflag.Flags
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
			// bus and forward them to the running chainbench-dashboard. Emission never
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

			eng, err := buildRunEngine(foldSpecEnv(runOpts{
				chain: chain, rpcURLs: rpcURLs, binary: binary,
				keysDir: keysDir, keysSource: keysSource,
				artifactRoot: artifactRoot, validators: validators,
				chainID: chainID, networkID: networkID, launchOpts: launchOpts,
				server: sf.Ref(), bus: bus,
			}, specs))
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
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "local: key set directory (read with --keys-source preset, written with generate)")
	cmd.Flags().StringVar(&keysSource, "keys-source", string(keysSourcePreset),
		"local: where node identities come from — preset (use --keys as-is) | generate (create a fresh set in --keys)")
	cmd.Flags().StringVar(&bootnode, "bootnode", "", "deprecated: ignored, BLS material is derived in process")
	_ = cmd.Flags().MarkDeprecated("bootnode", "no longer needed — BLS material is derived in process")
	cmd.Flags().StringVar(&artifactRoot, "artifact-root", "chainbench-out", "session artifact base directory")
	cmd.Flags().IntVar(&validators, "validators", 4, "local: validator node count")
	cmd.Flags().Int64Var(&chainID, "chain-id", 0, "local: override the manifest chain id in the built genesis (0 = manifest)")
	cmd.Flags().Int64Var(&networkID, "network-id", 0, "local: pin the devp2p network id on every node (0 = binary default)")
	cmd.Flags().StringArrayVar(&launchOpts, "launch-opt", nil,
		"local: high-precedence launch knob key=value (repeatable; bare key for boolean flags, e.g. nodiscover)")
	sf.Bind(cmd)
	cmd.Flags().StringVar(&dashboardURL, "dashboard", "", "chainbench-dashboard URL to stream run events to (e.g. http://127.0.0.1:8787)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the session summary as JSON instead of a table")
	return cmd
}

// keysSourceKind names where a local run's node identities come from. Typed so
// the accepted values live in one place instead of as literals at the parse site.
type keysSourceKind string

const (
	// keysSourcePreset uses the --keys directory exactly as it is on disk: the
	// same identities, and therefore the same chain, on every run.
	keysSourcePreset keysSourceKind = "preset"
	// keysSourceGenerate creates a fresh random identity set in --keys the first
	// time and reuses it afterwards.
	keysSourceGenerate keysSourceKind = "generate"
)

// runOpts are the resolved flags for one `run` invocation. Grouped into a struct
// because the option count had outgrown a readable parameter list.
type runOpts struct {
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
	// server selects the node placement (ports, host, capacity) from the
	// operator's server set; its zero value uses the built-in local plan.
	server app.ServerRef
	bus    *obs.Bus
}

// buildRunEngine selects attach mode (when --rpc endpoints are given) or local
// mode (when --binary is given). A non-nil bus streams orchestration events.
func buildRunEngine(o runOpts) (engine.Engine, error) {
	if o.chain == "" {
		return nil, fmt.Errorf("run: --chain is required")
	}
	switch {
	case len(o.rpcURLs) > 0:
		return engine.NewAttachEngine(engine.AttachConfig{
			Chain: o.chain, RPCURLs: o.rpcURLs, ArtifactRoot: o.artifactRoot, Bus: o.bus,
		})
	case o.binary != "":
		src, err := keySource(o)
		if err != nil {
			return nil, err
		}
		overrides, err := parseLaunchOverrides(o.launchOpts)
		if err != nil {
			return nil, err
		}
		placement, err := app.ResolveServer(app.Deps{}, o.server, runMinValidators, runPortBand)
		if err != nil {
			return nil, err
		}
		return engine.NewLocalEngine(engine.LocalConfig{
			Chain: o.chain, Binary: o.binary, Keys: src,
			ArtifactRoot: o.artifactRoot, Validators: o.validators,
			ChainID: o.chainID, NetworkID: o.networkID, LaunchOverrides: overrides,
			Placement: placement.Placement, Bus: o.bus,
		})
	default:
		return nil, fmt.Errorf("run: provide --rpc <url> (attach) or --binary <path> (local)")
	}
}

// Placement bounds for a local run: a BFT floor and the local port band.
const (
	runMinValidators = 1
	runPortBand      = 100
)

// keySource maps --keys-source to the engine seam that materializes identities.
func keySource(o runOpts) (engine.KeySource, error) {
	if o.keysDir == "" {
		return nil, fmt.Errorf("run: --keys is required for a local run")
	}
	switch keysSourceKind(o.keysSource) {
	case "", keysSourcePreset:
		return engine.PresetKeySource{Path: o.keysDir}, nil
	case keysSourceGenerate:
		return engine.GeneratedKeySource{
			Path: o.keysDir, Validators: o.validators,
		}, nil
	default:
		return nil, fmt.Errorf("run: unknown --keys-source %q (want %s or %s)",
			o.keysSource, keysSourcePreset, keysSourceGenerate)
	}
}

// parseLaunchOverrides maps --launch-opt key=value pairs onto typed launchopt
// overrides. Keys are the chain-agnostic knob names (launchopt.Key); whether a
// key exists for the target binary is checked at assembly time by the Builder,
// which classifies an unsupported knob as an error rather than dropping it.
func parseLaunchOverrides(opts []string) ([]launchopt.Override, error) {
	out := make([]launchopt.Override, 0, len(opts))
	for _, o := range opts {
		k, v, _ := strings.Cut(o, "=")
		if k == "" {
			return nil, fmt.Errorf("run: bad --launch-opt %q (want key=value or a bare boolean key)", o)
		}
		out = append(out, launchopt.Override{Key: launchopt.Key(k), Value: v})
	}
	return out, nil
}

// readSpecFiles reads each spec file into raw JSON bytes, resolving a v2
// case's "env": "<id>" reference against the case file's directory
// (<dir>/<id>.env.json, then <dir>/env/<id>.env.json).
func readSpecFiles(paths []string) ([][]byte, error) {
	specs := make([][]byte, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("run: read spec %s: %w", p, err)
		}
		dir := filepath.Dir(p)
		b, err = testspec.InlineEnv(b, func(id string) ([]byte, error) {
			for _, cand := range []string{
				filepath.Join(dir, id+".env.json"),
				filepath.Join(dir, "env", id+".env.json"),
			} {
				if eb, err := os.ReadFile(cand); err == nil {
					return eb, nil
				}
			}
			return nil, fmt.Errorf("no %s.env.json next to %s (or in its env/ subdirectory)", id, p)
		})
		if err != nil {
			return nil, fmt.Errorf("run: %w", err)
		}
		specs = append(specs, b)
	}
	return specs, nil
}

// foldSpecEnv folds a single spec's v2 env declarations (keys, launch) into
// the run options where the CLI did not already decide: explicit flags win
// over the spec, and multiple specs get no folding (their envs could
// disagree; the engine seams are per-invocation).
func foldSpecEnv(o runOpts, specs [][]byte) runOpts {
	if len(specs) != 1 {
		return o
	}
	s, err := testspec.Parse(specs[0])
	if err != nil {
		return o // the engine reports the parse error with full context
	}
	if k := s.EnvKeys; k != nil {
		if o.keysSource == "" || o.keysSource == string(keysSourcePreset) {
			o.keysSource = k.Source
		}
		if o.keysDir == "" || o.keysDir == "keys/preset" {
			if k.Ref != "" {
				o.keysDir = k.Ref
			}
		}
	}
	if len(s.EnvLaunch) > 0 && len(o.launchOpts) == 0 {
		for _, kv := range s.EnvLaunch {
			if kv.Value == "" || kv.Value == "true" {
				o.launchOpts = append(o.launchOpts, kv.Key)
			} else {
				o.launchOpts = append(o.launchOpts, kv.Key+"="+kv.Value)
			}
		}
	}
	return o
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
