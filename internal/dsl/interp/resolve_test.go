package interp_test

import (
	"github.com/0xmhha/chainbench/internal/testhelper"
	"reflect"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestUnresolved(t *testing.T) {
	reg := testhelper.Registry() // seeds built-ins (sendTx, waitBlock, chainId, ...)

	t.Run("all resolve", func(t *testing.T) {
		s := dsl.Spec{
			Steps:      []map[string]any{{"sendTx": map[string]any{"from": "0x1"}}, {"waitBlock": map[string]any{"target": 3}}},
			Assertions: []map[string]any{{"assert": "chainId", "expected": 1}, {"assert": "balanceAt", "address": "0x1", "expected": 0}},
		}
		if got := interp.Unresolved(s, reg); len(got) != 0 {
			t.Fatalf("expected no unresolved, got %v", got)
		}
	})

	t.Run("unknown names flagged", func(t *testing.T) {
		s := dsl.Spec{
			PreActions: []map[string]any{{"ensureChain": true}}, // not a built-in action
			Steps:      []map[string]any{{"sendTx": map[string]any{}}, {"teleport": map[string]any{}}},
			Assertions: []map[string]any{{"assert": "chainId"}, {"assert": "Nonexistent"}},
		}
		got := interp.Unresolved(s, reg)
		want := []string{"action:ensureChain", "action:teleport", "assert:Nonexistent"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Unresolved = %v, want %v", got, want)
		}
	})

	t.Run("empty entries flagged", func(t *testing.T) {
		s := dsl.Spec{
			Steps:      []map[string]any{{}},
			Assertions: []map[string]any{{"expected": 1}}, // no "assert"
		}
		got := interp.Unresolved(s, reg)
		want := []string{"action:(empty)", "assert:(missing)"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Unresolved = %v, want %v", got, want)
		}
	})
}

// A v2 case saves a value in a do step and references it from a later expect
// statement. The two run interleaved in one Sequence, so the binding must be
// visible to the assertion — the old Steps-then-Assertions split bound nothing
// from a step for the assertions and reported a false unresolved reference.
func TestUnresolved_V2SequenceBindsSaveForLaterExpect(t *testing.T) {
	reg := testhelper.Registry()
	spec := dsl.Spec{
		Sequence: []dsl.Statement{
			{Do: "read", Args: map[string]any{"source": "baseFee", "save": "bf0"}},
			{Do: "waitBlock", Args: map[string]any{"target": 3}},
			{Expect: "baseFee", Args: map[string]any{"compare": "Greater", "expected": "$bf0"}},
		},
	}
	if got := interp.Unresolved(spec, reg); len(got) != 0 {
		t.Fatalf("Unresolved = %v, want none (a v2 step's save must bind for a later expect)", got)
	}
}

// A v2 expect statement that references a value nothing saved is still flagged.
func TestUnresolved_V2SequenceReportsUnboundExpectReference(t *testing.T) {
	reg := testhelper.Registry()
	spec := dsl.Spec{
		Sequence: []dsl.Statement{
			{Do: "waitBlock", Args: map[string]any{"target": 3}},
			{Expect: "baseFee", Args: map[string]any{"compare": "Greater", "expected": "$bf0"}},
		},
	}
	got := interp.Unresolved(spec, reg)
	if len(got) != 1 || got[0] != "ref:bf0" {
		t.Fatalf("Unresolved = %v, want [ref:bf0]", got)
	}
}

func TestUnresolved_ReportsUnboundReferences(t *testing.T) {
	reg := testhelper.Registry()
	spec := dsl.Spec{
		Steps: []map[string]any{
			{"read": map[string]any{"source": "call", "to": "0xa", "data": "0xb", "save": "supply"}},
			{"sendTx": map[string]any{"from": "0xa", "to": "$supply"}},
		},
		Assertions: []map[string]any{
			{"assert": "call", "to": "0xa", "data": "0xb", "expected": "$missing"},
		},
	}
	got := interp.Unresolved(spec, reg)
	want := []string{"ref:missing"}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("Unresolved = %v, want %v", got, want)
	}
}

func TestUnresolved_ReferenceMustBeSavedEarlier(t *testing.T) {
	reg := testhelper.Registry()
	// The reference is used in step 0 but only saved in step 1 — ordering matters.
	spec := dsl.Spec{
		Steps: []map[string]any{
			{"sendTx": map[string]any{"from": "0xa", "to": "$later"}},
			{"read": map[string]any{"source": "call", "to": "0xa", "data": "0xb", "save": "later"}},
		},
		Assertions: []map[string]any{{"assert": "blockNumber", "expected": "1"}},
	}
	got := interp.Unresolved(spec, reg)
	if len(got) != 1 || got[0] != "ref:later" {
		t.Fatalf("Unresolved = %v, want [ref:later]", got)
	}
}

func TestUnresolved_BoundReferencesAreClean(t *testing.T) {
	reg := testhelper.Registry()
	spec := dsl.Spec{
		Steps: []map[string]any{
			{"sendTx": map[string]any{"from": "0xa", "to": "0xb", "save": "hash"}},
		},
		Assertions: []map[string]any{
			{"assert": "txStatus", "hash": "$hash", "expected": "0x1"},
		},
	}
	if got := interp.Unresolved(spec, reg); len(got) != 0 {
		t.Fatalf("Unresolved = %v, want none", got)
	}
}

func TestUnresolved_SaveKeyBindsReference(t *testing.T) {
	reg := testhelper.Registry()
	spec := dsl.Spec{
		Steps: []map[string]any{
			{"newAccount": map[string]any{"save": "acct", "saveKey": "acctKey"}},
			{"sendTx": map[string]any{"key": "$acctKey", "to": "0xb", "expect": "reject"}},
		},
		Assertions: []map[string]any{},
	}
	if got := interp.Unresolved(spec, reg); len(got) != 0 {
		t.Fatalf("Unresolved = %v, want none (saveKey should bind $acctKey)", got)
	}
}

func TestUnresolved_ReportsAnUnknownReadSource(t *testing.T) {
	reg := testhelper.Registry()
	spec := dsl.Spec{
		Steps: []map[string]any{
			{"read": map[string]any{"source": "nosuchreader", "save": "v"}},
		},
		Assertions: []map[string]any{{"assert": "blockNumber", "expected": "1"}},
	}
	got := interp.Unresolved(spec, reg)
	if len(got) != 1 || got[0] != "source:nosuchreader" {
		t.Fatalf("Unresolved = %v, want [source:nosuchreader]", got)
	}
}

func TestUnresolved_AcceptsAValidReadSource(t *testing.T) {
	reg := testhelper.Registry()
	spec := dsl.Spec{
		Steps: []map[string]any{
			{"read": map[string]any{"source": "rpcCall", "method": "eth_chainId", "save": "v"}},
		},
		Assertions: []map[string]any{{"assert": "chainId", "expected": "$v"}},
	}
	if got := interp.Unresolved(spec, reg); len(got) != 0 {
		t.Fatalf("Unresolved = %v, want none", got)
	}
}
