package testengine

import "github.com/0xmhha/chainbench/internal/core/statemachine"

// The run machine's messages.
//
// Everything this machine can be told and says to itself, in one file. The
// bands are the ones protocol.go describes for the chain: what the outside
// sends down, what the machine says to itself, and what something below posts
// up. A run has one public command so far, and the rest is its own.

// What the outside sends down.
const (
	// CmdRun starts the suite.
	CmdRun statemachine.What = statemachine.BaseTest + 0x001 + iota
)

// What this machine says to itself, one per stage that finished.
const (
	eventDeclarationRead statemachine.What = statemachine.BaseTest + 0x100 + iota
	eventSessionOpened
	eventNetworkReached
	eventChainPrepared
	eventCasesRun
	eventCollected
	// eventStageStopped: a stage could not finish, so the run goes to what it
	// can still gather rather than straight to an end.
	eventStageStopped
)

// runWhatNames is the name of every message this machine has.
var runWhatNames = map[statemachine.What]string{
	CmdRun: "CmdRun",

	eventDeclarationRead: "eventDeclarationRead",
	eventSessionOpened:   "eventSessionOpened",
	eventNetworkReached:  "eventNetworkReached",
	eventChainPrepared:   "eventChainPrepared",
	eventCasesRun:        "eventCasesRun",
	eventCollected:       "eventCollected",
	eventStageStopped:    "eventStageStopped",
}

// startRun begins the suite.
type startRun struct{}

// What says which message this is.
func (startRun) What() statemachine.What { return CmdRun }

// declarationRead: the documents are read and the network they ask for settled.
type declarationRead struct{}

func (declarationRead) What() statemachine.What { return eventDeclarationRead }

// sessionOpened: the run has somewhere to write itself down.
type sessionOpened struct{}

func (sessionOpened) What() statemachine.What { return eventSessionOpened }

// networkReached: there is a network, and it is the one the plan described.
type networkReached struct{}

func (networkReached) What() statemachine.What { return eventNetworkReached }

// chainPrepared: the fork is crossed, the height reached, the accounts funded.
type chainPrepared struct{}

func (chainPrepared) What() statemachine.What { return eventChainPrepared }

// casesRun: every declared case ran, whatever each reported.
type casesRun struct{}

func (casesRun) What() statemachine.What { return eventCasesRun }

// collected: the evidence is gathered and the network is down.
type collected struct{}

func (collected) What() statemachine.What { return eventCollected }

// stageStopped: a stage could not finish.
type stageStopped struct{}

func (stageStopped) What() statemachine.What { return eventStageStopped }
