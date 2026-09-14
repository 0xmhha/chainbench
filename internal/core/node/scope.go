package node

import (
	"fmt"
	"regexp"
	"strconv"
)

// ScopeAll is the scope that names every node.
const ScopeAll = "all"

// indexedScopeRE matches the scope that names one node by its 1-based index.
// A leading zero is refused so "node01" cannot mean node 1 under one reader and
// nothing under another.
var indexedScopeRE = regexp.MustCompile(`^node([1-9][0-9]*)$`)

// A scope says which nodes a per-node value applies to. There are three forms,
// and a value set in more than one is applied most-general-first, so the
// narrowest scope wins:
//
//	"all"      every node
//	a role     every node in that role ("bp", "en", "pn")
//	"node<N>"  one node, 1-based ("node3")
//
// The rule lives here because the role form is this package's vocabulary, and
// because it was previously written out by hand in the grammar and again in the
// workspace. Both hand-written lists named bp and en and forgot pn, so a
// topology could declare a proxy tier that no launch flag could then address —
// the node was launchable but not configurable, and nothing said so.

// ValidScope reports whether s names a set of nodes.
func ValidScope(s string) bool {
	if s == ScopeAll {
		return true
	}
	if ScopeIndex(s) > 0 {
		return true
	}
	_, err := NormalizeRole(s)
	return err == nil
}

// ScopeIndex returns the 1-based node index s names, or 0 when s is not an
// indexed scope. Zero is not a valid index, so it is unambiguous as "no".
func ScopeIndex(s string) int {
	m := indexedScopeRE.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0 // unreachable: the pattern admits digits only
	}
	return n
}

// ScopeFor returns the scopes that apply to one node, most-general-first. A
// caller folds values in this order, so the last one that sets a key wins.
//
// An unreadable role contributes no scope rather than an empty one, which would
// otherwise look up the "" key and find whatever a caller happened to store
// there.
func ScopeFor(role Role, index int) []string {
	out := []string{ScopeAll}
	if canonical, err := NormalizeRole(string(role)); err == nil {
		out = append(out, string(canonical))
	}
	if index > 0 {
		out = append(out, fmt.Sprintf("node%d", index))
	}
	return out
}

// ScopeRank orders a scope from most general to most specific, so a caller
// that holds several can apply them in the order the fold expects without
// re-deriving which is which. A scope that is not one ranks last; it is not
// applied, and ranking it here keeps a sort total rather than undefined.
func ScopeRank(s string) int {
	switch {
	case s == ScopeAll:
		return 0
	case ScopeIndex(s) > 0:
		return 2
	default:
		if _, err := NormalizeRole(s); err == nil {
			return 1
		}
		return 3
	}
}

// ScopeWords says what a scope may be, so every surface refuses one in the same
// words and names the same roles.
func ScopeWords() string {
	return fmt.Sprintf("%q, a role (%s, %s, %s), or \"node<N>\"", ScopeAll, RoleBP, RoleEN, RolePN)
}
