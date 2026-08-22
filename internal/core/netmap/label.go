package netmap

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// NodeLabel names one node in a network: "node1", "bp1".
//
// It is a named type because it crosses package boundaries and keys maps, and
// because the DSL, the logs, and the workspace must all mean the same node by
// the same string. It is deliberately not keyring.Label — a node and a ring
// entry are different concepts that sometimes share a spelling.
type NodeLabel string

// labelPrefix is the indexed-label convention the DSL and the workspace
// already use ("node1", "node2", ...).
const labelPrefix = "node"

// LabelFor returns the conventional label for a 1-based node index. It is the
// rule the DSL used to hard-code as "node"+itoa(i); owning it here means a
// change to the convention is one edit, and the DSL follows.
func LabelFor(index int) NodeLabel {
	return NodeLabel(labelPrefix + strconv.Itoa(index))
}

// Index returns the 1-based index of a conventional label, or an error for a
// label that does not follow the indexed convention — a named label ("faucet")
// has no index, and inventing one would misaddress a node.
func (l NodeLabel) Index() (int, error) {
	rest, ok := strings.CutPrefix(string(l), labelPrefix)
	if !ok {
		return 0, fmt.Errorf("netmap: %q is not an indexed node label", l)
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("netmap: %q is not an indexed node label", l)
	}
	return n, nil
}

// RoleLabel returns a node's role-scoped alias: the ord-th node of that role,
// 1-based ("en2" is the second endpoint).
//
// The alias is not the identity — the identity is the node's index, and
// LabelFor spells it ("node7"). Both name the same node and both are accepted
// wherever a node is addressed, because they answer different questions: an
// operator counting a network reads indexes, while a test definition written
// once and run on many topologies has to say "an endpoint", not "the seventh
// node". Only the identity reaches disk (datadir, log file, keyring entry), so
// the alias can be re-derived when a network is composed differently.
func RoleLabel(role node.Role, ord int) NodeLabel {
	return NodeLabel(string(role) + strconv.Itoa(ord))
}

// ParseRoleLabel splits a role-scoped alias into its canonical role and 1-based
// ordinal. Legacy spellings fold like everywhere else ("validator1" is bp 1).
//
// An indexed identity label ("node7") is not a role alias: "node" is not a
// role, so it returns an error here and is resolved through Index instead.
func ParseRoleLabel(l NodeLabel) (node.Role, int, error) {
	s := string(l)
	split := strings.LastIndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) + 1
	if split == 0 || split == len(s) {
		return "", 0, fmt.Errorf("netmap: %q is not a role label (want a role and a 1-based ordinal, e.g. en2)", l)
	}
	ord, err := strconv.Atoi(s[split:])
	if err != nil || ord < 1 {
		return "", 0, fmt.Errorf("netmap: %q has no 1-based ordinal", l)
	}
	role, err := NormalizeRole(s[:split])
	if err != nil {
		return "", 0, fmt.Errorf("netmap: %q: %w", l, err)
	}
	return role, ord, nil
}
