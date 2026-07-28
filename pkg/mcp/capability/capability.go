// Package capability is the layered, per-project feature registry behind the
// chainbench MCP surface. It answers "which features does chain X support, and
// how are they called" as data, so the tool set grows with chains/projects
// without editing a central switch.
//
// The model is hierarchical: a capability is addressed by
// <version>.<chain>.<name> (e.g. "v1.common.faucet.send" or
// "v1.stablenet.governance.propose_mint"). "common" capabilities apply to every
// chain and are implemented once (pkg/mcp/features/common); chain-specific ones
// live in that chain's project package (pkg/mcp/features/<chain>). Each project
// ships a declarative catalog (a .jsonl list of Descriptors) plus handlers, and
// registers both here at init(). A capability is EXPOSED only if its catalog
// entry has a bound handler — so the exposed surface is exactly what projects
// have registered.
package capability

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// CommonChain is the chain value for capabilities shared by every chain.
const CommonChain = "common"

// Param is one input parameter of a capability (for schema + discovery).
type Param struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Desc     string `json:"desc,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// Descriptor is a declarative catalog entry, decoded from a project's .jsonl.
type Descriptor struct {
	Version string  `json:"version"`
	Chain   string  `json:"chain"` // "common" or a chain id
	Name    string  `json:"name"`  // dotted feature name, e.g. "tx.send"
	Summary string  `json:"summary"`
	Params  []Param `json:"params"`
}

// Address is the hierarchical identifier "<version>.<chain>.<name>".
func (d Descriptor) Address() string { return d.Version + "." + d.Chain + "." + d.Name }

// Handler runs a capability with decoded args, returning agent-readable text.
type Handler func(ctx context.Context, args map[string]any) (string, error)

// Capability is a catalog descriptor bound to its handler.
type Capability struct {
	Descriptor
	Handler Handler
}

var (
	mu       sync.Mutex
	catalog  = map[string]Descriptor{} // address -> descriptor
	handlers = map[string]Handler{}    // address -> handler
)

// LoadCatalog parses a project's .jsonl catalog (one Descriptor per non-empty
// line) and records the descriptors. Call from a project package's init().
func LoadCatalog(jsonl []byte) error {
	mu.Lock()
	defer mu.Unlock()
	sc := bufio.NewScanner(bytes.NewReader(jsonl))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		b := bytes.TrimSpace(sc.Bytes())
		if len(b) == 0 || b[0] == '#' {
			continue
		}
		var d Descriptor
		if err := json.Unmarshal(b, &d); err != nil {
			return fmt.Errorf("capability: catalog line %d: %w", line, err)
		}
		if d.Version == "" || d.Chain == "" || d.Name == "" {
			return fmt.Errorf("capability: catalog line %d: version/chain/name required", line)
		}
		catalog[d.Address()] = d
	}
	return sc.Err()
}

// RegisterHandler binds a handler to a capability address. Call from a project
// package's init(), alongside LoadCatalog.
func RegisterHandler(version, chain, name string, h Handler) {
	mu.Lock()
	defer mu.Unlock()
	handlers[version+"."+chain+"."+name] = h
}

// All returns every EXPOSED capability (catalog entry with a bound handler),
// sorted by address for deterministic output.
func All() []Capability {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Capability, 0, len(catalog))
	for addr, d := range catalog {
		if h, ok := handlers[addr]; ok {
			out = append(out, Capability{Descriptor: d, Handler: h})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Address() < out[j].Address() })
	return out
}

// For returns the capabilities available to a chain: the common set plus that
// chain's own. An empty chain returns only the common set.
func For(chain string) []Capability {
	out := make([]Capability, 0)
	for _, c := range All() {
		if c.Chain == CommonChain || (chain != "" && c.Chain == chain) {
			out = append(out, c)
		}
	}
	return out
}

// Lookup returns the exposed capability at address, if any.
func Lookup(address string) (Capability, bool) {
	mu.Lock()
	defer mu.Unlock()
	d, ok := catalog[address]
	if !ok {
		return Capability{}, false
	}
	h, ok := handlers[address]
	if !ok {
		return Capability{}, false
	}
	return Capability{Descriptor: d, Handler: h}, true
}
