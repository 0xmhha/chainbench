package feature_test

import (
	"context"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/feature"
	"testing"
)

func TestNodeResetRegistrationInvokesPublicAdapter(t *testing.T) {
	d, ok := feature.Lookup("node.reset")
	if !ok {
		t.Fatal("node reset is missing from the feature registry")
	}
	if d.Stage != feature.StageCompose || d.ReadOnly || d.Summary == "" {
		t.Fatal("reset metadata does not describe its effects")
	}
	input, ok := d.Input().(*app.NodeResetIn)
	if !ok {
		t.Fatal("reset registration uses a different input contract")
	}
	if _, err := d.Invoke(context.Background(), app.Deps{}, input); err == nil {
		t.Fatal("invalid reset reached execution")
	}
}
