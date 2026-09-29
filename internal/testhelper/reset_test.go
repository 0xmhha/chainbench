package testhelper

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// fakeResetControl is a node control that can also reset a node
// (interp.NodeResetter).
type fakeResetControl struct {
	fakeNodeControl
	reset []int
}

func (c *fakeResetControl) Reset(_ context.Context, n node.Node) (node.Node, error) {
	c.reset = append(c.reset, n.Index)
	n.PID = 0
	return n, nil
}

func TestResetNodeAction_WiredAndResets(t *testing.T) {
	ctrl := &fakeResetControl{}
	d := faultDeps(ctrl)
	env := envWithNodes(t, 4, "http://unused")

	act, ok := d.Actions.Action(actionResetNode)
	if !ok {
		t.Fatal("resetNode not registered")
	}
	if err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{"on": "node4"}}); err != nil {
		t.Fatalf("resetNode: %v", err)
	}
	if len(ctrl.reset) != 1 || ctrl.reset[0] != 4 {
		t.Fatalf("reset = %v, want [4]", ctrl.reset)
	}
	n, err := env.Resolve("node4")
	if err != nil {
		t.Fatal(err)
	}
	if n.PID != 0 {
		t.Errorf("node4 pid = %d after reset, want 0 written back", n.PID)
	}
}

// TestResetNodeAction_ControlThatCannotReset: attach mode owns no datadirs, so
// the step says so instead of doing a plain stop that would leave the chain.
func TestResetNodeAction_ControlThatCannotReset(t *testing.T) {
	ctrl := &fakeNodeControl{}
	d := faultDeps(ctrl)
	env := envWithNodes(t, 4, "http://unused")
	act, _ := d.Actions.Action(actionResetNode)
	err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{"on": "node4"}})
	if err == nil || !strings.Contains(err.Error(), "cannot reset") {
		t.Fatalf("err = %v, want a refusal saying the control cannot reset", err)
	}
	if len(ctrl.stopped) != 0 {
		t.Errorf("the node was stopped anyway: %v", ctrl.stopped)
	}
}
