package node_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestValidScope_AcceptsEveryRole is why the rule moved here. Two callers wrote
// the scope list out by hand, both named bp and en, and both forgot pn: a
// topology could declare a proxy tier that no per-node value could address.
// Asking the vocabulary means a role is addressable the moment it exists.
func TestValidScope_AcceptsEveryRole(t *testing.T) {
	for _, r := range []node.Role{node.RoleBP, node.RoleEN, node.RolePN} {
		if !node.ValidScope(string(r)) {
			t.Errorf("role %q must be a scope", r)
		}
	}
	for _, s := range []string{node.ScopeAll, "node1", "node12"} {
		if !node.ValidScope(s) {
			t.Errorf("%q must be a scope", s)
		}
	}
	for _, s := range []string{"", "validator", "endpoint", "boot", "sideways", "node", "node0", "node01", "Node1", "node-1"} {
		if node.ValidScope(s) {
			t.Errorf("%q must not be a scope", s)
		}
	}
}

// TestScopeIndex_ReadsOnlyAnUnambiguousIndex: zero means "not an indexed
// scope", and a leading zero is refused so "node01" cannot mean node 1 under
// one reader and nothing under another.
func TestScopeIndex_ReadsOnlyAnUnambiguousIndex(t *testing.T) {
	for in, want := range map[string]int{
		"node1": 1, "node12": 12, "node100": 100,
		"node0": 0, "node01": 0, "node": 0, "all": 0, "bp": 0, "": 0, "nodeX": 0,
	} {
		if got := node.ScopeIndex(in); got != want {
			t.Errorf("ScopeIndex(%q) = %d, want %d", in, got, want)
		}
	}
}

// TestScopeFor_OrdersMostGeneralFirst pins the fold order every caller depends
// on: a value set for one node beats one set for its role, which beats one set
// for every node. A caller applies them in this order and lets the last write
// win, so the order IS the precedence.
func TestScopeFor_OrdersMostGeneralFirst(t *testing.T) {
	got := strings.Join(node.ScopeFor(node.RoleBP, 3), ",")
	if want := "all,bp,node3"; got != want {
		t.Errorf("ScopeFor(bp, 3) = %q, want %q", got, want)
	}
	// An unreadable role contributes no scope. An empty one would look up the
	// "" key and find whatever a caller happened to store there.
	got = strings.Join(node.ScopeFor(node.Role("validator"), 2), ",")
	if want := "all,node2"; got != want {
		t.Errorf("ScopeFor(validator, 2) = %q, want %q", got, want)
	}
	// Index 0 means "no particular node", so only the general scopes apply.
	got = strings.Join(node.ScopeFor(node.RoleEN, 0), ",")
	if want := "all,en"; got != want {
		t.Errorf("ScopeFor(en, 0) = %q, want %q", got, want)
	}
}

// TestScopeWords_NamesEveryRole keeps the refusal message honest: a surface
// that refuses a scope has to be able to say what it would have accepted, and
// the answer must not go stale when the vocabulary changes.
func TestScopeWords_NamesEveryRole(t *testing.T) {
	words := node.ScopeWords()
	for _, want := range []string{node.ScopeAll, string(node.RoleBP), string(node.RoleEN), string(node.RolePN), "node<N>"} {
		if !strings.Contains(words, want) {
			t.Errorf("ScopeWords() = %q, which does not name %q", words, want)
		}
	}
}
