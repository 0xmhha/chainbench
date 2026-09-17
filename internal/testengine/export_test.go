// Test seams for this package's unexported setup-failure handling.
//
// The behaviour under test is what happens when setting a network up fails,
// which a black-box test cannot reach: composed is unexported and the only way
// in is a real composition. These two expose the decision itself, so the test
// can drive both directions without launching anything.

package testengine

import (
	"context"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// ComposedForTest builds a composed network whose only live part is how it is
// taken down.
func ComposedForTest(teardown func(context.Context) error) composed {
	return composed{teardown: teardown}
}

// AfterFailedSetupForTest exposes afterFailedSetup with the gathering side
// disabled: it is driven with an empty workspace path, so the test drives the
// decision (stop or keep) without a network to gather from.
func AfterFailedSetupForTest(ctx context.Context, net composed, keepUp bool, setupErr error) error {
	var out RunSuiteOut
	return afterFailedSetup(ctx, chainsetup.Deps{}, "", net, keepUp, &out, setupErr)
}
