package node_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestIs_AsksTheVocabularyRatherThanComparingWords is the predicate every role
// decision has to go through. Two defects in this series came from comparing
// against one spelling: a producer launched without --mine (the chain stalls
// while every node reports healthy) and a selector that resolved bp1 to the
// wrong node. Neither failed until something else started emitting the other
// word.
//
// A word outside the vocabulary answers false. It does not match by accident,
// and it does not fold onto a role it resembles.
func TestIs_AsksTheVocabularyRatherThanComparingWords(t *testing.T) {
	for _, c := range []struct {
		role, canonical node.Role
		want            bool
	}{
		{node.RoleBP, node.RoleBP, true},
		{node.RoleEN, node.RoleEN, true},
		{node.RolePN, node.RolePN, true},
		{node.RoleEN, node.RoleBP, false},
		{node.RolePN, node.RoleEN, false},
		{node.RoleBP, node.RolePN, false},
		{node.Role("sideways"), node.RoleBP, false},
		{node.RoleBP, node.Role("sideways"), false},
	} {
		if got := node.Is(c.role, c.canonical); got != c.want {
			t.Errorf("Is(%q, %q) = %v, want %v", c.role, c.canonical, got, c.want)
		}
	}
}

// TestNormalizeRole_RefusesTheRetiredSpellings is the regression this change
// exists to create.
//
// "validator", "endpoint" and "boot" used to fold onto bp, en and bp. They were
// kept so state written before the vocabulary settled would still parse, and
// the comment saying so was read as fact long after nothing wrote them. Folding
// them also made "validator" look like a fourth role when it names what a bp
// does while another bp proposes, and made "boot" look like a way to connect
// nodes when pn is that.
//
// They are refused now, and the message says what to write instead.
func TestNormalizeRole_RefusesTheRetiredSpellings(t *testing.T) {
	for _, retired := range []string{"validator", "endpoint", "boot"} {
		got, err := node.NormalizeRole(retired)
		if err == nil {
			t.Errorf("NormalizeRole(%q) = %q, want an error", retired, got)
			continue
		}
		for _, want := range []string{"bp", "en", "pn"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("NormalizeRole(%q) error %q does not name %q", retired, err, want)
			}
		}
	}
}

// TestNormalizeRole_AcceptsTheVocabulary pins the other half: the three words
// resolve to themselves, and nothing else resolves at all.
func TestNormalizeRole_AcceptsTheVocabulary(t *testing.T) {
	for _, r := range []node.Role{node.RoleBP, node.RoleEN, node.RolePN} {
		got, err := node.NormalizeRole(string(r))
		if err != nil {
			t.Errorf("NormalizeRole(%q): %v", r, err)
			continue
		}
		if got != r {
			t.Errorf("NormalizeRole(%q) = %q, want %q", r, got, r)
		}
	}
	if _, err := node.NormalizeRole(""); err == nil {
		t.Error("NormalizeRole(\"\") must be an error: an empty role is a missing declaration")
	}
}
