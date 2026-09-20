package chainsetup

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// What a verb needs before it may run.
//
// composeNeeds (steps_compose.go) already answers half of this for the eight
// composition steps: which earlier step must be RECORDED. It answers nothing for
// the rest of the surface. Measured 2026-09-21: the workspace exports 39 verbs,
// six consult composeNeeds, and the others each roll their own check and write
// their own message. Four said "no node table" in four different wordings —
// health and preflight pointed at `chain place`, hardfork said "compose the
// network first", and verify-validators gave no guidance at all.
//
// That is the defect composeNeeds was written to end, still standing outside
// composition. So the same idea is widened rather than a second mechanism built:
// a verb declares what it needs, in one table, and one function enforces it.
//
// The vocabulary is two words because the questions are two. A composition step
// asks "has the step before me run?", and its answer lives in the recorded
// steps. An operating verb asks something the step record cannot answer — "is
// anything running right now?" — because a node can be down for reasons no step
// chose.

// runNeed is the state a verb needs the network to be in.
type runNeed int

const (
	// anyRun asks nothing of the network. It is the honest answer for an
	// accessor and for a verb that must work on a half-built workspace.
	anyRun runNeed = iota
	// placed needs a node table: the composition has decided which nodes exist
	// and where they live. It does not ask whether any of them is up.
	placed
	// stopped needs every node down. Destructive verbs ask for it, because
	// removing the data under a live process leaves a node running on files
	// that no longer describe it.
	stopped
)

// nodeNeed is what a verb needs of ONE node, for the verbs that act by index.
//
// It is a third word rather than a variant of runNeed because it answers about a
// node, not about the network: "every node is down" and "node 3 is down" are
// different questions, and a verb that relaunches one node must not be held to
// the first.
type nodeNeed int

const (
	// anyNode asks nothing of the node beyond its existing.
	anyNode nodeNeed = iota
	// launched needs the node to carry a recorded argv — it has been started at
	// least once, so there is a command to run again. Relaunching a node that
	// never ran has nothing to relaunch.
	launched
	// down needs the node to be stopped.
	down
)

// verbNeed is one verb's declaration.
type verbNeed struct {
	// step names this verb's composition step, when it is one. Its own
	// prerequisites stay in composeNeeds, so the order is still declared once.
	step string
	// run is the state the network must be in.
	run runNeed
	// node is what the verb's own node must satisfy. A verb that takes an index
	// declares these; they are checked by allowNode. It is a list because the
	// conditions are independent — relaunching a node needs it both stopped and
	// previously started, and neither implies the other.
	node []nodeNeed
	// nodeAll applies node to every node in the table rather than to one. A verb
	// that acts on the whole network but needs what a per-node condition asks —
	// hardfork needs every node's argv — says so here instead of writing the
	// loop again.
	nodeAll bool
	// why explains a declaration that asks for nothing. An empty need with no
	// reason is what the ratchet refuses: "this verb checks nothing" has to be a
	// decision someone wrote down, not a gap nobody noticed.
	why string
}

// verbNeeds declares, for every exported verb on Workspace, what it needs.
//
// It may not have holes: arch's ratchet walks the method set and fails on a verb
// that is missing here. That is the number this table exists to move — a verb
// added without a declaration stops the build rather than quietly checking
// nothing.
var verbNeeds = map[string]verbNeed{
	// Composition steps. Their order is composeNeeds'; they are listed here so
	// the table is the whole surface and not a second partial one.
	"Allocate":   {step: "place"},
	"Keys":       {step: "keys"},
	"Genesis":    {step: "genesis"},
	"Config":     {step: "config"},
	"Provision":  {step: "deploy"},
	"Init":       {step: "init"},
	"Start":      {step: "start"},
	"LaunchOpts": {step: "build"},

	// Verbs that read or act on the node table.
	"Health":           {run: placed},
	"Preflight":        {run: placed},
	"Hardfork":         {run: placed, node: []nodeNeed{launched}, nodeAll: true},
	"VerifyValidators": {run: placed},

	// Destructive.
	"Rm": {run: stopped},

	// Declared to need nothing, each for a reason.
	"New":             {why: "it creates the workspace, so there is nothing to require"},
	"Dir":             {why: "accessor"},
	"State":           {why: "accessor"},
	"RPCHost":         {why: "accessor"},
	"Have":            {why: "accessor: it answers whether a step ran, so requiring one would be circular"},
	"NodeSet":         {why: "accessor over the node table; an empty table is an empty set, which is the true answer"},
	"Netmap":          {why: "derives a view from whatever is placed; an empty table is an empty map"},
	"Save":            {why: "persists the record, which has to work at every stage including a failed one"},
	"Lock":            {why: "guards the workspace directory, so it runs before anything is known about it"},
	"SetDriver":       {why: "wiring, set before any verb runs"},
	"SetEnv":          {why: "wiring, set before any verb runs"},
	"Acquire":         {why: "resolves the machine set; it runs before place has anything to record"},
	"MarkStepFailed":  {why: "records a failure, which by definition happens where a requirement was not met"},
	"Reconcile":       {why: "its job is to meet a record that disagrees with reality, so it must run on any state"},
	"Retarget":        {why: "rewrites where the workspace points, which is what a stale target needs"},
	"Logs":            {why: "a dead node's log is the reason to ask for it; it refuses an unknown index by name"},
	"LogExcerpt":      {why: "same as Logs, which it calls"},
	"Stop":            {why: "stopping what is already stopped is the outcome the caller asked for"},
	"StopNode":        {why: "same as Stop, for one node"},
	"StartNode":       {node: []nodeNeed{down, launched}},
	"SwapNode":        {node: []nodeNeed{launched}},
	"Restart":         {why: "delegates to StopNode and StartNode, which each answer for themselves"},
	"CrossFork":       {why: "names the node or binary the fork has nobody to run on, which a table-wide state cannot"},
	"Compare":         {why: "reads a baseline file, not the network"},
	"CheckBaseline":   {why: "reads a baseline file, not the network"},
	"ObserveBaseline": {why: "records what is there now, including nothing"},
}

// allow reports whether verb may run, naming what is missing.
//
// One function, so one message shape. The four wordings this replaced sent a
// reader to three different places for the same missing thing.
func (w *Workspace) allow(verb string) error {
	need, declared := verbNeeds[verb]
	if !declared {
		return fmt.Errorf("chainsetup: %s: this verb declares no requirements — add it to verbNeeds", verb)
	}
	if need.step != "" {
		if err := w.require(need.step); err != nil {
			return err
		}
	}
	if need.nodeAll {
		for _, ns := range w.state.Nodes {
			if err := checkNode(verb, need.node, ns); err != nil {
				return err
			}
		}
	}
	switch need.run {
	case placed:
		if len(w.state.Nodes) == 0 {
			return fmt.Errorf("chainsetup: %s: the node table is empty — run `chain place` first", lower(verb))
		}
	case stopped:
		for _, ns := range w.state.Nodes {
			if ns.PID > 0 {
				return fmt.Errorf("chainsetup: %s: node%d is running (pid %d) — run `chain stop` first", lower(verb), ns.Index, ns.PID)
			}
		}
	}
	return nil
}

// allowNode reports whether verb may run against one node, naming what is
// missing. It is allow's per-node half: the verb's table-wide requirements are
// checked first, then the node's own.
func (w *Workspace) allowNode(verb string, index int) error {
	if err := w.allow(verb); err != nil {
		return err
	}
	// The lookup goes through nodeAt so an unknown index is refused in one
	// place. Writing the sentence again here is the very duplication this file
	// exists to remove, and it silently changed the wording a test pins.
	ni, err := w.nodeAt(index)
	if err != nil {
		return err
	}
	return checkNode(verb, verbNeeds[verb].node, w.state.Nodes[ni])
}

// checkNode holds one node to one condition. One function, so the refusal reads
// the same wherever it comes from — this replaced the same "no recorded argv"
// sentence written out in three places.
func checkNode(verb string, needs []nodeNeed, ns node.Record) error {
	for _, need := range needs {
		switch need {
		case launched:
			if len(ns.Args) == 0 {
				return fmt.Errorf("chainsetup: %s: node%d has no recorded argv — run `chain start` first", lower(verb), ns.Index)
			}
		case down:
			if ns.PID > 0 {
				return fmt.Errorf("chainsetup: %s: node%d is already running (pid %d)", lower(verb), ns.Index, ns.PID)
			}
		}
	}
	return nil
}

// lower spells a verb the way the command line does.
func lower(verb string) string {
	out := make([]rune, 0, len(verb)+2)
	for i, r := range verb {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				out = append(out, ' ')
			}
			r += 'a' - 'A'
		}
		out = append(out, r)
	}
	return string(out)
}
