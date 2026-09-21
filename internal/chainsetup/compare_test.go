package chainsetup

import (
	"context"
	"fmt"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/preflight"
)

// TestEveryVerdictIsAState holds the two vocabularies together.
//
// preflight answers how much has to be rebuilt; the state says where that
// answer puts the run. A verdict added without a state here would fall into the
// default and stop the walk, which is loud — but it would stop it at a
// comparison that had already been made, so the check is worth having at build
// time instead.
func TestEveryVerdictIsAState(t *testing.T) {
	for _, c := range []struct {
		v    preflight.Verdict
		want lifecycle.Status
	}{
		{preflight.Reuse, lifecycle.CompareChainSame},
		{preflight.RebuildNodes, lifecycle.CompareChainNodesDiffer},
		{preflight.RebuildAll, lifecycle.CompareChainNetworkDiffers},
		{preflight.Compose, lifecycle.CompareChainNothingComposed},
	} {
		got, err := stateOfVerdict(c.v)
		if err != nil {
			t.Errorf("%s: %v", c.v, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: %s, want %s", c.v, got, c.want)
		}
	}
	// A verdict nobody declared is refused rather than guessed at.
	if _, err := stateOfVerdict(preflight.Verdict(9)); err == nil {
		t.Error("an undeclared verdict was mapped to a state")
	}
}

// TestEachVerdictWalksItsOwnWay drives the four routes with the comparison and
// the steps replaced, because what is being checked is the walk: which states a
// verdict passes through, and which steps run as a result.
//
// The moves are the table's, so a route this could take and the table could not
// fails here rather than against a live network.
func TestEachVerdictWalksItsOwnWay(t *testing.T) {
	for _, c := range []struct {
		name  string
		at    lifecycle.Status
		steps string
	}{
		{"the network is the one wanted", lifecycle.CompareChainSame, ""},
		{"some nodes differ", lifecycle.CompareChainNodesDiffer, ""},
		{"a network-wide fact differs", lifecycle.CompareChainNetworkDiffers,
			"new,place,keys,genesis,config,build,deploy,init,start"},
		{"nothing is composed", lifecycle.CompareChainNothingComposed,
			"new,place,keys,genesis,config,build,deploy,init,start"},
	} {
		r := &recorder{}
		m, err := lifecycle.New(c.at, lifecycle.ChainVerify, walkOnly(c.at, r))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if err := m.Run(t.Context()); err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got := join(r.ran); got != c.steps {
			t.Errorf("%s: ran %q, want %q", c.name, got, c.steps)
		}
	}
}

// walkOnly is the comparison's moves with the work taken out, so the four
// routes can be walked without a workspace. The compose stages are the real
// handlers, driven by the recorder.
func walkOnly(from lifecycle.Status, r *recorder) map[lifecycle.Status]lifecycle.Handler {
	h := upHandlers(r.run)
	h[lifecycle.CompareChain] = func(_ context.Context, m *lifecycle.Machine, at lifecycle.Status) error {
		switch at {
		case lifecycle.CompareChainSame, lifecycle.CompareChainNodesDiffer:
			return m.Request(lifecycle.ChainVerify)
		case lifecycle.CompareChainNetworkDiffers, lifecycle.CompareChainNothingComposed:
			return m.Request(lifecycle.ChainOpenWorkspace)
		}
		return fmt.Errorf("the comparison was asked for %s", at)
	}
	return h
}

func join(xs []string) string {
	s := ""
	for i, x := range xs {
		if i > 0 {
			s += ","
		}
		s += x
	}
	return s
}
