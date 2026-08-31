package testengine_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"

	"github.com/0xmhha/chainbench/internal/testengine"
)

// collectEvents runs specs through the engine with an Emit hook capturing every
// published event, so tests can assert the dashboard-facing milestone stream.
func collectEvents(t *testing.T, h *harness, specs [][]byte, network string) []collector.Event {
	t.Helper()
	deps := h.deps(t)
	deps.Network = network
	var got []collector.Event
	deps.Emit = func(ev collector.Event) { got = append(got, ev) }
	if _, err := testengine.New(deps).Run(context.Background(), specs); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return got
}

func messages(evs []collector.Event) []string {
	out := make([]string, len(evs))
	for i, e := range evs {
		out[i] = e.Message
	}
	return out
}

func containsMsg(evs []collector.Event, msg string) bool {
	for _, e := range evs {
		if e.Message == msg {
			return true
		}
	}
	return false
}

func TestEngine_EmitsMilestoneEvents(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	evs := collectEvents(t, h, [][]byte{specJSON("T1", "wbft")}, "wbft")

	for _, want := range []string{"run started", "building environment", "running spec", "spec pass", "run complete"} {
		if !containsMsg(evs, want) {
			t.Fatalf("missing milestone %q in %v", want, messages(evs))
		}
	}
	// Every event carries the network label for the dashboard.
	for _, e := range evs {
		if e.Network != "wbft" {
			t.Fatalf("event %q network = %q, want wbft", e.Message, e.Network)
		}
	}
}

func TestEngine_EmitResultCarriesStatus(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	evs := collectEvents(t, h, [][]byte{specJSON("T1", "wbft")}, "wbft")

	var result *collector.Event
	for i := range evs {
		if evs[i].Kind == collector.KindResult && evs[i].Message == "spec pass" {
			result = &evs[i]
		}
	}
	if result == nil {
		t.Fatalf("no result event in %v", messages(evs))
	}
	if got := result.Fields["status"]; got != "pass" {
		t.Fatalf("result status field = %v, want pass", got)
	}
	if got := result.Fields["id"]; got != "T1" {
		t.Fatalf("result id field = %v, want T1", got)
	}
}

func TestEngine_EmitSkipsInapplicable(t *testing.T) {
	h := &harness{
		fpByChain:  map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"},
		applicable: func(s dsl.Spec) bool { return s.Chain.Name == "wbft" },
	}
	evs := collectEvents(t, h, [][]byte{specJSON("T1", "stablenet")}, "wbft")
	if !containsMsg(evs, "spec skipped") {
		t.Fatalf("missing skip event in %v", messages(evs))
	}
	if containsMsg(evs, "building environment") {
		t.Fatalf("inapplicable spec must not build: %v", messages(evs))
	}
}

func TestEngine_NilEmitIsNoop(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "aaaaaaaaaaaa0000"}}
	// deps() leaves Emit nil; Run must not panic.
	if _, err := testengine.New(h.deps(t)).Run(context.Background(), [][]byte{specJSON("T1", "wbft")}); err != nil {
		t.Fatalf("Run with nil Emit: %v", err)
	}
}
