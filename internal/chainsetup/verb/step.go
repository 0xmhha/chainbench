package verb

import (
	"context"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// step runs one composition step through the composition machine.
//
// Every standalone `chain <step>` command comes here. It used to call the step
// body itself, which meant the order the steps must run in was asked for by
// each body and answered in fourteen hand-written sentences. The machine is
// what knows the order now, and a command says which step it wants.
//
// in carries only the part of the request this step reads. It is the same
// request an `up` carries whole, so a knob means the same thing however it
// arrives.
func step(ctx context.Context, d chainsetup.Deps, dataDir, name string, in chainsetup.ChainUpIn) (chainsetup.StepOut, error) {
	in.DataDir = dataDir
	ws, err := chainsetup.Open(dataDir, d.Clock)
	if err != nil {
		return chainsetup.StepOut{}, err
	}
	detail, err := chainsetup.NewManager(d, ws).Step(ctx, name, in)
	return chainsetup.StepOut{Detail: detail}, err
}
