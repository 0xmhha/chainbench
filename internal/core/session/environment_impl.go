package session

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// env is the concrete Environment: a shared chain instance identified by its
// fingerprint, and the source of truth for node/endpoint resolution.
type env struct {
	id       string
	dir      string
	fp       Fingerprint
	dataPath string

	mu    sync.Mutex
	nodes []node.Node
}

func (e *env) ID() string               { return e.id }
func (e *env) Dir() string              { return e.dir }
func (e *env) Fingerprint() Fingerprint { return e.fp }
func (e *env) DataPath() string         { return e.dataPath }

// LogPath returns the tail-accumulated log path for a node.
func (e *env) LogPath(nodeName string) string {
	return filepath.Join(e.dir, dirLogs, nodeName+".log")
}

// ChainstateDir returns the directory holding collected chainstate.
func (e *env) ChainstateDir() string { return filepath.Join(e.dir, dirChainstate) }

// PopulateNodeTable fills the node table from a bring-up result.
func (e *env) PopulateNodeTable(ns node.NodeSet) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nodes = append(e.nodes[:0], ns.Nodes...)
}

// Nodes returns a copy of the node table.
func (e *env) Nodes() []node.Node {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]node.Node(nil), e.nodes...)
}

// Resolve maps a selector to a node. Forms: "node7" (identity), "bp1"/"en2"
// (role alias, 1-based ordinal), "bp:any" (first of role), "en:0" (role +
// 0-based index).
//
// Both spellings resolve because they answer different questions — see
// netmap.RoleLabel. The identity form is checked first: "node" is not a role,
// so the two grammars cannot collide.
func (e *env) Resolve(selector string) (node.Node, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if idx, err := netmap.NodeLabel(selector).Index(); err == nil {
		for _, n := range e.nodes {
			if n.Index == idx {
				return n, nil
			}
		}
		return node.Node{}, fmt.Errorf("session: selector %q names node %d, which this environment does not have (%d nodes)", selector, idx, len(e.nodes))
	}

	token, idx, anyForm, err := parseSelector(selector)
	if err != nil {
		return node.Node{}, err
	}
	roles := rolesForToken(token)
	if roles == nil {
		return node.Node{}, fmt.Errorf("session: unknown role %q in selector %q", token, selector)
	}

	var matched []node.Node
	for _, n := range e.nodes {
		if roles[n.Role] {
			matched = append(matched, n)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].Index < matched[j].Index })

	if anyForm {
		idx = 0
	}
	if idx < 0 || idx >= len(matched) {
		return node.Node{}, fmt.Errorf("session: selector %q out of range (%d nodes of role %q)", selector, len(matched), token)
	}
	return matched[idx], nil
}

// ResolveEach resolves several selectors, preserving order.
func (e *env) ResolveEach(selectors []string) ([]node.Node, error) {
	out := make([]node.Node, 0, len(selectors))
	for _, sel := range selectors {
		n, err := e.Resolve(sel)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// envDoc is the env.json schema.
type envDoc struct {
	EnvID       string      `json:"envId"`
	Fingerprint string      `json:"fingerprint"`
	DataPath    string      `json:"dataPath"`
	Nodes       []node.Node `json:"nodes"`
}

// Save writes env.json (fingerprint + node table + data path).
func (e *env) Save() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return writeJSON(filepath.Join(e.dir, fileEnv), envDoc{
		EnvID:       e.id,
		Fingerprint: string(e.fp),
		DataPath:    e.dataPath,
		Nodes:       e.nodes,
	})
}

// parseSelector splits a selector into a role token and a 0-based index. anyForm
// is true for "role:any". Name form "bpN" is converted to a 0-based index (N-1).
func parseSelector(sel string) (token string, index int, anyForm bool, err error) {
	if i := strings.IndexByte(sel, ':'); i >= 0 {
		token = sel[:i]
		rest := sel[i+1:]
		if rest == "any" {
			return token, 0, true, nil
		}
		n, e := strconv.Atoi(rest)
		if e != nil {
			return "", 0, false, fmt.Errorf("session: bad selector index in %q", sel)
		}
		return token, n, false, nil
	}
	split := strings.LastIndexFunc(sel, func(r rune) bool { return r < '0' || r > '9' }) + 1
	if split == 0 || split == len(sel) {
		return "", 0, false, fmt.Errorf("session: selector %q must be name (bp1) or role:index (en:0)", sel)
	}
	n, e := strconv.Atoi(sel[split:])
	if e != nil || n < 1 {
		return "", 0, false, fmt.Errorf("session: bad selector ordinal in %q", sel)
	}
	return sel[:split], n - 1, false, nil
}

// rolesForToken maps a DSL role token to the node roles it selects. Returns nil
// for an unknown token.
// rolesForToken folds a selector's role word onto every spelling a node may
// actually carry. The folding itself belongs to netmap — this function only
// adds the legacy spelling, because a running network can hold either while
// NM3 is in flight: a node composed today records "validator", one composed
// after the switch records "bp", and a selector has to match both or a spec
// silently addresses nothing.
func rolesForToken(tok string) map[node.Role]bool {
	canonical, err := netmap.NormalizeRole(tok)
	if err != nil {
		return nil
	}
	roles := map[node.Role]bool{canonical: true}
	if legacy := netmap.LegacySpelling(canonical); legacy != canonical {
		roles[legacy] = true
	}
	return roles
}
