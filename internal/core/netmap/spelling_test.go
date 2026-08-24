package netmap_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestIs_FoldsBothSpellings is the predicate every role decision has to go
// through. Two defects in this series came from comparing against one spelling:
// a producer launched without --mine (chain stalls, every node healthy) and a
// selector that resolved bp1 to the wrong node. Neither failed until something
// else started emitting the other word.
func TestIs_FoldsBothSpellings(t *testing.T) {
	for _, c := range []struct {
		role, canonical node.Role
		want            bool
	}{
		{node.RoleBP, node.RoleBP, true},
		{node.RoleValidator, node.RoleBP, true},
		{node.RoleBP, node.RoleValidator, true}, // symmetric: either side may be legacy
		{node.RoleEN, node.RoleEN, true},
		{node.RoleEndpoint, node.RoleEN, true},
		{node.RolePN, node.RolePN, true},
		{node.RoleEN, node.RoleBP, false},
		{node.RolePN, node.RoleEN, false},
		{node.RoleBoot, node.RoleBP, false}, // boot is its own role until it becomes an attribute of a bp
		{node.Role("sideways"), node.RoleBP, false},
		{node.RoleBP, node.Role("sideways"), false},
	} {
		if got := netmap.Is(c.role, c.canonical); got != c.want {
			t.Errorf("Is(%q, %q) = %v, want %v", c.role, c.canonical, got, c.want)
		}
	}
}
