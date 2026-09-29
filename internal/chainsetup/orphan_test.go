package chainsetup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// stoppingDriver is fakeIntrospectDriver that remembers what it was asked to
// stop, which is the whole verdict these tests want.
type stoppingDriver struct {
	fakeIntrospectDriver
	stopped []int
}

func (d *stoppingDriver) Stop(_ context.Context, h process.Handle) error {
	d.stopped = append(d.stopped, h.PID)
	return nil
}

// orphanWorkspace is a placed two-node composition with no pids recorded — a
// run killed between launching and saving. dataDir(i) is where node i would
// have been launched from.
func orphanWorkspace(t *testing.T) (*Workspace, func(int) string) {
	t.Helper()
	dir := t.TempDir()
	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: dir}
	w.state.Binary = "gstable"
	at := func(i int) string { return filepath.Join(dir, "node"+string(rune('0'+i))) }
	w.state.Nodes = []node.Record{
		{Index: 1, Label: "node1", DataDir: at(1)},
		{Index: 2, Label: "node2", DataDir: at(2)},
	}
	return w, at
}

// TestStopUnrecorded_StopsWhatTheRecordCannotName.
//
// The case this exists for: a run is killed between launching a node and
// saving its pid. The record then says nothing is running while the node is,
// `chain stop` reports "0 node(s) stopped", and the next composition on the
// machine is refused for ports held by a network nobody can name.
//
// The pid is lost, but --datadir is not: it is this workspace's.
func TestStopUnrecorded_StopsWhatTheRecordCannotName(t *testing.T) {
	w, at := orphanWorkspace(t)
	d := &stoppingDriver{fakeIntrospectDriver: fakeIntrospectDriver{
		byBinary: map[string][]int{"gstable": {111, 222}},
		cmdlines: map[int][]string{
			111: {"gstable", "--datadir", at(1)},
			222: {"gstable", "--datadir", at(2)},
		},
	}}
	w.SetDriver(func() (process.Driver, error) { return d, nil })

	n, err := w.StopUnrecorded(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("stopped %d, want both nodes", n)
	}
	if len(d.stopped) != 2 {
		t.Errorf("the driver was asked to stop %v, want two pids", d.stopped)
	}
}

// TestStopUnrecorded_LeavesAnotherCompositionAlone.
//
// The search is by binary name, so it turns up every node of that build on the
// machine — including ones belonging to a composition that is running quite
// happily. Stopping those would make "stop my network" mean "stop everyone's".
// The datadir is what separates them.
func TestStopUnrecorded_LeavesAnotherCompositionAlone(t *testing.T) {
	w, at := orphanWorkspace(t)
	d := &stoppingDriver{fakeIntrospectDriver: fakeIntrospectDriver{
		byBinary: map[string][]int{"gstable": {111, 999}},
		cmdlines: map[int][]string{
			111: {"gstable", "--datadir", at(1)},
			999: {"gstable", "--datadir", "/somewhere/else/node1"},
		},
	}}
	w.SetDriver(func() (process.Driver, error) { return d, nil })

	if _, err := w.StopUnrecorded(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(d.stopped) != 1 || d.stopped[0] != 111 {
		t.Errorf("stopped %v, want only this workspace's node", d.stopped)
	}
}

// TestStopUnrecorded_LeavesARecordedPidToStopNodes: a pid the record holds is
// stopped by the path that knows which node it is. Signalling it here too
// would pay the shutdown grace twice for one node.
func TestStopUnrecorded_LeavesARecordedPidToStopNodes(t *testing.T) {
	w, at := orphanWorkspace(t)
	w.state.Nodes[0].PID = 111
	d := &stoppingDriver{fakeIntrospectDriver: fakeIntrospectDriver{
		byBinary: map[string][]int{"gstable": {111, 222}},
		cmdlines: map[int][]string{
			111: {"gstable", "--datadir", at(1)},
			222: {"gstable", "--datadir", at(2)},
		},
	}}
	w.SetDriver(func() (process.Driver, error) { return d, nil })

	if _, err := w.StopUnrecorded(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(d.stopped) != 1 || d.stopped[0] != 222 {
		t.Errorf("stopped %v, want only the unrecorded node", d.stopped)
	}
}

// TestStopUnrecorded_NothingRunningIsNotAFailure keeps the ordinary path quiet:
// Stop calls this every time, and a composition that is already down must not
// start reporting errors for it.
func TestStopUnrecorded_NothingRunningIsNotAFailure(t *testing.T) {
	w, _ := orphanWorkspace(t)
	d := &stoppingDriver{fakeIntrospectDriver: fakeIntrospectDriver{
		byBinary: map[string][]int{"gstable": nil},
	}}
	w.SetDriver(func() (process.Driver, error) { return d, nil })

	n, err := w.StopUnrecorded(context.Background())
	if err != nil || n != 0 {
		t.Errorf("stopped %d, err %v; want 0 and no error", n, err)
	}
}
