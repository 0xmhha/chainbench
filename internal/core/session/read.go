package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// List returns the session IDs under root — directories that hold a session.json
// — sorted ascending (session IDs are timestamp-ordered, so this is oldest
// first). A missing root yields no sessions rather than an error, so a dashboard
// can point at an artifact root before any run has written to it.
func List(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("session: list %s: %w", root, err)
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, statErr := os.Stat(filepath.Join(root, e.Name(), fileSession)); statErr == nil {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// SessionDir returns a session's directory under root.
func SessionDir(root, id string) string {
	return filepath.Join(root, id)
}

// SessionFilePath returns the path to a session's session.json under root.
func SessionFilePath(root, id string) string {
	return filepath.Join(root, id, fileSession)
}

// ChainstatePaths returns the chainstate jsonl files persisted across a session's
// environments, sorted. It returns no paths (not an error) when the session has
// no collected chainstate.
func ChainstatePaths(root, id string) ([]string, error) {
	glob := filepath.Join(root, id, dirEnvironments, "*", dirChainstate, "*.jsonl")
	matches, err := filepath.Glob(glob)
	if err != nil {
		return nil, fmt.Errorf("session: chainstate glob for %s: %w", id, err)
	}
	sort.Strings(matches)
	return matches, nil
}
