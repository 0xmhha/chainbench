package engine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/engine"
	"github.com/0xmhha/chainbench/internal/testspec"
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
	applicable                          func(testspec.Spec) bool
}

func (h *harness) deps(t *testing.T) engine.Deps {
	return engine.Deps{
		Command: "test",
		NewSession: func(_ context.Context, cmd string) (session.Session, error) {
			return session.New(t.TempDir(), cmd, time.Unix(0, 0).UTC(), nil)
		},
		Fingerprint: func(s testspec.Spec) session.Fingerprint {
			return h.fpByChain[s.Chain.Name]
		},
		Applicable: h.applicable,
		BuildEnv: func(_ context.Context, _ session.Environment, _ testspec.Spec) (node.NodeSet, engine.TeardownFunc, error) {
			h.buildCount++
			ns := node.NodeSet{Nodes: []node.Node{{Index: 1, Role: node.RoleValidator}}}
			return ns, func(context.Context) error { h.teardownCount++; return nil }, nil
		},
		RunSpec: func(_ context.Context, _ testspec.Spec, _ session.Environment, rec session.TestRecord) (session.TestStatus, error) {
			h.runCount++
			rec.Status(session.StatusPass)
			return session.StatusPass, nil
		},
	}
}

func TestEngine_ReusesEnvByFingerprint(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	e := engine.New(h.deps(t))

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

func TestEngine_DifferentFingerprintsBuildTwice(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{
		"wbft":      "aaaaaaaaaaaa0000",
		"stablenet": "bbbbbbbbbbbb1111",
	}}
	e := engine.New(h.deps(t))
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
		applicable: func(s testspec.Spec) bool { return s.Chain.Name == "wbft" },
	}
	e := engine.New(h.deps(t))
	if _, err := e.Run(context.Background(), [][]byte{specJSON("T1", "stablenet")}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if h.buildCount != 0 || h.runCount != 0 {
		t.Fatalf("inapplicable spec must not build/run (build=%d run=%d)", h.buildCount, h.runCount)
	}
}

func TestEngine_MalformedSpecBlocked(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{}}
	e := engine.New(h.deps(t))
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
