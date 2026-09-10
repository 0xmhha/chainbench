package chainsetup

import (
	"context"
	"github.com/0xmhha/chainbench/internal/core/node"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
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
			"ssh: {user: dev, password: "+canary+"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	clock := func() time.Time { return time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC) }
	w, err := Open(filepath.Join(dir, "ws"), clock)
	if err != nil {
		t.Fatal(err)
	}
	w.state.Chain = "stablenet"
	w.state.ServerSet = setPath
	w.state.Target = resource.Spec{Server: "box1", Host: "192.0.2.11", DataRoot: "/data/cb"}
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
	// The genesis is recorded by the path the composition actually wrote, which
	// is what the record reads back.
	w.state.GenesisPath = filepath.Join(tdir, "genesis.json")
	acc := &resource.Access{DataRoot: tdir, Files: files}

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

// TestRecordRun_CollectsTheGenesisTheCompositionActuallyUsed is MON-008.
//
// The record used to build a fresh Layout from the data root alone, which
// resolves to the flat <dataRoot>/genesis.json. A composition with a
// workspace-config writes its genesis under the composition's runtime directory
// instead, so the record either held no genesis at all or — with a leftover from
// another composition on the same data root — copied that one and presented it
// as this run's. Both files exist here, and only the real one may be collected.
func TestRecordRun_CollectsTheGenesisTheCompositionActuallyUsed(t *testing.T) {
	dir := t.TempDir()
	tdir := t.TempDir()
	files := filestore.Local{}
	ctx := context.Background()

	const stale = `{"note":"a different composition's leftover"}`
	const real = `{"note":"this run's genesis"}`
	if err := files.Write(ctx, filepath.Join(tdir, "genesis.json"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	realPath := filepath.Join(tdir, "runtime", "abcdef123456", "genesis.json")
	if err := files.Write(ctx, realPath, []byte(real), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(tdir, "runtime", "abcdef123456", "configs", "node1.toml")
	if err := files.Write(ctx, cfgPath, []byte("[Node]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	clock := func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) }
	w, err := Open(filepath.Join(dir, "ws"), clock)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Chain = "stablenet"
	w.state.Target = resource.Spec{DataRoot: tdir}
	w.state.GenesisPath = realPath
	w.state.Nodes = []node.Record{{Index: 1, Role: "bp", ConfigPath: cfgPath, PID: 11}}

	rec, err := w.recordRun(ctx, &resource.Access{DataRoot: tdir, Files: files}, "gstable")
	if err != nil {
		t.Fatalf("recordRun: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(rec, "genesis.json"))
	if err != nil {
		t.Fatalf("record is missing the genesis: %v", err)
	}
	if string(got) == stale {
		t.Fatal("the record collected another composition's genesis")
	}
	if string(got) != real {
		t.Fatalf("record genesis = %s, want this run's", got)
	}
	// The node configs answer "what was this node started with", so they belong
	// in the record too.
	if b, err := os.ReadFile(filepath.Join(rec, "config_node1.toml")); err != nil || string(b) != "[Node]\n" {
		t.Fatalf("record is missing node1's config: %q err=%v", b, err)
	}
	// Nothing was missing, so no note.
	if _, err := os.Stat(filepath.Join(rec, "missing-inputs.txt")); err == nil {
		t.Fatal("a complete record must not claim inputs were missing")
	}
}

// TestRecordRun_SaysWhenAnInputCouldNotBeCollected: a record that quietly lacks
// the genesis looks complete and is not.
func TestRecordRun_SaysWhenAnInputCouldNotBeCollected(t *testing.T) {
	dir := t.TempDir()
	tdir := t.TempDir()
	clock := func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) }
	w, err := Open(filepath.Join(dir, "ws"), clock)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Chain = "stablenet"
	w.state.Target = resource.Spec{DataRoot: tdir}
	w.state.GenesisPath = filepath.Join(tdir, "runtime", "nope", "genesis.json") // never written
	w.state.Nodes = []node.Record{{Index: 1, Role: "bp", PID: 11}}

	rec, err := w.recordRun(context.Background(), &resource.Access{DataRoot: tdir, Files: filestore.Local{}}, "gstable")
	if err != nil {
		t.Fatalf("recordRun: %v", err)
	}
	note, err := os.ReadFile(filepath.Join(rec, "missing-inputs.txt"))
	if err != nil {
		t.Fatalf("a record that could not collect the genesis must say so: %v", err)
	}
	if !strings.Contains(string(note), "genesis") {
		t.Fatalf("the note should name the genesis: %s", note)
	}
}
