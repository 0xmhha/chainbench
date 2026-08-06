package testspec

import (
	"errors"

	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// errNotImplemented marks a frozen-but-unimplemented contract (T0.1).
var errNotImplemented = errors.New("testspec: not implemented")

// ChainSpec selects the chain, its binary/binaries, and genesis overlay for a
// test. A single Binary applies to all nodes; Binaries maps roles to binaries
// for mixed (handoff) environments.
type ChainSpec struct {
	Name           string
	Binary         string
	Binaries       map[string]string
	Config         string
	GenesisOverlay map[string]any
}

// Spec is a parsed, validated test definition (schema in design §4.3).
type Spec struct {
	SchemaVersion    string
	ID               string
	ApplicableChains string
	Chain            ChainSpec
	Topology         map[string]any
	Hardforks        map[string]int
	Placement        string
	DefaultOn        string
	PreActions       []map[string]any
	Steps            []map[string]any
	Assertions       []map[string]any
	PostActions      []map[string]any
	Timeouts         map[string]string
}

// Parse validates raw JSON against the schema (required: schemaVersion, id,
// chain, assertions) and returns the parsed Spec.
func Parse(raw []byte) (Spec, error) {
	return Spec{}, errNotImplemented
}

// Fingerprint hashes the resolved declared values
// (binaries+genesis+config+topology+hardforks+placement) to a reuse key. config
// comes from resolved; the rest come from the receiver. It never touches a chain.
func (s Spec) Fingerprint(resolved config.Values) session.Fingerprint {
	return session.Fingerprint("")
}

// Get resolves a dot-path (a.b.c) within the spec, parsing comma-separated
// multiples. ok is false when the path is absent.
func (s Spec) Get(dotPath string) (any, bool) {
	return nil, false
}
