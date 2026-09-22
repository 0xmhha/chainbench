package dsl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxPresetSearchDepth bounds how far ReadFiles walks up from a case file
// looking for the chain-preset/ directory that holds its declaration. Case files
// sit at most a few levels below the suite root
// (tests/tc/<chain>/<group>/<domain>), so a small bound keeps a typo from
// scanning the whole filesystem.
const maxPresetSearchDepth = 8

// sharedPresetDir is where chain-presets live when they are not beside the
// cases that use them: one directory at the repository root, beside the key
// presets. A suite may still keep its own in a chain-preset/ directory next to
// its cases, which is why the walk exists at all; this is the last place looked,
// so a local declaration wins over the shared one of the same name.
const sharedPresetDir = "presets/chain"

// ReadFiles reads each spec file into raw JSON bytes, resolving a v2 case's
// "chainPreset": "<id>" reference against the case file's directory and then
// each ancestor up to maxPresetSearchDepth levels: at every level it looks for
// <dir>/<id>.json and <dir>/chain-preset/<id>.json, and finally in
// sharedPresetDir. That lets a tree of case directories share one declaration
// without naming a path. It is the one place a spec path becomes the bytes an
// engine runs, so every surface resolves chain-preset references the same way.
func ReadFiles(paths []string) ([][]byte, error) { return ReadFilesWithChainPreset(paths, "") }

// ReadFilesWithChainPreset reads the specs the way ReadFiles does, with every case moved
// onto the env named by presetRef. An empty presetRef reads the specs as written.
//
// This is the run-time half of keeping mainnet differences out of the cases: the
// declaration already lives in its own file, and this is what chooses which file
// without editing the case. What the case itself overrode is kept, because that
// is the part that belongs to the test rather than to the chain.
//
// presetRef is an env id, resolved the same way a case's own reference is. A value
// that looks like a path (it contains a separator or ends in .json) is read as
// one instead, so a declaration for a chain that has no home in this tree yet
// can still be run against.
func ReadFilesWithChainPreset(paths []string, presetRef string) ([][]byte, error) {
	var presetID string
	var presetRaw []byte
	if presetRef != "" {
		var err error
		if presetID, presetRaw, err = resolveChainPresetRef(paths, presetRef); err != nil {
			return nil, err
		}
	}
	specs := make([][]byte, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("dsl: read spec %s: %w", p, err)
		}
		if b, err = UseChainPreset(b, presetID); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		dir := filepath.Dir(p)
		b, err = InlineChainPreset(b, func(id string) ([]byte, error) {
			if id == presetID && presetRaw != nil {
				return presetRaw, nil
			}
			if eb, ok := findChainPresetFile(dir, id); ok {
				return eb, nil
			}
			return nil, fmt.Errorf("no chain-preset %q beside %s, in a chain-preset/ directory above it, or in %s", id, p, sharedPresetDir)
		})
		if err != nil {
			return nil, fmt.Errorf("dsl: %w", err)
		}
		specs = append(specs, b)
	}
	return specs, nil
}

// resolveChainPresetRef turns the caller's env reference into the id the cases will
// name and the bytes that id resolves to.
//
// A path is read directly and supplies its own id; an id is searched for from
// the first spec's directory, which is where a case's own reference is searched
// from, so one rule covers both.
func resolveChainPresetRef(paths []string, presetRef string) (string, []byte, error) {
	if looksLikePath(presetRef) {
		raw, err := os.ReadFile(presetRef)
		if err != nil {
			return "", nil, fmt.Errorf("dsl: read env %s: %w", presetRef, err)
		}
		env, err := ParseChainPreset(raw)
		if err != nil {
			return "", nil, fmt.Errorf("dsl: env %s: %w", presetRef, err)
		}
		return env.ID, raw, nil
	}
	if len(paths) == 0 {
		return "", nil, fmt.Errorf("dsl: env %q cannot be resolved without a spec to search from", presetRef)
	}
	raw, ok := findChainPresetFile(filepath.Dir(paths[0]), presetRef)
	if !ok {
		return "", nil, fmt.Errorf("dsl: no chain-preset %q beside %s, in a chain-preset/ directory above it, or in %s", presetRef, paths[0], sharedPresetDir)
	}
	return presetRef, raw, nil
}

// looksLikePath distinguishes a file from an id without touching the disk, so
// the same input always means the same thing.
func looksLikePath(ref string) bool {
	return strings.ContainsRune(ref, filepath.Separator) || strings.HasSuffix(ref, ".json")
}

// findChainPresetFile walks up from dir looking for the env declaration named id.
func findChainPresetFile(dir, id string) ([]byte, bool) {
	for depth := 0; depth < maxPresetSearchDepth; depth++ {
		for _, cand := range []string{
			filepath.Join(dir, id+".json"),
			filepath.Join(dir, "chain-preset", id+".json"),
			filepath.Join(dir, sharedPresetDir, id+".json"),
		} {
			if b, err := os.ReadFile(cand); err == nil {
				return b, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, false
}
