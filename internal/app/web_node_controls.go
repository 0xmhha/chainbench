package app

import (
	"context"

	"github.com/0xmhha/chainbench/internal/resource"
)

func verifyWebNodeProcesses(ctx context.Context, state State, selected []string, lookup resource.Lookup) error {
	if len(selected) != 1 {
		return ErrDeploymentConflict
	}
	for _, ns := range state.Nodes {
		if string(ns.NodeLabel()) != selected[0] || ns.PID == 0 {
			continue
		}
		access, err := (resource.Opener{Lookup: lookup}).Open(state.Target)
		if err != nil {
			return err
		}
		target, err := (resource.Opener{Lookup: lookup}).Inspect(ctx, state.Target)
		if err != nil {
			return err
		}
		observer := webObserverFor(access, target)
		if observer == nil {
			return ErrDeploymentConflict
		}
		binary := state.Binary
		if ns.Binary != "" {
			binary = state.Binaries[ns.Binary]
		}
		if observeWebNodeProcess(ctx, observer, binary, ns).State != "running" {
			return ErrDeploymentConflict
		}
	}
	return nil
}
