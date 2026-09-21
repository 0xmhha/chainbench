package app

import (
	"context"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
)

// Hardfork verbs live in the chainsetup module; app wraps them for MCP.

type (
	// HardforkPlanIn shapes hardfork plan.
	HardforkPlanIn = verb.HardforkPlanIn
	// HardforkPlanOut is the plan report.
	HardforkPlanOut = verb.HardforkPlanOut
	// HardforkExecuteIn shapes hardfork execute.
	HardforkExecuteIn = verb.HardforkExecuteIn
	// HardforkExecuteOut is the execution report.
	HardforkExecuteOut = verb.HardforkExecuteOut
)

// HardforkPlan reports what a hardfork would do.
func HardforkPlan(ctx context.Context, d Deps, in HardforkPlanIn) (HardforkPlanOut, error) {
	return verb.HardforkPlan(ctx, d.chainsetupDeps(), in)
}

// HardforkExecute performs the hardfork.
func HardforkExecute(ctx context.Context, d Deps, in HardforkExecuteIn) (HardforkExecuteOut, error) {
	return verb.HardforkExecute(ctx, d.chainsetupDeps(), in)
}
