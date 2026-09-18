package blueprint

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// Parse reads a blueprint and checks it against itself.
//
// Unknown fields are refused. A declaration is the one place a typo has no
// symptom — a misspelled key is simply absent, the value is resolved from
// somewhere else, and the network comes up looking fine and configured
// differently from what the document says. That failure is silent and late,
// which is the worst combination, so the strictness is not a nicety.
func Parse(raw []byte) (Blueprint, error) {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)

	var bp Blueprint
	if err := dec.Decode(&bp); err != nil {
		if errors.Is(err, io.EOF) {
			// An empty document is an empty blueprint, not a failure: a
			// generator writes the shell before anything fills it.
			return Blueprint{}, nil
		}
		return Blueprint{}, fmt.Errorf("blueprint: %w", err)
	}
	// A second document in the stream would be silently dropped, and a person
	// who wrote one meant it to be read.
	if err := dec.Decode(new(Blueprint)); !errors.Is(err, io.EOF) {
		return Blueprint{}, fmt.Errorf("blueprint: a blueprint is one document; this stream has more than one")
	}
	bp.normalize()
	if err := bp.Validate(); err != nil {
		return Blueprint{}, err
	}
	return bp, nil
}

// Marshal writes a blueprint back out.
//
// It round-trips: what Parse read, Marshal writes, and Parse reads again to the
// same value. That is not a tidiness property. `net blueprint --from-preset`
// generates a document a person then edits, so a field this package cannot
// write is a field the generator cannot offer.
func Marshal(bp Blueprint) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(bp); err != nil {
		return nil, fmt.Errorf("blueprint: encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("blueprint: encode: %w", err)
	}
	return buf.Bytes(), nil
}

// normalize folds an empty collection onto an absent one.
//
// `nodes: []` and no `nodes:` key say the same thing, and a document that says
// it the first way must not become a different blueprint from one that says it
// the second. Without this the round trip is not a round trip: the encoder
// omits an empty list, so writing and re-reading turns [] into absent — and a
// generator's output would compare unequal to its own input. The fuzzer found
// exactly this, on `nodes: [] `.
func (bp *Blueprint) normalize() {
	clear := func(s *[]string) {
		if len(*s) == 0 {
			*s = nil
		}
	}
	if len(bp.Nodes) == 0 {
		bp.Nodes = nil
	}
	if len(bp.Alloc) == 0 {
		bp.Alloc = nil
	}
	for i := range bp.Nodes {
		if len(bp.Nodes[i].Launch) == 0 {
			bp.Nodes[i].Launch = nil
		}
	}
	if b := bp.Binaries; b != nil {
		if len(b.Overrides) == 0 {
			b.Overrides = nil
		}
		for i := range b.Overrides {
			clear(&b.Overrides[i].Nodes)
		}
	}
	if v := bp.Validators; v != nil {
		clear(&v.Explicit)
	}
	if g := bp.Genesis; g != nil && len(g.Overrides) == 0 {
		g.Overrides = nil
	}
	if g := bp.Governance; g != nil {
		if len(g.Roles) == 0 {
			g.Roles = nil
		}
		if len(g.Env) == 0 {
			g.Env = nil
		}
	}
}
