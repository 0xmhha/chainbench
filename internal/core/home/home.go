// Package home is where chainbench keeps what it makes.
//
// It exists because "the default location" was three different answers. A key
// set defaulted to keys/default, a session to chainbench-out, and a
// composition to ~/.chainbench/<stamp> — the first two relative to whatever
// directory the operator happened to be standing in. So a ring created in one
// place was invisible from another, and sessions scattered across the
// filesystem one cwd at a time.
//
// The rule is the one the requirement states: name a path and the assets go
// there; name none and they go to one promised place. This package owns that
// place, and it sits at the bottom so the key store, the composition and the
// surfaces can all reach the same answer instead of each keeping their own.
package home

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir is the directory under the operator's home that holds chainbench's own
// state: key sets, compositions, and session artifacts.
const Dir = ".chainbench"

// Root is the promised location: ~/.chainbench.
func Root() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home: no home directory to place chainbench's files under: %w", err)
	}
	return filepath.Join(h, Dir), nil
}

// Under joins parts beneath the root, so a caller names what it keeps rather
// than where the root is.
func Under(parts ...string) (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{root}, parts...)...), nil
}

// KeySets is where key sets live when none is named: ~/.chainbench/keys.
func KeySets() (string, error) { return Under("keys") }

// Sessions is where session artifacts land when no root is named:
// ~/.chainbench/sessions. A composition keeps its own sessions beside its
// workspace instead; this is the answer for a run that has no composition —
// attaching to a network somebody else is running.
func Sessions() (string, error) { return Under("sessions") }
