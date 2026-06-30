package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// remoteNameRE mirrors network.json's name pattern: ^[a-z0-9][a-z0-9_-]*$
// Enforced at handler boundary and re-checked here (defense-in-depth against
// path traversal / reserved-name misuse).
var remoteNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ErrReservedName reports an attempt to save a network under the reserved "local" name.
var ErrReservedName = errors.New("state: 'local' is reserved for the local network")

// ErrInvalidName reports a network name that violates the schema pattern.
var ErrInvalidName = errors.New("state: network name must match [a-z0-9][a-z0-9_-]*")

// IsReservedRemoteName reports whether s is reserved by the state layer and
// cannot be used as a remote network name. Currently only "local" is reserved.
// Centralizes the reserved-name rule so IsValidRemoteName, SaveRemote, and
// loadRemote stay in sync if the reserved set grows.
func IsReservedRemoteName(s string) bool {
	return s == "local"
}

// IsValidRemoteName reports whether s is a structurally-valid remote network name.
// Matches the network.json schema pattern and rejects reserved names.
// Handlers use this for input validation before attempting a probe or state write.
func IsValidRemoteName(s string) bool {
	return !IsReservedRemoteName(s) && remoteNameRE.MatchString(s)
}

// SaveRemote persists a remote attached network under
// <stateDir>/networks/<name>.json. The write is atomic (temp file + rename).
// Overwriting an existing file is allowed.
func SaveRemote(stateDir string, net *types.Network) error {
	if net == nil {
		return fmt.Errorf("state: nil network")
	}
	if IsReservedRemoteName(net.Name) {
		return ErrReservedName
	}
	if !remoteNameRE.MatchString(net.Name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, net.Name)
	}
	dir := filepath.Join(stateDir, "networks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("state: mkdir networks: %w", err)
	}
	raw, err := json.MarshalIndent(net, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal network: %w", err)
	}
	finalPath := filepath.Join(dir, net.Name+".json")
	tmpPath := finalPath + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o644); err != nil {
		return fmt.Errorf("state: write temp: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("state: rename: %w", err)
	}
	return nil
}

// loadRemote is package-private; external callers use LoadActive which
// routes on Name.
func loadRemote(stateDir, name string) (*types.Network, error) {
	if IsReservedRemoteName(name) {
		return nil, ErrReservedName
	}
	if !remoteNameRE.MatchString(name) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	path := filepath.Join(stateDir, "networks", name+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: no attached network named %q", ErrStateNotFound, name)
		}
		return nil, fmt.Errorf("state: read %s: %w", path, err)
	}
	var net types.Network
	if err := json.Unmarshal(raw, &net); err != nil {
		return nil, fmt.Errorf("state: decode %s: %w", path, err)
	}
	// Integrity check: file contents must agree with filename stem, otherwise
	// a copy-paste or rename mistake silently serves the wrong network.
	if net.Name != name {
		return nil, fmt.Errorf("state: filename %q has mismatched network name %q", name, net.Name)
	}
	return &net, nil
}

// ListRemotes returns the attached networks under <stateDir>/networks/, sorted
// by name. The local network (pids.json) is not included. A missing networks
// directory yields an empty slice (not an error). Files that fail to parse or
// whose contents disagree with the filename are skipped rather than failing the
// whole listing.
func ListRemotes(stateDir string) ([]*types.Network, error) {
	dir := filepath.Join(stateDir, "networks")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.Network{}, nil
		}
		return nil, fmt.Errorf("state: read networks dir: %w", err)
	}
	nets := make([]*types.Network, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		stem := strings.TrimSuffix(name, ".json")
		net, err := loadRemote(stateDir, stem)
		if err != nil {
			continue // skip malformed / reserved / mismatched entries
		}
		nets = append(nets, net)
	}
	sort.Slice(nets, func(i, j int) bool { return nets[i].Name < nets[j].Name })
	return nets, nil
}

// RemoveRemote deletes <stateDir>/networks/<name>.json (the inverse of
// SaveRemote). Reserved/invalid names are rejected; a missing network yields a
// wrapped ErrStateNotFound.
func RemoveRemote(stateDir, name string) error {
	if IsReservedRemoteName(name) {
		return ErrReservedName
	}
	if !remoteNameRE.MatchString(name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	path := filepath.Join(stateDir, "networks", name+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: no attached network named %q", ErrStateNotFound, name)
		}
		return fmt.Errorf("state: remove %s: %w", path, err)
	}
	return nil
}
