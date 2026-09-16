package dsl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxEnvSearchDepth bounds how far ReadFiles walks up from a case file looking
// for the env/ directory that holds its environment declaration. Case files sit
// at most a few levels below the suite root (tests/tc/<chain>/<group>/<domain>),
// so a small bound keeps a typo from scanning the whole filesystem.
const maxEnvSearchDepth = 8

// ReadFiles reads each spec file into raw JSON bytes, resolving a v2 case's
// "env": "<id>" reference against the case file's directory and then each
// ancestor up to maxEnvSearchDepth levels: at every level it looks for
// <dir>/<id>.env.json and <dir>/env/<id>.env.json. That lets a tree of case
// directories share one env/ directory at the suite root. It is the one place a
// spec path becomes the bytes an engine runs, so every surface resolves env
// references the same way.
func ReadFiles(paths []string) ([][]byte, error) { return ReadFilesWithEnv(paths, "") }

// ReadFilesWithEnv reads the specs the way ReadFiles does, with every case moved
// onto the env named by envRef. An empty envRef reads the specs as written.
//
// This is the run-time half of keeping mainnet differences out of the cases: the
// declaration already lives in its own file, and this is what chooses which file
// without editing the case. What the case itself overrode is kept, because that
// is the part that belongs to the test rather than to the chain.
//
// envRef is an env id, resolved the same way a case's own reference is. A value
// that looks like a path (it contains a separator or ends in .json) is read as
// one instead, so a declaration for a chain that has no home in this tree yet
// can still be run against.
func ReadFilesWithEnv(paths []string, envRef string) ([][]byte, error) {
	var envID string
	var envRaw []byte
	if envRef != "" {
		var err error
		if envID, envRaw, err = resolveEnvRef(paths, envRef); err != nil {
			return nil, err
		}
	}
	specs := make([][]byte, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("dsl: read spec %s: %w", p, err)
		}
		if b, err = UseEnv(b, envID); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		dir := filepath.Dir(p)
		b, err = InlineEnv(b, func(id string) ([]byte, error) {
			if id == envID && envRaw != nil {
				return envRaw, nil
			}
			if eb, ok := findEnvFile(dir, id); ok {
				return eb, nil
			}
			return nil, fmt.Errorf("no %s.env.json beside %s or in an env/ directory at any level above it", id, p)
		})
		if err != nil {
			return nil, fmt.Errorf("dsl: %w", err)
		}
		specs = append(specs, b)
	}
	return specs, nil
}

// resolveEnvRef turns the caller's env reference into the id the cases will
// name and the bytes that id resolves to.
//
// A path is read directly and supplies its own id; an id is searched for from
// the first spec's directory, which is where a case's own reference is searched
// from, so one rule covers both.
func resolveEnvRef(paths []string, envRef string) (string, []byte, error) {
	if looksLikePath(envRef) {
		raw, err := os.ReadFile(envRef)
		if err != nil {
			return "", nil, fmt.Errorf("dsl: read env %s: %w", envRef, err)
		}
		env, err := ParseEnv(raw)
		if err != nil {
			return "", nil, fmt.Errorf("dsl: env %s: %w", envRef, err)
		}
		return env.ID, raw, nil
	}
	if len(paths) == 0 {
		return "", nil, fmt.Errorf("dsl: env %q cannot be resolved without a spec to search from", envRef)
	}
	raw, ok := findEnvFile(filepath.Dir(paths[0]), envRef)
	if !ok {
		return "", nil, fmt.Errorf("dsl: no %s.env.json beside %s or in an env/ directory at any level above it", envRef, paths[0])
	}
	return envRef, raw, nil
}

// looksLikePath distinguishes a file from an id without touching the disk, so
// the same input always means the same thing.
func looksLikePath(ref string) bool {
	return strings.ContainsRune(ref, filepath.Separator) || strings.HasSuffix(ref, ".json")
}

// findEnvFile walks up from dir looking for the env declaration named id.
func findEnvFile(dir, id string) ([]byte, bool) {
	for depth := 0; depth < maxEnvSearchDepth; depth++ {
		for _, cand := range []string{
			filepath.Join(dir, id+".env.json"),
			filepath.Join(dir, "env", id+".env.json"),
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
