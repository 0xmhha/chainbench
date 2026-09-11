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

// dirObservations is the per-test folder holding failure evidence (E8), mirrored
// from the session layout.
const dirObservations = "observations"

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
	Seq int    `json:"seq"`
	ID  string `json:"id"`
	Env string `json:"env,omitempty"`
	// Session names the run this test came from. It is empty in a single run's
	// report, where the report itself names the session, and set in a combined
	// one, where "seq 1" means nothing without it -- several runs each have a seq 1.
	Session  string   `json:"session,omitempty"`
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
	link := func(p string) {
		if rel, err := filepath.Rel(sessionDir, p); err == nil {
			out = append(out, rel)
		} else {
			out = append(out, p)
		}
	}
	for _, name := range evidenceFiles {
		p := filepath.Join(testDir, name)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		link(p)
	}
	// A failed test's gathered evidence (E8) lives under observations/; link each
	// file so the report points at the logs, process, and RPC/block snapshot
	// behind a failure. ReadDir returns names sorted, so the order is stable.
	obs := filepath.Join(testDir, dirObservations)
	if entries, err := os.ReadDir(obs); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				link(filepath.Join(obs, e.Name()))
			}
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

// Combine merges several runs' reports into one.
//
// A run per spec file is the normal way to use this harness, and until now each
// one answered only for itself: the verdict over a batch had to be assembled by
// eye, and "the most recent session" is the wrong answer to "did the batch pass".
// Combining is not a rendering concern -- the tally has to be summed in one
// place, or two surfaces will sum it differently.
//
// Tests keep their own seq and gain the session they came from, because seq is
// unique within a run and not across runs. Order follows the order given, which
// callers supply oldest-first, so a reader scans a batch in the order it ran.
func Combine(reps []Report) Report {
	out := Report{Session: CombinedSession}
	for _, r := range reps {
		if out.StartedAt == "" || (r.StartedAt != "" && r.StartedAt < out.StartedAt) {
			out.StartedAt = r.StartedAt
		}
		out.Summary.Pass += r.Summary.Pass
		out.Summary.Fail += r.Summary.Fail
		out.Summary.Blocked += r.Summary.Blocked
		out.Summary.Skip += r.Summary.Skip
		for _, t := range r.Tests {
			if t.Session == "" {
				t.Session = r.Session
			}
			out.Tests = append(out.Tests, t)
		}
	}
	return out
}

// CombinedSession is the session id a combined report carries. It is not a
// session that exists, and saying so is the point: a reader who follows it to a
// directory should find nothing rather than another run's evidence.
const CombinedSession = "(combined)"

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
