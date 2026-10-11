package app

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestFailedDeploymentNamesEachNodesKnownState(t *testing.T) {
	state := State{Nodes: []node.Record{
		{Index: 1, Label: "node1", DataDir: "/d/nodes/c/node1", PID: 4100},
		{Index: 2, Label: "node2", DataDir: "/d/nodes/c/node2"},
		{Index: 3, Label: "node3", DataDir: "/d/nodes/c/node3"},
	}}
	exists := func(path string) (bool, error) { return path != "/d/nodes/c/node3", nil }
	effects, unresolved := webRecordedNodeEffects(state, exists)
	joined := strings.Join(effects, "; ")
	for _, want := range []string{"node1: data directory present, launched with recorded PID 4100", "node2: data directory present, not launched", "node3: no data directory"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in %v", want, effects)
		}
	}
	if strings.Join(unresolved, ",") != "/d/nodes/c/node1,/d/nodes/c/node2" {
		t.Errorf("unresolved %v", unresolved)
	}
	unknown := func(string) (bool, error) { return false, errFakeInspection }
	effects, unresolved = webRecordedNodeEffects(state, unknown)
	if !strings.Contains(strings.Join(effects, "; "), "node2: state could not be checked") || len(unresolved) != 3 {
		t.Errorf("an uncheckable node must stay unresolved: %v %v", effects, unresolved)
	}
}

func TestFailureDispositionFollowsTheFailedPhase(t *testing.T) {
	for phase, want := range map[string]string{"new": "retained", "place": "retained", "build": "retained", "deploy": "unknown", "init": "unknown", "start": "unknown"} {
		if got := webFailureDisposition(phase); got != want {
			t.Errorf("%s: %s, want %s", phase, got, want)
		}
	}
}

var errFakeInspection = errFake("target unreachable")

type errFake string

func (e errFake) Error() string { return string(e) }
