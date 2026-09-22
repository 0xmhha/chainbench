// Package preset owns the pre-written documents a run starts from, and the
// reading of them.
//
// A preset exists so that standing a network up does not mean writing the same
// declaration again. This package owns one kind: a KEY preset, which says which
// identities a network runs as. It is hand-written, committed, and read rather
// than generated.
//
// It used to own a second, a CHAIN preset read from presets/chain/*.yaml. That
// document said the same things a chain-preset declaration says, in another
// format and another vocabulary, and a case reached it through one — so a
// handoff was declared twice. The declaration is the DSL's now
// (internal/dsl.ChainPresetV2) and this package is the key half.
//
// The rule that makes the module worth having is that the reading lives here
// and only here. A consumer asks for the decoded document — [Key] — and works
// with that; it does not know the file layout, the codec, or where the
// directory is. Moving the family is then a change to this package instead of a
// change to everyone who reads it.
//
// That is why the directories are named here too. They are a fact about where
// preset documents live, which is this package's business; a default spelled
// out at each call site is the same defect one step smaller, and it was real —
// "presets/keys" was written out in seven places, one of which had already
// noticed and made a package-local constant of its own.
package preset

// KeysDir is where the key preset lives, relative to the repository root. A
// caller that needs the default asks for it rather than spelling it, so moving
// the family is a change here.
//
// There was a ChainDir beside it, naming presets/chain. Nothing ever asked for
// it — the one consumer kept a constant of its own with the same string — and
// the documents it named are gone, so it went with them.
const KeysDir = "presets/keys"
