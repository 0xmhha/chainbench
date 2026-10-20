package testengine

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/nodemonitor"
)

func forkHeightFixture() (health.Report, node.NodeSet, forkGate) {
	report := health.Report{Producing: true}
	nodes := node.NodeSet{}
	for i := 1; i <= 5; i++ {
		height := uint64(21)
		if i >= 4 {
			height = 19
		}
		report.Nodes = append(report.Nodes, health.NodeInfo{Index: i, OK: true, BlockNumber: height, PeerCount: 3})
		nodes.Nodes = append(nodes.Nodes, node.Node{Index: i, Role: node.RoleBP, PID: 100 + i})
	}
	return report, nodes, forkGate{at: 20, handedOver: true, preFork: map[int]bool{5: true}}
}

func TestHandoverReadinessRejectsPreForkNodeWithinLagTolerance(t *testing.T) {
	report, nodes, fork := forkHeightFixture()
	facts := factsFromReport(report, nodes, fork)
	for _, f := range facts {
		want := nodemonitor.Ready
		if f.Node == 4 {
			want = nodemonitor.Waitable
		}
		got := nodemonitor.Classify(f)
		if got.Verdict != want {
			t.Errorf("node%d at%d: %v, want%v (%v)", f.Node, f.Height, got.Verdict, want, got.Reasons)
		}
	}
}

type forkHeightObserver struct {
	report  health.Report
	nodes   node.NodeSet
	fork    forkGate
	rounds  int
	catchup bool
}

func (o *forkHeightObserver) Observe(context.Context) ([]nodemonitor.Facts, error) {
	o.rounds++
	if o.catchup && o.rounds > 1 {
		o.report.Nodes[3].BlockNumber = 20
	}
	return factsFromReport(o.report, o.nodes, o.fork), nil
}

type forkHeightClock struct{ calls int }

func (c *forkHeightClock) Sleep(context.Context, time.Duration) error { c.calls++; return nil }

type forkHeightNoRestart struct{}

func (forkHeightNoRestart) Restart(context.Context, int) error {
	return fmt.Errorf("a syncing successor must not be restarted")
}

func TestHandoverGateWaitsForActivationAndKeepsDeadline(t *testing.T) {
	for _, catchup := range []bool{true, false} {
		t.Run(fmt.Sprint(catchup), func(t *testing.T) {
			report, nodes, fork := forkHeightFixture()
			obs := &forkHeightObserver{report: report, nodes: nodes, fork: fork, catchup: catchup}
			clock := &forkHeightClock{}
			steps := []string{}
			result, err := nodemonitor.Gate(t.Context(), obs, forkHeightNoRestart{}, clock, stepSink{steps: &steps}, nodemonitor.Options{MaxNodeMonitorTimeout: 2 * time.Millisecond, WaitInterval: time.Millisecond})
			if err != nil || result.OK != catchup {
				t.Fatalf("catchup%v: result%+v err%v", catchup, result, err)
			}
			if catchup {
				if clock.calls != 1 || obs.rounds != 2 {
					t.Fatal("gate returned before successor activation", clock.calls, obs.rounds)
				}
			} else if clock.calls != 2 || !strings.Contains(result.Terminate, "still not ready") {
				t.Fatal("lost wait budget", clock.calls, result.Terminate)
			}
		})
	}
}

func TestHandoverHeightFloorPreservesOtherReadinessContracts(t *testing.T) {
	for _, mode := range []string{"active with sampling lag", "pre-handover parking", "restart before activation", "no fork"} {
		t.Run(mode, func(t *testing.T) {
			report, nodes, fork := forkHeightFixture()
			switch mode {
			case "active with sampling lag":
				report.Nodes[0].BlockNumber = 22
				report.Nodes[3].BlockNumber = 20
			case "pre-handover parking":
				fork.handedOver = false
				report.Producing = false
				for i := range report.Nodes {
					report.Nodes[i].BlockNumber = 19
				}
			case "restart before activation":
				fork.restart = true
			case "no fork":
				fork = forkGate{}
			}
			for _, fact := range factsFromReport(report, nodes, fork) {
				if got := nodemonitor.Classify(fact); got.Verdict != nodemonitor.Ready {
					t.Fatal(mode, got)
				}
				want := uint64(0)
				if mode == "active with sampling lag" && fact.Node != 5 {
					want = 20
				}
				if fact.WantMinHeight != want {
					t.Fatal("incorrect activation constraint", fact)
				}
			}
		})
	}
	report, nodes, fork := forkHeightFixture()
	fact := factsFromReport(report, nodes, fork)[3]
	fact.ChainID, fact.WantChainID = 1, 2
	if got := nodemonitor.Classify(fact); got.Verdict != nodemonitor.Fatal {
		t.Fatal("activation wait hid fatal wrong genesis", got)
	}
}
