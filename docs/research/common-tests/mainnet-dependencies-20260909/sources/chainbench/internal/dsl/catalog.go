package dsl

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SpecInfo is one test case as a catalog lists it: enough to choose and run it
// without opening the file. Chain is best-effort — a case that names its env by
// reference (rather than inline) does not carry the chain in the file, so it is
// left empty rather than guessed.
type SpecInfo struct {
	// Path is the spec file, relative to the listed root.
	Path string `json:"path"`
	// ID is the case id.
	ID string `json:"id"`
	// Kind is "case" (v2), or empty for a v1 spec.
	Kind string `json:"kind"`
	// Chain is the chain family the case runs on, when the file states it.
	Chain string `json:"chain,omitempty"`
	// Description is the case's prose description, when it has one.
	Description string `json:"description,omitempty"`
}

// catalogEntry is the lenient shape ListSpecs decodes: it reads only the header
// fields, so a case whose env is a reference (unresolved) still lists.
type catalogEntry struct {
	ID          string          `json:"id"`
	Kind        string          `json:"kind"`
	Description string          `json:"description"`
	Chain       json.RawMessage `json:"chain"` // v1: {"name": ...}
	Env         json.RawMessage `json:"env"`   // v2: inline {"chain": ...} or an id string
}

// ListSpecs walks root for test-case JSON files and returns one SpecInfo each,
// sorted by path. It skips env declaration files (kind "env") and any file that
// is not a JSON object with an id, so a directory holding both cases and envs
// lists only the runnable cases. It parses nothing beyond the header, so a case
// that references its env by id lists without the env being resolvable.
func ListSpecs(root string) ([]SpecInfo, error) {
	var out []SpecInfo
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		var e catalogEntry
		if json.Unmarshal(data, &e) != nil || e.ID == "" || e.Kind == KindEnv {
			return nil // not a runnable case
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			rel = p
		}
		out = append(out, SpecInfo{
			Path: rel, ID: e.ID, Kind: e.Kind,
			Chain: e.chain(), Description: e.Description,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("dsl: list specs under %s: %w", root, err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// chain reads the chain family from whichever form the file uses: a v1 top-level
// chain object, or a v2 inline env object. An env named by reference leaves it
// empty.
func (e catalogEntry) chain() string {
	if len(e.Chain) > 0 {
		var c struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(e.Chain, &c) == nil {
			return c.Name
		}
	}
	if len(e.Env) > 0 {
		var env struct {
			Chain string `json:"chain"`
		}
		if json.Unmarshal(e.Env, &env) == nil {
			return env.Chain
		}
	}
	return ""
}
