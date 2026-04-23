package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

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

// SaveRemote persists a remote attached network under
// <stateDir>/networks/<name>.json. The write is atomic (temp file + rename).
// Overwriting an existing file is allowed.
func SaveRemote(stateDir string, net *types.Network) error {
	if net == nil {
		return fmt.Errorf("state: nil network")
	}
	if net.Name == "local" {
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
	if name == "local" {
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
	return &net, nil
}
