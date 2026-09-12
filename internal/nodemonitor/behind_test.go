package nodemonitor_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/nodemonitor"
)

// fit is a node that passes every other condition, so each case below turns on
// the height lag alone.
func fit() nodemonitor.Facts {
	return nodemonitor.Facts{
		Node: 4, Label: "node4", Wanted: true,
		PIDAlive: true, RPCUp: true, Advancing: true, Syncing: false,
		Height: 30, WantHeight: 30,
	}
}

// TestClassify_ANodeFarBehindTheNetworkIsNotReady is the condition neither
// "syncing" nor "advancing" covered. Syncing is false both for a node that has
// caught up and for one that has not started — with no peers yet it is idle, not
// syncing — and advancing is a fact about the network, so the nodes that are
// producing make it true for the one that is not.
//
// Measured before this existed: a fourth validator reported ready while 33 blocks
// behind, every turn the round robin gave it was lost to a round-change timeout,
// and the chain rotated among three for its first 33 blocks.
func TestClassify_ANodeFarBehindTheNetworkIsNotReady(t *testing.T) {
	f := fit()
	f.Height, f.WantHeight = 0, 33
	got := nodemonitor.Classify(f)
	if got.Verdict != nodemonitor.Waitable {
		t.Fatalf("verdict = %v, want Waitable (detail: %s)", got.Verdict, strings.Join(got.Reasons, "; "))
	}
	for _, want := range []string{"behind the network", "0", "33"} {
		if !strings.Contains(strings.Join(got.Reasons, "; "), want) {
			t.Errorf("detail %q does not name %q — the reader has to guess which node is behind and by how much", strings.Join(got.Reasons, "; "), want)
		}
	}
}

// TestClassify_SamplingSkewIsNotBeingBehind keeps the check from making every
// run wait forever. Heights are read one node at a time while the chain keeps
// moving, so the last node read is routinely a block or two behind the first.
func TestClassify_SamplingSkewIsNotBeingBehind(t *testing.T) {
	for lag := uint64(0); lag <= 2; lag++ {
		f := fit()
		f.WantHeight = f.Height + lag
		if got := nodemonitor.Classify(f); got.Verdict != nodemonitor.Ready {
			t.Errorf("lag %d: verdict = %v, want Ready (detail: %s)", lag, got.Verdict, strings.Join(got.Reasons, "; "))
		}
	}
	f := fit()
	f.WantHeight = f.Height + 3
	if got := nodemonitor.Classify(f); got.Verdict != nodemonitor.Waitable {
		t.Errorf("lag 3: verdict = %v, want Waitable — the tolerance must have an edge", got.Verdict)
	}
}

// TestClassify_NoNetworkHeightLeavesTheLagUnchecked pins the opt-in: a caller
// that does not know the network's head must not have every node held back by a
// comparison against zero.
func TestClassify_NoNetworkHeightLeavesTheLagUnchecked(t *testing.T) {
	f := fit()
	f.Height, f.WantHeight = 0, 0
	if got := nodemonitor.Classify(f); got.Verdict != nodemonitor.Ready {
		t.Errorf("verdict = %v, want Ready when the observer supplied no network height (detail: %s)", got.Verdict, strings.Join(got.Reasons, "; "))
	}
}

// TestClassify_BeingBehindDoesNotOutrankSomethingWorse keeps the ordering: a
// node that is behind AND dead is a restart, not a wait, or the gate would sit
// out its whole budget waiting for a process that is gone.
func TestClassify_BeingBehindDoesNotOutrankSomethingWorse(t *testing.T) {
	f := fit()
	f.Height, f.WantHeight = 0, 33
	f.PIDAlive = false
	if got := nodemonitor.Classify(f); got.Verdict != nodemonitor.Restartable {
		t.Errorf("verdict = %v, want Restartable for a dead process (detail: %s)", got.Verdict, strings.Join(got.Reasons, "; "))
	}
}
