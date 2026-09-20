package blueprint

import (
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// FromPresetIn describes the network a preset should be written out as.
type FromPresetIn struct {
	// Dir is the key set directory the document will point its nodekeys at.
	Dir string
	// Chain names the chain plugin, or Manifest names an external one.
	Chain    string
	Manifest string
	// Producers and Endpoints size the network. Producers defaults to every
	// identity the set declares as a validator, Endpoints to zero.
	Producers int
	Endpoints int
	// Binary is the node executable, when the caller knows it.
	Binary string
	// Peering is the peer graph; empty leaves it at the default.
	Peering string
}

// FromPreset writes the network a key set would compose, as a document.
//
// This is the inversion the whole track is for (design §3.3). The composition
// used to read `preset -> (internal assembly) -> network`, and the middle was
// neither visible nor editable. It now reads `preset -> blueprint -> network`,
// where the blueprint is a file a person can open, diff, edit and commit.
//
// The keys are referenced by path, never copied into the document. A blueprint
// is meant to be read, shared and put under version control, and a generator
// that inlined private keys would make that unsafe by default — the one way to
// turn a convenience into a leak.
//
// It is deliberately the reverse of the raw path rather than a shortcut past
// it: what this writes must resolve to the identities the preset holds, which
// is what TestFromPreset_ComposesTheSameNetwork holds it to.
func FromPreset(set keyring.Preset, in FromPresetIn) (Blueprint, error) {
	if in.Dir == "" {
		return Blueprint{}, fmt.Errorf("blueprint: from preset: the key set directory is what the document points its keys at")
	}
	if len(set.Nodes) == 0 {
		return Blueprint{}, fmt.Errorf("blueprint: from preset: %s holds no identities", in.Dir)
	}

	producers := in.Producers
	if producers <= 0 {
		// Every identity the set declares as a validator. A set that declares
		// none is a set of endpoints, and saying so beats inventing a producer.
		producers = len(set.Network.Validators)
	}
	total := producers + in.Endpoints
	if total > len(set.Nodes) {
		return Blueprint{}, fmt.Errorf("blueprint: from preset: %d nodes were asked for and %s holds %d identities", total, in.Dir, len(set.Nodes))
	}
	if total == 0 {
		return Blueprint{}, fmt.Errorf("blueprint: from preset: %s declares no validators, so the network size has to be given", in.Dir)
	}

	bp := Blueprint{Version: Version, Chain: in.Chain, Manifest: in.Manifest, Peering: in.Peering}
	if in.Binary != "" {
		bp.Binaries = &Binaries{Node: in.Binary}
	}
	ord := map[node.Role]int{}
	for i := 0; i < total; i++ {
		role := node.RoleBP
		if i >= producers {
			role = node.RoleEN
		}
		ord[role]++
		e := set.Nodes[i]
		bp.Nodes = append(bp.Nodes, Node{
			Name: string(node.RoleLabel(role, ord[role])),
			Role: string(role),
			// By path, not by value. See the doc comment.
			NodeKey: &NodeKeyRef{File: filepath.Join(in.Dir, entryDir(e, i+1), "nodekey")},
		})
	}
	// Round-tripping here rather than at the call site: a generator that emits
	// a document its own parser rejects is worse than one that fails, because
	// the failure surfaces at whatever reads it next.
	if err := bp.Validate(); err != nil {
		return Blueprint{}, fmt.Errorf("blueprint: from preset: the generated document is not valid: %w", err)
	}
	return bp, nil
}

// entryDir names the directory an entry's key lives in. A preset labels its
// entries by index, and a ring built by a command may label its own.
func entryDir(e keyring.Entry, i int) string {
	if e.Label != "" {
		return string(e.Label)
	}
	return fmt.Sprintf("node%d", i)
}
