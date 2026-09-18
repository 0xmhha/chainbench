package testhelper

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// crossForkStepTimeout is the wait a step takes when it names none. The
// composition's own default is longer; a case that names the step usually knows
// roughly how far the fork is and can say.
const crossForkStepTimeout = 5 * time.Minute

// seedCrossForkBuiltins registers the fork-crossing action.
func seedCrossForkBuiltins(r interp.Registry) {
	r.RegisterAction(dsl.ActionCrossFork, crossForkAction{})
}

// crossForkAction waits for the network to reach the block before its declared
// hardfork and hands production to the build that seals after it.
//
// A case names it when it has to act BEFORE the fork and then ask whether what
// it did survived: send a transaction, deploy a contract, register something,
// and only then cross. A case that has nothing to do beforehand says nothing
// and the composition crosses the fork on its own.
//
// It takes no "on" selector. The handover is the network's, not a node's — who
// stops and who starts producing is decided by which build each node runs, and
// that is recorded in the composition rather than chosen by a step.
//
// Args: timeout (duration string, optional).
type crossForkAction struct{}

func (crossForkAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	if ac.Deps == nil || ac.Deps.Nodes == nil {
		return fmt.Errorf("dsl: %s needs node control, which this run has none of (attach mode does not own the node processes)", dsl.ActionCrossFork)
	}
	if ac.Env == nil {
		return fmt.Errorf("dsl: %s: no environment", dsl.ActionCrossFork)
	}
	crosser, ok := ac.Deps.Nodes.(interp.ForkCrosser)
	if !ok {
		return fmt.Errorf("dsl: %s: this run's node control cannot cross a fork", dsl.ActionCrossFork)
	}
	timeout := crossForkStepTimeout
	if raw, given := ac.Args["timeout"].(string); given && raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("dsl: %s: timeout %q: %w", dsl.ActionCrossFork, raw, err)
		}
		timeout = d
	}
	nodes, err := crosser.CrossFork(ctx, timeout)
	if err != nil {
		return fmt.Errorf("dsl: %s: %w", dsl.ActionCrossFork, err)
	}
	// Every node, because crossing changes more than the ones that moved: the
	// successors have a new role and a new pid, and a table holding the old pid
	// would stop the wrong process later.
	for _, n := range nodes {
		ac.Env.UpdateNode(n)
	}
	return nil
}
