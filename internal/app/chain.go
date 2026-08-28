package app

import (
	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// ResolveChain returns the chain plugin: an external, project-supplied manifest
// when one is named (the hybrid model), otherwise the embedded chain registered
// for the id. Exported because every surface that acts on a chain resolves it
// the same way; the implementation lives in chains/external, which is the one
// package that knows both halves.
func ResolveChain(chain, manifestPath, templatePath string) (registry.ChainPlugin, error) {
	return external.ResolveChain(chain, manifestPath, templatePath)
}
