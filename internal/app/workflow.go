package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/report"
	"github.com/0xmhha/chainbench/internal/core/session"
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

// Report renders a run's report as text for the MCP surface. dir is a session
// directory or a root holding several sessions (then the most recent is used).
// MCP reaches the report through here rather than importing the report/session
// core packages directly (architecture-v2 §2: MCP goes through app).
func Report(dir string) (string, error) {
	sessionDir := dir
	if ids, _ := session.List(dir); len(ids) > 0 {
		sessionDir = session.SessionDir(dir, ids[len(ids)-1])
	}
	rep, err := report.Read(sessionDir)
	if err != nil {
		rep, err = report.Build(sessionDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "no runs recorded", nil
			}
			return "", err
		}
	}
	if len(rep.Tests) == 0 {
		return "no runs recorded", nil
	}
	var b strings.Builder
	for _, t := range rep.Tests {
		fmt.Fprintf(&b, "%d %s [%s] %s\n", t.Seq, t.ID, t.Env, t.Status)
	}
	fmt.Fprintf(&b, "session=%s pass=%d fail=%d blocked=%d skip=%d",
		rep.Session, rep.Summary.Pass, rep.Summary.Fail, rep.Summary.Blocked, rep.Summary.Skip)
	return b.String(), nil
}
