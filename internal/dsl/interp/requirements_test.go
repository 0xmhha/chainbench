package interp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// table is a node table and nothing else: the four methods the interpreter
// actually needs, with no directory, no fingerprint and no disk.
//
// It is the point of the narrowing. Before it, every test that ran the
// interpreter built a real session under a temp dir and populated an
// environment, because implementing eleven methods was the more expensive
// option. Four is cheaper than a session.
type table struct {
	nodes   []node.Node
	updated []node.Node
}

func (t *table) Nodes() []node.Node { return t.nodes }

func (t *table) Resolve(sel string) (node.Node, error) {
	for _, n := range t.nodes {
		if sel == fmt.Sprintf("node%d", n.Index) {
			return n, nil
		}
	}
	return node.Node{}, fmt.Errorf("no node %q", sel)
}

func (t *table) ResolveEach(sels []string) ([]node.Node, error) {
	var out []node.Node
	for _, s := range sels {
		n, err := t.Resolve(s)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func (t *table) UpdateNode(n node.Node) { t.updated = append(t.updated, n) }

// sink records what the interpreter writes, in four methods rather than twelve.
type sink struct {
	steps  []session.StepResult
	assert []session.AssertResult
	posts  []session.PostResult
	status session.TestStatus
}

func (s *sink) Step(_ int, r session.StepResult) { s.steps = append(s.steps, r) }
func (s *sink) Assert(r session.AssertResult)    { s.assert = append(s.assert, r) }
func (s *sink) PostAction(r session.PostResult)  { s.posts = append(s.posts, r) }
func (s *sink) Status(st session.TestStatus)     { s.status = st }

// noop is an action that does nothing and reports it ran.
type noop struct{ ran *bool }

func (a noop) Do(context.Context, *interp.ActionCtx) error { *a.ran = true; return nil }

// TestRun_NeedsNoSession is B2's outcome, stated as a capability rather than an
// import count: the interpreter runs a spec against four small methods, with no
// session, no environment on disk and no temp directory.
//
// The full detach — moving StepResult and its siblings into this package — was
// considered and rejected. It would leave two structs that have to agree, and a
// field added to one and forgotten in the other drops evidence from an artifact
// silently. That is the failure this codebase has spent the year removing, and
// trading it for one fewer import is a bad trade.
func TestRun_NeedsNoSession(t *testing.T) {
	reg := interp.NewRegistry()
	ran := false
	reg.RegisterAction("tx", noop{ran: &ran})

	tbl := &table{nodes: []node.Node{{Index: 1, Role: node.RoleBP, RPCURL: "http://n1"}}}
	rec := &sink{}

	i := interp.NewInterpreter(interp.Deps{Actions: reg})
	status, err := i.Run(context.Background(), dsl.Spec{
		ID:         "no-session",
		Steps:      []map[string]any{{"tx": map[string]any{"on": "node1"}}},
		Assertions: []map[string]any{},
	}, tbl, rec)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !ran {
		t.Error("the step did not run")
	}
	if status != session.StatusPass {
		t.Errorf("status = %q, want pass", status)
	}
	if len(rec.steps) != 1 {
		t.Errorf("recorded %d steps, want 1", len(rec.steps))
	}
}

// TestNodeTable_IsSatisfiedByAnEnvironment keeps the narrowing honest: the
// engine still passes a session environment, and it must go on satisfying the
// requirement without knowing the requirement exists.
func TestNodeTable_IsSatisfiedByAnEnvironment(t *testing.T) {
	var _ interp.NodeTable = (session.Environment)(nil)
	var _ interp.Recorder = (session.TestRecord)(nil)
}
