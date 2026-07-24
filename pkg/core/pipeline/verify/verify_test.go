package verify_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/verify"
)

// fakeProber returns fixed info and an incrementing block height so the
// producing check sees strictly increasing heights.
type fakeProber struct {
	chainID uint64
	peers   uint64
	block   *atomic.Uint64
	failAll bool
}

func (f *fakeProber) ChainID(context.Context) (uint64, error) {
	if f.failAll {
		return 0, context.DeadlineExceeded
	}
	return f.chainID, nil
}
func (f *fakeProber) BlockNumber(context.Context) (uint64, error) {
	if f.failAll {
		return 0, context.DeadlineExceeded
	}
	return f.block.Add(1), nil // increments each call
}
func (f *fakeProber) PeerCount(context.Context) (uint64, error) { return f.peers, nil }
func (f *fakeProber) Syncing(context.Context) (bool, error)     { return false, nil }

func TestRun_ProducingAndInfo(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{
		{RPCURL: "http://n1"}, {RPCURL: "http://n2"},
	})

	block := &atomic.Uint64{}
	opts := verify.Options{
		Dial:          func(string) verify.Prober { return &fakeProber{chainID: 8283, peers: 2, block: block} },
		ProgressDelay: time.Millisecond,
		Sleep:         func(time.Duration) {}, // no real waiting
	}
	rep, err := verify.Run(context.Background(), ns, opts, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.Producing {
		t.Error("expected Producing=true (block height increments)")
	}
	if len(rep.Nodes) != 2 {
		t.Fatalf("nodes: %d", len(rep.Nodes))
	}
	for _, ni := range rep.Nodes {
		if !ni.OK || ni.ChainID != 8283 || ni.PeerCount != 2 {
			t.Errorf("node %d: %+v", ni.Index, ni)
		}
	}
}

func TestRun_NodeFailureRecorded(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://dead"}})
	opts := verify.Options{
		Dial:  func(string) verify.Prober { return &fakeProber{failAll: true, block: &atomic.Uint64{}} },
		Sleep: func(time.Duration) {},
	}
	rep, err := verify.Run(context.Background(), ns, opts, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Nodes[0].OK || rep.Nodes[0].Err == "" {
		t.Errorf("expected node failure recorded: %+v", rep.Nodes[0])
	}
	if rep.Producing {
		t.Error("dead node should not be producing")
	}
}

func TestRun_NotProducingWhenStatic(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://n1"}})
	// static prober: block never changes.
	staticProber := &staticBlock{height: 100, chainID: 1}
	opts := verify.Options{
		Dial:  func(string) verify.Prober { return staticProber },
		Sleep: func(time.Duration) {},
	}
	rep, _ := verify.Run(context.Background(), ns, opts, nil)
	if rep.Producing {
		t.Error("static height should report Producing=false")
	}
	if rep.Nodes[0].BlockNumber != 100 {
		t.Errorf("block: %d", rep.Nodes[0].BlockNumber)
	}
}

type staticBlock struct {
	height  uint64
	chainID uint64
}

func (s *staticBlock) ChainID(context.Context) (uint64, error)     { return s.chainID, nil }
func (s *staticBlock) BlockNumber(context.Context) (uint64, error) { return s.height, nil }
func (s *staticBlock) PeerCount(context.Context) (uint64, error)   { return 0, nil }
func (s *staticBlock) Syncing(context.Context) (bool, error)       { return false, nil }

// ensure node package import is used (NodeSet type surfaced via attach).
var _ = node.NodeSet{}
