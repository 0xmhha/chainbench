package chainsetup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// MON-017. Discovery searched one binary name — the composition's single one —
// while the comparison it feeds builds each node's candidate from binaryFor.
// A node table may give each node its own binary, and shipped specs do:
// go-wemix/handoff runs producers on gwemix and validators on gwbft, and the
// stablenet binary-swap spec pairs gstable with gstable-genesis-mismatch. pgrep
// is an exact-name match, so every node on the other binary was invisible.
//
// The tests below go through mergeRunning rather than stopping at discovery,
// because the defect is not "the map was short" but "the wrong pid was attached
// — or none was".

// binNames is the per-node binaries map shared by these tests: two distinct
// executables, in two directories, so neither the name nor the path collides.
var binNames = map[string]string{
	"alpha": "/data/bin/node-alpha",
	"beta":  "/data/bin/node-beta",
}

// twoBinaryWorkspace builds a workspace whose node1 runs node-alpha and node2
// runs node-beta, with both processes already up.
func twoBinaryWorkspace(t *testing.T) (*Workspace, map[int]string) {
	t.Helper()
	dir := t.TempDir()

	cfg1 := filepath.Join(dir, "node1.toml")
	cfg2 := filepath.Join(dir, "node2.toml")
	if err := os.WriteFile(cfg1, []byte("config one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg2, []byte("config two"), 0o600); err != nil {
		t.Fatal(err)
	}
	data1 := filepath.Join(dir, "node1")
	data2 := filepath.Join(dir, "node2")

	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: dir}
	// The composition's single binary is node-alpha; node2 names beta itself.
	w.state.Binary = binNames["alpha"]
	w.state.Binaries = binNames
	w.state.Nodes = []node.Record{
		{Index: 1, Label: "node1", ConfigPath: cfg1, DataDir: data1},
		{Index: 2, Label: "node2", Binary: "beta", ConfigPath: cfg2, DataDir: data2},
	}

	// A driver that answers per exact name, the way pgrep does.
	w.SetDriver(func() (process.Driver, error) {
		return &fakeIntrospectDriver{
			byBinary: map[string][]int{
				"node-alpha": {901},
				"node-beta":  {902},
			},
			cmdlines: map[int][]string{
				901: {binNames["alpha"], "--datadir", data1, "--config", cfg1},
				902: {binNames["beta"], "--datadir", data2, "--config", cfg2},
			},
		}, nil
	})
	return w, map[int]string{1: cfg1, 2: cfg2}
}

// TestMergeRunning_FindsNodesOnDifferentBinaries is the defect itself: both
// nodes are already up, on different executables, and both must be attached to
// their own pid.
func TestMergeRunning_FindsNodesOnDifferentBinaries(t *testing.T) {
	w, cfgs := twoBinaryWorkspace(t)

	hash := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return filestore.Hash(b)
	}
	after := []nodeTarget{
		{Index: 1, Label: "node1", ConfigHash: hash(cfgs[1]), Binary: binNames["alpha"]},
		{Index: 2, Label: "node2", ConfigHash: hash(cfgs[2]), Binary: binNames["beta"]},
	}

	before, alive, attach, refuse, err := w.mergeRunning(context.Background(), after,
		reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}})
	if err != nil {
		t.Fatalf("mergeRunning: %v", err)
	}
	if refuse != "" {
		t.Fatalf("both nodes match what this run would compose, so nothing should be refused: %s", refuse)
	}
	if attach[1] != 901 {
		t.Fatalf("node1 (node-alpha) attached pid %d, want 901", attach[1])
	}
	// The one the single-name search could never see.
	if attach[2] != 902 {
		t.Fatalf("node2 (node-beta) attached pid %d, want 902 — a node on a second binary was missed", attach[2])
	}
	if !alive[1] || !alive[2] {
		t.Fatalf("both recovered nodes must count as alive, got %v", alive)
	}
	if before[2].Binary != binNames["beta"] {
		t.Fatalf("node2 baseline binary = %q, want the beta path", before[2].Binary)
	}
}

// TestMergeRunning_SharedBinaryIsNotSearchedTwice guards the fix's own failure
// mode. pgrep returns the same pid for the same name, so searching a name once
// per node would hand the same process to the one-per-datadir rule twice and
// refuse a perfectly ordinary network as if two processes shared a datadir.
func TestMergeRunning_SharedBinaryIsNotSearchedTwice(t *testing.T) {
	dir := t.TempDir()
	cfgs := make([]string, 3)
	datas := make([]string, 3)
	nodes := make([]node.Record, 3)
	cmdlines := map[int][]string{}
	pids := []int{801, 802, 803}
	for i := range nodes {
		cfgs[i] = filepath.Join(dir, "n.toml")
		if i > 0 {
			cfgs[i] = filepath.Join(dir, "n"+string(rune('0'+i))+".toml")
		}
		if err := os.WriteFile(cfgs[i], []byte("same config"), 0o600); err != nil {
			t.Fatal(err)
		}
		datas[i] = filepath.Join(dir, "node", string(rune('1'+i)))
		nodes[i] = node.Record{Index: i + 1, Label: "node" + string(rune('1'+i)), ConfigPath: cfgs[i], DataDir: datas[i]}
		cmdlines[pids[i]] = []string{"/data/bin/shared", "--datadir", datas[i], "--config", cfgs[i]}
	}

	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: dir}
	w.state.Binary = "/data/bin/shared"
	w.state.Nodes = nodes
	w.SetDriver(func() (process.Driver, error) {
		return &fakeIntrospectDriver{
			byBinary: map[string][]int{"shared": pids},
			cmdlines: cmdlines,
		}, nil
	})

	b, _ := os.ReadFile(cfgs[0])
	after := make([]nodeTarget, 3)
	for i := range after {
		after[i] = nodeTarget{Index: i + 1, Label: nodes[i].Label, ConfigHash: filestore.Hash(b), Binary: "/data/bin/shared"}
	}
	_, _, attach, refuse, err := w.mergeRunning(context.Background(), after,
		reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}})
	if err != nil {
		t.Fatalf("three nodes on one binary must not error: %v", err)
	}
	if refuse != "" {
		t.Fatalf("three nodes on one binary must not be refused: %s", refuse)
	}
	for i, want := range pids {
		if attach[i+1] != want {
			t.Fatalf("node%d attached pid %d, want %d", i+1, attach[i+1], want)
		}
	}
}

// TestMergeRunning_SecondBinaryMismatchIsRefused: finding the node is not the
// same as accepting it. A node discovered on its own binary but running a
// different config than this run would give it is still foreign.
func TestMergeRunning_SecondBinaryMismatchIsRefused(t *testing.T) {
	w, cfgs := twoBinaryWorkspace(t)
	b1, err := os.ReadFile(cfgs[1])
	if err != nil {
		t.Fatal(err)
	}
	after := []nodeTarget{
		{Index: 1, Label: "node1", ConfigHash: filestore.Hash(b1), Binary: binNames["alpha"]},
		// node2 is up, and this run would give it a different config.
		{Index: 2, Label: "node2", ConfigHash: "sha256:something-else", Binary: binNames["beta"]},
	}
	_, _, _, refuse, err := w.mergeRunning(context.Background(), after,
		reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}})
	if err != nil {
		t.Fatalf("mergeRunning: %v", err)
	}
	if refuse == "" {
		t.Fatal("a node found on the second binary with a different config must be refused, not adopted")
	}
}

// TestIntrospectRunning_TwoProcessesOneDatadirStillRefusedAcrossBinaries: the
// one-per-datadir rule is about real collisions, and searching more names must
// not weaken it. Two genuinely different processes out of one datadir stay a
// refusal even when they run different executables.
func TestIntrospectRunning_TwoProcessesOneDatadirStillRefusedAcrossBinaries(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "node1.toml")
	if err := os.WriteFile(cfg, []byte("c"), 0o600); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(dir, "node1")

	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: dir}
	w.state.Binary = binNames["alpha"]
	w.state.Binaries = binNames
	w.state.Nodes = []node.Record{
		{Index: 1, Label: "node1", ConfigPath: cfg, DataDir: data},
		{Index: 2, Label: "node2", Binary: "beta", ConfigPath: cfg, DataDir: filepath.Join(dir, "node2")},
	}
	w.SetDriver(func() (process.Driver, error) {
		return &fakeIntrospectDriver{
			byBinary: map[string][]int{"node-alpha": {701}, "node-beta": {702}},
			cmdlines: map[int][]string{
				701: {binNames["alpha"], "--datadir", data, "--config", cfg},
				// A second, different process out of the same datadir.
				702: {binNames["beta"], "--datadir", data, "--config", cfg},
			},
		}, nil
	})

	_, err = w.introspectRunning(context.Background(), w.state.Binary)
	if err == nil {
		t.Fatal("two processes out of one datadir must still be refused")
	}
}
