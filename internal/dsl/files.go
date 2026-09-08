package dsl

import (
	"fmt"
	"os"
	"path/filepath"
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
func ReadFiles(paths []string) ([][]byte, error) {
	specs := make([][]byte, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("dsl: read spec %s: %w", p, err)
		}
		dir := filepath.Dir(p)
		b, err = InlineEnv(b, func(id string) ([]byte, error) {
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
