package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// sessionFile is the session artifact file name written under a session root.
const sessionFile = "session.json"

// TestOutcome is one test's recorded result in a session summary.
type TestOutcome struct {
	Seq    int    `json:"seq"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Summary is the parsed session.json verdict: the per-test outcomes and the
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

// ReadSessionSummary reads and parses the session.json under root. It is the one
// reader the CLI and MCP surfaces share so both report a run identically.
func ReadSessionSummary(root string) (Summary, error) {
	data, err := os.ReadFile(filepath.Join(root, sessionFile))
	if err != nil {
		return Summary{}, fmt.Errorf("engine: read session: %w", err)
	}
	var s Summary
	if err := json.Unmarshal(data, &s); err != nil {
		return Summary{}, fmt.Errorf("engine: parse session: %w", err)
	}
	return s, nil
}
