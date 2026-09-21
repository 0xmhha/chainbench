package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/report"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// The workflow is what MCP exists to reach (architecture-v2 §2): one call
// that takes DSL inputs, composes the chain the specs declare, runs the specs
// through the test engine, collects the session, and reports. Since R4 the
// engine owns that whole flow (testengine.RunSuite); this layer only adapts
// the shared Deps and keeps the MCP-facing names stable.

// RunSuiteIn is one whole workflow request.
type RunSuiteIn = testengine.RunSuiteIn

// RunSuiteOut is the workflow's report.
type RunSuiteOut = testengine.RunSuiteOut

// RunSuite delegates the whole flow to the test engine, which composes the
// declared chain through chainsetup and runs the specs against it.
func RunSuite(ctx context.Context, d Deps, in RunSuiteIn) (RunSuiteOut, error) {
	return testengine.RunSuite(ctx, d.chainsetupDeps(), in)
}

// ComposePlan is the network a run is about to compose, after the declaration
// and the request's overrides are merged.
type ComposePlan = testengine.ComposePlan

// RunSummary is a collected session result.
type RunSummary = testengine.Summary

// AttachRunIn runs specs against an already-running network.
type AttachRunIn struct {
	Chain        string
	RPCURLs      []string
	ArtifactRoot string
	Caps         []string
	// KeysDir is the key set the running network was composed from, when the
	// operator knows it. It is what lets a spec name an account by label while
	// attached: the addresses are the key set's, and without it the run has no
	// way to turn "node1" into one.
	KeysDir string
	// Specs are raw DSL JSON blobs (already env-resolved).
	Specs [][]byte
	// Nodes is the network as its composer recorded it, with the roles a spec
	// addresses by. Set from DataDir; a caller who names endpoints has no such
	// record and gets the roleless table attach has always used.
	Nodes NodeSet
	// Bus receives orchestration events; nil disables emission.
	//
	// A bus rather than a dashboard URL because the dashboard is itself a
	// surface (L6) and this layer sits below it. Opening the stream is
	// dashboard.Stream's job, which both surfaces call.
	Bus *collector.Bus
	// Declared is what the specs' env.attach said, carried by a Start whose
	// state came from one. It supplies what the caller did not: the endpoints,
	// the key set, the capabilities and the chain.
	//
	// The rule is that what the caller said outranks the document, and here
	// that is expressed by "empty means not said" — a surface passes a flag's
	// value only when the flag was given.
	Declared *AttachDecl
	// At is the adopt state this run starts in, from StartFor. It decides which
	// of the three attach paths runs.
	//
	// Required. It used to be worked out here from whether DataDir was set,
	// which is the same decision the two surfaces were each making again with
	// their own words; a caller that has not decided is not a caller this can
	// decide for.
	At lifecycle.Status
	// DataDir attaches to the network a workspace already composed, instead of
	// naming its endpoints. The workspace supplies both: the endpoints, and
	// the capabilities the composition advertised.
	//
	// The capabilities are why this exists. A spec may declare that it needs
	// one — the proposal-expiry regression needs "short-expiry", which a
	// genesis overlay grants — and a caller who only passes endpoints cannot
	// say what the network has, so a gated spec always skips. Composing again
	// to satisfy the gate would test a different network from the one the
	// operator set up.
	DataDir string
}

// AttachRun attaches the test engine to a running network and runs the specs,
// returning the session root.
func AttachRun(ctx context.Context, d Deps, in AttachRunIn) (string, error) {
	// Pre-flight the specs the way RunSuite does before composing (WA10): an
	// unresolved action/assertion/reader/reference or a malformed selector fails
	// here with a clear reason. The attach path skipped this, so such a mistake
	// reached the interpreter and vanished into an AssertResult that has no field
	// to carry it. A spec that does not parse is left for the engine to record
	// per-spec.
	parsed := make([]dsl.Spec, 0, len(in.Specs))
	for _, raw := range in.Specs {
		if s, perr := dsl.Parse(raw); perr == nil {
			parsed = append(parsed, s)
		}
	}
	if err := testengine.Precheck(parsed); err != nil {
		return "", fmt.Errorf("app: attach run: %w", err)
	}
	// Attaching to a workspace whose network is up goes through the engine's
	// workspace-attach path, which wires the readiness gate (E6), failure-
	// evidence collection (E8), fault control, and the composition manifest just
	// as the compose path does — the plain NewAttachEngine below wires none of
	// those, so a bare-URL attach (which owns no workspace or processes) is the
	// only thing that should use it (WA10).
	if in.At.Block() != lifecycle.AdoptChain {
		return "", fmt.Errorf("app: attach run: %s is not a state a run attaches in — resolve one with StartFor", in.At)
	}
	if in.At == lifecycle.AdoptChainByDeclaration {
		if in.Declared == nil {
			return "", fmt.Errorf("app: attach run: %s carries no declaration", in.At)
		}
		if in.Chain == "" {
			in.Chain = in.Declared.Chain
		}
		if in.KeysDir == "" {
			in.KeysDir = in.Declared.KeysDir
		}
		if len(in.RPCURLs) == 0 {
			in.RPCURLs = in.Declared.RPCURLs
		}
		if len(in.Caps) == 0 {
			in.Caps = in.Declared.Provides
		}
	}
	if in.At == lifecycle.AdoptChainByWorkspace {
		if in.DataDir == "" {
			return "", fmt.Errorf("app: attach run: %s needs the workspace whose network is up", in.At)
		}
		return testengine.AttachWorkspaceRun(ctx, d.chainsetupDeps(), testengine.AttachWorkspaceIn{
			DataDir: in.DataDir, Chain: in.Chain, ArtifactRoot: in.ArtifactRoot,
			Caps: in.Caps, Specs: in.Specs,
		})
	}
	if in.Chain == "" {
		return "", fmt.Errorf("app: attach run: a chain is required to attach")
	}
	if len(in.RPCURLs) == 0 {
		return "", fmt.Errorf("app: attach run: no endpoint to attach to")
	}
	eng, err := testengine.NewAttachEngine(testengine.AttachConfig{
		Chain: in.Chain, RPCURLs: in.RPCURLs,
		ArtifactRoot: in.ArtifactRoot, Caps: in.Caps, Clock: d.Clock,
		KeysDir: in.KeysDir, Bus: in.Bus, Nodes: in.Nodes,
	})
	if err != nil {
		return "", fmt.Errorf("app: attach run: %w", err)
	}
	return eng.Run(ctx, in.Specs)
}

// DeclaredAttach is the attach declaration the given specs share, or nil when
// none of them declares one.
//
// One run runs against one network, so the specs have to agree about it — the
// same rule sameComposition keeps for a composed network, for the same reason.
// A run that took the first spec's endpoints and answered the rest from them
// would report on a network those cases never named.
//
// Specs are read the way every surface reads them, so an env reference is
// resolved here too and a case that names an attaching env behaves like one
// that writes it inline.
func DeclaredAttach(paths []string) (*AttachDecl, error) {
	specs, err := dsl.ReadFiles(paths)
	if err != nil {
		return nil, err
	}
	return declaredAttachIn(specs, paths)
}

// declaredAttachIn is DeclaredAttach over specs already read.
//
// The surfaces do not all hold paths: the CLI names files and the tool is
// handed the blobs. Splitting the reading from the agreeing is what lets both
// ask the same question, which is why the tool had no env.attach branch at all
// until now.
func declaredAttachIn(specs [][]byte, labels []string) (*AttachDecl, error) {
	var (
		want *AttachDecl
		from string
	)
	for i, raw := range specs {
		sp, perr := dsl.Parse(raw)
		if perr != nil {
			// Not this function's verdict: the engine reports a spec that does
			// not parse, per spec, with the reason.
			continue
		}
		label := labelAt(labels, i)
		if sp.EnvAttach == nil {
			if want != nil {
				return nil, fmt.Errorf("app: %s declares an attach env and %s does not — one run runs against one network", from, label)
			}
			continue
		}
		got := &AttachDecl{
			Chain:    sp.Chain.Name,
			RPCURLs:  sp.EnvAttach.RPC,
			KeysDir:  sp.EnvAttach.KeysDir,
			Provides: sp.EnvAttach.Provides,
		}
		if want == nil {
			if i > 0 {
				return nil, fmt.Errorf("app: %s declares an attach env and %s does not — one run runs against one network", label, labelAt(labels, 0))
			}
			want, from = got, label
			continue
		}
		if !slices.Equal(want.RPCURLs, got.RPCURLs) || want.Chain != got.Chain {
			return nil, fmt.Errorf("app: %s and %s attach to different networks — split the run", from, label)
		}
	}
	return want, nil
}

// labelAt names a spec in a refusal. A caller that handed over blobs with no
// names gets a position, which is still enough to say which two disagree.
func labelAt(labels []string, i int) string {
	if i < len(labels) {
		return labels[i]
	}
	return fmt.Sprintf("spec %d", i+1)
}

// AttachDecl is the network a case's own env names, in the terms a surface
// needs: where it is, whose keys it was built from, and what it offers.
//
// It is app's type and not the DSL's on purpose. A surface reaches a feature
// through app, so handing it a dsl type would make every caller of this
// function name the module directly — which the architecture guard reports, and
// which is how a surface comes to depend on a parse shape it has no business
// knowing.
type AttachDecl struct {
	// Chain is the chain id the env declares. Required: a spec that names a
	// contract needs the chain's table before the first call.
	Chain string
	// RPCURLs are the endpoints, still as written — "${VAR:-default}" is
	// expanded by the surface, which owns the process environment.
	RPCURLs []string
	// KeysDir is the key set the running network was composed from, empty when
	// the declaration does not say.
	KeysDir string
	// Provides is what the running network offers, for capability gating.
	Provides []string
}

// SpecInfo is one test case as the catalog lists it.
type SpecInfo = dsl.SpecInfo

// ListSpecs enumerates the runnable test cases under dir, so an operator or an
// agent can discover what is there before running one. It is the catalog behind
// the CLI `test list` and the MCP test_list tool.
func ListSpecs(dir string) ([]SpecInfo, error) { return dsl.ListSpecs(dir) }

// ReadSpecFiles reads DSL spec files, resolving each against its environment.
//
// Every surface that runs specs reads them, and reading is where a relative
// path or an env reference is settled; two surfaces settling it separately is
// two answers to where a spec's environment lives.
func ReadSpecFiles(paths []string) ([][]byte, error) { return dsl.ReadFiles(paths) }

// SessionSummary reads a session's collected summary.
func SessionSummary(root string) (RunSummary, error) {
	return testengine.ReadSessionSummary(root)
}

// ValidateResult is one spec's verdict: whether it parses, and what is wrong
// with it if not.
type ValidateResult = testengine.ValidateResult

// Validate runs the shared offline DSL validation for the MCP surface: the same
// parse, name-resolution, selector, and capability checks the CLI `validate`
// runs, so both surfaces reach the same verdict. It writes and composes nothing.
func Validate(paths []string, chain string) ([]testengine.ValidateResult, error) {
	return testengine.ValidateSpecs(paths, chain)
}

// ValidateContent validates spec bytes (the form MCP passes: a spec is a JSON
// string, not a file on the host), reaching the same verdict as Validate.
func ValidateContent(raws [][]byte, labels []string, chain string) ([]testengine.ValidateResult, error) {
	return testengine.ValidateContent(raws, labels, chain)
}

// ReportDoc is a run's report as a surface receives it: the verdict tally and
// one entry per test.
type ReportDoc = report.Report

// ReportIn names the session to read.
type ReportIn struct {
	Dir string `cb:"workspace-dir,required" help:"session directory, or a root holding sessions"`
	// All combines every session under Dir instead of reading the most recent.
	// A run per spec file is the normal way to use this harness, and "the most
	// recent session" is the wrong answer to "did the batch pass".
	All bool `cb:"all" help:"combine every session under the directory into one tally, instead of reading the most recent"`
}

// Report reads a run's report from a session directory, or from a root holding
// several sessions, in which case the most recent is read.
//
// It answers with the report rather than with prose, because the two surfaces
// lay it out differently — a table for a person, JSON for a program — and a
// layer that renders is a layer each surface has to work around. Reading it is
// what they share: prefer the persisted report.json, and fall back to building
// it from session.json so a run recorded before report.json existed still
// shows.
func Report(_ context.Context, _ Deps, in ReportIn) (ReportDoc, error) {
	dir := in.Dir
	ids, _ := session.List(dir)
	if in.All {
		return combinedReport(dir, ids)
	}
	sessionDir := dir
	if len(ids) > 0 {
		sessionDir = session.SessionDir(dir, ids[len(ids)-1])
	}
	rep, err := readReport(sessionDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ReportDoc{}, nil
		}
		return ReportDoc{}, err
	}
	return rep, nil
}

// CombinedSession is relayed for the surfaces, which lay a combined report out
// differently (a session column per row, a run count instead of a session id) and
// so must be able to tell one. They may not import core (architecture-v2 §2,
// enforced by arch.TestSurfacesReachThroughApp), and without a relay each surface
// would carry its own copy of the literal — the same shape the argument decoders
// were in before app/args.go.
const CombinedSession = report.CombinedSession

// ReportSessions reports how many distinct runs a report covers. A combined
// report's session id names no directory, so this is what a reader gets instead.
func ReportSessions(rep ReportDoc) int {
	seen := map[string]bool{}
	for _, t := range rep.Tests {
		seen[t.Session] = true
	}
	return len(seen)
}

// combinedReport reads every session under dir and merges them.
//
// A session that cannot be read is skipped rather than failing the whole answer:
// a batch is usually read while something is still writing, and refusing to
// report on nine finished runs because a tenth is mid-write would make the
// combined view useless exactly when it is wanted.
func combinedReport(dir string, ids []string) (ReportDoc, error) {
	if len(ids) == 0 {
		// Dir may itself be one session rather than a root of them.
		rep, err := readReport(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return ReportDoc{}, nil
			}
			return ReportDoc{}, err
		}
		return rep, nil
	}
	reps := make([]report.Report, 0, len(ids))
	for _, id := range ids {
		rep, err := readReport(session.SessionDir(dir, id))
		if err != nil {
			continue
		}
		reps = append(reps, rep)
	}
	if len(reps) == 0 {
		return ReportDoc{}, nil
	}
	return report.Combine(reps), nil
}

// readReport prefers the persisted report.json and falls back to building one
// from session.json, so a run recorded before report.json existed still shows.
func readReport(sessionDir string) (report.Report, error) {
	rep, err := report.Read(sessionDir)
	if err == nil {
		return rep, nil
	}
	return report.Build(sessionDir)
}
