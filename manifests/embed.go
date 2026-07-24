// Package manifests embeds the declarative per-chain manifest JSON so chain
// plugins can load their profile without filesystem path resolution.
package manifests

import (
	"embed"
	"fmt"
)

//go:embed chains/*.json
var chainsFS embed.FS

// Raw returns the raw manifest JSON for a chain id, or an error if absent.
func Raw(id string) ([]byte, error) {
	b, err := chainsFS.ReadFile("chains/" + id + ".json")
	if err != nil {
		return nil, fmt.Errorf("manifests: no manifest for chain %q: %w", id, err)
	}
	return b, nil
}
