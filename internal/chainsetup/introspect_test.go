package chainsetup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// fakeIntrospectDriver answers the two machine questions introspectRunning asks
// — which pids run a binary, and a pid's argv — from fixed tables.
type fakeIntrospectDriver struct {
	byBinary map[string][]int
	cmdlines map[int][]string
}

func (f *fakeIntrospectDriver) Provision(context.Context, process.NodeSpec) error { return nil }
func (f *fakeIntrospectDriver) Launch(context.Context, process.NodeSpec) (process.Handle, error) {
	return process.Handle{}, nil
}
func (f *fakeIntrospectDriver) Stop(context.Context, process.Handle) error { return nil }
func (f *fakeIntrospectDriver) PIDAlive(_ context.Context, pid int) (bool, error) {
	return len(f.cmdlines[pid]) > 0, nil
}
func (f *fakeIntrospectDriver) FindBinary(_ context.Context, name string) ([]int, error) {
	return f.byBinary[name], nil
}
func (f *fakeIntrospectDriver) Cmdline(_ context.Context, pid int) ([]string, error) {
	argv, ok := f.cmdlines[pid]
	if !ok {
		return nil, fmt.Errorf("no such pid %d", pid)
	}
	return argv, nil
}

// TestIntrospectRunning_RecoversConfigHashByLabel: a running node's config is
// read back from its argv and hashed on the machine, keyed by the datadir's
// label — the same hash the compose steps record.
func TestIntrospectRunning_RecoversConfigHashByLabel(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "node1.toml")
	const cfg = "[Node]\nDataDir = \"x\"\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(dir, "node1")

	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: dir}
	w.state.Binary = "gstable"
	w.state.Nodes = []node.Record{{Index: 1, Label: "node1", ConfigPath: cfgPath, DataDir: dataDir}}

	fake := &fakeIntrospectDriver{
		byBinary: map[string][]int{"gstable": {4242}},
		cmdlines: map[int][]string{4242: {"gstable", "--datadir", dataDir, "--config", cfgPath}},
	}
	w.SetDriver(func() (process.Driver, error) { return fake, nil })

	got, err := w.introspectRunning(context.Background(), "gstable")
	if err != nil {
		t.Fatalf("introspectRunning: %v", err)
	}
	r, ok := got["node1"]
	if !ok {
		t.Fatalf("no running node recovered for label node1: %v", got)
	}
	if r.PID != 4242 || r.Binary != "gstable" {
		t.Fatalf("recovered = %+v, want pid 4242 binary gstable", r)
	}
	if want := filestore.Hash([]byte(cfg)); r.ConfigHash != want {
		t.Fatalf("config hash = %q, want %q", r.ConfigHash, want)
	}
}

// TestIntrospectRunning_SkipsProcessesWithoutDatadirOrConfig: a process whose
// argv names no datadir or config is not a node this can place, so it is left
// out rather than mis-keyed.
func TestIntrospectRunning_SkipsProcessesWithoutDatadirOrConfig(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: dir}
	w.state.Binary = "gstable"
	w.state.Nodes = []node.Record{{Index: 1, Label: "node1", DataDir: filepath.Join(dir, "node1")}}

	fake := &fakeIntrospectDriver{
		byBinary: map[string][]int{"gstable": {7}},
		cmdlines: map[int][]string{7: {"gstable", "version"}}, // no --datadir/--config
	}
	w.SetDriver(func() (process.Driver, error) { return fake, nil })

	got, err := w.introspectRunning(context.Background(), "gstable")
	if err != nil {
		t.Fatalf("introspectRunning: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected nothing recovered, got %v", got)
	}
}
