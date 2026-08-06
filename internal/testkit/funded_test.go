package testkit_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// A case that reads FundedKey and skips when it is absent must be recorded as a
// skip without a funded key and as a pass with one — the RunOpts key never
// touches the NodeSet.
func TestFundedKeyAndSkip(t *testing.T) {
	ns := node.NodeSet{Chain: "x", Capabilities: []string{"rpc"}}
	c := testkit.Case{Name: "funded-probe", Fn: func(tt *testkit.T) {
		if _, ok := tt.FundedKey(); !ok {
			tt.Skip("no funded key")
		}
	}}

	if r := testkit.RunCaseWith(context.Background(), c, ns, nil, testkit.RunOpts{}); r.Status != testkit.StatusSkip {
		t.Errorf("without key: status %s, want skip", r.Status)
	}
	if r := testkit.RunCaseWith(context.Background(), c, ns, nil, testkit.RunOpts{FundedKey: []byte{1, 2, 3}}); r.Status != testkit.StatusPass {
		t.Errorf("with key: status %s, want pass", r.Status)
	}
}
