// Test seams for this package's unexported setup-failure handling.
//
// The behaviour under test is what happens when setting a network up fails,
// which a black-box test cannot reach: composed is unexported and the only way
// in is a real composition. These two expose the decision itself, so the test
// can drive both directions without launching anything.

package testengine

import "context"

// ComposedForTest builds a composed network whose only live part is how it is
// taken down.
func ComposedForTest(teardown func(context.Context) error) composed {
	return composed{teardown: teardown}
}

// StopAfterFailedSetupForTest exposes stopAfterFailedSetup.
func StopAfterFailedSetupForTest(ctx context.Context, net composed, keepUp bool, setupErr error) error {
	return stopAfterFailedSetup(ctx, net, keepUp, setupErr)
}
