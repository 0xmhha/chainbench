package testengine

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// TestOutcome is one test's recorded result in a session summary.
type TestOutcome struct {
	Seq    int    `json:"seq"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Summary is the session.json verdict: the per-test outcomes and the
// pass/fail/blocked/skip counts.
type Summary struct {
	Tests   []TestOutcome `json:"tests"`
	Summary struct {
		Pass    int `json:"pass"`
		Fail    int `json:"fail"`
		Blocked int `json:"blocked"`
		Skip    int `json:"skip"`
	} `json:"summary"`
}

// Failed reports whether any test failed or was blocked (a non-passing run).
func (s Summary) Failed() bool {
	return s.Summary.Fail > 0 || s.Summary.Blocked > 0
}

// ReadSessionSummary reads the session.json under root through the session
// package — the single reader of the schema — and maps it to a Summary. It is
// the one summary reader the CLI and MCP surfaces share so both report a run
// identically.
func ReadSessionSummary(root string) (Summary, error) {
	res, err := session.LoadDir(root)
	if err != nil {
		return Summary{}, fmt.Errorf("engine: %w", err)
	}
	s := Summary{Tests: make([]TestOutcome, 0, len(res.Tests))}
	for _, t := range res.Tests {
		s.Tests = append(s.Tests, TestOutcome{Seq: t.Seq, ID: t.ID, Status: t.Status})
	}
	s.Summary.Pass = res.Summary.Pass
	s.Summary.Fail = res.Summary.Fail
	s.Summary.Blocked = res.Summary.Blocked
	s.Summary.Skip = res.Summary.Skip
	return s, nil
}
