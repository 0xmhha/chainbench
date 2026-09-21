package chainsetup

// Marking an error with the kind of failure it is.
//
// A stage's handler has to put the machine in the state its failure is, and the
// errors the steps return are sentences: they name the node, the file, the fork
// and what to do about it, which is what makes them worth reading and is not
// something a caller can switch on.
//
// So the kind rides alongside the message rather than in front of it. Wrapping
// with "%w: %w" would put the kind's own words into every refusal an operator
// reads, saying the same thing twice; this keeps the sentence byte for byte and
// lets errors.Is find the kind.

// marked is an error with its kind attached.
type marked struct {
	kind error
	err  error
}

func (m marked) Error() string   { return m.err.Error() }
func (m marked) Unwrap() []error { return []error{m.kind, m.err} }

// ofKind marks err as being of this kind. A nil err stays nil so a return site
// can be wrapped without a branch around it.
func ofKind(kind error, err error) error {
	if err == nil {
		return nil
	}
	return marked{kind: kind, err: err}
}
