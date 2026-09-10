package feature

import "github.com/0xmhha/chainbench/internal/app"

// The test stage: running DSL test definitions against a network.
//
// One definition and several are the same feature at one and many — several is
// that same run repeated in the order given, keeping the network up between
// them so each definition's own preflight decides whether to reuse it. They are
// registered together so a surface cannot offer one without the other.
func init() {
	Register(Registration{
		Name: "test.run", Stage: StageTest,
		Summary: "Run a DSL test definition: compose the network it declares, then run it",
	}, app.RunSuite)
	Register(Registration{
		Name: "test.run-sequence", Stage: StageTest,
		Summary: "Run several DSL test definitions in order, reusing the chain when consecutive ones match",
	}, app.RunSuites)
}
