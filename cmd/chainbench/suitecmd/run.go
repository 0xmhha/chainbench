// Package suitecmd owns RUNNING test specs: composing or attaching to the
// network a spec declares and executing it (run), checking a spec without running
// it (validate), and lifting a v1 spec to v2 (migrate-spec).
//
// validate is here rather than in catalogcmd because it resolves a spec against
// the registered vocabulary, which is the same resolution run performs — one
// answer, two moments.
package suitecmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/exitcode"

	"github.com/0xmhha/chainbench/cmd/chainbench/resourcecmd"
	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/home"
	"github.com/0xmhha/chainbench/internal/dashboard"
	"github.com/0xmhha/chainbench/internal/preset"
)

// runReport is the --json shape for a run: the session path plus the verdict
// (app.RunSummary is embedded so its tests/summary fields flatten in).
type runReport struct {
	Session string `json:"session"`
	// Error is why the run produced no session at all — a chain that would not
	// compose, most often. It is omitted when there is one, so a successful
	// document is byte for byte what it was before this field existed.
	Error string `json:"error,omitempty"`
	// FailedAt is the state the run failed in, so a consumer can tell a
	// declaration that is wrong from a network that would not stand up without
	// matching on the message.
	FailedAt string `json:"failedAt,omitempty"`
	app.RunSummary
}

// NewRun runs DSL test specs through the test engine. With --workspace-dir
// it composes the network the specs declare through the workspace steps (a
// handoff env composes the handoff) and runs against that; with --rpc it
// attaches to a running network. The engine's self-assembly build path is
// gone (R4): composition belongs to chainsetup alone.
func NewRun() *cobra.Command {
	var (
		chain           string
		rpcURLs         []string
		binary          string
		keysDir         string
		keysSource      string
		artifactRoot    string
		bpCount         int
		chainID         int64
		networkID       int64
		launchOpts      []string
		dashboardURL    string
		jsonOut         bool
		noSkips         bool
		workspaceDir    string
		workspaceConfig string
		keepUp          bool
		waitBlocks      uint64
		nodeMonitorT    time.Duration
		docker          bool
		attach          bool
		planOnly        bool
		presetRef       string
		sf              resourcecmd.ServerFlags
	)
	cmd := &cobra.Command{
		Use:   "run [spec.json ...]",
		Short: "Run DSL test specs (compose the declared network, or attach)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Which of the four ways this run was asked to reach a network.
			// The answer is a state, and it is worked out in one place that
			// every surface shares — this switch used to be written here, in
			// the MCP tool, and again inside AttachRun, and the three did not
			// agree.
			start, serr := app.StartFor(app.Named{
				Spell: app.CLISpelling, RPCURLs: rpcURLs, Attach: attach,
				WorkspaceDir: workspaceDir,
				Specs: func() ([][]byte, []string, error) {
					specs, err := app.ReadSpecFiles(args)
					return specs, args, err
				},
			})
			if serr != nil {
				return fmt.Errorf("run: %w", serr)
			}
			if start.Adopts() {
				// ${VAR} in a declared endpoint is expanded here because the
				// surface owns the process environment. A committed case must
				// not have to carry a machine's address.
				if start.Declared != nil {
					start.Declared.RPCURLs = expandEach(start.Declared.RPCURLs)
				}
				// A flag's value is passed only when the flag was given, which
				// is how "what the caller said outranks the document" is said
				// to the layer that applies it.
				if !cmd.Flags().Changed("keys") {
					keysDir = ""
				}
				return runAttached(cmd, args, start, workspaceDir, chain, rpcURLs,
					artifactRoot, keysDir, dashboardURL, jsonOut, noSkips)
			}
			in := app.RunSuiteIn{
				SpecPaths: args, DataDir: workspaceDir, Chain: chain, Env: presetRef,
				Binary: binary, Server: sf.Ref(), Docker: docker, KeepUp: keepUp, WaitBlocks: waitBlocks,
				ChainID: chainID, NetworkID: networkID, LaunchOpts: launchOpts,
				NodeMonitorTimeout: nodeMonitorT,
			}
			if cmd.Flags().Changed("keys") {
				in.KeysDir = keysDir
			}
			if cmd.Flags().Changed("keys-source") {
				in.KeysSource = keysSource
			}
			if cmd.Flags().Changed("bp") {
				in.BPCount = bpCount
			}
			if cmd.Flags().Changed("artifact-root") {
				in.ArtifactRoot = artifactRoot
			}
			if cmd.Flags().Changed("workspace-config") {
				in.WorkspaceConfigPath = workspaceConfig
			}
			if planOnly {
				return showPlan(cmd, in, jsonOut)
			}
			// The plan goes to stderr, not stdout: --json promises a document
			// and a run that printed prose above it would break every reader.
			in.OnPlan = planPrinter(cmd.ErrOrStderr())
			if len(args) > 1 {
				// Several definitions are the same run repeated, in the order
				// given; the network is kept up between them so each one's own
				// preflight decides whether to reuse it.
				return runComposedSequence(cmd, in, jsonOut, noSkips)
			}
			return runComposed(cmd, in, jsonOut, noSkips)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (e.g. stablenet); required to attach, with --workspace-dir it must agree with what the specs declare and may be omitted")
	cmd.Flags().StringVar(&workspaceDir, "workspace-dir", "", "compose: workspace where the network the specs declare is set up, then run against it")
	cmd.Flags().StringVar(&workspaceConfig, "workspace-config", "", "compose: environment file owning the target dataRoot and its purpose directories; the same DSL runs across targets by swapping this file")
	cmd.Flags().StringVar(&presetRef, "chain-preset", "", "compose: run every case on this chain-preset instead of the one it names (an id, or a path to a chain-preset file); what a case overrode is kept")
	cmd.Flags().BoolVar(&planOnly, "plan", false, "compose: print the network the specs and flags resolve to, then stop without composing it")
	cmd.Flags().BoolVar(&keepUp, "keep-up", false, "compose: leave the network running after the run")
	cmd.Flags().Uint64Var(&waitBlocks, "wait-blocks", 0, "compose: wait until the head reaches this height before running")
	cmd.Flags().DurationVar(&nodeMonitorT, "node-monitor-timeout", 0, "compose: how long the readiness gate waits on nodes still coming up (0 = default; raise for a large/slow bring-up, e.g. 5m for a 15-node poa network over docker)")
	cmd.Flags().StringArrayVar(&rpcURLs, "rpc", nil, "attach: node RPC URL (repeatable) — runs against a live network")
	cmd.Flags().BoolVar(&attach, "attach", false,
		"attach: the network --workspace-dir composed is already up — run against it, with the capabilities it advertised, instead of composing again")
	cmd.Flags().StringVar(&binary, "binary", "", "compose: node binary path, overriding what the specs declare")
	cmd.Flags().StringVar(&keysDir, "keys", preset.KeysDir, "compose: key set directory, overriding what the specs declare")
	cmd.Flags().StringVar(&keysSource, "keys-source", "keyPreset",
		"compose: where node identities come from — keyPreset (use --keys as-is) | generate (create a fresh set in --keys)")
	cmd.Flags().StringVar(&artifactRoot, "artifact-root", defaultArtifactRoot(),
		"session artifact base directory; overrides workspace-config's control.artifactRoot "+
			"(default: ~/.chainbench/sessions — outside the workspace, so removing the workspace "+
			"does not remove the record of what was tested)")
	cmd.Flags().IntVar(&bpCount, "bp", 4, "compose: bp node count, overriding what the specs declare")
	cmd.Flags().Int64Var(&chainID, "chain-id", 0, "compose: override the chain id in the built genesis (0 = declared/manifest)")
	cmd.Flags().Int64Var(&networkID, "network-id", 0, "compose: pin the devp2p network id on every node (0 = binary default)")
	cmd.Flags().StringArrayVar(&launchOpts, "launch-opt", nil,
		"compose: high-precedence launch knob key=value (repeatable; bare key for boolean flags, e.g. nodiscover)")
	sf.Bind(cmd)
	cmd.Flags().BoolVar(&docker, "docker", false,
		"compose: the server set's hosts are local docker containers — translate this tool's dials via the localmap next to the server set (addresses only; docker itself is untouched)")
	cmd.Flags().StringVar(&dashboardURL, "dashboard", "", "attach: chainbench-dashboard URL to stream run events to (e.g. http://127.0.0.1:8787)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the session summary as JSON instead of a table")
	cmd.Flags().BoolVar(&noSkips, "no-skips", false,
		"treat a skipped case as a failure — for a run whose point is that the cases were ELIGIBLE, "+
			"such as one booting a network with a capability the cases gate on. A silent skip there means the "+
			"capability never reached them, and the run says pass=0 skip=N and exits 0 without it")
	return cmd
}

// defaultArtifactRoot is where a run's artifacts land when no root is named.
//
// One answer for every run, composed or attached. A composed run used to keep
// its artifacts beside its workspace, which tied the record of what was tested
// to the scratch directory it was tested in: removing the workspace to reclaim
// disk, or to force a clean compose, took every verdict and every log with it.
// Before that it was the relative "chainbench-out", which scattered them across
// the filesystem one working directory at a time.
func defaultArtifactRoot() string {
	d, err := home.Sessions()
	if err != nil {
		return "chainbench-out"
	}
	return d
}

// runAttached runs the specs against a network that is already up.
//
// One function for all three ways in, because they only ever differed in where
// the network came from and that is now the start state. What they shared —
// reading the specs, opening the event stream, printing the session — is all a
// surface owes, and having it written three times is how the declared path came
// to expand ${VAR} and the other two did not.
func runAttached(cmd *cobra.Command, args []string, start app.Start,
	workspaceDir, chain string, rpcURLs []string,
	artifactRoot, keysDir, dashboardURL string, jsonOut, noSkips bool) error {
	specs, err := app.ReadSpecFiles(args)
	if err != nil {
		return err
	}
	bus, flush := dashboard.Stream(dashboardURL)
	defer flush()
	root, err := app.AttachRun(cmd.Context(), surface.Deps(cmd), app.AttachRunIn{
		At: start.At, Declared: start.Declared,
		Chain: chain, RPCURLs: rpcURLs, DataDir: workspaceDir,
		ArtifactRoot: artifactRoot, KeysDir: keysDir, Specs: specs, Bus: bus,
	})
	if err != nil {
		return err
	}
	return printSession(cmd.OutOrStdout(), root, jsonOut, noSkips)
}

// expandEach resolves ${VAR} and ${VAR:-default} in each endpoint, the same
// expansion a declared binary path gets. An endpoint is a fact about a machine,
// and a committed case must not have to carry one.
func expandEach(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = os.Expand(s, envOrDefault)
	}
	return out
}

// envOrDefault expands NAME or NAME:-fallback from the process environment.
func envOrDefault(spec string) string {
	name, fallback, hasDefault := strings.Cut(spec, ":-")
	if v, ok := os.LookupEnv(name); ok && v != "" {
		return v
	}
	if hasDefault {
		return fallback
	}
	return ""
}

// runComposed composes the network the specs declare and runs them against
// it, printing the setup steps before the session.
func runComposed(cmd *cobra.Command, in app.RunSuiteIn, jsonOut, noSkips bool) error {
	// Under --json the whole of stdout is the document; the setup narration is
	// progress, so it goes to stderr. Without it, both share stdout as before.
	notes := progressWriter(cmd, jsonOut)
	res, err := app.RunSuite(cmd.Context(), surface.Deps(cmd), in)
	for _, step := range res.SetupSteps {
		fmt.Fprintln(notes, step)
	}
	if res.Preflight != "" {
		fmt.Fprintf(notes, "preflight: %s\n", res.Preflight)
	}
	if err != nil {
		return setupFailure(cmd.OutOrStdout(), err, failedAtOf(res), jsonOut)
	}
	return printSession(cmd.OutOrStdout(), res.SessionRoot, jsonOut, noSkips)
}

// setupFailure reports a run that never got as far as a session.
//
// It used to return the error and nothing else, which made one definition
// answer differently from several for the very same failure: a broken binary
// exited 1 with an empty stdout on its own, and 2 with a full document when a
// second definition was named alongside it. Nothing about the failure differs,
// so nothing about the report should.
//
// The code is 2 either way, --json or not, matching what sequenceExit already
// returns for a definition that could not run: "the chain would not come up" is not the
// same news as "a test ran and failed", and CI gates on the difference.
func setupFailure(out io.Writer, cause error, at string, jsonOut bool) error {
	if jsonOut {
		// The document goes out even though the run failed, because under
		// --json stdout is the whole answer: a consumer that gets nothing
		// cannot tell a compose failure from a crash.
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(runReport{Error: cause.Error(), FailedAt: at}); err != nil {
			return err
		}
	}
	// The state is appended rather than woven in: the message is a sentence
	// somebody wrote to be read, and the state is for whoever asks "which stage
	// was that?" without reading it.
	if at != "" {
		cause = fmt.Errorf("%w (at %s)", cause, at)
	}
	return &exitcode.Error{Code: 2, Err: cause}
}

// failedAtOf names the states a run failed in, outermost first, or "" when it
// failed in none this build knows.
//
// There are two when the chain refused: the run's own stage, and which of the
// chain's stages did the refusing. They are one suffix because they used to be
// two — "(at ChainLaunchNodesFailPortBusy) (at TestReachNetworkFailCompose)" —
// and two suffixes that look alike read as one thing said twice rather than as
// a stage and the stage inside it.
//
// The surface takes words rather than the state's own type, because naming that
// type here would be reaching past app for it.
func failedAtOf(res app.RunSuiteOut) string {
	if res.FailedAt == 0 {
		return ""
	}
	at := res.FailedAt.String()
	if res.ComposeFailedAt != 0 {
		at += " / " + res.ComposeFailedAt.String()
	}
	return at
}

// progressWriter is where narration goes: stderr when stdout has to parse as a
// document, stdout otherwise so a person reads one stream in order.
func progressWriter(cmd *cobra.Command, jsonOut bool) io.Writer {
	if jsonOut {
		return cmd.ErrOrStderr()
	}
	return cmd.OutOrStdout()
}

// runComposedSequence runs several test definitions in order, each through the
// same path a single one takes, and prints each definition's setup, preflight
// and session under its own heading. It ends with one line per definition so a
// long run's outcome is readable without scrolling back.
func runComposedSequence(cmd *cobra.Command, in app.RunSuiteIn, jsonOut, noSkips bool) error {
	notes := progressWriter(cmd, jsonOut)
	res, err := app.RunSuites(cmd.Context(), surface.Deps(cmd), in)
	if err != nil {
		return err
	}

	report := sequenceReport{Runs: make([]sequenceRunReport, 0, len(res.Runs))}
	for i, r := range res.Runs {
		fmt.Fprintf(notes, "\n=== [%d/%d] %s ===\n", i+1, len(res.Runs), r.Spec)
		for _, step := range r.Out.SetupSteps {
			fmt.Fprintln(notes, step)
		}
		if r.Out.Preflight != "" {
			fmt.Fprintf(notes, "preflight: %s\n", r.Out.Preflight)
		}
		if r.Err != "" {
			fmt.Fprintf(notes, "error: %s\n", r.Err)
		}
		if jsonOut {
			report.Runs = append(report.Runs, sequenceRunReport{
				Spec: r.Spec, Session: r.Out.SessionRoot, Error: r.Err, RunSummary: r.Out.Summary,
			})
			continue
		}
		if r.Err == "" {
			// A failed definition is reported in the tally below, so its own
			// verdict must not stop the remaining ones from printing.
			_ = printSession(cmd.OutOrStdout(), r.Out.SessionRoot, false, false)
		}
	}

	if jsonOut {
		// One document for the whole command: a reader parses stdout once.
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "\n=== %d definition(s) ===\n", len(res.Runs))
		for i, r := range res.Runs {
			s := r.Out.Summary.Summary
			status := fmt.Sprintf("pass=%d fail=%d blocked=%d skip=%d", s.Pass, s.Fail, s.Blocked, s.Skip)
			if r.Err != "" {
				status = "error: " + r.Err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d. %s — %s\n", i+1, r.Spec, status)
		}
	}
	return sequenceExit(res, noSkips)
}

// sequenceRunReport is one definition's entry in the --json document. It carries
// the same summary a single run emits, so a reader handles both the same way.
type sequenceRunReport struct {
	Spec    string `json:"spec"`
	Session string `json:"session,omitempty"`
	Error   string `json:"error,omitempty"`
	app.RunSummary
}

// sequenceReport is the whole command's --json document.
type sequenceReport struct {
	Runs []sequenceRunReport `json:"runs"`
}

// sequenceExit maps the run to the exit code the CLI promises CI: 0 all pass,
// 1 a test failed, 2 blocked or an infrastructure error. A definition that could
// not run at all is infrastructure, which outranks a plain test failure — losing
// that distinction is what a single generic error did.
func sequenceExit(res app.RunSuitesOut, noSkips bool) error {
	setupErrors, failed, blocked := res.Totals()
	if skipped := res.Skipped(); noSkips && skipped > 0 {
		return &exitcode.Error{Code: 2, Err: fmt.Errorf(
			"run: %d test(s) skipped and --no-skips was given — the run answered nothing about them", skipped)}
	}
	switch {
	case setupErrors > 0 || blocked > 0:
		return &exitcode.Error{Code: 2, Err: fmt.Errorf(
			"run: %d definition(s) could not run, %d test(s) blocked, %d failed", setupErrors, blocked, failed)}
	case failed > 0:
		return &exitcode.Error{Code: 1, Err: fmt.Errorf("run: %d test(s) failed", failed)}
	default:
		return nil
	}
}

// printSession reads the saved session and prints a table plus a summary,
// returning a non-nil error when any test failed or was blocked.
func printSession(out io.Writer, root string, jsonOut, noSkips bool) error {
	doc, err := app.SessionSummary(root)
	if err != nil {
		return err
	}

	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(runReport{Session: root, RunSummary: doc}); err != nil {
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
	// A skip is not a failure, and with --no-skips it is: the caller said the
	// point of this run was that the cases could run at all. It maps to 2 for
	// the same reason blocked does — the run answered nothing, which is a
	// different thing from answering "no".
	if noSkips && doc.Summary.Skip > 0 {
		return &exitcode.Error{Code: 2, Err: fmt.Errorf(
			"run: %d test(s) skipped and --no-skips was given — the run answered nothing about them", doc.Summary.Skip)}
	}
	if doc.Failed() {
		// Blocked/infrastructure errors are more severe than a plain test
		// failure, so they map to exit code 2 (F16-O5).
		code := 1
		if doc.Summary.Blocked > 0 {
			code = 2
		}
		return &exitcode.Error{Code: code, Err: fmt.Errorf("run: %d failed, %d blocked", doc.Summary.Fail, doc.Summary.Blocked)}
	}
	return nil
}

// planPrinter returns a sink that writes each compose plan to w, skipping one
// that repeats the last.
//
// A run of many definitions composes once and reuses the network, so printing
// every definition's plan would bury the one thing worth seeing: the moment the
// network changes between definitions. Repetition is silence; a difference is a
// new block.
func planPrinter(w io.Writer) func(app.ComposePlan) {
	var last string
	return func(p app.ComposePlan) {
		s := p.String()
		if s == last {
			return
		}
		last = s
		fmt.Fprintf(w, "composing:\n%s", s)
	}
}

// showPlan resolves what the run would compose and prints it without composing.
func showPlan(cmd *cobra.Command, in app.RunSuiteIn, jsonOut bool) error {
	plans, err := app.PlanSuites(cmd.Context(), in)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(plans)
	}
	for _, p := range plans {
		fmt.Fprintf(out, "%s\n%s\n", p.Spec, p.Plan.String())
	}
	return nil
}
