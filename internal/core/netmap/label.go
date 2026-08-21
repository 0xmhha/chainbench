package netmap

import (
	"fmt"
	"strconv"
	"strings"
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
