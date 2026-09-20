package chainsetup

import "fmt"

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

// verbNeed is one verb's declaration.
type verbNeed struct {
	// step names this verb's composition step, when it is one. Its own
	// prerequisites stay in composeNeeds, so the order is still declared once.
	step string
	// run is the state the network must be in.
	run runNeed
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
	"Hardfork":         {run: placed},
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
	"StartNode":       {why: "refuses a running node and an unknown index by name, which is finer than a table-wide state"},
	"SwapNode":        {why: "same as StartNode: it names the node it cannot swap"},
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
