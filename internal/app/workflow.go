package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/collector"
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
	if in.DataDir != "" {
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

// Validate runs the shared offline DSL validation for the MCP surface: the same
// parse, name-resolution, selector, and capability checks the CLI `validate`
// runs, so both surfaces reach the same verdict. It writes and composes nothing.
// ValidateResult is one spec's verdict: whether it parses, and what is wrong
// with it if not.
type ValidateResult = testengine.ValidateResult

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

// Report reads a run's report from a session directory, or from a root holding
// several sessions, in which case the most recent is read.
//
// It answers with the report rather than with prose, because the two surfaces
// lay it out differently — a table for a person, JSON for a program — and a
// layer that renders is a layer each surface has to work around. Reading it is
// what they share: prefer the persisted report.json, and fall back to building
// it from session.json so a run recorded before report.json existed still
// shows.
// ReportIn names the session to read.
type ReportIn struct {
	Dir string `cb:"workspace-dir,required" help:"session directory, or a root holding sessions"`
}

func Report(_ context.Context, _ Deps, in ReportIn) (ReportDoc, error) {
	dir := in.Dir
	sessionDir := dir
	if ids, _ := session.List(dir); len(ids) > 0 {
		sessionDir = session.SessionDir(dir, ids[len(ids)-1])
	}
	rep, err := report.Read(sessionDir)
	if err != nil {
		rep, err = report.Build(sessionDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return ReportDoc{}, nil
			}
			return ReportDoc{}, err
		}
	}
	return rep, nil
}
