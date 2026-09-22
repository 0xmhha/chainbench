package chainsetup

// StepOut is what one mutating step did.
//
// It is the step's type as well as the verb's. They were two types with the
// same fields for one commit, and a verb that copied one into the other is a
// place where the two can be made to disagree.
//
// It used to carry the states the step went through, as a list each step
// appended to as it ran. The machine walks those states now, so the list was a
// second account of the same walk kept by hand — and the two disagreed where a
// step reported a path it had not taken.
type StepOut struct {
	Detail string
	// Shipped is how many identity files a step sent to a remote target, and 0
	// for a local one. The deploy step is the only one that sets it, and it is
	// here rather than parsed back out of Detail because which way that step
	// went is a fact about the composition, not a phrase in a sentence.
	Shipped int
}
