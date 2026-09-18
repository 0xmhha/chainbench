package process_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/process"
)

// TestLedger_RecordsQueriesAndSurvivesReopen pins the ledger's contract: what
// one command records, a later command reads back — machine, binary, command
// line, pid — and the file is the persistence, not the process.
func TestLedger_RecordsQueriesAndSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	l, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := process.Proc{
		PID: 4242, Label: "node1", Binary: "gstable",
		Command: "gstable --config config_node1.toml", Host: "192.0.2.11",
	}
	if err := l.Record(p); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := l.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	back, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := back.Get("node1")
	if !ok || got != p {
		t.Fatalf("reopened ledger lost the entry: %+v", got)
	}
	if found := back.FindBinary("192.0.2.11", "gstable"); len(found) != 1 {
		t.Fatalf("FindBinary: %+v", found)
	}
	if found := back.FindBinary("192.0.2.12", "gstable"); len(found) != 0 {
		t.Fatalf("FindBinary matched the wrong machine: %+v", found)
	}
}

// TestLedger_RefusesADoubleLaunch: recording a label that is already recorded
// is the caller starting something twice, refused with both pids named.
func TestLedger_RefusesADoubleLaunch(t *testing.T) {
	l, err := process.OpenLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Record(process.Proc{PID: 100, Label: "node1"}); err != nil {
		t.Fatal(err)
	}
	err = l.Record(process.Proc{PID: 200, Label: "node1"})
	if err == nil {
		t.Fatal("double launch recorded")
	}
	if !strings.Contains(err.Error(), "100") || !strings.Contains(err.Error(), "200") {
		t.Errorf("refusal names neither pid: %v", err)
	}

	// Stop, clear, relaunch: the normal cycle.
	if _, ok := l.Clear("node1"); !ok {
		t.Fatal("clear lost the entry")
	}
	if err := l.Record(process.Proc{PID: 200, Label: "node1"}); err != nil {
		t.Fatalf("relaunch after clear: %v", err)
	}
}

// TestLedger_RejectsUnusableEntries: a record without a label cannot be
// queried or cleared, and a non-positive pid is not a live process.
func TestLedger_RejectsUnusableEntries(t *testing.T) {
	l, err := process.OpenLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Record(process.Proc{PID: 1}); err == nil {
		t.Error("recorded an entry with no label")
	}
	if err := l.Record(process.Proc{PID: 0, Label: "x"}); err == nil {
		t.Error("recorded pid 0")
	}
}
