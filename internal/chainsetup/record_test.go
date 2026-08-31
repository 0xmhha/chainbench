package chainsetup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// TestRecordRun_WritesTheFactsAndNeverASecret pins the record's two
// contracts: everything a debugger asks first (chain, target, node table,
// launch commands, genesis) is in the folder, and nothing that logs in is —
// the server set's ssh section stays outside by construction, checked here
// with a canary password in the referenced file.
func TestRecordRun_WritesTheFactsAndNeverASecret(t *testing.T) {
	dir := t.TempDir()
	setPath := filepath.Join(dir, "server-set.yaml")
	const canary = "CANARY-NEVER-IN-RECORD"
	if err := os.WriteFile(setPath, []byte(
		"version: 2\npool:\n  hosts: [{name: box1, addr: 192.0.2.11}]\n"+
			"ssh: {user: dev, password: "+canary+"}\ndataRoot: /data/cb\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	clock := func() time.Time { return time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC) }
	w, err := Open(filepath.Join(dir, "ws"), clock)
	if err != nil {
		t.Fatal(err)
	}
	w.state.Chain = "stablenet"
	w.state.ServerSet = setPath
	w.state.Target = machine.Spec{Server: "box1", Host: "192.0.2.11", DataRoot: "/data/cb"}
	w.state.Nodes = []node.Record{{Index: 1, Role: "bp", Host: "192.0.2.11", PID: 4242}}
	if err := w.ledger.Record(process.Proc{
		PID: 4242, Label: "node1", Binary: "gstable",
		Command: "gstable --config config_node1.toml", Host: "192.0.2.11",
	}); err != nil {
		t.Fatal(err)
	}

	// The target is faked local so the genesis read-back has something to read.
	tdir := t.TempDir()
	files := filestore.Local{}
	if err := files.Write(context.Background(), filepath.Join(tdir, "genesis.json"), []byte(`{"config":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	w.state.Target.DataRoot = tdir
	acc := &machine.Access{DataRoot: tdir, Files: files}

	rec, err := w.recordRun(context.Background(), acc, "/data/cb/bin/gstable")
	if err != nil {
		t.Fatalf("recordRun: %v", err)
	}

	wantFiles := []string{"manifest.json", "launch-commands.txt", "genesis.json"}
	var all strings.Builder
	for _, f := range wantFiles {
		b, err := os.ReadFile(filepath.Join(rec, f))
		if err != nil {
			t.Fatalf("record is missing %s: %v", f, err)
		}
		all.Write(b)
	}
	got := all.String()
	for _, want := range []string{"stablenet", "node1", "gstable --config", `"server": "box1"`} {
		if !strings.Contains(got, want) {
			t.Errorf("record lost %q", want)
		}
	}
	// The canary is the actual secret value. The word "password" itself may
	// appear legitimately — a launch argv names the keystore password FILE by
	// path — so the gate is on values, not words.
	if strings.Contains(got, canary) {
		t.Fatalf("the record carries the server set's password value")
	}
	if !strings.Contains(rec, "20260825-120000") {
		t.Errorf("run folder not stamped by the injected clock: %s", rec)
	}
}
