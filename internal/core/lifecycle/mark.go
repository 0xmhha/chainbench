package lifecycle

// Marking an error with the kind of failure it is.
//
// A stage's handler has to put the machine in the state its failure is, and the
// errors the work returns are sentences: they name the node, the file, the fork
// and what to do about it, which is what makes them worth reading and is not
// something a caller can switch on.
//
// So the kind rides alongside the message rather than in front of it. Wrapping
// with "%w: %w" would put the kind's own words into every refusal an operator
// reads, saying the same thing twice; this keeps the sentence byte for byte and
// lets errors.Is find the kind.
//
// It lives here rather than in one area's package because every area needs it
// for the same reason. The chain area had it first and the test area would have
// copied it, and two copies of the rule that turns an error into a state is how
// the two come to classify differently.

// marked is an error with its kind attached.
type marked struct {
	kind error
	err  error
}

func (m marked) Error() string   { return m.err.Error() }
func (m marked) Unwrap() []error { return []error{m.kind, m.err} }

// Mark attaches kind to err without changing what err says. A nil err stays nil
// so a return site can be wrapped without a branch around it.
func Mark(kind error, err error) error {
	if err == nil {
		return nil
	}
	return marked{kind: kind, err: err}
}
