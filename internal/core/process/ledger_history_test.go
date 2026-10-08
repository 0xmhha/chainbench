package process_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/process"
)

func TestLedgerRelaunchContinuesArchivedRevision(t *testing.T) {
	for _, operation := range []string{"record", "supersede"} {
		t.Run(operation, func(t *testing.T) {
			dir := t.TempDir()
			l, err := process.OpenLedger(dir)
			if err != nil {
				t.Fatal(err)
			}
			if err = l.Record(process.Proc{Label: "node1", PID: 100}); err != nil {
				t.Fatal(err)
			}
			if _, _, err = l.Supersede(process.Proc{Label: "node1", PID: 200}); err != nil {
				t.Fatal(err)
			}
			l.Clear("node1")
			if err = l.Save(); err != nil {
				t.Fatal(err)
			}
			l, err = process.OpenLedger(dir)
			if err != nil {
				t.Fatal(err)
			}
			next := process.Proc{Label: "node1", PID: 300}
			if operation == "record" {
				err = l.Record(next)
			} else {
				_, _, err = l.Supersede(next)
			}
			if err != nil {
				t.Fatal(err)
			}
			current, ok := l.Get("node1")
			if !ok || current.Revision != 1 || len(l.History("node1")) != 1 {
				t.Fatal("relaunch reset the archived revision or changed history", current, l.History("node1"))
			}
		})
	}
}

func TestLedgerRetirePersistsOnceAndContinuesExplicitRetry(t *testing.T) {
	dir := t.TempDir()
	l, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	original := process.Proc{Label: "node1", PID: 100, Command: "old --config owned", Revision: 4}
	if err = l.Record(original); err != nil {
		t.Fatal(err)
	}
	if retired, ok := l.Retire("node1"); !ok || retired != original {
		t.Fatal("retire lost the prior command", retired)
	}
	if _, ok := l.Retire("node1"); ok {
		t.Fatal("retiring twice duplicated a stopped entry")
	}
	if err = l.Save(); err != nil {
		t.Fatal(err)
	}
	l, err = process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := l.Get("node1"); ok {
		t.Fatal("retired process remained current after restart")
	}
	h := l.History("node1")
	if len(h) != 1 || h[0] != original {
		t.Fatal("retired history changed", h)
	}
	if err = l.Record(process.Proc{Label: "node1", PID: 200}); err != nil {
		t.Fatal(err)
	}
	current, ok := l.Get("node1")
	if !ok || current.Revision != 5 {
		t.Fatal("explicit retry reset retained revision", current)
	}
}
