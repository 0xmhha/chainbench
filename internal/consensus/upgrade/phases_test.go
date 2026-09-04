package upgrade

import (
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/wemix" // register the poa-family plugin
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// poaFamilyPlugin is the wemix plugin, whose family declares the bring-up order
// a handoff has to follow.
func poaFamilyPlugin(t *testing.T) registry.ChainPlugin {
	t.Helper()
	p, err := registry.Get("wemix")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestPhases_TheProducerGoesUpAlone.
//
// A poa network's etcd cluster forms only while its producer is alone: with the
// others already running, admin.etcdInit() returns without error and creates
// nothing, and the producer then never seals. The handoff used to launch every
// node and bootstrap afterwards, which is why it failed at verify-etcd while
// the same network came up correctly through the composition path.
//
// The order is the consensus family's to declare. This pins that the handoff
// asks rather than deciding, and that the answer is rendered as the plan's own
// 0-based positions.
func TestPhases_TheProducerGoesUpAlone(t *testing.T) {
	h := &Handoff{}
	h.Plan.Nodes = []NodeSpec{
		{Producer: true}, {Producer: false}, {Producer: false}, {Producer: false},
	}
	h.From = poaFamilyPlugin(t)

	boot, rest := h.phases()
	if len(boot) != 1 || boot[0] != 0 {
		t.Fatalf("boot = %v, want only the producer at position 0", boot)
	}
	if len(rest) != 3 {
		t.Fatalf("rest = %v, want the other three", rest)
	}
	for i, want := range []int{1, 2, 3} {
		if rest[i] != want {
			t.Fatalf("rest = %v, want [1 2 3] — positions are the plan's, 0-based", rest)
		}
	}
}

// TestPhases_ALoneProducerHasNoSecondStep: with nothing else to start, the
// bring-up is one step and there is no empty phase to launch.
func TestPhases_ALoneProducerHasNoSecondStep(t *testing.T) {
	h := &Handoff{}
	h.Plan.Nodes = []NodeSpec{{Producer: true}}
	h.From = poaFamilyPlugin(t)

	boot, rest := h.phases()
	if len(boot) != 1 || len(rest) != 0 {
		t.Fatalf("boot = %v, rest = %v; want the producer alone and nothing after", boot, rest)
	}
}

// TestLaunchOptions_OnlySelectsPositions: an empty selection is every node,
// which is what a family with nothing to order asks for.
func TestLaunchOptions_OnlySelectsPositions(t *testing.T) {
	all := LaunchOptions{}
	for i := range 3 {
		if !all.wants(i) {
			t.Errorf("an empty selection skipped position %d", i)
		}
	}
	some := LaunchOptions{Only: []int{0, 2}}
	for i, want := range map[int]bool{0: true, 1: false, 2: true} {
		if some.wants(i) != want {
			t.Errorf("wants(%d) = %v, want %v", i, some.wants(i), want)
		}
	}
}
