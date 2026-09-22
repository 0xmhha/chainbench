package testengine

import (
	"github.com/0xmhha/chainbench/internal/core/lifecycle"

	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Reading a node table out of a declaration.
//
// A declaration says its shape one of two ways: counts by role, or a node list
// naming each one. The list form is what a mixed-binary network needs, since
// only it can say which build a given node runs. Both arrive here as the same
// request, so nothing downstream has to know which spelling was used.

func topologyHasKeys(t map[string]any) bool {
	list, ok := t["nodes"].([]any)
	if !ok {
		return false
	}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := m["key"].(string); ok && s != "" {
			return true
		}
	}
	return false
}

// topologyHasProxy reports whether a node-table topology declares a pn, so the
// composer selects the proxied graph for it exactly as it does for the count
// form. It is nil-safe: the count form passes no table.
func topologyHasProxy(t *node.Topology) bool {
	if t == nil {
		return false
	}
	for _, n := range t.Nodes {
		if node.Is(n.NodeRole(), node.RolePN) {
			return true
		}
	}
	return false
}

// inlineTopologyOf builds an in-memory node table from a topology.nodes[]
// declaration, so a spec can name each node's role and binary in one file. It
// returns nil when the declaration uses the count form (no nodes key), which
// keeps the existing validators/endpoints path.
//
// The second result maps each binary name a node references to its resolved
// path, drawn from the env's binaries map; the third is a fallback binary (the
// first node's) for a node that names none.
func inlineTopologyOf(chain string, t map[string]any, binaries map[string]string) (*node.Topology, map[string]string, string, error) {
	raw, ok := t["nodes"]
	if !ok {
		return nil, nil, "", nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, nil, "", lifecycle.Mark(errMalformed, fmt.Errorf("topology.nodes must be a list"))
	}
	if len(list) == 0 {
		return nil, nil, "", lifecycle.Mark(errIncomplete, fmt.Errorf("topology.nodes is empty"))
	}
	topo := &node.Topology{Chain: chain, Nodes: make([]node.Entry, 0, len(list))}
	resolved := map[string]string{}
	fallback := ""
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, nil, "", lifecycle.Mark(errMalformed, fmt.Errorf("topology.nodes[%d] must be a mapping", i))
		}
		entry := node.Entry{Index: i + 1}
		for k, v := range m {
			s, isStr := v.(string)
			switch k {
			case "role":
				if !isStr {
					return nil, nil, "", lifecycle.Mark(errMalformed, fmt.Errorf("topology.nodes[%d].role must be a string", i))
				}
				entry.Role = s
			case "binary":
				if !isStr {
					return nil, nil, "", lifecycle.Mark(errMalformed, fmt.Errorf("topology.nodes[%d].binary must be a string", i))
				}
				entry.Binary = s
			case "sync", topoSyncMode, topoSyncModeSnak:
				if !isStr {
					return nil, nil, "", lifecycle.Mark(errMalformed, fmt.Errorf("topology.nodes[%d].%s must be a string", i, k))
				}
				entry.SyncMode = s
			case "bootnode":
				b, isBool := v.(bool)
				if !isBool {
					return nil, nil, "", lifecycle.Mark(errMalformed, fmt.Errorf("topology.nodes[%d].bootnode must be a boolean", i))
				}
				entry.Bootnode = b
			case "config":
				if !isStr {
					return nil, nil, "", lifecycle.Mark(errMalformed, fmt.Errorf("topology.nodes[%d].config must be a string (a path to a pre-written config file)", i))
				}
				entry.Config = expand(s)
			case "key":
				if !isStr {
					return nil, nil, "", lifecycle.Mark(errMalformed, fmt.Errorf("topology.nodes[%d].key must be a string (a key file path or 0x-hex)", i))
				}
				entry.Key = expand(s)
			case "index":
				n, ferr := countOf("nodes[].index", v)
				if ferr != nil {
					return nil, nil, "", ferr
				}
				entry.Index = n
			default:
				return nil, nil, "", lifecycle.Mark(errMalformed, fmt.Errorf("topology.nodes[%d].%s is not a key the composer knows (role, binary, sync, bootnode, index, config, key)", i, k))
			}
		}
		if entry.Role == "" {
			return nil, nil, "", lifecycle.Mark(errIncomplete, fmt.Errorf("topology.nodes[%d] needs a role", i))
		}
		if entry.Binary != "" {
			path, named := binaries[entry.Binary]
			if !named {
				return nil, nil, "", lifecycle.Mark(errUnknownName, fmt.Errorf("topology.nodes[%d].binary %q is not declared in binaries", i, entry.Binary))
			}
			p := expand(path)
			resolved[entry.Binary] = p
			if fallback == "" {
				fallback = p
			}
		}
		topo.Nodes = append(topo.Nodes, entry)
	}
	if err := topo.Validate(); err != nil {
		return nil, nil, "", err
	}
	return topo, resolved, fallback, nil
}

// Topology keys a declaration may use for its node counts.
const (
	topoBP           = "bp"
	topoEN           = "en"
	topoPN           = "pn"
	topoSyncMode     = "syncMode"
	topoSyncModeSnak = "sync_mode"
	// topoMax is the bp value that fills the network to the server set instead
	// of naming a count: bp becomes one node per server, less the pn and en.
	topoMax = "max"
)

// topologyOf reads the node counts a declaration gives: validators (or bp),
// endpoints (or en), and the endpoints' sync mode. A key it does not know is
// an error rather than a silently ignored intention.
//
// bp may be the word "max" instead of a number: autoBP is then true and the
// validator count is left for the composer to fill from the server set.
func topologyOf(t map[string]any) (validators, endpoints, proxies int, syncMode string, autoBP bool, err error) {
	for k, v := range t {
		switch k {
		case topoBP:
			if s, ok := v.(string); ok {
				if s != topoMax {
					return 0, 0, 0, "", false, fmt.Errorf("topology.%s must be a number or %q, got %q", k, topoMax, s)
				}
				autoBP = true
				break
			}
			validators, err = countOf(k, v)
		case topoEN:
			endpoints, err = countOf(k, v)
		case topoPN:
			proxies, err = countOf(k, v)
		case topoSyncMode, topoSyncModeSnak:
			s, ok := v.(string)
			if !ok {
				err = lifecycle.Mark(errMalformed, fmt.Errorf("topology.%s must be a string", k))
			}
			syncMode = s
		default:
			err = lifecycle.Mark(errMalformed, fmt.Errorf("topology.%s is not a key the composer knows (bp, en, pn, syncMode)", k))
		}
		if err != nil {
			return 0, 0, 0, "", false, err
		}
	}
	return validators, endpoints, proxies, syncMode, autoBP, nil
}

// countOf reads a node count, which JSON hands over as a float.
func countOf(key string, v any) (int, error) {
	switch n := v.(type) {
	case float64:
		if n < 0 || n != float64(int(n)) {
			return 0, lifecycle.Mark(errMalformed, fmt.Errorf("topology.%s must be a whole non-negative number, got %v", key, v))
		}
		return int(n), nil
	case int:
		if n < 0 {
			return 0, lifecycle.Mark(errMalformed, fmt.Errorf("topology.%s must be non-negative, got %d", key, n))
		}
		return n, nil
	default:
		return 0, lifecycle.Mark(errMalformed, fmt.Errorf("topology.%s must be a number, got %T", key, v))
	}
}

// hardforkSets renders declared fork heights as the genesis step's config
// overrides: {"boho": 10} becomes bohoBlock=10. Sorted, so the recorded step
// detail is stable.
