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
