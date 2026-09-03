package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Result is the session.json schema and the read-back view of a completed run.
// It is written by Save and read by LoadDir; one struct owns the schema so a
// reporter or a summary display never re-derives it.
type Result struct {
	ID        string       `json:"id"`
	Command   string       `json:"command"`
	StartedAt string       `json:"startedAt"`
	Tests     []TestResult `json:"tests"`
	Summary   Counts       `json:"summary"`
}

// TestResult is one test's entry in session.json.
type TestResult struct {
	Seq    int    `json:"seq"`
	ID     string `json:"id"`
	Env    string `json:"env,omitempty"`
	Status string `json:"status"`
}

// Counts is the verdict tally over a session's tests.
type Counts struct {
	Pass    int `json:"pass"`
	Fail    int `json:"fail"`
	Blocked int `json:"blocked"`
	Skip    int `json:"skip"`
}

// LoadDir reads the session.json in a session directory into a Result. It is the
// single reader of the session schema; report building and summary display both
// go through it rather than parsing session.json themselves.
func LoadDir(dir string) (Result, error) {
	data, err := os.ReadFile(filepath.Join(dir, fileSession))
	if err != nil {
		return Result{}, fmt.Errorf("session: read %s: %w", fileSession, err)
	}
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return Result{}, fmt.Errorf("session: parse %s: %w", fileSession, err)
	}
	return r, nil
}

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

// TestDir returns a test record's directory (tests/<NNN>_<id>) inside a session
// directory. The layout lives here, its one owner, so a reader (report building)
// locates a test's artifacts without re-deriving the folder name.
func TestDir(sessionDir string, seq int, id string) string {
	return filepath.Join(sessionDir, dirTests, fmt.Sprintf("%03d_%s", seq, id))
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
