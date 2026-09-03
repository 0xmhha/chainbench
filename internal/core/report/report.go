package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// fileReport is the root report artifact name.
const fileReport = "report.json"

// evidenceFiles are the per-test artifacts a report links to when present. They
// are listed in a fixed order so the report is deterministic.
var evidenceFiles = []string{
	"spec.json",
	"steps.json",
	"assert.json",
	"status.json",
	"artifacts.json",
	"postaction.json",
}

// Report is the run-level report: the session's verdicts with a link to each
// test's evidence and the overall tally. It carries no verdict logic of its own
// — it mirrors what the session recorded.
type Report struct {
	// Session is the run's session id.
	Session string `json:"session"`
	// Command is the command line that produced the run.
	Command string `json:"command"`
	// StartedAt is the run start time (RFC3339).
	StartedAt string `json:"startedAt"`
	// Summary is the verdict tally over all tests.
	Summary session.Counts `json:"summary"`
	// Tests is one entry per test, verdict plus evidence links.
	Tests []TestReport `json:"tests"`
}

// TestReport is one test's verdict and the session-relative paths to the
// evidence that backs it (only files that exist are listed).
type TestReport struct {
	Seq      int      `json:"seq"`
	ID       string   `json:"id"`
	Env      string   `json:"env,omitempty"`
	Status   string   `json:"status"`
	Dir      string   `json:"dir"`
	Evidence []string `json:"evidence,omitempty"`
}

// Build reads a session directory's session.json and per-test evidence and
// assembles a Report. It does not write anything; Write persists the result.
func Build(sessionDir string) (Report, error) {
	res, err := session.LoadDir(sessionDir)
	if err != nil {
		return Report{}, fmt.Errorf("report: %w", err)
	}
	rep := Report{
		Session:   res.ID,
		Command:   res.Command,
		StartedAt: res.StartedAt,
		Summary:   res.Summary,
		Tests:     make([]TestReport, 0, len(res.Tests)),
	}
	for _, t := range res.Tests {
		td := session.TestDir(sessionDir, t.Seq, t.ID)
		rel, err := filepath.Rel(sessionDir, td)
		if err != nil {
			rel = td
		}
		rep.Tests = append(rep.Tests, TestReport{
			Seq:      t.Seq,
			ID:       t.ID,
			Env:      t.Env,
			Status:   t.Status,
			Dir:      rel,
			Evidence: evidenceUnder(sessionDir, td),
		})
	}
	return rep, nil
}

// evidenceUnder lists the session-relative paths of the evidence files that
// exist in a test's directory, in evidenceFiles order.
func evidenceUnder(sessionDir, testDir string) []string {
	var out []string
	for _, name := range evidenceFiles {
		p := filepath.Join(testDir, name)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if rel, err := filepath.Rel(sessionDir, p); err == nil {
			out = append(out, rel)
		} else {
			out = append(out, p)
		}
	}
	return out
}

// Write persists a report as report.json at the session root. It reuses the
// session package's atomic write so report.json is written the same way as every
// other artifact in the tree, not through a second copy of temp+rename.
func Write(sessionDir string, rep Report) error {
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("report: marshal: %w", err)
	}
	if err := session.WriteFileAtomic(filepath.Join(sessionDir, fileReport), b, 0o644); err != nil {
		return fmt.Errorf("report: write %s: %w", fileReport, err)
	}
	return nil
}

// Generate builds the report for a session directory and writes report.json.
// It is the one call the engine makes after the session is saved.
func Generate(sessionDir string) (Report, error) {
	rep, err := Build(sessionDir)
	if err != nil {
		return Report{}, err
	}
	if err := Write(sessionDir, rep); err != nil {
		return Report{}, err
	}
	return rep, nil
}

// Read loads a previously written report.json from a session directory, for
// display by the CLI and MCP report surfaces.
func Read(sessionDir string) (Report, error) {
	data, err := os.ReadFile(filepath.Join(sessionDir, fileReport))
	if err != nil {
		return Report{}, fmt.Errorf("report: read %s: %w", fileReport, err)
	}
	var rep Report
	if err := json.Unmarshal(data, &rep); err != nil {
		return Report{}, fmt.Errorf("report: parse %s: %w", fileReport, err)
	}
	return rep, nil
}
