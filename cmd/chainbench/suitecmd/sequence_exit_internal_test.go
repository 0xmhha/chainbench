package suitecmd

import (
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/cmd/chainbench/exitcode"
	"github.com/0xmhha/chainbench/internal/app"
)

// TestSequenceExit_MapsTheThreeOutcomesToCodes pins the half of MON-015 that
// running one passing definition cannot show.
//
// The sequential path used to return one generic error, so CI could not tell a
// definition that never ran from a test that ran and failed. Totals() now keeps
// the three apart and this turns them into the codes: 2 when something could not
// run or was blocked, 1 when tests ran and failed, 0 otherwise. The blocked
// operand in particular had no test of its own — the live check that exercised
// code 2 got there through a setup error, which is the other half of the same
// condition.
func TestSequenceExit_MapsTheThreeOutcomesToCodes(t *testing.T) {
	mk := func(pass, fail, blocked int) app.SuiteRunResult {
		var r app.SuiteRunResult
		r.Out.Summary.Summary.Pass = pass
		r.Out.Summary.Summary.Fail = fail
		r.Out.Summary.Summary.Blocked = blocked
		return r
	}
	setupErr := app.SuiteRunResult{Spec: "a.json", Err: "compose failed"}

	cases := []struct {
		name string
		runs []app.SuiteRunResult
		want int // 0 means no error at all
	}{
		{"all passed", []app.SuiteRunResult{mk(3, 0, 0), mk(1, 0, 0)}, 0},
		{"a test failed", []app.SuiteRunResult{mk(1, 1, 0)}, 1},
		{"a test was blocked", []app.SuiteRunResult{mk(1, 0, 1)}, 2},
		{"a definition could not run", []app.SuiteRunResult{mk(1, 0, 0), setupErr}, 2},
		// Blocked outranks failed: a run that could not be carried out is worse
		// news than one that was and came out wrong, and CI gates on the
		// difference.
		{"blocked and failed together", []app.SuiteRunResult{mk(0, 2, 1)}, 2},
		{"setup error and failure together", []app.SuiteRunResult{mk(0, 1, 0), setupErr}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := sequenceExit(app.RunSuitesOut{Runs: tc.runs})
			if tc.want == 0 {
				if err != nil {
					t.Fatalf("a clean sequence must not error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want exit code %d, got no error", tc.want)
			}
			var ec *exitcode.Error
			if !errors.As(err, &ec) {
				t.Fatalf("the error must carry an exit code, got %T: %v", err, err)
			}
			if ec.Code != tc.want {
				t.Fatalf("exit code = %d, want %d (%v)", ec.Code, tc.want, err)
			}
		})
	}
}
