package chainsetup

import (
	"context"
	"io/fs"
	"os"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// MON-010, the half past discovery. Keying the running-node map by server was
// verified; what happens to that map afterwards was not. This drives the whole
// chain — mergeRunning to reconcileReuse to recordLaunch to the run ledger — and
// then reopens the workspace, because a pid attached to the wrong node is a
// wrong answer whether it happens at the lookup or at the write.
//
// The shape is the one that has nothing else to go on: two servers, the same
// datadir string, the same config PATH, the same binary. Only the file each
// server's store returns for that path differs, so a read from the wrong store
// produces the wrong hash, and only the server in the key separates the two
// nodes at all.

// serverStore is one machine's files: a fixed body for any path asked of it, so
// two servers can disagree about the contents of the same path.
type serverStore struct{ body []byte }

func (s serverStore) Exists(context.Context, string) (bool, error) { return true, nil }
func (s serverStore) Read(context.Context, string) ([]byte, error) { return s.body, nil }
func (s serverStore) Remove(context.Context, string) error         { return nil }

func (s serverStore) Write(context.Context, string, []byte, fs.FileMode) error {
	return nil
}
func (s serverStore) Checksum(_ context.Context, _ string) (string, error) {
	return filestore.Hash(s.body), nil
}

const (
	sharedDataDir = "/data/chainbench/node1"
	sharedCfgPath = "/data/chainbench/configs/node1.toml"
	sharedBinary  = "/data/chainbench/bin/gstable"
)

// twoServerWorkspace places node1 on server6 and node2 on server7, both running
// out of sharedDataDir with sharedCfgPath and the same binary, and seeds a
// machine per server whose store answers with its own body.
func twoServerWorkspace(t *testing.T, dir, body6, body7 string, pid6, pid7 int) *Workspace {
	t.Helper()
	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: "/data/chainbench"}
	w.state.Binary = sharedBinary
	w.state.Nodes = []node.Record{
		{Index: 1, Label: "node1", Server: "server6", Host: "10.0.0.6", ConfigPath: sharedCfgPath, DataDir: sharedDataDir},
		{Index: 2, Label: "node2", Server: "server7", Host: "10.0.0.7", ConfigPath: sharedCfgPath, DataDir: sharedDataDir},
	}
	argv := []string{sharedBinary, "--datadir", sharedDataDir, "--config", sharedCfgPath}
	w.machines = map[string]*resource.Access{
		"server6": {
			Spec: resource.Spec{Server: "server6"}, DataRoot: "/data/chainbench",
			Files: serverStore{body: []byte(body6)},
			Driver: &fakeIntrospectDriver{
				byBinary: map[string][]int{"gstable": {pid6}},
				cmdlines: map[int][]string{pid6: argv},
			},
		},
		"server7": {
			Spec: resource.Spec{Server: "server7"}, DataRoot: "/data/chainbench",
			Files: serverStore{body: []byte(body7)},
			Driver: &fakeIntrospectDriver{
				byBinary: map[string][]int{"gstable": {pid7}},
				cmdlines: map[int][]string{pid7: argv},
			},
		},
	}
	return w
}

// TestReconcileReuse_AttachesAndRecordsEachServersOwnPID walks the full path and
// then reopens the workspace.
func TestReconcileReuse_AttachesAndRecordsEachServersOwnPID(t *testing.T) {
	dir := t.TempDir()
	const body6, body7 = "server six config", "server seven config"
	const pid6, pid7 = 606, 707
	w := twoServerWorkspace(t, dir, body6, body7, pid6, pid7)

	// What this run would compose: each node's candidate matches the file its
	// own server holds, so both are reusable in place.
	// The candidate map is keyed by config path, and both nodes share one — so
	// the per-node comparison has to come from the running side. Give the
	// candidate the hash node1's server holds and check node2 is judged on its
	// own server's file rather than on this one.
	cands := candidateInputs{
		Genesis: "",
		Configs: map[string]string{sharedCfgPath: filestore.Hash([]byte(body6))},
	}

	snap := reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}}
	plan, err := w.reconcileReuse(context.Background(), snap, cands)
	if err != nil {
		t.Fatalf("reconcileReuse: %v", err)
	}
	// node2's server holds a different file than the candidate, so it is foreign
	// and the whole reuse is refused. That refusal is itself the proof that the
	// two servers were read separately: with one store, both would have matched.
	if plan.Refuse == "" {
		t.Fatal("node2's server holds a different config, so the reuse must be refused — both servers were read from one store")
	}

	// Now make both servers agree with the candidate, and the attach must land
	// each server's own pid on its own node.
	w2 := twoServerWorkspace(t, t.TempDir(), body6, body6, pid6, pid7)
	same := candidateInputs{Configs: map[string]string{sharedCfgPath: filestore.Hash([]byte(body6))}}
	plan, err = w2.reconcileReuse(context.Background(), reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}}, same)
	if err != nil {
		t.Fatalf("reconcileReuse: %v", err)
	}
	if plan.Refuse != "" {
		t.Fatalf("both servers match the candidate, so nothing should be refused: %s", plan.Refuse)
	}
	for _, d := range plan.Nodes {
		if !d.Reuse {
			t.Fatalf("node%d was not reused: %s", d.Index, d.Reason)
		}
	}
	if got := w2.state.Nodes[0].PID; got != pid6 {
		t.Fatalf("node1 (server6) recorded pid %d, want %d", got, pid6)
	}
	if got := w2.state.Nodes[1].PID; got != pid7 {
		t.Fatalf("node2 (server7) recorded pid %d, want %d — the servers' pids were swapped", got, pid7)
	}

	// The ledger has to carry the same answer, with each node's own host and
	// datadir, and survive a reopen: Open reattaches pids from it.
	if err := w2.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	reopened, err := Open(w2.Dir(), nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if len(reopened.state.Nodes) != 2 {
		t.Fatalf("reopened node table has %d nodes", len(reopened.state.Nodes))
	}
	for i, want := range []struct {
		pid  int
		host string
	}{{pid6, "10.0.0.6"}, {pid7, "10.0.0.7"}} {
		ns := reopened.state.Nodes[i]
		if ns.PID != want.pid {
			t.Fatalf("after reopen node%d has pid %d, want %d", i+1, ns.PID, want.pid)
		}
		if ns.Host != want.host {
			t.Fatalf("after reopen node%d has host %q, want %q", i+1, ns.Host, want.host)
		}
		if ns.DataDir != sharedDataDir || ns.ConfigPath != sharedCfgPath {
			t.Fatalf("after reopen node%d lost its paths: datadir=%q config=%q", i+1, ns.DataDir, ns.ConfigPath)
		}
	}
}

// TestReconcileReuse_IdenticalInputsStillKeepServersApart is the case with the
// least to go on: same binary, same datadir, same config path AND the same
// bytes. Nothing distinguishes the two nodes except which server each process
// was found on, and the pids must not cross.
func TestReconcileReuse_IdenticalInputsStillKeepServersApart(t *testing.T) {
	const body = "byte for byte identical"
	const pid6, pid7 = 616, 717
	w := twoServerWorkspace(t, t.TempDir(), body, body, pid6, pid7)

	cands := candidateInputs{Configs: map[string]string{sharedCfgPath: filestore.Hash([]byte(body))}}
	plan, err := w.reconcileReuse(context.Background(), reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}}, cands)
	if err != nil {
		t.Fatalf("reconcileReuse: %v", err)
	}
	if plan.Refuse != "" {
		t.Fatalf("identical inputs on both servers must reuse, not refuse: %s", plan.Refuse)
	}
	if w.state.Nodes[0].PID != pid6 || w.state.Nodes[1].PID != pid7 {
		t.Fatalf("pids crossed: node1=%d node2=%d, want %d and %d",
			w.state.Nodes[0].PID, w.state.Nodes[1].PID, pid6, pid7)
	}
}
