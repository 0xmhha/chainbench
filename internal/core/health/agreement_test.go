package health_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// verify answered "is this network healthy" without ever asking whether the
// nodes were on the same chain. Producing reads the PRIMARY node's height
// rising, which is equally true of a network that has split: every partition
// keeps producing, on its own fork, and each node is up and in sync with itself.
//
// These pin the check that closes that, and — as much — pin that it refuses to
// answer when it could not look.

// chainProber answers as a node on a named chain: the hash it reports at a
// height encodes which chain it is on, so two probers with different chain tags
// are a fork and two with the same tag are not.
type chainProber struct {
	chainTag string
	height   uint64
	hashErr  error
}

func (p *chainProber) ChainID(context.Context) (uint64, error)     { return 1, nil }
func (p *chainProber) BlockNumber(context.Context) (uint64, error) { return p.height, nil }
func (p *chainProber) PeerCount(context.Context) (uint64, error)   { return 3, nil }
func (p *chainProber) Syncing(context.Context) (bool, error)       { return false, nil }
func (p *chainProber) BlockHashAt(_ context.Context, n uint64) (string, error) {
	if p.hashErr != nil {
		return "", p.hashErr
	}
	return fmt.Sprintf("0x%s%04d", p.chainTag, n), nil
}

// blindProber can do everything a Prober must and nothing more: it cannot report
// a hash, which is the case the optional interface exists for.
type blindProber struct{ height uint64 }

func (p *blindProber) ChainID(context.Context) (uint64, error)     { return 1, nil }
func (p *blindProber) BlockNumber(context.Context) (uint64, error) { return p.height, nil }
func (p *blindProber) PeerCount(context.Context) (uint64, error)   { return 3, nil }
func (p *blindProber) Syncing(context.Context) (bool, error)       { return false, nil }

func nodesAt(urls ...string) node.NodeSet {
	ns := node.NodeSet{Network: "t"}
	for i, u := range urls {
		ns.Nodes = append(ns.Nodes, node.Node{Index: i + 1, RPCURL: u})
	}
	return ns
}

func runWith(t *testing.T, ns node.NodeSet, probers map[string]health.Prober) health.Report {
	t.Helper()
	rep, err := health.Run(context.Background(), ns, health.Options{
		Dial:  func(u string) health.Prober { return probers[u] },
		Sleep: func(time.Duration) {},
	}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return rep
}

// TestAgreement_SplitNetworkIsNotHealthy is the case that used to pass. Both
// partitions are up, in sync with themselves, and producing.
func TestAgreement_SplitNetworkIsNotHealthy(t *testing.T) {
	ns := nodesAt("a", "b", "c")
	rep := runWith(t, ns, map[string]health.Prober{
		"a": &chainProber{chainTag: "aaaa", height: 20},
		"b": &chainProber{chainTag: "aaaa", height: 20},
		"c": &chainProber{chainTag: "bbbb", height: 20}, // the other side of the split
	})
	if !rep.Agreement.Checked {
		t.Fatalf("the comparison should have been possible: %s", rep.Agreement.Detail)
	}
	if rep.Agreement.Agreed {
		t.Fatal("a network split across two chains reported agreement")
	}
	if !strings.Contains(rep.Agreement.Detail, "3") && !strings.Contains(rep.Agreement.Detail, "[3]") {
		t.Errorf("the detail should name which nodes are where: %q", rep.Agreement.Detail)
	}
}

// TestAgreement_OneChainAgrees: the ordinary case still reads as agreement, and
// names the block it compared.
func TestAgreement_OneChainAgrees(t *testing.T) {
	ns := nodesAt("a", "b")
	rep := runWith(t, ns, map[string]health.Prober{
		"a": &chainProber{chainTag: "aaaa", height: 20},
		"b": &chainProber{chainTag: "aaaa", height: 20},
	})
	if !rep.Agreement.Checked || !rep.Agreement.Agreed {
		t.Fatalf("one chain should agree: %+v", rep.Agreement)
	}
	if rep.Agreement.Height == 0 {
		t.Error("the report should name the block it compared")
	}
}

// TestAgreement_ALaggingNodeIsNotAFork is why the comparison is at a height and
// not at the head. A node one block behind holds a different HEAD hash while
// being on exactly the same chain.
func TestAgreement_ALaggingNodeIsNotAFork(t *testing.T) {
	ns := nodesAt("a", "b")
	rep := runWith(t, ns, map[string]health.Prober{
		"a": &chainProber{chainTag: "aaaa", height: 20},
		"b": &chainProber{chainTag: "aaaa", height: 17}, // three behind, same chain
	})
	if !rep.Agreement.Agreed {
		t.Fatalf("a lagging node on the same chain is not a fork: %s", rep.Agreement.Detail)
	}
	if rep.Agreement.Height != 17 {
		t.Errorf("compared block %d, want the highest block BOTH have (17)", rep.Agreement.Height)
	}
}

// TestAgreement_UncheckableIsNotAgreement is the vacuous-truth guard. A prober
// that cannot read a hash, or a set with one node, must report that it could not
// look — not that everything is fine.
func TestAgreement_UncheckableIsNotAgreement(t *testing.T) {
	t.Run("prober cannot read a hash", func(t *testing.T) {
		ns := nodesAt("a", "b")
		rep := runWith(t, ns, map[string]health.Prober{
			"a": &blindProber{height: 20},
			"b": &blindProber{height: 20},
		})
		if rep.Agreement.Agreed {
			t.Fatal("a prober that cannot compare reported agreement")
		}
		if rep.Agreement.Checked {
			t.Fatal("it must say the check did not happen")
		}
		if !strings.Contains(rep.Agreement.Detail, "hash") {
			t.Errorf("the detail should say what it could not do: %q", rep.Agreement.Detail)
		}
	})

	t.Run("a single node", func(t *testing.T) {
		ns := nodesAt("a")
		rep := runWith(t, ns, map[string]health.Prober{"a": &chainProber{chainTag: "aaaa", height: 20}})
		if rep.Agreement.Checked || rep.Agreement.Agreed {
			t.Fatalf("one node cannot agree with anyone: %+v", rep.Agreement)
		}
	})

	t.Run("a node that errors on the hash", func(t *testing.T) {
		ns := nodesAt("a", "b")
		rep := runWith(t, ns, map[string]health.Prober{
			"a": &chainProber{chainTag: "aaaa", height: 20},
			"b": &chainProber{chainTag: "aaaa", height: 20, hashErr: errNoHash},
		})
		if rep.Agreement.Checked || rep.Agreement.Agreed {
			t.Fatalf("a node that would not answer cannot be counted as agreeing: %+v", rep.Agreement)
		}
	})
}

var errNoHash = fmt.Errorf("no hash")
