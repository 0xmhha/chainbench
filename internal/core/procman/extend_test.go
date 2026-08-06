package procman_test

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/procman"
)

func TestTrackProc_DataDirsUnique(t *testing.T) {
	m := procman.New()
	m.TrackProc(procman.Proc{PID: 1001, Label: "node1", DataDir: "/data/a"})
	m.TrackProc(procman.Proc{PID: 1002, Label: "node2", DataDir: "/data/b"})
	m.TrackProc(procman.Proc{PID: 1003, Label: "node3", DataDir: "/data/a"}) // dup dir
	m.TrackProc(procman.Proc{PID: 1004, Label: "node4"})                     // no dir

	got := m.DataDirs()
	sort.Strings(got)
	want := []string{"/data/a", "/data/b"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("DataDirs = %v, want %v", got, want)
	}
}

func TestRemoveDataDirs(t *testing.T) {
	base := t.TempDir()
	dirA := filepath.Join(base, "a")
	dirB := filepath.Join(base, "b")
	for _, d := range []string{dirA, dirB} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m := procman.New()
	m.TrackProc(procman.Proc{PID: 2001, DataDir: dirA})
	m.TrackProc(procman.Proc{PID: 2002, DataDir: dirB})

	if errs := m.RemoveDataDirs(); len(errs) != 0 {
		t.Fatalf("RemoveDataDirs errors: %v", errs)
	}
	for _, d := range []string{dirA, dirB} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Fatalf("datadir %s still exists (err=%v)", d, err)
		}
	}
}

func TestStopRemote_RemoteOnly(t *testing.T) {
	m := procman.New()
	m.TrackProc(procman.Proc{PID: 3001, Label: "local"})                // local
	m.TrackProc(procman.Proc{PID: 3002, Label: "r1", Host: "10.0.0.1"}) // remote
	m.TrackProc(procman.Proc{PID: 3003, Label: "r2", Host: "10.0.0.2"}) // remote

	var killed []string
	errs := m.StopRemote(func(host string, pid int) error {
		killed = append(killed, host+":"+strconv.Itoa(pid))
		return nil
	})
	if len(errs) != 0 {
		t.Fatalf("StopRemote errors: %v", errs)
	}
	sort.Strings(killed)
	want := []string{"10.0.0.1:3002", "10.0.0.2:3003"}
	if len(killed) != 2 || killed[0] != want[0] || killed[1] != want[1] {
		t.Fatalf("killed = %v, want %v (local must be skipped)", killed, want)
	}
}

func TestDedup_HostPidComposite(t *testing.T) {
	m := procman.New()
	m.TrackProc(procman.Proc{PID: 4000, Label: "local"})                  // local pid 4000
	m.TrackProc(procman.Proc{PID: 4000, Label: "remote", Host: "h1"})     // remote pid 4000 (distinct)
	m.TrackProc(procman.Proc{PID: 4000, Label: "dup-remote", Host: "h1"}) // dup of remote
	if m.Count() != 2 {
		t.Fatalf("Count = %d, want 2 (local+remote pid 4000 distinct, dup collapsed)", m.Count())
	}
}
