package supervisor_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/procman"
	"github.com/0xmhha/chainbench/internal/core/supervisor"
)

func nodeSet(pids ...int) node.NodeSet {
	ns := node.NodeSet{Chain: "wbft"}
	for i, p := range pids {
		ns.Nodes = append(ns.Nodes, node.Node{Index: i + 1, Role: node.RoleValidator, PID: p})
	}
	return ns
}

func TestBringUp_Success(t *testing.T) {
	pm := procman.New()
	launched := nodeSet(4001, 4002)
	deps := supervisor.Deps{
		Launch: func(_ context.Context, _ driver.Plan, _ []int) (supervisor.LaunchResult, error) {
			return supervisor.LaunchResult{
				Nodes: launched,
				Procs: []procman.Proc{{PID: 4001, DataDir: "/d/1"}, {PID: 4002, DataDir: "/d/2"}},
			}, nil
		},
		HealthGate: func(_ context.Context, _ node.NodeSet) (supervisor.Diagnosis, error) {
			return supervisor.Diagnosis{OK: true}, nil
		},
		Procman: pm,
		Sleep:   func(time.Duration) {},
	}
	s := supervisor.New(deps)

	ns, diag, err := s.BringUp(context.Background(), driver.Plan{}, supervisor.Options{MaxAttempts: 3})
	if err != nil {
		t.Fatalf("BringUp: %v", err)
	}
	if !diag.OK || len(ns.Nodes) != 2 {
		t.Fatalf("diag=%+v nodes=%d", diag, len(ns.Nodes))
	}
	if pm.Count() != 2 {
		t.Fatalf("procman tracked %d, want 2", pm.Count())
	}
}

func TestBringUp_RetryThenFail(t *testing.T) {
	pm := procman.New()
	attempts := 0
	deps := supervisor.Deps{
		Launch: func(_ context.Context, _ driver.Plan, _ []int) (supervisor.LaunchResult, error) {
			attempts++
			return supervisor.LaunchResult{Nodes: nodeSet(5001), Procs: []procman.Proc{{PID: 5001, DataDir: t.TempDir()}}}, nil
		},
		HealthGate: func(_ context.Context, _ node.NodeSet) (supervisor.Diagnosis, error) {
			return supervisor.Diagnosis{OK: false, Mode: supervisor.ForkNotCrossed, Detail: "fork not reached"}, nil
		},
		Procman: pm,
		Sleep:   func(time.Duration) {},
	}
	s := supervisor.New(deps)

	_, diag, err := s.BringUp(context.Background(), driver.Plan{}, supervisor.Options{MaxAttempts: 3})
	if err == nil {
		t.Fatal("expected failure after retries")
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if diag.Mode != supervisor.ForkNotCrossed {
		t.Fatalf("diag.Mode = %v, want ForkNotCrossed", diag.Mode)
	}
}

func TestBringUp_LaunchError(t *testing.T) {
	s := supervisor.New(supervisor.Deps{
		Launch: func(_ context.Context, _ driver.Plan, _ []int) (supervisor.LaunchResult, error) {
			return supervisor.LaunchResult{}, errors.New("exec failed")
		},
		HealthGate: func(_ context.Context, _ node.NodeSet) (supervisor.Diagnosis, error) {
			return supervisor.Diagnosis{OK: true}, nil
		},
		Procman: procman.New(),
		Sleep:   func(time.Duration) {},
	})
	if _, _, err := s.BringUp(context.Background(), driver.Plan{}, supervisor.Options{MaxAttempts: 1}); err == nil {
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

	pm := procman.New()
	pm.TrackProc(procman.Proc{PID: pid, DataDir: dataDir})
	s := supervisor.New(supervisor.Deps{Procman: pm, Sleep: func(time.Duration) {}})

	if err := s.Teardown(context.Background(), nodeSet(pid), supervisor.TeardownOpts{RemoveDataDir: true, Grace: time.Second}); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if procman.Alive(pid) {
		t.Fatal("process still alive after Teardown")
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatalf("datadir not removed (err=%v)", err)
	}
}
