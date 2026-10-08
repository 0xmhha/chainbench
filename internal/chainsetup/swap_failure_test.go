package chainsetup_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// TestSwapFailurePersistsStoppedStateAndPriorExecution reopens the saved record
// after the selected process has stopped but replacement cannot complete.
func TestSwapFailurePersistsStoppedStateAndPriorExecution(t *testing.T) {
	for _, failure := range []string{"launch", "configuration"} {
		t.Run(failure, func(t *testing.T) {
			dir, stub, deps := launchedNetwork(t)
			in := verb.NodeSwapIn{DataDir: dir, Index: 1, Binary: "/opt/replacement"}
			if failure == "launch" {
				deps.Driver = func() (process.Driver, error) { return restartFailureDriver{stub}, nil }
			} else {
				in.Binary, in.Config = "", []string{"unknown-option=true"}
			}
			if _, err := verb.NodeSwap(context.Background(), deps, in); err == nil {
				t.Fatal("failed replacement reported success")
			}
			ws, err := chainsetup.Open(dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			st := ws.State()
			if st.Nodes[0].PID != 0 {
				t.Fatalf("stopped node still recorded as running after %s failure: pid %d", failure, st.Nodes[0].PID)
			}
			if st.Nodes[1].PID != 1002 {
				t.Fatal("replacement changed sibling PID", st.Nodes)
			}
			if len(stub.stopped) != 1 || stub.stopped[0] != 1 {
				t.Fatal("replacement stopped outside selected node", stub.stopped)
			}
			led, err := process.OpenLedger(dir)
			if err != nil {
				t.Fatal(err)
			}
			label := string(node.LabelFor(1))
			if _, ok := led.Get(label); ok {
				t.Fatal("stopped process remained a current ledger entry")
			}
			history := led.History(label)
			if len(history) != 1 || history[0].PID != 1001 {
				t.Fatal("failed replacement lost prior execution history", history)
			}
			if failure == "launch" {
				retry, err := verb.NodeSwap(context.Background(), chainsetup.Deps{Driver: func() (process.Driver, error) { return stub, nil }}, in)
				if err != nil || retry.Node.PID != 2001 {
					t.Fatal("explicit replacement retry failed", retry, err)
				}
				led, err = process.OpenLedger(dir)
				if err != nil {
					t.Fatal(err)
				}
				current, ok := led.Get(label)
				if !ok || current.Revision != 1 || len(led.History(label)) != 1 {
					t.Fatal("retry lost revision continuity or duplicated history", current, led.History(label))
				}
			}
		})
	}
}

func TestSwapStopFailurePreservesCurrentExecutionAndHistory(t *testing.T) {
	dir, stub, deps := launchedNetwork(t)
	stub.stopErr = errors.New("stop refused")
	if _, err := verb.NodeSwap(context.Background(), deps, verb.NodeSwapIn{DataDir: dir, Index: 1, Binary: "/opt/replacement"}); err == nil {
		t.Fatal("stop failure reported replacement success")
	}
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ws.State().Nodes[0].PID != 1001 || len(stub.launched) != 0 {
		t.Fatal("failed stop cleared the PID or launched a replacement")
	}
	led, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	label := string(node.LabelFor(1))
	current, ok := led.Get(label)
	if !ok || current.PID != 1001 || len(led.History(label)) != 0 {
		t.Fatal("failed stop retired a process that was not stopped", current, led.History(label))
	}
}

// TestSwapLaunchFailurePreservesRealProcessesAndData uses the target's actual
// process driver, two isolated process groups and a missing replacement file.
func TestSwapLaunchFailurePreservesRealProcessesAndData(t *testing.T) {
	dir := t.TempDir()
	nodes := make([]node.Record, 0, 2)
	markers := make([]string, 0, 2)
	for i := 1; i <= 2; i++ {
		child := exec.Command("/bin/sleep", "120")
		child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() { _ = child.Wait(); close(done) }()
		t.Cleanup(func() {
			_ = child.Process.Kill()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("fixture process did not exit")
			}
		})
		n := record(dir, i, 8600+(i-1)*10, child.Process.Pid)
		n.Args = []string{"120"}
		if err := os.MkdirAll(n.DataDir, 0700); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(n.DataDir, "retained-data")
		if err := os.WriteFile(marker, []byte("preserve owned data"), 0600); err != nil {
			t.Fatal(err)
		}
		markers = append(markers, marker)
		nodes = append(nodes, n)
	}
	seedWorkspace(t, dir, "wbft", "/bin/sleep", nodes)
	initialLedger, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		spec := process.SpecOf(n)
		spec.Binary = "/bin/sleep"
		if err = initialLedger.Record(process.ProcFor(spec, n.PID)); err != nil {
			t.Fatal(err)
		}
	}
	if err = initialLedger.Save(); err != nil {
		t.Fatal(err)
	}
	_, err = verb.NodeSwap(context.Background(), chainsetup.Deps{}, verb.NodeSwapIn{DataDir: dir, Index: 1, Binary: filepath.Join(dir, "missing-replacement")})
	if err == nil {
		t.Fatal("missing executable replacement reported success")
	}
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ws.State().Nodes[0].PID != 0 || process.Alive(nodes[0].PID) {
		t.Fatal("stopped real process remained current or alive")
	}
	if ws.State().Nodes[1].PID != nodes[1].PID || !process.Alive(nodes[1].PID) {
		t.Fatal("replacement failure affected sibling process")
	}
	for _, path := range markers {
		raw, err := os.ReadFile(path)
		if err != nil || string(raw) != "preserve owned data" {
			t.Fatal("replacement failure changed node data", err)
		}
	}
	led, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	history := led.History(string(node.LabelFor(1)))
	if len(history) != 1 || history[0].PID != nodes[0].PID || history[0].Command != "/bin/sleep 120" || history[0].Binary != "sleep" {
		t.Fatal("real stopped process lost its execution history", history)
	}
}
