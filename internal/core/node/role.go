package node

import (
	"fmt"
)

// NormalizeRole folds a role spelling onto the canonical vocabulary
// (bp / en / pn). The legacy spellings keep working because they
// are written into persisted workspaces and topology files; unknown spellings
// are an error rather than a silently-invented role.
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

// LegacySpelling maps a canonical role back to the spelling persisted state
// and the launch flows still carry ("bp" → "validator").
//
// It exists only for the transition: the composition still writes and compares
// the legacy words. Flipping them is a migration of its own — deferred because
// Is makes both spellings safe to compare, while the flip changes argv
// and persisted workspaces and so needs its own live re-verification (tracked
// as NM6 in docs/dev/netmap-design.md). When that migration lands, this
// function goes with it.
func LegacySpelling(r Role) Role {
	switch r {
	case RoleBP:
		return RoleValidator
	case RoleEN:
		return RoleEndpoint
	default:
		return r
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
