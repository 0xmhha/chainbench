// Package preset owns the pre-written documents a run starts from, and the
// reading of them.
//
// A preset exists so that standing a network up does not mean writing the same
// declaration again. There are two families, and they answer different
// questions: a KEY preset says which identities the network runs as, a CHAIN
// preset says how it is configured. Both are hand-written, committed, and read
// rather than generated.
//
// The rule that makes the module worth having is that the reading lives here
// and only here. A consumer asks for the decoded document — [Key], [Chain] —
// and works with that; it does not know the file layout, the codec, or where
// the directory is. Adding a preset family, or moving one, is then a change to
// this package instead of a change to everyone who reads one.
//
// That is why the directories are named here too. They are a fact about where
// preset documents live, which is this package's business; a default spelled
// out at each call site is the same defect one step smaller, and it was real —
// "presets/keys" was written out in seven places, one of which had already
// noticed and made a package-local constant of its own.
package preset

// KeysDir and ChainDir are where each family's documents live, relative to the
// repository root. A caller that needs a default asks for it rather than
// spelling it, so moving a family is a change here.
const (
	// KeysDir holds one directory per node, read by [LoadKeyPreset].
	KeysDir = "presets/keys"
	// ChainDir holds one YAML document per chain preset, read by
	// [LoadChainPreset].
	ChainDir = "presets/chain"
)
