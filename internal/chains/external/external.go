// Package external loads a chain plugin from a project-supplied manifest file,
// so chainbench can bench a chain it does not embed — on an existing consensus
// family — without a code change. It is the composition boundary for the hybrid
// model: first-party chains stay embedded (pkg/chains/*), while a project points
// `--manifest`/`--genesis-template` at its own files. Only a genuinely new
// consensus family still needs a plugin (a new pkg/consensus/*).
package external

import (
	"fmt"
	"os"

	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// Load builds a ChainPlugin from an external manifest file and an optional
// genesis template file. The manifest's consensus_family must be a built-in
// family (wbft|poa); its accounts protocol resolves via the SDK by the
// manifest's "protocol" field (or, when empty, its id). A template path is
// required when the manifest declares a genesis template.
func Load(manifestPath, templatePath string) (registry.ChainPlugin, error) {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("external manifest: %w", err)
	}
	m, err := registry.ParseManifest(raw)
	if err != nil {
		return nil, err
	}
	fam, err := familyByName(m.ConsensusFamily)
	if err != nil {
		return nil, err
	}
	protoName := m.Protocol
	if protoName == "" {
		protoName = m.ID
	}
	proto, err := protocol.ByName(protoName)
	if err != nil {
		return nil, fmt.Errorf("external manifest %q: unknown accounts protocol %q "+
			"(set \"protocol\" to a built-in: stablenet|wbft|wemix): %w", m.ID, protoName, err)
	}

	var tmpl []byte
	switch {
	case templatePath != "":
		if tmpl, err = os.ReadFile(templatePath); err != nil {
			return nil, fmt.Errorf("external genesis template: %w", err)
		}
	case m.Genesis.Template != "":
		return nil, fmt.Errorf("external manifest %q declares genesis.template %q "+
			"but no --genesis-template path was given", m.ID, m.Genesis.Template)
	}

	return registry.StaticPlugin{M: m, Fam: fam, Proto: proto, Tmpl: tmpl}, nil
}

// familyByName resolves a built-in consensus family. This switch lives in the
// composition layer (not core), so core keeps its no-family-import boundary.
func familyByName(name string) (registry.ConsensusFamily, error) {
	switch name {
	case "wbft":
		return wbft.New(), nil
	case "poa":
		return poa.New(), nil
	default:
		return nil, fmt.Errorf("external manifest: consensus family %q is not built in "+
			"(a new family needs a pkg/consensus/* plugin); built-in: wbft|poa", name)
	}
}

// ResolveChain returns the plugin a caller means: the external, project-supplied
// manifest when one is named (the hybrid model), otherwise the embedded chain
// registered for the id.
//
// It lives here because this is the only package that already knows both halves
// — registry.Get for embedded chains and Load for manifests — and putting it in
// registry would make registry depend on the loaders that register into it.
// Every surface that acts on a chain resolves it through this one function.
func ResolveChain(chain, manifestPath, templatePath string) (registry.ChainPlugin, error) {
	if manifestPath != "" {
		return Load(manifestPath, templatePath)
	}
	return registry.Get(chain)
}
