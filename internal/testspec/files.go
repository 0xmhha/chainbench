package testspec

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReadFiles reads each spec file into raw JSON bytes, resolving a v2 case's
// "env": "<id>" reference against the case file's directory
// (<dir>/<id>.env.json, then <dir>/env/<id>.env.json, then the parent's
// env/ directory — a suite of case directories sharing one set of envs). It
// is the one place a spec path becomes the bytes an engine runs, so every
// surface resolves env references the same way.
func ReadFiles(paths []string) ([][]byte, error) {
	specs := make([][]byte, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("testspec: read spec %s: %w", p, err)
		}
		dir := filepath.Dir(p)
		b, err = InlineEnv(b, func(id string) ([]byte, error) {
			for _, cand := range []string{
				filepath.Join(dir, id+".env.json"),
				filepath.Join(dir, "env", id+".env.json"),
				filepath.Join(dir, "..", "env", id+".env.json"),
			} {
				if eb, err := os.ReadFile(cand); err == nil {
					return eb, nil
				}
			}
			return nil, fmt.Errorf("no %s.env.json next to %s (or in an env/ directory beside it or its parent)", id, p)
		})
		if err != nil {
			return nil, fmt.Errorf("testspec: %w", err)
		}
		specs = append(specs, b)
	}
	return specs, nil
}
