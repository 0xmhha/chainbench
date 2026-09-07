package blueprint

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// nodeName is what a node may be called. It becomes a datadir, a config file
// and a log file (node.Layout), so it has to survive being a path segment.
var nodeName = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// hexAddress and hexKey are the two written-out forms the document accepts.
var (
	hexAddress = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	hexKey     = regexp.MustCompile(`^(0x)?[0-9a-fA-F]{64}$`)
)

// Validate checks a blueprint against itself.
//
// Against ITSELF is the whole boundary. Whether the chain is registered,
// whether a keystore file is there, whether the ports are free: none of that is
// answerable from the document, and answering it here would put the same
// question in two places. What is checked is what the document alone decides —
// that its names are usable, that its references point at nodes it declares,
// and that it does not say two contradictory things.
func (bp Blueprint) Validate() error {
	if bp.Version != 0 && bp.Version != Version {
		return fmt.Errorf("blueprint: version %d is not supported (this build reads version %d)", bp.Version, Version)
	}
	if bp.Chain != "" && bp.Manifest != "" {
		return fmt.Errorf("blueprint: chain %q and manifest %q both name the chain — keep one", bp.Chain, bp.Manifest)
	}
	// Neither is allowed: Resolve decides where the chain comes from, and a
	// partial document that has not said yet is the normal case.

	if _, err := node.ParsePeering(bp.Peering); err != nil {
		return fmt.Errorf("blueprint: %w", err)
	}

	declared, err := bp.validateNodes()
	if err != nil {
		return err
	}
	if err := bp.validateBinaries(declared); err != nil {
		return err
	}
	if err := bp.validateValidators(declared); err != nil {
		return err
	}
	return bp.validateAlloc(declared)
}

// validateNodes checks each node and returns the set of names declared.
func (bp Blueprint) validateNodes() (map[string]bool, error) {
	declared := make(map[string]bool, len(bp.Nodes))
	for i, n := range bp.Nodes {
		where := fmt.Sprintf("node %d", i+1)
		if n.Name != "" {
			where = fmt.Sprintf("node %q", n.Name)
			if !nodeName.MatchString(n.Name) {
				return nil, fmt.Errorf("blueprint: %s: a name is lower-case and starts with a letter (it becomes a directory)", where)
			}
			if declared[n.Name] {
				return nil, fmt.Errorf("blueprint: two nodes are named %q", n.Name)
			}
			declared[n.Name] = true
		}
		role, err := n.role()
		if err != nil {
			return nil, fmt.Errorf("blueprint: %s: %w", where, err)
		}
		if n.Account != nil {
			if n.Account.Keystore == "" {
				return nil, fmt.Errorf("blueprint: %s: an account says nothing without a keystore", where)
			}
			// Only a sealing node holds one. Saying otherwise is a
			// misunderstanding worth reporting, not a field to drop quietly.
			if role != "" && !node.Is(role, node.RoleBP) {
				return nil, fmt.Errorf("blueprint: %s: role %s does not seal, so it has no account", where, n.Role)
			}
		}
		if err := n.NodeKey.validate(); err != nil {
			return nil, fmt.Errorf("blueprint: %s: %w", where, err)
		}
		if err := validatePorts(n.Ports); err != nil {
			return nil, fmt.Errorf("blueprint: %s: %w", where, err)
		}
	}
	return declared, nil
}

// role folds this node's declared role, or returns "" when it left the role out.
func (n Node) role() (node.Role, error) {
	if n.Role == "" {
		return "", nil
	}
	return node.NormalizeRole(n.Role)
}

// validate checks that a key reference names exactly one source.
func (k *NodeKeyRef) validate() error {
	if k == nil {
		return nil
	}
	switch {
	case k.File == "" && k.Hex == "":
		return fmt.Errorf("a nodekey needs a file or a hex value")
	case k.File != "" && k.Hex != "":
		return fmt.Errorf("a nodekey names a file or a hex value, not both")
	case k.Hex != "" && !hexKey.MatchString(k.Hex):
		return fmt.Errorf("a nodekey hex value is 32 bytes (64 hex digits)")
	}
	return nil
}

// validatePorts rejects a port number no socket can carry.
func validatePorts(p *node.Endpoints) error {
	if p == nil {
		return nil
	}
	for _, c := range []struct {
		name string
		port int
	}{
		{"p2p", p.P2P}, {"etcd", p.Etcd}, {"etcd_client", p.EtcdClient},
		{"http", p.HTTP}, {"ws", p.WS}, {"auth", p.Auth}, {"metrics", p.Metrics},
	} {
		if c.port < 0 || c.port > 65535 {
			return fmt.Errorf("%s port %d is outside 1-65535", c.name, c.port)
		}
	}
	return nil
}

// validateBinaries checks that an override covers nodes the document declares.
func (bp Blueprint) validateBinaries(declared map[string]bool) error {
	if bp.Binaries == nil {
		return nil
	}
	for i, o := range bp.Binaries.Overrides {
		if len(o.Nodes) == 0 {
			return fmt.Errorf("blueprint: binaries override %d names no nodes", i+1)
		}
		if o.Node == "" {
			return fmt.Errorf("blueprint: binaries override %d names no binary", i+1)
		}
		if err := known(declared, o.Nodes, fmt.Sprintf("binaries override %d", i+1)); err != nil {
			return err
		}
	}
	return nil
}

// validateValidators checks the sealing set names one source and real nodes.
func (bp Blueprint) validateValidators(declared map[string]bool) error {
	v := bp.Validators
	if v == nil {
		return nil
	}
	if v.From != "" && len(v.Explicit) > 0 {
		return fmt.Errorf("blueprint: validators are derived from %q or listed explicitly, not both", v.From)
	}
	if v.From != "" && v.From != "role" {
		return fmt.Errorf("blueprint: validators from %q is not a rule this build knows (want role)", v.From)
	}
	return known(declared, v.Explicit, "validators")
}

// validateAlloc checks each pre-funded balance names one subject.
func (bp Blueprint) validateAlloc(declared map[string]bool) error {
	for i, a := range bp.Alloc {
		where := fmt.Sprintf("alloc %d", i+1)
		switch {
		case a.Account == "" && a.Address == "":
			return fmt.Errorf("blueprint: %s names neither an account nor an address", where)
		case a.Account != "" && a.Address != "":
			return fmt.Errorf("blueprint: %s names both account %q and address %q — keep one", where, a.Account, a.Address)
		case a.Address != "" && !hexAddress.MatchString(a.Address):
			return fmt.Errorf("blueprint: %s: %q is not a 20-byte address", where, a.Address)
		}
		if a.Account != "" {
			if err := known(declared, []string{a.Account}, where); err != nil {
				return err
			}
		}
	}
	return nil
}

// known reports the first reference that names no declared node.
//
// It lists what IS declared. A reference that resolves to nothing is almost
// always a typo, and the answer a person needs is the spelling they meant.
func known(declared map[string]bool, refs []string, where string) error {
	for _, r := range refs {
		if declared[r] {
			continue
		}
		names := make([]string, 0, len(declared))
		for n := range declared {
			names = append(names, n)
		}
		if len(names) == 0 {
			return fmt.Errorf("blueprint: %s names node %q, but the document declares no named nodes", where, r)
		}
		sort.Strings(names)
		return fmt.Errorf("blueprint: %s names node %q; the document declares %s", where, r, strings.Join(names, ", "))
	}
	return nil
}
