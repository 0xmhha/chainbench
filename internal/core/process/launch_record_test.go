package process_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// specFor is a minimal launch spec for the record tests.
func specFor(index int, binary string, args ...string) process.NodeSpec {
	return process.NodeSpec{
		Index:   index,
		Role:    node.RoleValidator,
		Host:    "10.0.0.5",
		Binary:  binary,
		DataDir: "/data/node",
		Args:    args,
	}
}

// TestProcFor pins the launch record's shape: label from the node index, the
// binary's base name, and the full command line (binary + args).
func TestProcFor(t *testing.T) {
	p := process.ProcFor(specFor(0, "/opt/bin/gstable", "--datadir", "/data/node"), 4242)
	if p.PID != 4242 {
		t.Errorf("PID = %d, want 4242", p.PID)
	}
	if want := string(node.LabelFor(0)); p.Label != want {
		t.Errorf("Label = %q, want %q", p.Label, want)
	}
	if p.Binary != "gstable" {
		t.Errorf("Binary = %q, want %q (base name)", p.Binary, "gstable")
	}
	if want := "/opt/bin/gstable --datadir /data/node"; p.Command != want {
		t.Errorf("Command = %q, want %q", p.Command, want)
	}
	if p.Host != "10.0.0.5" || p.DataDir != "/data/node" {
		t.Errorf("Host/DataDir = %q/%q, want 10.0.0.5//data/node", p.Host, p.DataDir)
	}
}

// TestLaunchAndRecord_RecordsLaunchedPID: the pid the ledger holds is exactly
// the one the driver returned, and the fresh launch is refused a second time
// (a double launch) while a nil ledger launches without recording.
func TestLaunchAndRecord_RecordsLaunchedPID(t *testing.T) {
	d := &fakeDriver{}
	l, err := process.OpenLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	spec := specFor(0, "gstable")
	h, err := process.LaunchAndRecord(context.Background(), d, l, spec)
	if err != nil {
		t.Fatalf("LaunchAndRecord: %v", err)
	}
	got, ok := l.Get(string(node.LabelFor(0)))
	if !ok {
		t.Fatal("launched node not recorded")
	}
	if got.PID != h.PID {
		t.Errorf("recorded pid %d != launched pid %d", got.PID, h.PID)
	}

	// A second launch of the same label is a double launch: refused.
	if _, err := process.LaunchAndRecord(context.Background(), d, l, spec); err == nil {
		t.Error("second LaunchAndRecord of the same label = nil, want double-launch error")
	}

	// A nil ledger launches without recording.
	if _, err := process.LaunchAndRecord(context.Background(), d, nil, specFor(1, "gstable")); err != nil {
		t.Fatalf("LaunchAndRecord(nil ledger): %v", err)
	}
	if len(d.launched) != 3 {
		t.Errorf("driver launched %d times, want 3", len(d.launched))
	}
}

// TestLedgerSupersede_PreservesPriorAsRevision: a swap replaces the current
// entry, archives the prior pid/command in history, and bumps the revision —
// unlike Record it does not refuse the existing label.
func TestLedgerSupersede_PreservesPriorAsRevision(t *testing.T) {
	l, err := process.OpenLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	label := string(node.LabelFor(0))
	first := process.ProcFor(specFor(0, "gstable-old"), 100)
	if prev, ok, err := l.Supersede(first); err != nil || ok {
		t.Fatalf("first Supersede: prev=%v ok=%v err=%v, want !ok nil", prev, ok, err)
	}
	second := process.ProcFor(specFor(0, "gstable-new"), 200)
	prev, ok, err := l.Supersede(second)
	if err != nil {
		t.Fatalf("second Supersede: %v", err)
	}
	if !ok || prev.PID != 100 {
		t.Fatalf("prev = %+v ok=%v, want the pid-100 entry", prev, ok)
	}
	cur, _ := l.Get(label)
	if cur.PID != 200 {
		t.Errorf("current pid = %d, want 200", cur.PID)
	}
	if cur.Revision != 1 {
		t.Errorf("current revision = %d, want 1", cur.Revision)
	}
	hist := l.History(label)
	if len(hist) != 1 || hist[0].PID != 100 {
		t.Fatalf("history = %+v, want one entry with pid 100", hist)
	}
	if hist[0].Binary != "gstable-old" {
		t.Errorf("history binary = %q, want gstable-old", hist[0].Binary)
	}

	// History survives a reopen.
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
}

// TestLedgerSupersede_SurvivesReopen: the archived revision is persisted, so a
// later command still sees what ran before the swap.
func TestLedgerSupersede_SurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	l, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	label := string(node.LabelFor(0))
	if _, _, err := l.Supersede(process.ProcFor(specFor(0, "gstable-old"), 100)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := l.Supersede(process.ProcFor(specFor(0, "gstable-new"), 200)); err != nil {
		t.Fatal(err)
	}
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	reopened, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	hist := reopened.History(label)
	if len(hist) != 1 || hist[0].PID != 100 {
		t.Fatalf("reopened history = %+v, want one entry with pid 100", hist)
	}
	cur, _ := reopened.Get(label)
	if cur.PID != 200 || cur.Revision != 1 {
		t.Errorf("reopened current = pid %d rev %d, want pid 200 rev 1", cur.PID, cur.Revision)
	}
}
