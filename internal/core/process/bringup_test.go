package process_test

import (
	"context"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/process"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func nodeSet(pids ...int) node.NodeSet {
	ns := node.NodeSet{Chain: "wbft"}
	for i, p := range pids {
		ns.Nodes = append(ns.Nodes, node.Node{Index: i + 1, Role: node.RoleValidator, PID: p})
	}
	return ns
}

func TestBringUp_Success(t *testing.T) {
	pm := process.New()
	launched := nodeSet(4001, 4002)
	deps := process.Deps{
		Launch: func(_ context.Context, _ process.Plan, _ []int) (process.Result, error) {
			return process.Result{
				Nodes: launched,
				Procs: []process.Proc{{PID: 4001, DataDir: "/d/1"}, {PID: 4002, DataDir: "/d/2"}},
			}, nil
		},
		HealthGate: func(_ context.Context, _ node.NodeSet) (process.Diagnosis, error) {
			return process.Diagnosis{OK: true}, nil
		},
		Procman: pm,
		Sleep:   func(time.Duration) {},
	}
	s := process.NewLauncher(deps)

	ns, diag, err := s.BringUp(context.Background(), process.Plan{}, process.Options{MaxAttempts: 3})
	if err != nil {
		t.Fatalf("BringUp: %v", err)
	}
	if !diag.OK || len(ns.Nodes) != 2 {
		t.Fatalf("diag=%+v nodes=%d", diag, len(ns.Nodes))
	}
	if pm.Count() != 2 {
		t.Fatalf("process ledger tracked %d, want 2", pm.Count())
	}
}

func TestBringUp_RetryThenFail(t *testing.T) {
	pm := process.New()
	attempts := 0
	deps := process.Deps{
		Launch: func(_ context.Context, _ process.Plan, _ []int) (process.Result, error) {
			attempts++
			return process.Result{Nodes: nodeSet(5001), Procs: []process.Proc{{PID: 5001, DataDir: t.TempDir()}}}, nil
		},
		HealthGate: func(_ context.Context, _ node.NodeSet) (process.Diagnosis, error) {
			return process.Diagnosis{OK: false, Mode: process.ForkNotCrossed, Detail: "fork not reached"}, nil
		},
		Procman: pm,
		Sleep:   func(time.Duration) {},
	}
	s := process.NewLauncher(deps)

	_, diag, err := s.BringUp(context.Background(), process.Plan{}, process.Options{MaxAttempts: 3})
	if err == nil {
		t.Fatal("expected failure after retries")
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if diag.Mode != process.ForkNotCrossed {
		t.Fatalf("diag.Mode = %v, want process.ForkNotCrossed", diag.Mode)
	}
}

func TestBringUp_LaunchError(t *testing.T) {
	s := process.NewLauncher(process.Deps{
		Launch: func(_ context.Context, _ process.Plan, _ []int) (process.Result, error) {
			return process.Result{}, errors.New("exec failed")
		},
		HealthGate: func(_ context.Context, _ node.NodeSet) (process.Diagnosis, error) {
			return process.Diagnosis{OK: true}, nil
		},
		Procman: process.New(),
		Sleep:   func(time.Duration) {},
	})
	if _, _, err := s.BringUp(context.Background(), process.Plan{}, process.Options{MaxAttempts: 1}); err == nil {
		t.Fatal("launch error must fail bring-up")
	}
}

func TestTeardown_StopsAndRemovesDataDir(t *testing.T) {
	// A real backgrounded process to stop, plus a datadir to remove.
	out, err := exec.Command("sh", "-c", "sleep 300 >/dev/null 2>&1 & echo $!").Output()
	if err != nil {
		t.Fatalf("start sleeper: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parse pid: %v", err)
	}
	dataDir := t.TempDir()
	sub := filepath.Join(dataDir, "chaindata")
	_ = os.MkdirAll(sub, 0o755)

	pm := process.New()
	pm.TrackProc(process.Proc{PID: pid, DataDir: dataDir})
	s := process.NewLauncher(process.Deps{Procman: pm, Sleep: func(time.Duration) {}})

	if err := s.Teardown(context.Background(), nodeSet(pid), process.TeardownOpts{RemoveDataDir: true, Grace: time.Second}); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if process.Alive(pid) {
		t.Fatal("process still alive after Teardown")
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatalf("datadir not removed (err=%v)", err)
	}
}
