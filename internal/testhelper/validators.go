package testhelper

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// Asking a chain who its validators are.
//
// Every chain here answers the question and no two answer it the same way.
// wbft and stablenet serve istanbul_getValidators. wemix serves no such
// method — go-wemix's "wemix" namespace holds only the brioche reward calls —
// and its set comes from the governance contract its nodes agree on.
//
// A case that wrote either route would run on one chain and be refused by the
// next, and the refusal is not even honest: measured 2026-09-28, a case that
// named the method the wemix manifest declares got
// "wemix_getValidators does not exist" from a chain that knows its validators
// perfectly well.
//
// So the case names the question. registry.RunningValidators picks the route,
// and it is the only place that picks it; the run carries that one choice in
// as Deps.Validators rather than making it again here.

// assertValidators is the validator-set assertion and read source.
const assertValidators = "validators"

// readValidators returns the chain's current validator set as the addresses
// its own consensus recognizes.
//
// A slice, not a count, so a spec can weigh it however it means to — Len for
// "there are four", Contains for "this node is one of them". Len is the
// default comparator because counting is the common case.
func readValidators(ctx context.Context, d *interp.Deps, c *rpc.Client, _ map[string]any) (any, error) {
	if d == nil || d.Validators == nil {
		return nil, fmt.Errorf("dsl: validators: this run resolved no chain, so there is no way to know how this one answers")
	}
	vals, err := d.Validators(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("dsl: validators: %w", err)
	}
	return vals, nil
}
