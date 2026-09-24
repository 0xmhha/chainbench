package testengine

import (
	"context"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// upProber answers every probe as a healthy, advancing node.
type upProber struct{ h uint64 }

func (p *upProber) ChainID(context.Context) (uint64, error)     { return 1, nil }
func (p *upProber) BlockNumber(context.Context) (uint64, error) { p.h++; return p.h, nil }
func (p *upProber) PeerCount(context.Context) (uint64, error)   { return 1, nil }
func (p *upProber) Syncing(context.Context) (bool, error)       { return false, nil }

// TestHealthObserver_SeesThePIDARestartRecorded is design-v3 state-machine-06
// §10 E5, and it failed on the code it was written against.
//
// The gate restarts a node it finds dead, and the restart records the new pid
// in the workspace. The observer held the node set it was built with, so the
// next round read the old pid — zero — and called the restarted node dead
// again. A restart could never be judged a success; the gate spent its one
// restart and gave up with "exhausted 1 restart(s): process not alive".
func TestHealthObserver_SeesThePIDARestartRecorded(t *testing.T) {
	before := node.NodeSet{Nodes: []node.Node{{Index: 1, Host: "127.0.0.1", RPCURL: "http://127.0.0.1:8545"}}}
	after := node.NodeSet{Nodes: []node.Node{{Index: 1, Host: "127.0.0.1", RPCURL: "http://127.0.0.1:8545", PID: 4242}}}
	dial := &upProber{}
	obs := healthObserver{
		nodes: before,
		load:  func(context.Context) (node.NodeSet, error) { return after, nil },
		opts:  health.Options{Dial: func(string) health.Prober { return dial }, Sleep: func(time.Duration) {}},
	}
	facts, err := obs.Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 || !facts[0].PIDAlive {
		t.Fatalf("after the restart recorded pid 4242 the observer still sees %+v", facts)
	}
}
