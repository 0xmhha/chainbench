package chainsetup

import (
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// TestNamedBinariesAreCheckedBeforeTheLaunch: a declaration that names several
// binaries has to have all of them on the target before anything starts, not
// only the one the first phase launches.
//
// Measured 2026-10-06 on the docker servers. The swap case was run with its
// upgrade binary pointed at a name no container held. The run composed, booted
// five nodes, swapped node5 onto the missing file, and then sat in a waitFor
// until the deadline:
//
//	result: fail
//	reason: step 11 (waitFor) failed: dsl: waitFor rpcCall: condition unmet
//	        (last read error: rpc: eth_blockNumber: Post "…:18605": EOF)
//
// Three things are wrong with that. It took over three minutes. It is recorded
// as a case failure, when the case was never asked. And the words "binary" and
// "gstable-hardfork" appear nowhere in the reason, so the operator reads a
// closed port and goes looking at the network.
//
// So every name in the binaries map is checked where the single binary already
// was, and the refusal carries errLaunchNoBinary — the kind that classifies as
// ChainLaunchNodesFailNoBinary, which a sweep reports as blocked rather than as
// a verdict about the case.
func TestNamedBinariesAreCheckedBeforeTheLaunch(t *testing.T) {
	// The missing one is named, and named by its map key, so a reader knows
	// which declaration to fix rather than which file to go hunting for.
	err := missingNamedBinaries(map[string]string{
		"default":  "/data/chainbench/bin/gstable",
		"upgrade":  "/data/chainbench/bin/gstable-hardfork",
		"postfork": "/data/chainbench/bin/gstable-postfork",
	}, func(path string) (bool, error) {
		return path == "/data/chainbench/bin/gstable", nil
	})
	if len(err) != 2 {
		t.Fatalf("reported %d missing, want 2: %v", len(err), err)
	}
	joined := strings.Join(err, "\n")
	for _, want := range []string{"binaries.upgrade", "gstable-hardfork", "binaries.postfork", "gstable-postfork"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the report does not name %q:\n%s", want, joined)
		}
	}
	// Sorted, so two runs of the same broken declaration read the same.
	if i, j := strings.Index(joined, "binaries.postfork"), strings.Index(joined, "binaries.upgrade"); i > j {
		t.Errorf("the report is not in name order:\n%s", joined)
	}

	// Nothing missing is nothing to say.
	if got := missingNamedBinaries(map[string]string{"default": "/bin/x"},
		func(string) (bool, error) { return true, nil }); len(got) != 0 {
		t.Errorf("reported %v for a map whose binaries are all present", got)
	}

	// A single-binary declaration carries no map, and the single binary is
	// already checked by name — this must not double-report it.
	if got := missingNamedBinaries(nil, func(string) (bool, error) { return false, nil }); len(got) != 0 {
		t.Errorf("reported %v for a declaration with no named binaries", got)
	}

	// A probe that cannot answer is not an absent binary: saying "missing" on a
	// broken ssh connection sends the operator to rebuild a file that is there.
	boom := errors.New("ssh: connection reset")
	got := missingNamedBinaries(map[string]string{"upgrade": "/bin/y"},
		func(string) (bool, error) { return false, boom })
	if len(got) != 1 || !strings.Contains(got[0], "connection reset") {
		t.Errorf("a probe failure is not reported as such: %v", got)
	}
	if strings.Contains(got[0], "not on the target") {
		t.Errorf("a probe failure is reported as an absent binary: %v", got)
	}
}

// TestNoBinaryClassifiesAsBlocked pins the half that makes the sweep read
// right: the kind the named-binary refusal carries has to be the one the
// launch classifier turns into ChainLaunchNodesFailNoBinary. A refusal with no
// kind lands in the unclassified state, which a sweep shows as a plain failure.
func TestNoBinaryClassifiesAsBlocked(t *testing.T) {
	marked := lifecycle.Mark(errLaunchNoBinary, errors.New("binaries.upgrade /data/bin/gstable-hardfork: not on the target"))
	if got := LaunchFailure(marked); got != lifecycle.ChainLaunchNodesFailNoBinary {
		t.Fatalf("classified as %s, want ChainLaunchNodesFailNoBinary", got)
	}
}

// TestTheAdviceMatchesWhatIsMissing: the closing line of the pre-launch refusal
// has to point at the fix for what was actually absent.
//
// Measured 2026-10-06. A run whose only problem was a named binary nobody had
// placed ended with "run the earlier steps (`chain genesis`, `chain config`,
// `chain init`) or check --binary". Every one of those steps had already run
// and all of them had succeeded, so the advice sent the reader to redo work
// that was fine. The fix was to build the second binary and put it on the
// target, or to point the variable at it, and the message said neither.
func TestTheAdviceMatchesWhatIsMissing(t *testing.T) {
	onlyBinaries := missingAdvice(true, false)
	if strings.Contains(onlyBinaries, "chain genesis") {
		t.Errorf("binaries-only advice sends the reader to redo the composition steps: %q", onlyBinaries)
	}
	for _, want := range []string{"place", "variable"} {
		if !strings.Contains(onlyBinaries, want) {
			t.Errorf("binaries-only advice does not mention %q: %q", want, onlyBinaries)
		}
	}

	// Anything else missing, and the composition steps are the lead again —
	// a genesis or a datadir that is not there is what they produce.
	withOthers := missingAdvice(true, true)
	if !strings.Contains(withOthers, "chain genesis") {
		t.Errorf("mixed advice drops the composition steps: %q", withOthers)
	}
	if got := missingAdvice(false, true); !strings.Contains(got, "chain genesis") {
		t.Errorf("non-binary advice drops the composition steps: %q", got)
	}
}
