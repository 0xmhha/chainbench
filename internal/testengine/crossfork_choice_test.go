package testengine

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// TestCasesCrossFork_TheStepIsHowACaseClaimsTheMoment.
//
// The composition crosses the declared fork on its own, because a case that
// only wants a chain past the fork should not have to say so. A case that has
// to act BEFORE the fork names the step, and then the composition leaves the
// fork alone — otherwise the network would already be across by the time the
// case's first statement ran.
func TestCasesCrossFork_TheStepIsHowACaseClaimsTheMoment(t *testing.T) {
	plain := dsl.Spec{Sequence: []dsl.Statement{
		{Do: "waitBlock"}, {Expect: "blockNumber"},
	}}
	claiming := dsl.Spec{Sequence: []dsl.Statement{
		{Do: "sendTx"}, {Do: dsl.ActionCrossFork}, {Expect: "rpcCall"},
	}}

	if casesCrossFork([]dsl.Spec{plain}) {
		t.Error("a case that never names the step was read as claiming the fork")
	}
	if !casesCrossFork([]dsl.Spec{claiming}) {
		t.Error("a case that names the step was not read as claiming the fork")
	}
	// A suite is one network. One case claiming the moment is enough: crossing
	// during composition would put every case past the fork, including that one.
	if !casesCrossFork([]dsl.Spec{plain, claiming}) {
		t.Error("one case claiming the fork did not hold for the suite it shares a network with")
	}
	if casesCrossFork(nil) {
		t.Error("an empty suite claimed the fork")
	}
}
