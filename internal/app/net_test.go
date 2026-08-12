package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
)

func fixedClock() time.Time { return time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC) }

func TestNetNewThenStatus(t *testing.T) {
	dir := t.TempDir()
	d := app.Deps{Clock: fixedClock}

	out, err := app.NetNew(context.Background(), d, app.NetNewIn{
		DataDir: dir, Chain: "stablenet",
	})
	if err != nil {
		t.Fatalf("NetNew: %v", err)
	}
	if out.Detail == "" {
		t.Fatal("NetNew returned empty detail")
	}

	st, err := app.NetStatus(context.Background(), d, app.NetStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetStatus: %v", err)
	}
	if st.State.Chain != "stablenet" {
		t.Fatalf("chain = %q", st.State.Chain)
	}
	if step, ok := st.State.Steps["new"]; !ok || !step.Done {
		t.Fatalf("new step not recorded: %+v", st.State.Steps)
	}
	// The persisted step timestamp comes from the injected clock, proving Deps
	// reach the workspace.
	if step := st.State.Steps["new"]; step.At != "2026-08-12T00:00:00Z" {
		t.Fatalf("step stamped %q, want the injected clock", step.At)
	}
}

func TestNetNewRejectsUnknownChain(t *testing.T) {
	_, err := app.NetNew(context.Background(), app.Deps{}, app.NetNewIn{
		DataDir: t.TempDir(), Chain: "nope",
	})
	if err == nil {
		t.Fatal("unknown chain must fail")
	}
}
