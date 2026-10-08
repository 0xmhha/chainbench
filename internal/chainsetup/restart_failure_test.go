package chainsetup_test

import (
	"context"
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

type restartFailureDriver struct{ *stubDriver }

func (d restartFailureDriver) Launch(context.Context, process.NodeSpec) (process.Handle, error) {
	return process.Handle{}, errors.New("selected node launch failed")
}

func TestRestartFailurePersistsStoppedNodeAndPreservesSibling(t *testing.T) {
	dir, stub, deps := launchedNetwork(t)
	deps.Driver = func() (process.Driver, error) { return restartFailureDriver{stub}, nil }
	_, err := verb.ChainRestart(context.Background(), deps, verb.ChainRestartIn{DataDir: dir, Node: 1})
	if err == nil {
		t.Fatal("failed launch reported a successful restart")
	}
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	st := ws.State()
	if st.Nodes[0].PID != 0 || st.Nodes[1].PID != 1002 {
		t.Fatal("failed restart lost the stopped state or changed a sibling", st.Nodes)
	}
	if len(stub.stopped) != 1 || stub.stopped[0] != 1 {
		t.Fatal("restart stopped outside the selected node", stub.stopped)
	}
	ledger, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ledger.Get(string(node.LabelFor(1))); ok {
		t.Fatal("failed restart retained a stale running ledger entry")
	}
}

func TestRestartStopFailureDoesNotLaunchOrClearPID(t *testing.T) {
	dir, stub, deps := launchedNetwork(t)
	stub.stopErr = errors.New("selected node could not stop")
	_, err := verb.ChainRestart(context.Background(), deps, verb.ChainRestartIn{DataDir: dir, Node: 1})
	if err == nil {
		t.Fatal("failed stop reported a successful restart")
	}
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ws.State().Nodes[0].PID != 1001 || len(stub.launched) != 0 {
		t.Fatal("failed stop cleared the existing PID or launched another process")
	}
}
