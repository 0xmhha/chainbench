package chainsetup

import "github.com/0xmhha/chainbench/internal/core/lifecycle"

// StepOut is what one mutating step did: its recorded detail line, and the
// states it went through when the step knows them.
//
// It is the step's type as well as the verb's. They were two types with the
// same two fields for one commit, and a verb that copied one into the other is
// a place where the two can be made to disagree.
type StepOut struct {
	Detail string
	// Passed is the states this step went through, in order, ending at the one
	// it finished in. It is empty for a step whose work has not moved into its
	// handler yet, and the handler then walks a path it assumed instead of the
	// one that happened. A step that fills this has stopped being guessed at.
	Passed []lifecycle.Status
	// Shipped is how many identity files a step sent to a remote target, and 0
	// for a local one. The deploy step is the only one that sets it, and it is
	// here rather than parsed back out of Detail because which way that step
	// went is a fact about the composition, not a phrase in a sentence.
	Shipped int
}
