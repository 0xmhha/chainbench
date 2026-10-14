//go:build e2e

package main

import (
	"context"
)

// finishHandoffWorkspace follows the handoff's two teardown operations. The
// stop callback must prove termination before disposable data can be reclaimed.
func finishHandoffWorkspace(ctx context.Context, workspace string, stop, reclaim func(context.Context, string) error) error {
	if err := stop(ctx, workspace); err != nil {
		return err
	}
	return reclaim(ctx, workspace)
}
