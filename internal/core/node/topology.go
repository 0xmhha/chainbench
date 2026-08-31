// Package topology is the config-file model for a local test network's node
// layout: which node index plays which role (block producer / endpoint / boot),
// its sync mode (full / snap / archive), and which node is the boot It
// replaces the positional "N validators + M endpoints" counts with an explicit,
// per-node, inspectable description, so a test's environment is set up by one
// consistent rule and you can tell from the file exactly how the chain is
// configured.
//
// It is a pure config model (load + validate + normalize); the setup pipeline
// consumes a Topology to build the per-node launch specs.
package node

import (
	"fmt"
	"os"
	"sort"

	"go.yaml.in/yaml/v3"
)

// Topology is the declarative node layout for a local network.
type Topology struct {
	Chain   string          `yaml:"chain"`
	Network string          `yaml:"network,omitempty"`
	Nodes   []TopologyEntry `yaml:"nodes"`
}

// Entry is one node as the topology file declares it — index, role, sync mode,
// bootnode flag. A declaration, not the node's fact record (Record): the
// composition reads this and writes that.
// it is the boot
type TopologyEntry struct {
	Index    int    `yaml:"index"`
	Role     string `yaml:"role"`                // bp|validator, en|endpoint, boot
	SyncMode string `yaml:"sync_mode,omitempty"` // full (default) | snap | archive
	Bootnode bool   `yaml:"bootnode,omitempty"`
}

// Load reads and validates a topology YAML file.
func LoadTopology(path string) (Topology, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Topology{}, fmt.Errorf("topology: read %s: %w", path, err)
	}
	var t Topology
	if err := yaml.Unmarshal(b, &t); err != nil {
		return Topology{}, fmt.Errorf("topology: parse %s: %w", path, err)
	}
	if err := t.Validate(); err != nil {
		return Topology{}, err
	}
	return t, nil
}

var validSyncModes = map[string]bool{"": true, "full": true, "snap": true, "archive": true}

// Validate checks the topology is internally consistent: at least one node, a
// contiguous 1..N index set, recognized roles and sync modes, at least one block
// producer, and at most one boot
func (t Topology) Validate() error {
	if t.Chain == "" {
		return fmt.Errorf("topology: chain is required")
	}
	if len(t.Nodes) == 0 {
		return fmt.Errorf("topology: at least one node is required")
	}
	idx := make([]int, 0, len(t.Nodes))
	producers, bootnodes := 0, 0
	for _, n := range t.Nodes {
		role, err := NormalizeRole(n.Role)
		if err != nil {
			return fmt.Errorf("topology: node %d has unknown role %q (want bp|validator, en|endpoint, boot)", n.Index, n.Role)
		}
		if !validSyncModes[n.SyncMode] {
			return fmt.Errorf("topology: node %d has unknown sync_mode %q (want full, snap, archive)", n.Index, n.SyncMode)
		}
		if role == RoleBP {
			producers++
		}
		if n.Bootnode {
			bootnodes++
		}
		idx = append(idx, n.Index)
	}
	if producers == 0 {
		return fmt.Errorf("topology: need at least one block-producer (bp/validator) node")
	}
	if bootnodes > 1 {
		return fmt.Errorf("topology: at most one bootnode (found %d)", bootnodes)
	}
	sort.Ints(idx)
	for i, v := range idx {
		if v != i+1 {
			return fmt.Errorf("topology: node indices must be a contiguous 1..%d set, got %v", len(idx), idx)
		}
	}
	return nil
}

// NodeRole returns the Role for n (validated by Validate), in the legacy
// spelling the composition still persists and compares. The alias table this
// package used to keep is gone — netmap owns the folding, and this method
// switches to the canonical spelling when the launch flows migrate to it.
func (n TopologyEntry) NodeRole() Role {
	role, err := NormalizeRole(n.Role)
	if err != nil {
		return "" // unreachable after Validate; an invalid role never launches
	}
	return LegacySpelling(role)
}

// EffectiveSyncMode returns n's sync mode, defaulting an unset value to "full".
func (n TopologyEntry) EffectiveSyncMode() string {
	if n.SyncMode == "" {
		return "full"
	}
	return n.SyncMode
}

// Sorted returns the nodes ordered by index (ascending).
func (t Topology) Sorted() []TopologyEntry {
	out := append([]TopologyEntry(nil), t.Nodes...)
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

// BootnodeIndex returns the 1-based index of the bootnode, or 0 if none is set.
func (t Topology) BootnodeIndex() int {
	for _, n := range t.Nodes {
		if n.Bootnode {
			return n.Index
		}
	}
	return 0
}

// Counts returns the number of block-producer and endpoint nodes.
func (t Topology) Counts() (producers, endpoints int) {
	for _, n := range t.Nodes {
		switch n.NodeRole() {
		case RoleValidator, RoleBP:
			producers++
		case RoleEndpoint, RoleEN:
			endpoints++
		}
	}
	return producers, endpoints
}
