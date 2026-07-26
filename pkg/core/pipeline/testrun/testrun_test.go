package testrun_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/obs"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// fakeClient is a testkit.Client that needs no network.
type fakeClient struct{}

func (fakeClient) Call(context.Context, string, any, ...any) error { return nil }
func (fakeClient) BlockNumber(context.Context) (uint64, error)     { return 10, nil }
func (fakeClient) ChainID(context.Context) (uint64, error)         { return 8283, nil }

func init() {
	testkit.Register(testkit.Case{
		Name: "tr-ok", Category: "unit", RequiresCaps: []string{"rpc"},
		Fn: func(t *testkit.T) {
			id, err := t.Primary().ChainID(t.Ctx())
			t.NoErr(err, "chain id")
			t.Truef(id == 8283, "want 8283, got %d", id)
		},
	})
	testkit.Register(testkit.Case{
		Name: "tr-fail", Category: "unit",
		Fn: func(t *testkit.T) { t.Errorf("intentional failure") },
	})
	testkit.Register(testkit.Case{
		Name: "tr-gated-chain", Category: "unit", ChainCompat: []string{"ethereum"},
		Fn: func(t *testkit.T) { t.Errorf("should not run") },
	})
	testkit.Register(testkit.Case{
		Name: "tr-gated-cap", Category: "unit", RequiresCaps: []string{"consensus"},
		Fn: func(t *testkit.T) { t.Errorf("should not run") },
	})
}

func TestRun_GatingRunningReporting(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	// attach sets Capabilities ["rpc"]; "consensus" is absent -> tr-gated-cap skips.

	store := obs.NewMemStore()
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{
		Names:   []string{"tr-ok", "tr-fail", "tr-gated-chain", "tr-gated-cap"},
		Factory: func(string) testkit.Client { return fakeClient{} },
		Store:   store,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	byName := map[string]testkit.Result{}
	for _, r := range rep.Results {
		byName[r.Name] = r
	}
	if byName["tr-ok"].Status != testkit.StatusPass {
		t.Errorf("tr-ok: %+v", byName["tr-ok"])
	}
	if byName["tr-fail"].Status != testkit.StatusFail || byName["tr-fail"].Message == "" {
		t.Errorf("tr-fail: %+v", byName["tr-fail"])
	}
	if byName["tr-gated-chain"].Status != testkit.StatusSkip {
		t.Errorf("tr-gated-chain should skip (wrong chain): %+v", byName["tr-gated-chain"])
	}
	if byName["tr-gated-cap"].Status != testkit.StatusSkip {
		t.Errorf("tr-gated-cap should skip (missing cap): %+v", byName["tr-gated-cap"])
	}

	pass, fail, skip := rep.Counts()
	if pass != 1 || fail != 1 || skip != 2 {
		t.Errorf("counts: pass=%d fail=%d skip=%d", pass, fail, skip)
	}
	if !rep.Failed() {
		t.Error("report should be Failed()")
	}
	// Coverage: applicable = tr-ok, tr-fail, tr-gated-cap (tr-gated-chain is
	// chain-incompatible, so excluded); ran = 2 -> 2/3 = 66%.
	if rep.Applicable != 3 {
		t.Errorf("applicable: %d, want 3", rep.Applicable)
	}
	if got := rep.Coverage(); got != 66 {
		t.Errorf("coverage: %d%%, want 66%%", got)
	}
	// Store got a record per case.
	if len(store.ListRuns()) != 4 {
		t.Errorf("store runs: %d, want 4", len(store.ListRuns()))
	}
	if r, ok := store.GetRun("test/tr-fail"); !ok || r.Status != obs.RunFailed {
		t.Errorf("tr-fail record: %+v ok=%v", r, ok)
	}
	// A skip is persisted as RunSkipped, never RunSucceeded (no false green).
	if r, ok := store.GetRun("test/tr-gated-cap"); !ok || r.Status != obs.RunSkipped {
		t.Errorf("tr-gated-cap record should be RunSkipped: %+v ok=%v", r, ok)
	}
	if r, ok := store.GetRun("test/tr-ok"); !ok || r.Status != obs.RunSucceeded {
		t.Errorf("tr-ok record should be RunSucceeded: %+v ok=%v", r, ok)
	}
}
