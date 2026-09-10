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
	r, ok := got[nodeAddr{DataDir: dataDir}]
	if !ok {
		t.Fatalf("no running node recovered for %s: %v", dataDir, got)
	}
	if r.PID != 4242 || r.Binary != "gstable" {
		t.Fatalf("recovered = %+v, want pid 4242 binary gstable", r)
	}
	if want := filestore.Hash([]byte(cfg)); r.ConfigHash != want {
		t.Fatalf("config hash = %q, want %q", r.ConfigHash, want)
	}
}

// TestMergeRunning_RefusesForeignMismatch: a node already up on the target with
// a config different from the one this run would give it is a foreign process,
// so the reuse is refused rather than composed over.
func TestMergeRunning_RefusesForeignMismatch(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "node1.toml")
	if err := os.WriteFile(cfgPath, []byte("running config"), 0o600); err != nil {
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

	// The run would compose a different config for node1 (a hash that is not the
	// running file's), so the running node is foreign.
	after := []nodeTarget{{Index: 1, Label: "node1", ConfigHash: "sha256:different", Binary: "gstable"}}
	_, _, _, refuse, err := w.mergeRunning(context.Background(), after, reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}})
	if err != nil {
		t.Fatalf("mergeRunning: %v", err)
	}
	if refuse == "" {
		t.Fatal("expected a foreign-mismatch refusal")
	}
}

// TestMergeRunning_ReusesMatchingInPlace: a node up with the config this run
// would give it is folded into the baseline and queued to be attached.
func TestMergeRunning_ReusesMatchingInPlace(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "node1.toml")
	const cfg = "matching config"
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

	after := []nodeTarget{{Index: 1, Label: "node1", ConfigHash: filestore.Hash([]byte(cfg)), Binary: "gstable"}}
	before, alive, attach, refuse, err := w.mergeRunning(context.Background(), after, reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}})
	if err != nil || refuse != "" {
		t.Fatalf("mergeRunning: refuse=%q err=%v", refuse, err)
	}
	if _, ok := before[1]; !ok {
		t.Fatal("matching running node was not folded into the baseline")
	}
	if !alive[1] {
		t.Fatal("matching running node not marked alive")
	}
	if attach[1] != 4242 {
		t.Fatalf("attach pid = %d, want 4242", attach[1])
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

// TestIntrospectRunning_TwoCompositionsOnOneServerDoNotCollide is MON-010.
//
// Every composition names its nodes node1..nodeN, and the datadir's last element
// is that label. Keying discovery by the label therefore merged the node1 of one
// composition with the node1 of another sharing the data root — the composition
// id in the path is exactly what tells them apart. Whichever process was listed
// last won, so a foreign pid could be adopted and later stopped as ours.
func TestIntrospectRunning_TwoCompositionsOnOneServerDoNotCollide(t *testing.T) {
	dir := t.TempDir()
	// Two compositions, same server, same node label, different datadirs.
	mine := filepath.Join(dir, "node", "aaaaaaaaaaaa", "node1")
	theirs := filepath.Join(dir, "node", "bbbbbbbbbbbb", "node1")
	myCfg := filepath.Join(dir, "mine.toml")
	theirCfg := filepath.Join(dir, "theirs.toml")
	if err := os.WriteFile(myCfg, []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(theirCfg, []byte("theirs"), 0o600); err != nil {
		t.Fatal(err)
	}

	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: dir}
	w.state.Binary = "gstable"
	w.state.Nodes = []node.Record{{Index: 1, Label: "node1", ConfigPath: myCfg, DataDir: mine}}

	fake := &fakeIntrospectDriver{
		byBinary: map[string][]int{"gstable": {100, 200}},
		cmdlines: map[int][]string{
			100: {"gstable", "--datadir", mine, "--config", myCfg},
			200: {"gstable", "--datadir", theirs, "--config", theirCfg},
		},
	}
	w.SetDriver(func() (process.Driver, error) { return fake, nil })

	got, err := w.introspectRunning(context.Background(), "gstable")
	if err != nil {
		t.Fatalf("introspectRunning: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("both processes must be kept apart, got %d entries", len(got))
	}
	if r := got[nodeAddr{DataDir: mine}]; r.PID != 100 {
		t.Fatalf("this composition's node1 = pid %d, want 100", r.PID)
	}
	if r := got[nodeAddr{DataDir: theirs}]; r.PID != 200 {
		t.Fatalf("the other composition's node1 = pid %d, want 200", r.PID)
	}

	// And the merge must pick THIS composition's process, whatever the listing
	// order was: matched by datadir, not by the shared label.
	after := []nodeTarget{{Index: 1, Label: "node1", ConfigHash: filestore.Hash([]byte("mine")), Binary: "gstable"}}
	_, _, attach, refuse, err := w.mergeRunning(context.Background(), after,
		reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}})
	if err != nil || refuse != "" {
		t.Fatalf("mergeRunning: refuse=%q err=%v", refuse, err)
	}
	if attach[1] != 100 {
		t.Fatalf("attached pid %d — the other composition's node was adopted", attach[1])
	}
}

// TestIntrospectRunning_TwoProcessesOneDatadirIsRefused: picking a winner would
// bind this composition to whichever pid the listing happened to return first.
func TestIntrospectRunning_TwoProcessesOneDatadirIsRefused(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "node1")
	cfg := filepath.Join(dir, "n.toml")
	if err := os.WriteFile(cfg, []byte("c"), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: dir}
	w.state.Binary = "gstable"
	w.state.Nodes = []node.Record{{Index: 1, Label: "node1", ConfigPath: cfg, DataDir: dataDir}}
	fake := &fakeIntrospectDriver{
		byBinary: map[string][]int{"gstable": {7, 8}},
		cmdlines: map[int][]string{
			7: {"gstable", "--datadir", dataDir, "--config", cfg},
			8: {"gstable", "--datadir", dataDir, "--config", cfg},
		},
	}
	w.SetDriver(func() (process.Driver, error) { return fake, nil })

	if _, err := w.introspectRunning(context.Background(), "gstable"); err == nil {
		t.Fatal("two processes out of one datadir must be refused, not silently resolved")
	}
}
