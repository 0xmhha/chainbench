package app

import (
	"context"
	"fmt"

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
	// Specs are raw DSL JSON blobs (already env-resolved).
	Specs [][]byte
}

// AttachRun attaches the test engine to a running network and runs the specs,
// returning the session root.
func AttachRun(ctx context.Context, d Deps, in AttachRunIn) (string, error) {
	eng, err := testengine.NewAttachEngine(testengine.AttachConfig{
		Chain: in.Chain, RPCURLs: in.RPCURLs,
		ArtifactRoot: in.ArtifactRoot, Caps: in.Caps, Clock: d.Clock,
	})
	if err != nil {
		return "", fmt.Errorf("app: attach run: %w", err)
	}
	return eng.Run(ctx, in.Specs)
}

// SessionSummary reads a session's collected summary.
func SessionSummary(root string) (RunSummary, error) {
	return testengine.ReadSessionSummary(root)
}
