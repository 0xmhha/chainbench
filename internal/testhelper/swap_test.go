package testhelper

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// fakeSwapControl is a node control that also swaps binaries (interp.NodeSwapper).
type fakeSwapControl struct {
	fakeNodeControl
	swapped []int
	binary  string
}

func (c *fakeSwapControl) Swap(_ context.Context, n node.Node, binary string) (node.Node, error) {
	c.swapped = append(c.swapped, n.Index)
	c.binary = binary
	n.PID = 7777
	return n, nil
}

func TestSwapNodeAction_WiredAndSwaps(t *testing.T) {
	ctrl := &fakeSwapControl{}
	d := faultDeps(ctrl)
	env := envWithNodes(t, 4, "http://unused")

	act, ok := d.Actions.Action(actionSwapNode)
	if !ok {
		t.Fatal("swapNode not registered")
	}
	err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{
		"on": "bp3", "binary": "/opt/gwbft",
	}})
	if err != nil {
		t.Fatalf("swapNode: %v", err)
	}
	if len(ctrl.swapped) != 1 || ctrl.swapped[0] != 3 {
		t.Fatalf("swapped = %v, want [3]", ctrl.swapped)
	}
	if ctrl.binary != "/opt/gwbft" {
		t.Errorf("binary = %q, want /opt/gwbft", ctrl.binary)
	}
	// The relaunched node (new pid) is written back to the env table.
	n, err := env.Resolve("bp3")
	if err != nil {
		t.Fatal(err)
	}
	if n.PID != 7777 {
		t.Errorf("node3 pid = %d after swap, want the relaunched 7777", n.PID)
	}
}

func TestSwapNodeAction_RequiresBinary(t *testing.T) {
	d := faultDeps(&fakeSwapControl{})
	env := envWithNodes(t, 4, "http://unused")
	act, _ := d.Actions.Action(actionSwapNode)
	if err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{"on": "bp1"}}); err == nil {
		t.Fatal("want an error swapping with no binary")
	}
}

// TestSwapNodeAction_UnsupportedControl: a control that cannot swap (plain
// attach's Stop/Start-only control) reports that, rather than failing obscurely.
func TestSwapNodeAction_UnsupportedControl(t *testing.T) {
	d := faultDeps(&fakeNodeControl{})
	env := envWithNodes(t, 4, "http://unused")
	act, _ := d.Actions.Action(actionSwapNode)
	err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{
		"on": "bp1", "binary": "/opt/gwbft",
	}})
	if err == nil {
		t.Fatal("want an error when the control cannot swap")
	}
}
