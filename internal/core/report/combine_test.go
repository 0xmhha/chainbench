package report_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/report"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// A run per spec file is the normal shape, and until Combine existed each run
// answered only for itself: the verdict over a batch had to be assembled by eye,
// and "the most recent session" is the wrong answer to "did the batch pass".

func run(id, startedAt string, c session.Counts, tests ...report.TestReport) report.Report {
	return report.Report{Session: id, StartedAt: startedAt, Summary: c, Tests: tests}
}

// TestCombine_SumsTheTally is the whole point: one verdict over several runs.
func TestCombine_SumsTheTally(t *testing.T) {
	got := report.Combine([]report.Report{
		run("A", "2026-09-11T01:00:00Z", session.Counts{Pass: 2, Fail: 1},
			report.TestReport{Seq: 1, ID: "a1", Status: "pass"},
			report.TestReport{Seq: 2, ID: "a2", Status: "fail"},
			report.TestReport{Seq: 3, ID: "a3", Status: "pass"}),
		run("B", "2026-09-11T02:00:00Z", session.Counts{Pass: 1, Blocked: 1, Skip: 3},
			report.TestReport{Seq: 1, ID: "b1", Status: "pass"}),
	})
	want := session.Counts{Pass: 3, Fail: 1, Blocked: 1, Skip: 3}
	if got.Summary != want {
		t.Errorf("summary = %+v, want %+v", got.Summary, want)
	}
	if len(got.Tests) != 4 {
		t.Errorf("combined %d tests, want 4", len(got.Tests))
	}
}

// TestCombine_NamesTheRunEachTestCameFrom: seq is unique within a run, not
// across runs, so two runs each have a seq 1 and the number alone misleads.
func TestCombine_NamesTheRunEachTestCameFrom(t *testing.T) {
	got := report.Combine([]report.Report{
		run("A", "", session.Counts{}, report.TestReport{Seq: 1, ID: "a1"}),
		run("B", "", session.Counts{}, report.TestReport{Seq: 1, ID: "b1"}),
	})
	for _, tr := range got.Tests {
		if tr.Session == "" {
			t.Fatalf("test %q carries no session, so its seq %d is ambiguous", tr.ID, tr.Seq)
		}
	}
	if got.Tests[0].Session == got.Tests[1].Session {
		t.Errorf("both tests claim session %q", got.Tests[0].Session)
	}
}

// TestCombine_KeepsAnAlreadyAttributedTest: combining a combined report must not
// relabel rows with the outer session, which names no run.
func TestCombine_KeepsAnAlreadyAttributedTest(t *testing.T) {
	inner := report.Combine([]report.Report{
		run("A", "", session.Counts{Pass: 1}, report.TestReport{Seq: 1, ID: "a1"}),
	})
	outer := report.Combine([]report.Report{inner})
	if got := outer.Tests[0].Session; got != "A" {
		t.Errorf("test attributed to %q, want the original run A", got)
	}
}

// TestCombine_StartedAtIsTheEarliest: the batch began when its first run began.
func TestCombine_StartedAtIsTheEarliest(t *testing.T) {
	got := report.Combine([]report.Report{
		run("late", "2026-09-11T05:00:00Z", session.Counts{}),
		run("early", "2026-09-11T01:00:00Z", session.Counts{}),
	})
	if got.StartedAt != "2026-09-11T01:00:00Z" {
		t.Errorf("startedAt = %q, want the earliest run's", got.StartedAt)
	}
}

// TestCombine_SessionIdNamesNoDirectory: a reader who follows the id should find
// nothing rather than another run's evidence, so it must not look like a real id.
func TestCombine_SessionIdNamesNoDirectory(t *testing.T) {
	got := report.Combine([]report.Report{run("UTC-20260911-010000", "", session.Counts{})})
	if got.Session == "UTC-20260911-010000" {
		t.Fatal("a combined report took a real session's id, so its evidence paths point at one run")
	}
	if !strings.Contains(got.Session, "combined") {
		t.Errorf("session id %q should say it is a combination", got.Session)
	}
}

// TestCombine_MarshalsTheAttribution: the JSON surface has to carry the session
// per test, or a program reading it cannot tell the runs apart either.
func TestCombine_MarshalsTheAttribution(t *testing.T) {
	got := report.Combine([]report.Report{
		run("A", "", session.Counts{}, report.TestReport{Seq: 1, ID: "a1"}),
	})
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"session":"A"`) {
		t.Errorf("the marshalled report does not attribute its test: %s", raw)
	}
}

// TestCombine_EmptyIsEmpty: nothing to combine is not an error and not a verdict.
func TestCombine_EmptyIsEmpty(t *testing.T) {
	got := report.Combine(nil)
	if len(got.Tests) != 0 || got.Summary != (session.Counts{}) {
		t.Errorf("combining nothing produced %+v", got)
	}
}
