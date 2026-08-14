package testspec_test

import (
	"reflect"
	"testing"

	"github.com/0xmhha/chainbench/internal/testspec"
)

func TestUnresolved(t *testing.T) {
	reg := testspec.NewRegistry(true) // seeds built-ins (sendTx, waitBlock, chainId, ...)

	t.Run("all resolve", func(t *testing.T) {
		s := testspec.Spec{
			Steps:      []map[string]any{{"sendTx": map[string]any{"from": "0x1"}}, {"waitBlock": map[string]any{"target": 3}}},
			Assertions: []map[string]any{{"assert": "chainId", "expected": 1}, {"assert": "balanceAt", "address": "0x1", "expected": 0}},
		}
		if got := testspec.Unresolved(s, reg); len(got) != 0 {
			t.Fatalf("expected no unresolved, got %v", got)
		}
	})

	t.Run("unknown names flagged", func(t *testing.T) {
		s := testspec.Spec{
			PreActions: []map[string]any{{"ensureChain": true}}, // not a built-in action
			Steps:      []map[string]any{{"sendTx": map[string]any{}}, {"teleport": map[string]any{}}},
			Assertions: []map[string]any{{"assert": "chainId"}, {"assert": "Nonexistent"}},
		}
		got := testspec.Unresolved(s, reg)
		want := []string{"action:ensureChain", "action:teleport", "assert:Nonexistent"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Unresolved = %v, want %v", got, want)
		}
	})

	t.Run("empty entries flagged", func(t *testing.T) {
		s := testspec.Spec{
			Steps:      []map[string]any{{}},
			Assertions: []map[string]any{{"expected": 1}}, // no "assert"
		}
		got := testspec.Unresolved(s, reg)
		want := []string{"action:(empty)", "assert:(missing)"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Unresolved = %v, want %v", got, want)
		}
	})
}

func TestUnresolved_ReportsUnboundReferences(t *testing.T) {
	reg := testspec.NewRegistry(true)
	spec := testspec.Spec{
		Steps: []map[string]any{
			{"read": map[string]any{"source": "call", "to": "0xa", "data": "0xb", "save": "supply"}},
			{"sendTx": map[string]any{"from": "0xa", "to": "$supply"}},
		},
		Assertions: []map[string]any{
			{"assert": "call", "to": "0xa", "data": "0xb", "expected": "$missing"},
		},
	}
	got := testspec.Unresolved(spec, reg)
	want := []string{"ref:missing"}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("Unresolved = %v, want %v", got, want)
	}
}

func TestUnresolved_ReferenceMustBeSavedEarlier(t *testing.T) {
	reg := testspec.NewRegistry(true)
	// The reference is used in step 0 but only saved in step 1 — ordering matters.
	spec := testspec.Spec{
		Steps: []map[string]any{
			{"sendTx": map[string]any{"from": "0xa", "to": "$later"}},
			{"read": map[string]any{"source": "call", "to": "0xa", "data": "0xb", "save": "later"}},
		},
		Assertions: []map[string]any{{"assert": "blockNumber", "expected": "1"}},
	}
	got := testspec.Unresolved(spec, reg)
	if len(got) != 1 || got[0] != "ref:later" {
		t.Fatalf("Unresolved = %v, want [ref:later]", got)
	}
}

func TestUnresolved_BoundReferencesAreClean(t *testing.T) {
	reg := testspec.NewRegistry(true)
	spec := testspec.Spec{
		Steps: []map[string]any{
			{"sendTx": map[string]any{"from": "0xa", "to": "0xb", "save": "hash"}},
		},
		Assertions: []map[string]any{
			{"assert": "txStatus", "hash": "$hash", "expected": "0x1"},
		},
	}
	if got := testspec.Unresolved(spec, reg); len(got) != 0 {
		t.Fatalf("Unresolved = %v, want none", got)
	}
}

func TestUnresolved_SaveKeyBindsReference(t *testing.T) {
	reg := testspec.NewRegistry(true)
	spec := testspec.Spec{
		Steps: []map[string]any{
			{"newAccount": map[string]any{"save": "acct", "saveKey": "acctKey"}},
			{"sendTx": map[string]any{"key": "$acctKey", "to": "0xb", "expect": "reject"}},
		},
		Assertions: []map[string]any{},
	}
	if got := testspec.Unresolved(spec, reg); len(got) != 0 {
		t.Fatalf("Unresolved = %v, want none (saveKey should bind $acctKey)", got)
	}
}

func TestUnresolved_ReportsAnUnknownReadSource(t *testing.T) {
	reg := testspec.NewRegistry(true)
	spec := testspec.Spec{
		Steps: []map[string]any{
			{"read": map[string]any{"source": "nosuchreader", "save": "v"}},
		},
		Assertions: []map[string]any{{"assert": "blockNumber", "expected": "1"}},
	}
	got := testspec.Unresolved(spec, reg)
	if len(got) != 1 || got[0] != "source:nosuchreader" {
		t.Fatalf("Unresolved = %v, want [source:nosuchreader]", got)
	}
}

func TestUnresolved_AcceptsAValidReadSource(t *testing.T) {
	reg := testspec.NewRegistry(true)
	spec := testspec.Spec{
		Steps: []map[string]any{
			{"read": map[string]any{"source": "rpcCall", "method": "eth_chainId", "save": "v"}},
		},
		Assertions: []map[string]any{{"assert": "chainId", "expected": "$v"}},
	}
	if got := testspec.Unresolved(spec, reg); len(got) != 0 {
		t.Fatalf("Unresolved = %v, want none", got)
	}
}
