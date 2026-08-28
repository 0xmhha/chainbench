package chainsetup_test

import (
	"context"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

func TestHardforkPlan_ReadsTheFromChainFromTheNetwork(t *testing.T) {
	dir, _, _ := launchedNetwork(t)

	out, err := chainsetup.HardforkPlan(context.Background(), chainsetup.Deps{}, chainsetup.HardforkPlanIn{
		DataDir: dir, ToChain: "wbft", Block: 100,
	})
	if err != nil {
		t.Fatalf("HardforkPlan: %v", err)
	}
	// The from-chain is never guessed: it comes from the running network.
	if out.Plan.FromChain != "stablenet" || out.Plan.ToChain != "wbft" {
		t.Errorf("plan = %s -> %s, want stablenet -> wbft", out.Plan.FromChain, out.Plan.ToChain)
	}
	if len(out.Plan.Swaps) != 2 {
		t.Errorf("got %d swaps, want one per node", len(out.Plan.Swaps))
	}
	if out.To.Manifest().ID != "wbft" {
		t.Errorf("target plugin = %s, want wbft", out.To.Manifest().ID)
	}
}

func TestHardforkPlan_SameChainNeedsAnExplicitBinary(t *testing.T) {
	// Both sides resolve to the same manifest binary name, so without an
	// explicit post-fork build there is literally nothing to swap.
	dir, _, _ := launchedNetwork(t)

	_, err := chainsetup.HardforkPlan(context.Background(), chainsetup.Deps{}, chainsetup.HardforkPlanIn{
		DataDir: dir, ToChain: "stablenet", Block: 100,
	})
	if err == nil {
		t.Fatal("want an error for a same-chain hardfork with no binary")
	}
	if !strings.Contains(err.Error(), "post-fork binary") {
		t.Errorf("error should say what is missing, got: %v", err)
	}

	// With one it plans normally.
	if _, err := chainsetup.HardforkPlan(context.Background(), chainsetup.Deps{}, chainsetup.HardforkPlanIn{
		DataDir: dir, ToChain: "stablenet", ToBinary: "/opt/gstable-postfork", Block: 100,
	}); err != nil {
		t.Errorf("same-chain hardfork with a binary should plan: %v", err)
	}
}

func TestHardforkPlan_RequiresADataDirAndTarget(t *testing.T) {
	dir, _, _ := launchedNetwork(t)
	if _, err := chainsetup.HardforkPlan(context.Background(), chainsetup.Deps{}, chainsetup.HardforkPlanIn{ToChain: "wbft"}); err == nil {
		t.Error("want an error without a data dir")
	}
	if _, err := chainsetup.HardforkPlan(context.Background(), chainsetup.Deps{}, chainsetup.HardforkPlanIn{DataDir: dir}); err == nil {
		t.Error("want an error without a target chain")
	}
}

func TestHardforkExecute_RecordsThePostForkBinaryChainAndPids(t *testing.T) {
	dir, stub, deps := launchedNetwork(t)
	planned, err := chainsetup.HardforkPlan(context.Background(), deps, chainsetup.HardforkPlanIn{
		DataDir: dir, ToChain: "wbft", Block: 100,
	})
	if err != nil {
		t.Fatalf("HardforkPlan: %v", err)
	}

	const postFork = "/opt/gwbft"
	res, err := chainsetup.HardforkExecute(context.Background(), deps, chainsetup.HardforkExecuteIn{
		Plan: planned, DataDir: dir, Binary: postFork,
	})
	if err != nil {
		t.Fatalf("HardforkExecute: %v", err)
	}
	if len(res.Nodes.Nodes) != 2 || len(stub.launched) != 2 {
		t.Errorf("upgraded %d nodes, driver launched %v", len(res.Nodes.Nodes), stub.launched)
	}
	// A later single-node restart must start from the binary the network is
	// actually running, on the chain it now is, with the pids it now has.
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatalf("reopen workspace: %v", err)
	}
	st := ws.State()
	if st.Binary != postFork || st.Chain != "wbft" {
		t.Errorf("workspace after the fork: binary %q chain %q, want %q / wbft", st.Binary, st.Chain, postFork)
	}
	for _, rec := range st.Nodes {
		if rec.PID != 2000+rec.Index {
			t.Errorf("node%d pid = %d, want the relaunched pid %d", rec.Index, rec.PID, 2000+rec.Index)
		}
	}
	if !st.Steps["hardfork"].Done {
		t.Error("the fork must be recorded as a step")
	}
}

func TestHardforkExecute_RequiresAResolvedBinary(t *testing.T) {
	dir, _, deps := launchedNetwork(t)
	planned, err := chainsetup.HardforkPlan(context.Background(), deps, chainsetup.HardforkPlanIn{
		DataDir: dir, ToChain: "wbft", Block: 100,
	})
	if err != nil {
		t.Fatalf("HardforkPlan: %v", err)
	}

	if _, err := chainsetup.HardforkExecute(context.Background(), deps, chainsetup.HardforkExecuteIn{
		Plan: planned, DataDir: dir,
	}); err == nil {
		t.Error("want an error without a post-fork binary")
	}
}
