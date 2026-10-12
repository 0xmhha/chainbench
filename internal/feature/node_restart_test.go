package feature_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/feature"
)

func TestNodeRestartRegistrationInvokesPublicAdapter(t *testing.T) {
	d, ok := feature.Lookup("node.restart")
	if !ok || d.ReadOnly || d.Stage != feature.StageCompose {
		t.Fatal("restart effect is absent or classified as a read")
	}
	input, ok := d.Input().(*app.ChainRestartIn)
	if !ok {
		t.Fatal("restart registration uses a different adapter")
	}
	if _, err := d.Invoke(context.Background(), app.Deps{}, input); err == nil {
		t.Fatal("invalid restart input reached execution")
	}
}
