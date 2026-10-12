package feature_test

import (
	"context"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/feature"
	"testing"
)

func TestNodeSwapRegistrationInvokesPublicAdapter(t *testing.T) {
	d, ok := feature.Lookup("node.swap")
	if !ok || d.ReadOnly || d.Stage != feature.StageCompose {
		t.Fatal("node replacement effect absent or classified as a read")
	}
	input, ok := d.Input().(*app.NodeSwapIn)
	if !ok {
		t.Fatal("replacement uses a different adapter")
	}
	if _, err := d.Invoke(context.Background(), app.Deps{}, input); err == nil {
		t.Fatal("invalid replacement reached execution")
	}
}
