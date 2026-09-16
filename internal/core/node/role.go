package node

import (
	"fmt"
)

// NormalizeRole resolves a role to the vocabulary, or says why it cannot.
//
// There is nothing to fold. A role is written by a declaration or read from a
// record this build wrote, and both spell it bp, en or pn. The spellings this
// function used to accept — validator, endpoint, boot — are gone, along with
// the code that read them: keeping a reader for an old spelling kept teaching
// it, and a comment saying the old words "survive in persisted state" was read
// as fact long after nothing wrote them.
//
// An unknown spelling is an error rather than a silently-invented role. This is
// the one place that decides, so a caller never compares against whichever
// word it happens to know.
func NormalizeRole(s string) (Role, error) {
	switch Role(s) {
	case RoleBP, RoleEN, RolePN:
		return Role(s), nil
	default:
		return "", fmt.Errorf("node: unknown role %q (want %s, %s or %s)", s, RoleBP, RoleEN, RolePN)
	}
}

// Is reports whether role is the named one.
//
// Every decision that turns on a role asks this way rather than comparing
// strings. Comparing against one spelling is how a producer came to be
// launched without --mine and how a selector came to resolve to the wrong
// node, and asking here means an unknown role answers false instead of
// matching by accident.
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
