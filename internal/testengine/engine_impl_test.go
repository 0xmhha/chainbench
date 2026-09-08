package testengine_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/report"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"

	"github.com/0xmhha/chainbench/internal/testengine"
)

func specJSON(id, chain string) []byte {
	b, _ := json.Marshal(map[string]any{
		"schemaVersion": "1",
		"id":            id,
		"chain":         map[string]any{"name": chain, "binary": "go-" + chain},
		"assertions":    []any{map[string]any{"assert": "True"}},
	})
	return b
}

// harness wires the engine to a real session with controllable fakes.
type harness struct {
	buildCount, runCount, teardownCount int
	fpByChain                           map[string]session.Fingerprint
	applicable                          func(dsl.Spec) bool
}

func (h *harness) deps(t *testing.T) testengine.Deps {
	return testengine.Deps{
		Command: "test",
		NewSession: func(_ context.Context, cmd string) (session.Session, error) {
			return session.New(t.TempDir(), cmd, time.Unix(0, 0).UTC())
		},
		Fingerprint: func(s dsl.Spec) session.Fingerprint {
			return h.fpByChain[s.Chain.Name]
		},
		Applicable: h.applicable,
		BuildEnv: func(_ context.Context, _ session.Environment, _ dsl.Spec) (node.NodeSet, testengine.TeardownFunc, error) {
			h.buildCount++
			ns := node.NodeSet{Nodes: []node.Node{{Index: 1, Role: node.RoleValidator}}}
			return ns, func(context.Context) error { h.teardownCount++; return nil }, nil
		},
		RunSpec: func(_ context.Context, _ dsl.Spec, _ session.Environment, rec session.TestRecord) (session.TestStatus, error) {
			h.runCount++
			rec.Status(session.StatusPass)
			return session.StatusPass, nil
		},
	}
}

func TestEngine_ReusesEnvByFingerprint(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	e := testengine.New(h.deps(t))

	root, err := e.Run(context.Background(), [][]byte{specJSON("T1", "wbft"), specJSON("T2", "wbft")})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if h.buildCount != 1 {
		t.Fatalf("buildCount = %d, want 1 (env reused)", h.buildCount)
	}
	if h.runCount != 2 {
		t.Fatalf("runCount = %d, want 2", h.runCount)
	}
	if h.teardownCount != 1 {
		t.Fatalf("teardownCount = %d, want 1", h.teardownCount)
	}
	if _, err := os.Stat(filepath.Join(root, "session.json")); err != nil {
		t.Fatalf("session.json missing: %v", err)
	}
}

func TestEngine_PreSpecGateBlocksTest(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	deps := h.deps(t)
	deps.PreSpec = func(context.Context, session.Environment) error {
		return errors.New("network not ready to test: node1 FATAL: chain diverged")
	}
	e := testengine.New(deps)
	if _, err := e.Run(context.Background(), [][]byte{specJSON("T1", "wbft")}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The gate ran before the test: the network was built, but the test was
	// blocked rather than run against an unfit network.
	if h.buildCount != 1 {
		t.Errorf("buildCount = %d, want 1", h.buildCount)
	}
	if h.runCount != 0 {
		t.Errorf("runCount = %d, want 0 (gate blocked the test)", h.runCount)
	}
}

func TestEngine_OnFailGathersOnFailedTest(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	deps := h.deps(t)
	deps.RunSpec = func(_ context.Context, _ dsl.Spec, _ session.Environment, rec session.TestRecord) (session.TestStatus, error) {
		rec.Status(session.StatusFail)
		return session.StatusFail, nil
	}
	fails := 0
	deps.OnFail = func(context.Context, session.Environment, session.TestRecord) error { fails++; return nil }
	e := testengine.New(deps)
	if _, err := e.Run(context.Background(), [][]byte{specJSON("T1", "wbft")}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fails != 1 {
		t.Errorf("OnFail called %d times on a failing test, want 1", fails)
	}
}

func TestEngine_OnFailSkippedOnPass(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	deps := h.deps(t) // default RunSpec passes
	fails := 0
	deps.OnFail = func(context.Context, session.Environment, session.TestRecord) error { fails++; return nil }
	e := testengine.New(deps)
	if _, err := e.Run(context.Background(), [][]byte{specJSON("T1", "wbft")}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fails != 0 {
		t.Errorf("OnFail called %d times on a passing test, want 0", fails)
	}
}

func TestEngine_PreSpecGatePassLetsTestRun(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	deps := h.deps(t)
	called := 0
	deps.PreSpec = func(context.Context, session.Environment) error { called++; return nil }
	e := testengine.New(deps)
	if _, err := e.Run(context.Background(), [][]byte{specJSON("T1", "wbft")}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if called != 1 || h.runCount != 1 {
		t.Errorf("gate called %d, runCount %d, want 1/1", called, h.runCount)
	}
}

func TestEngine_DifferentFingerprintsBuildTwice(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{
		"wbft":      "aaaaaaaaaaaa0000",
		"stablenet": "bbbbbbbbbbbb1111",
	}}
	e := testengine.New(h.deps(t))
	if _, err := e.Run(context.Background(), [][]byte{specJSON("T1", "wbft"), specJSON("T2", "stablenet")}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if h.buildCount != 2 || h.teardownCount != 2 {
		t.Fatalf("build=%d teardown=%d, want 2/2", h.buildCount, h.teardownCount)
	}
}

func TestEngine_SkipsInapplicable(t *testing.T) {
	h := &harness{
		fpByChain:  map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"},
		applicable: func(s dsl.Spec) bool { return s.Chain.Name == "wbft" },
	}
	e := testengine.New(h.deps(t))
	if _, err := e.Run(context.Background(), [][]byte{specJSON("T1", "stablenet")}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if h.buildCount != 0 || h.runCount != 0 {
		t.Fatalf("inapplicable spec must not build/run (build=%d run=%d)", h.buildCount, h.runCount)
	}
}

func TestEngine_MalformedSpecBlocked(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{}}
	e := testengine.New(h.deps(t))
	root, err := e.Run(context.Background(), [][]byte{[]byte("{bad json")})
	if err != nil {
		t.Fatalf("Run should not fail on a malformed spec: %v", err)
	}
	if h.buildCount != 0 || h.runCount != 0 {
		t.Fatal("malformed spec must not build/run")
	}
	if _, err := os.Stat(filepath.Join(root, "session.json")); err != nil {
		t.Fatalf("session.json missing: %v", err)
	}
}

// TestEngine_RecordsArtifactsManifest pins WA11: the composition manifest the
// run was given is written into each test's artifacts.json, so a verdict is
// traceable to the genesis it ran against rather than the field being empty.
func TestEngine_RecordsArtifactsManifest(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	deps := h.deps(t)
	deps.Artifacts = []session.ArtifactRef{{Kind: "genesis", Ref: "genesis.json"}}
	e := testengine.New(deps)

	root, err := e.Run(context.Background(), [][]byte{specJSON("T1", "wbft")})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	var found string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && filepath.Base(p) == "artifacts.json" {
			found = p
		}
		return nil
	})
	if found == "" {
		t.Fatal("no artifacts.json written for the test (WA11)")
	}
	b, err := os.ReadFile(found)
	if err != nil {
		t.Fatalf("read artifacts.json: %v", err)
	}
	var got session.TestArtifacts
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal artifacts.json: %v", err)
	}
	if len(got.Refs) != 1 || got.Refs[0].Kind != "genesis" || got.Refs[0].Ref != "genesis.json" {
		t.Fatalf("artifacts manifest = %+v, want a single genesis ref", got.Refs)
	}
}

// TestEngine_GeneratesReport pins WA23: a full engine run (compose -> run ->
// record) generates report.json without a live binary, so the
// compose->run->report pipeline has non-live CI coverage rather than only the
// GSTABLE_BIN-gated live tests.
func TestEngine_GeneratesReport(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	e := testengine.New(h.deps(t))

	root, err := e.Run(context.Background(), [][]byte{specJSON("T1", "wbft"), specJSON("T2", "wbft")})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	rep, err := report.Read(root)
	if err != nil {
		t.Fatalf("report.Read: %v (a run must generate report.json)", err)
	}
	if len(rep.Tests) != 2 || rep.Summary.Pass != 2 {
		t.Fatalf("report = %d tests, pass=%d; want 2 tests, 2 pass", len(rep.Tests), rep.Summary.Pass)
	}
}

// TestAttachWorkspaceRun_RefusesNoWorkspace pins WA10's attach entry: it fails
// cleanly (no panic) when given no workspace or one that composed nothing,
// rather than attaching to nothing. The gate/evidence wiring itself is the same
// wiredAttachEngine the compose path uses, covered by the engine PreSpec/OnFail
// tests above.
func TestAttachWorkspaceRun_RefusesNoWorkspace(t *testing.T) {
	sd := chainsetup.Deps{Clock: func() time.Time { return time.Unix(0, 0).UTC() }}
	if _, err := testengine.AttachWorkspaceRun(context.Background(), sd, testengine.AttachWorkspaceIn{}); err == nil {
		t.Fatal("attach with no workspace must fail")
	}
	if _, err := testengine.AttachWorkspaceRun(context.Background(), sd, testengine.AttachWorkspaceIn{
		DataDir: t.TempDir(), Chain: "wbft",
	}); err == nil {
		t.Fatal("attach to a workspace that composed nothing must fail, not attach to nothing")
	}
}
