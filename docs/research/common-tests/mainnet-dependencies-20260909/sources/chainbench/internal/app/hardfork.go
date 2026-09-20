package app

import (
	"context"

	chainsetupmod "github.com/0xmhha/chainbench/internal/chainsetup"
)

// Hardfork verbs live in the chainsetup module; app wraps them for MCP.

type (
	// HardforkPlanIn shapes hardfork plan.
	HardforkPlanIn = chainsetupmod.HardforkPlanIn
	// HardforkPlanOut is the plan report.
	HardforkPlanOut = chainsetupmod.HardforkPlanOut
	// HardforkExecuteIn shapes hardfork execute.
	HardforkExecuteIn = chainsetupmod.HardforkExecuteIn
	// HardforkExecuteOut is the execution report.
	HardforkExecuteOut = chainsetupmod.HardforkExecuteOut
)

// HardforkPlan reports what a hardfork would do.
func HardforkPlan(ctx context.Context, d Deps, in HardforkPlanIn) (HardforkPlanOut, error) {
	return chainsetupmod.HardforkPlan(ctx, d.chainsetupDeps(), in)
}

// HardforkExecute performs the hardfork.
func HardforkExecute(ctx context.Context, d Deps, in HardforkExecuteIn) (HardforkExecuteOut, error) {
	return chainsetupmod.HardforkExecute(ctx, d.chainsetupDeps(), in)
}
