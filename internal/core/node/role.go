package node

import (
	"fmt"
)

// NormalizeRole folds a role spelling onto the canonical vocabulary
// (bp / en / pn). The legacy spellings keep working because they are written
// into topology files and into workspaces composed before NM6; unknown
// spellings are an error rather than a silently-invented role.
//
// Since NM6 the folding runs one way only. Nothing emits "validator" or
// "endpoint" any more, so there is no mapping back: [Node.UnmarshalJSON] and
// [Entry.NodeRole] fold what is read, and everything above them works in the
// canonical vocabulary alone.
//
// This is the one place the folding lives. Before it, topology kept its own
// alias table and every consumer compared against whichever spelling it
// happened to know.
func NormalizeRole(s string) (Role, error) {
	switch Role(s) {
	case RoleBP, RoleValidator:
		return RoleBP, nil
	case RoleEN, RoleEndpoint:
		return RoleEN, nil
	case RolePN:
		return RolePN, nil
	case RoleBoot:
		// Still a role until the poa bring-up treats boot as an attribute.
		return RoleBoot, nil
	default:
		return "", fmt.Errorf("node: unknown role %q (want bp, en, pn, or a legacy spelling)", s)
	}
}

// Is reports whether role names canonical under any spelling — Is(r, RoleBP)
// is true for "bp" and for the legacy "validator".
//
// Every decision that turns on a role has to ask this way. Comparing against
// one spelling is how a producer came to be launched without --mine and how a
// selector came to resolve to the wrong node: both compared against the word
// they happened to know, and neither failed until something else emitted the
// other word.
func Is(role, canonical Role) bool {
	got, err := NormalizeRole(string(role))
	if err != nil {
		return false
	}
	want, err := NormalizeRole(string(canonical))
	if err != nil {
		return false
	}
	return got == want
}
