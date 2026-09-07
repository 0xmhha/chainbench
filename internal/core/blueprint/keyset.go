package blueprint

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// FileReader is how a blueprint reaches a key it named by path.
//
// It is injected rather than called, so this package still opens nothing. That
// matters beyond tidiness: a resolved network may describe a remote target, and
// the file naming a key is on whichever machine the operator ran from.
type FileReader func(path string) ([]byte, error)

// PresetFrom builds a key set out of the keys a resolved network carries.
//
// This is the raw path (design §3.2). Until now a composition could not start
// without a preset directory — store.LoadPreset was the only way a Preset came
// into being, so "preset is optional" was untrue no matter what the documents
// said. A blueprint that writes its own nodekeys now produces the same Preset
// the rest of the composition already consumes, so nothing downstream has to
// learn a second way to obtain keys.
//
// derivation says how much to compute: the wbft family needs BLS material and
// nothing else does, and asking for it costs real time.
func PresetFrom(r ResolvedNetwork, derivation derive.Derivation, read FileReader) (keyring.Preset, error) {
	var ks keyring.Preset
	for _, n := range r.Nodes {
		key, err := privateKey(n, read)
		if err != nil {
			return keyring.Preset{}, err
		}
		id, err := derive.Derive(key, derivation)
		if err != nil {
			return keyring.Preset{}, fmt.Errorf("blueprint: preset: %s: %w", n.Name, err)
		}
		ks.Nodes = append(ks.Nodes, keyring.Entry{
			Label:    keyring.Label(node.LabelFor(n.Index)),
			Index:    n.Index,
			Nodekey:  key,
			Identity: id,
		})
	}
	// The sealing set is the network's, in the order Resolve fixed. Recomputing
	// it from the entries would let a ring built from one blueprint disagree
	// with the blueprint about who seals.
	for _, name := range r.Validators {
		addr, ok := addressOf(r, ks, name)
		if !ok {
			return keyring.Preset{}, fmt.Errorf("blueprint: preset: validator %q is no node in this network", name)
		}
		ks.Network.Validators = append(ks.Network.Validators, addr)
	}
	return ks, nil
}

// privateKey reads a node's declared key, from the document or from the file it
// names.
func privateKey(n ResolvedNode, read FileReader) (derive.PrivateKey, error) {
	switch {
	case n.NodeKey.Hex != "":
		k, err := derive.ParsePrivateKey(n.NodeKey.Hex)
		if err != nil {
			return derive.PrivateKey{}, fmt.Errorf("blueprint: preset: %s: %w", n.Name, err)
		}
		return k, nil
	case n.NodeKey.File != "":
		if read == nil {
			return derive.PrivateKey{}, fmt.Errorf("blueprint: preset: %s names key file %q, but this call was given no way to read one", n.Name, n.NodeKey.File)
		}
		b, err := read(n.NodeKey.File)
		if err != nil {
			return derive.PrivateKey{}, fmt.Errorf("blueprint: preset: %s: read %s: %w", n.Name, n.NodeKey.File, err)
		}
		// The file holds the hex, usually with a trailing newline; whitespace
		// and an 0x prefix are ParsePrivateKey's business, and trimming here
		// too would be a second opinion about the same format.
		k, err := derive.ParsePrivateKey(string(b))
		if err != nil {
			return derive.PrivateKey{}, fmt.Errorf("blueprint: preset: %s: %s: %w", n.Name, n.NodeKey.File, err)
		}
		return k, nil
	}
	return derive.PrivateKey{}, fmt.Errorf("blueprint: preset: %s has no nodekey", n.Name)
}

// addressOf finds the account address a validator name stands for.
func addressOf(r ResolvedNetwork, ks keyring.Preset, name string) (string, bool) {
	for i, n := range r.Nodes {
		if n.Name != name {
			continue
		}
		if i < len(ks.Nodes) {
			return ks.Nodes[i].Address, true
		}
	}
	return "", false
}
