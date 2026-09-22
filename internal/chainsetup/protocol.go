package chainsetup

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// The composition machine's messages.
//
// Everything this machine can be told, says to itself, or reports upward is in
// this one file, because a protocol scattered across the states that use it is
// a protocol nobody can read end to end. The states are elsewhere; what they
// may say to each other is here.
//
// The value says the direction, so a number in a log is readable without the
// name. Within [statemachine.BaseChain]:
//
//	+0x001  a Cmd the outside sends down
//	+0x040  a Cmd this machine sends up to its controller
//	+0x100  an Event this machine leaves itself
//	+0x180  an Event something below posts up
//
// Exported or not says the same thing to the compiler: a message another
// package may send is exported, and one that only ever travels inside this
// machine is not. protocol_test.go holds the two to each other.

// What the outside sends down. A controller — the CLI through a verb, or the
// test engine — has these four and nothing else.
const (
	// CmdCompose composes a network from a request. It is accepted only when
	// the machine is at rest.
	CmdCompose statemachine.What = statemachine.BaseChain + 0x001 + iota
	// CmdStep runs one composition step on a network that stopped part way.
	CmdStep
	// CmdStop takes the network down.
	CmdStop
	// CmdClearError is the only way out of a failed composition, so that a
	// failure is something somebody decided to leave rather than something the
	// next command walked past.
	CmdClearError
)

// What this machine sends up to its controller. These are Cmd rather than
// Event because to the controller they are "now do this": the reference names
// a child's report CMD_POST_DHCP_ACTION for the same reason.
const (
	// CmdPostCompose reports that composing finished, either way. It is sent
	// once.
	CmdPostCompose statemachine.What = statemachine.BaseChain + 0x040 + iota
	// CmdOnQuit reports that this machine has stopped and its controller need
	// not wait for it any longer.
	CmdOnQuit
)

// What this machine leaves itself. A state's Enter cannot move, so it says what
// happened and the stage above it decides what that means.
const (
	// eventWorkspaceOpened: the workspace is open and locked.
	eventWorkspaceOpened statemachine.What = statemachine.BaseChain + 0x100 + iota
	// eventNodeTableBuilt: the nodes, their roles, hosts and ports are decided.
	eventNodeTableBuilt
	// eventKeysEnsured: every node has an identity.
	eventKeysEnsured
	// eventGenesisBuilt: the genesis exists, however it was arrived at.
	eventGenesisBuilt
	// eventInputsDeployed: every launch input is present on its target.
	eventInputsDeployed
	// eventPhaseLaunched: one launch phase is up. A launch is several, so the
	// stage counts them.
	eventPhaseLaunched
	// eventStageDone: a stage finished and has nothing of its own to report.
	// This is what the adapter around the old verbs says while the real leaf
	// states are still being moved over, one per commit.
	eventStageDone
	// eventStageFailed: a stage could not finish. The stage above writes the
	// reason down and goes to failed.
	eventStageFailed
)

// What something below posts up to this machine. A node monitor watching a
// launched network is outside the composition but reports into it.
const (
	// EventNodeDied: a node that was up is not any more.
	EventNodeDied statemachine.What = statemachine.BaseChain + 0x180 + iota
)

// whatNames is the name of every message this machine has.
//
// It is what makes a log readable, and protocol_test.go fails on a message that
// is missing from it — the reference gets the same list by reflection, and a
// test that names the gap is the same guarantee with the reflection left out.
var whatNames = map[statemachine.What]string{
	CmdCompose:    "CmdCompose",
	CmdStep:       "CmdStep",
	CmdStop:       "CmdStop",
	CmdClearError: "CmdClearError",

	CmdPostCompose: "CmdPostCompose",
	CmdOnQuit:      "CmdOnQuit",

	eventWorkspaceOpened: "eventWorkspaceOpened",
	eventNodeTableBuilt:  "eventNodeTableBuilt",
	eventKeysEnsured:     "eventKeysEnsured",
	eventGenesisBuilt:    "eventGenesisBuilt",
	eventInputsDeployed:  "eventInputsDeployed",
	eventPhaseLaunched:   "eventPhaseLaunched",
	eventStageDone:       "eventStageDone",
	eventStageFailed:     "eventStageFailed",

	EventNodeDied: "EventNodeDied",
}

// WhatName is what a message of this machine is called, for a log or an error.
//
// A message this machine does not have comes back as its number, because the
// caller is holding something from somewhere else and saying so is more useful
// than an empty string.
func WhatName(w statemachine.What) string {
	if name, ok := whatNames[w]; ok {
		return name
	}
	return fmt.Sprintf("What(%#x)", int(w))
}

// ---- The messages themselves.
//
// Compose is spelled without a suffix, as the design writes it. RunStep is the
// exception: this package already has a Step, which is a step of a recorded
// composition, and two things called Step in one package is not a choice Go
// offers.

// Compose asks for a whole network, from the request that describes it.
type Compose struct {
	Request ChainUpIn
	// From is the step to begin at, for a resume; empty begins at the first.
	//
	// It is here because the record does not yet say where a composition got
	// to. When it does, a resume starts from the recorded state path and this
	// field goes.
	From string
}

// What says which message this is.
func (Compose) What() statemachine.What { return CmdCompose }

// RunStep asks for one composition step by name, on a network that stopped part
// way. The names are UpStepNames.
type RunStep struct{ Name string }

// What says which message this is.
func (RunStep) What() statemachine.What { return CmdStep }

// Stop asks for the network to be taken down.
type Stop struct{}

// What says which message this is.
func (Stop) What() statemachine.What { return CmdStop }

// ClearError leaves a failed composition, having been read.
type ClearError struct{}

// What says which message this is.
func (ClearError) What() statemachine.What { return CmdClearError }

// PostCompose reports the end of composing to the controller.
//
// It carries the reason rather than a status, because the controller's next
// move depends on why: a port already held is somebody else's network, and a
// binary that will not start is this one's.
type PostCompose struct {
	OK    bool
	Steps []string
	Err   error
}

// What says which message this is.
func (PostCompose) What() statemachine.What { return CmdPostCompose }

// OnQuit reports that this machine has stopped.
type OnQuit struct{}

// What says which message this is.
func (OnQuit) What() statemachine.What { return CmdOnQuit }

// NodeDied reports that a node of the composed network is no longer running.
type NodeDied struct{ Index int }

// What says which message this is.
func (NodeDied) What() statemachine.What { return EventNodeDied }

// ---- What the machine says to itself.

// stageReport is what a finished stage says.
//
// One interface rather than a case per stage. The stage parent does the same
// three things with every one of them — note the line, look up what comes next,
// move — and nine stages each with their own case would be that written nine
// times. A failure is deliberately not one of these: it is the one report the
// parent treats differently.
type stageReport interface {
	statemachine.Message
	// stage is which step finished and the line a person reads about it.
	stage() (step, detail string)
}

// stageDone is a stage that finished with nothing of its own to say.
type stageDone struct {
	Step   string
	Detail string
}

func (stageDone) What() statemachine.What { return eventStageDone }

func (e stageDone) stage() (string, string) { return e.Step, e.Detail }

// workspaceOpened: the workspace holds the chain it was asked for, and the
// request that asked for it.
type workspaceOpened struct{ Detail string }

func (workspaceOpened) What() statemachine.What { return eventWorkspaceOpened }

func (e workspaceOpened) stage() (string, string) { return stepNew, e.Detail }

// stageFailed is a stage that could not finish, and why.
type stageFailed struct {
	Step string
	Err  error
}

func (stageFailed) What() statemachine.What { return eventStageFailed }

// nodeTableBuilt and the rest are declared with their leaf states, one commit
// each. Their What values are above so that the whole protocol is one file to
// read, and the band test holds every one of them to the private range whether
// or not a state sends it yet.
