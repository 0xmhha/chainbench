// Named-network registry (part of package session): the attached
// (already-running) networks chainbench knows about, recorded under the
// networks directory as one JSON file per name, so a later command or MCP call
// can address a network by name instead of repeating its endpoints.
//
// It backs the live network attach/list/info/detach surface. Its lifetime
// differs from the session's other state: a session is one run's artifacts, a
// composition is one workspace, and this is a machine-wide inventory keyed by
// name. (Formerly the standalone core/state package, folded into session in R1.)
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// networkNameRE is the allowed remote-network name pattern; it doubles as a
// defense against path traversal since names become file stems.
var networkNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Errors from the named-network registry. Callers classify with errors.Is.
var (
	// ErrReservedName rejects the reserved "local" name (the per-datadir nodeset).
	ErrReservedName = errors.New("session: 'local' is reserved for the local network")
	// ErrInvalidName rejects a name that violates the pattern.
	ErrInvalidName = errors.New("session: network name must match [a-z0-9][a-z0-9_-]*")
	// ErrNetworkNotFound reports no attached network under the given name.
	ErrNetworkNotFound = errors.New("session: no attached network")
)

// IsReservedNetworkName reports whether s is reserved by the state layer and
// cannot name a saved network. Only "local" (the per-datadir nodeset) is reserved.
func IsReservedNetworkName(s string) bool { return s == "local" }

// IsValidNetworkName reports whether s is a structurally valid remote network
// name (matches the pattern and is not reserved). Handlers use this to validate
// input before a probe or state write.
func IsValidNetworkName(s string) bool {
	return !IsReservedNetworkName(s) && networkNameRE.MatchString(s)
}

// networksDir is the directory holding one JSON file per named network.
func networksDir(stateDir string) string { return filepath.Join(stateDir, "networks") }

// SaveNetwork persists a named (attached/remote) network under
// <stateDir>/networks/<ns.Network>.json. The write is atomic (temp + rename);
// overwriting an existing entry is allowed. The reserved "local" name and
// invalid names are rejected.
func SaveNetwork(stateDir string, ns node.NodeSet) error {
	name := ns.Network
	if IsReservedNetworkName(name) {
		return ErrReservedName
	}
	if !networkNameRE.MatchString(name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	dir := networksDir(stateDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("session: mkdir networks: %w", err)
	}
	raw, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal network: %w", err)
	}
	final := filepath.Join(dir, name+".json")
	if err := WriteFileAtomic(final, raw, 0o644); err != nil {
		return fmt.Errorf("session: write network %s: %w", name, err)
	}
	return nil
}

// LoadNetwork reads the named network from <stateDir>/networks/<name>.json. A
// missing entry yields a wrapped ErrNetworkNotFound. The file's Network field
// must agree with the filename stem, else a rename/copy mistake would silently
// serve the wrong network.
func LoadNetwork(stateDir, name string) (node.NodeSet, error) {
	if IsReservedNetworkName(name) {
		return node.NodeSet{}, ErrReservedName
	}
	if !networkNameRE.MatchString(name) {
		return node.NodeSet{}, fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	path := filepath.Join(networksDir(stateDir), name+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return node.NodeSet{}, fmt.Errorf("%w named %q", ErrNetworkNotFound, name)
		}
		return node.NodeSet{}, fmt.Errorf("session: read %s: %w", path, err)
	}
	var ns node.NodeSet
	if err := json.Unmarshal(raw, &ns); err != nil {
		return node.NodeSet{}, fmt.Errorf("session: decode %s: %w", path, err)
	}
	if ns.Network != name {
		return node.NodeSet{}, fmt.Errorf("session: filename %q has mismatched network name %q", name, ns.Network)
	}
	return ns, nil
}

// ListNetworks returns the saved networks under <stateDir>/networks/, sorted by
// name. A missing directory yields an empty slice, not an error. Entries that
// fail to parse or whose contents disagree with the filename are skipped rather
// than failing the whole listing.
func ListNetworks(stateDir string) ([]node.NodeSet, error) {
	entries, err := os.ReadDir(networksDir(stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return []node.NodeSet{}, nil
		}
		return nil, fmt.Errorf("session: read networks dir: %w", err)
	}
	nets := make([]node.NodeSet, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		ns, err := LoadNetwork(stateDir, strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue // skip malformed / reserved / mismatched entries
		}
		nets = append(nets, ns)
	}
	sort.Slice(nets, func(i, j int) bool { return nets[i].Network < nets[j].Network })
	return nets, nil
}

// RemoveNetwork deletes <stateDir>/networks/<name>.json (the inverse of
// SaveNetwork). Reserved/invalid names are rejected; a missing entry yields a
// wrapped ErrNetworkNotFound.
func RemoveNetwork(stateDir, name string) error {
	if IsReservedNetworkName(name) {
		return ErrReservedName
	}
	if !networkNameRE.MatchString(name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	path := filepath.Join(networksDir(stateDir), name+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w named %q", ErrNetworkNotFound, name)
		}
		return fmt.Errorf("session: remove %s: %w", path, err)
	}
	return nil
}
