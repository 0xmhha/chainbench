// The compose plan: what a run will build, said out loud before it builds it.
//
// A declaration reaches the composer through three layers — the shared chain
// declaration, the case's own overrides, and what the command passed — and
// merging is silent. Reading a case file therefore does not tell an operator
// which chain is about to come up, and reading the workspace afterwards tells
// them only what did. The plan is the merged answer, rendered from the very
// value the composer is handed, so it cannot describe a different network than
// the one that launches.

package testengine

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// ComposePlan is the network a run is about to compose, after the declaration
// and the command's overrides have been merged.
//
// It is derived from the composer's own input rather than re-read from the
// specs: a plan assembled a second way would drift from what launches, and a
// plan that can drift is worse than none.
type ComposePlan struct {
	// Chain is the chain the specs declare.
	Chain string `json:"chain"`
	// Workspace is the directory the composition writes into.
	Workspace string `json:"workspace"`
	// Target says where the nodes run: this machine, or a server set entry.
	Target string `json:"target"`
	// Placement is the value Target was rendered from. It is kept beside the
	// phrase because a phrase cannot be compared: a verify holds the plan
	// against what the workspace recorded, and the workspace records the value.
	Placement resource.Spec `json:"placement,omitempty"`
	// Binary is the node executable every node runs unless its table row names
	// another; Binaries maps those per-node names to what they resolved to.
	Binary   string            `json:"binary,omitempty"`
	Binaries map[string]string `json:"binaries,omitempty"`

	Nodes   PlanNodes   `json:"nodes"`
	Keys    PlanKeys    `json:"keys"`
	Genesis PlanGenesis `json:"genesis"`

	// Launch and Config are the overrides that reach the nodes, by scope
	// ("all", a role, or "node<N>"). They are what a reader most often wants:
	// a knob that surprises them was set in one of these.
	Launch map[string][]PlanKnob `json:"launch,omitempty"`
	Config map[string][]string   `json:"config,omitempty"`

	// From says who chose each value that could have come from more than one
	// place. A handoff composes from its profile and leaves this empty.
	From map[PlanField]PlanSource `json:"from,omitempty"`

	// Handoff, when set, means this env composes a consensus handoff instead of
	// a plain network, and the fields above that a handoff does not use are
	// empty.
	Handoff *PlanHandoff `json:"handoff,omitempty"`
}

// PlanNodes is the layout: how many of each role, and how they peer.
type PlanNodes struct {
	BP int `json:"bp"`
	EN int `json:"en"`
	PN int `json:"pn"`
	// Declared is true when a node table named every node, so the counts above
	// were read off the table rather than requested as counts.
	Declared bool `json:"declared"`
	// AutoSize means the bp count is filled from the server set at compose
	// time, so the bp above is a floor and not the final number.
	AutoSize bool   `json:"autoSize,omitempty"`
	Peering  string `json:"peering"`
	SyncMode string `json:"syncMode,omitempty"`
}

// PlanKeys says where node identities come from.
type PlanKeys struct {
	Source string `json:"source"`
	Dir    string `json:"dir"`
	// Validators is how many of a generated set join the validator set
	// (0 = all); it has effect only when Source generates.
	Validators int `json:"validators,omitempty"`
	// Blueprint, when set, is the declaration the keys and layout come from.
	Blueprint string `json:"blueprint,omitempty"`
}

// PlanGenesis says which genesis the nodes will start from.
type PlanGenesis struct {
	// Existing names a finished genesis used verbatim; empty builds from the
	// chain's template.
	Existing string `json:"existing,omitempty"`
	// ChainID overrides the manifest's chain id when non-zero.
	ChainID int64 `json:"chainID,omitempty"`
	// Hardforks are the fork knobs the declaration sets.
	Hardforks []string `json:"hardforks,omitempty"`
	// Overlay is the rendered overlay file, when the declaration carried one.
	Overlay string `json:"overlay,omitempty"`
}

// PlanSource is where one value in the plan came from.
//
// Merging is silent, so after it a value cannot say who chose it. That matters
// whenever someone has to go change the value: the declaration is a file they
// can open, the command is the line they just typed, and a harness default is
// neither — it is a value nobody asked for, which is the one a reader is most
// likely to be surprised by.
//
// These name the same idea as nodeconfig.Layer, which orders the layers that
// assemble one node's argv. They are a separate vocabulary because they answer
// a different question: that one asks which layer wins for a knob, this one
// asks who chose a value the plan shows, including values no argv ever carries
// (the binary, the node counts, the target).
type PlanSource string

const (
	// SourceDeclaration is the merged document: the chain declaration plus the
	// case's own overrides. A reader goes and edits a file.
	SourceDeclaration PlanSource = "declaration"
	// SourceCommand is the invocation. It names no layer and wins over every
	// document. A reader goes and edits the line they typed.
	SourceCommand PlanSource = "command"
	// SourceHarness is a default this code chose because neither of the other
	// two named the value. A reader has nothing to edit, which is exactly why
	// the plan has to say so out loud.
	SourceHarness PlanSource = "harness"
)

// PlanField names one value of the plan that can come from more than one
// source. Values with a single possible source are absent on purpose: a field
// whose answer is always the same word says nothing.
type PlanField string

const (
	FieldBinary     PlanField = "binary"
	FieldTarget     PlanField = "target"
	FieldNodesBP    PlanField = "nodes.bp"
	FieldNodesEN    PlanField = "nodes.en"
	FieldNodesPN    PlanField = "nodes.pn"
	FieldKeysSource PlanField = "keys.source"
	FieldKeysDir    PlanField = "keys.dir"
)

// planFields is the render order, which is the order the values appear above.
var planFields = []PlanField{
	FieldBinary, FieldTarget,
	FieldNodesBP, FieldNodesEN, FieldNodesPN,
	FieldKeysSource, FieldKeysDir,
}

// PlanKnob is one launch override and who asked for it.
type PlanKnob struct {
	Knob string     `json:"knob"`
	From PlanSource `json:"from"`
}

func (k PlanKnob) String() string { return k.Knob + " (" + string(k.From) + ")" }

// PlanHandoff is the upgrade form: two binaries and a profile, not a layout.
//
// The size fields are read from the profile, which owns them. A handoff's
// network is not sized by the declaration — the profile carries the producer
// and validator counts AND the per-node material that has to match them (the
// identity order, the validator addresses, the BLS keys, the pre-computed
// extradata), so a second document naming a size could only disagree with it.
// Showing them here is how an operator still sees the size without a case
// repeating it.
type PlanHandoff struct {
	Profile    string `json:"profile"`
	Template   string `json:"template,omitempty"`
	FromBinary string `json:"fromBinary"`
	ToBinary   string `json:"toBinary"`
	Producers  int    `json:"producers,omitempty"`
	Validators int    `json:"validators,omitempty"`
	AtFork     string `json:"atFork,omitempty"`
	ForkBlock  int64  `json:"forkBlock,omitempty"`
	NetworkID  int64  `json:"networkID,omitempty"`
}

// planOf renders the plan from the composer's input. It reads only what is
// already decided: nothing here computes a value the composition does not
// already hold.
func planOf(c composition, chain string) ComposePlan {
	if c.handoff != nil {
		h := c.handoff
		hp := &PlanHandoff{
			Profile: h.ProfilePath, Template: h.Template,
			FromBinary: h.FromBinary, ToBinary: h.ToBinary,
		}
		// Best effort on purpose. A profile that cannot be read fails the run a
		// moment later, with the message that knows why; a display that invents
		// a second failure path for the same cause only makes the first one
		// harder to find. The size fields stay zero and the renderer omits them.
		if prof, err := upgrade.LoadProfile(h.ProfilePath); err == nil {
			hp.Producers, hp.Validators = prof.Roles.Producers, prof.Roles.Validators
			hp.AtFork, hp.ForkBlock = prof.Upgrade.AtFork, prof.Upgrade.ForkBlock
			hp.NetworkID = prof.Upgrade.NetworkID
		}
		return ComposePlan{
			Chain:     chain,
			Workspace: h.DataDir,
			Target:    "this machine",
			Keys:      PlanKeys{Dir: h.KeysDir},
			Genesis:   PlanGenesis{Overlay: h.GenesisOverlay},
			Handoff:   hp,
		}
	}
	up := c.up
	p := ComposePlan{
		Chain:     up.Chain,
		Workspace: up.DataDir,
		Target:    describeTarget(up),
		Placement: up.Target,
		Binary:    up.Binary,
		Binaries:  up.Binaries,
		Nodes:     planNodes(up),
		Keys: PlanKeys{
			Source: up.KeysSource, Dir: up.KeysDir,
			Validators: up.KeysValidators, Blueprint: up.BlueprintPath,
		},
		Genesis: PlanGenesis{
			Existing: up.GenesisExisting, ChainID: up.ChainID,
			Hardforks: up.GenesisSet, Overlay: up.OverlayPath,
		},
		Config: up.ConfigSet,
		From:   c.from,
	}
	if p.Keys.Source == "" {
		p.Keys.Source = keySourceKeyPreset
	}
	p.Launch = mergeScopes(up.LaunchScoped, up.LaunchSet)
	return p
}

// mergeScopes folds the flat "all"-scope list into the scoped map so a reader
// sees one table, keeping which side each knob came from. The flat list is what
// the command passed and the scoped map is what the declaration asked for, and
// the command wins, so it goes last.
func mergeScopes(scoped map[string][]string, all []string) map[string][]PlanKnob {
	if len(scoped) == 0 && len(all) == 0 {
		return nil
	}
	out := make(map[string][]PlanKnob, len(scoped)+1)
	for k, v := range scoped {
		for _, knob := range v {
			out[k] = append(out[k], PlanKnob{Knob: knob, From: SourceDeclaration})
		}
	}
	for _, knob := range all {
		out[node.ScopeAll] = append(out[node.ScopeAll], PlanKnob{Knob: knob, From: SourceCommand})
	}
	return out
}

// planNodes reads the layout off whichever form the composition carries: a node
// table names every node, and its absence leaves the counts.
func planNodes(up *chainsetup.NetUpIn) PlanNodes {
	n := PlanNodes{Peering: up.Peering, SyncMode: up.EndpointSyncMode}
	if n.Peering == "" {
		n.Peering = "mesh"
	}
	if up.Topology == nil {
		n.BP, n.EN, n.PN = up.BPCount, up.ENCount, up.PNCount
		n.AutoSize = up.AutoSize
		return n
	}
	n.Declared = true
	for _, e := range up.Topology.Nodes {
		switch node.Role(e.Role) {
		case node.RoleBP:
			n.BP++
		case node.RoleEN:
			n.EN++
		case node.RolePN:
			n.PN++
		}
	}
	return n
}

// describeTarget says where the nodes will run.
//
// Two fields answer that and both have to be read. Server is the server-set
// entry a command selected; Target is the placement a declaration named, and it
// carries a host and data root of its own. Reading only Server printed "this
// machine" for a case whose env.target sent every node to another host — a plan
// that describes a different network than the one that launches, which is the
// one thing this file exists to prevent.
//
// What a placement reads as is resource's to say, not this row's: it owns the
// locality rule, so no display site decides what counts as remote
// (architecture-v2 §4).
func describeTarget(up *chainsetup.NetUpIn) string {
	var where string
	switch {
	case up.Server.All:
		where = "every server in the set"
	case up.Server.Name != "":
		where = "server " + up.Server.Name
	case up.Server.SetPath != "":
		where = "the server set's default entry"
	case up.Target != (resource.Spec{}):
		where = up.Target.Describe()
	default:
		where = "this machine"
	}
	if up.Docker {
		where += " (docker)"
	}
	return where
}

// String renders the plan for a terminal: one labelled line per decision, in
// the order an operator asks the questions.
func (p ComposePlan) String() string {
	var b strings.Builder
	row := func(label, value string) {
		if value != "" {
			fmt.Fprintf(&b, "  %-10s %s\n", label, value)
		}
	}
	if p.Handoff != nil {
		row("chain", p.Chain+"  (consensus handoff)")
		row("workspace", p.Workspace)
		row("profile", p.Handoff.Profile)
		row("template", p.Handoff.Template)
		row("binaries", fmt.Sprintf("from %s -> to %s", p.Handoff.FromBinary, p.Handoff.ToBinary))
		row("nodes", p.Handoff.nodesLine())
		row("fork", p.Handoff.forkLine())
		row("keys", p.Keys.Dir)
		row("overlay", p.Genesis.Overlay)
		return b.String()
	}
	row("chain", p.Chain)
	row("workspace", p.Workspace)
	row("target", p.Target)
	row("binary", p.binaryLine())
	row("nodes", p.Nodes.line())
	row("keys", p.Keys.line())
	row("genesis", p.Genesis.line())
	writeKnobScopes(row, "launch", p.Launch)
	writeScopes(row, "config", p.Config)
	row("chosen by", p.chosenByLine())
	return b.String()
}

// chosenByLine names the values the declaration did not choose.
//
// It lists only those, and the row disappears when there are none. A reader
// scanning a plan already assumes the document decided; what they need told is
// where that assumption is wrong, and a table that repeats "declaration" seven
// times buries the two rows that do not say it.
func (p ComposePlan) chosenByLine() string {
	var parts []string
	for _, f := range planFields {
		src := p.From[f]
		if src == "" || src == SourceDeclaration {
			continue
		}
		parts = append(parts, string(f)+": "+string(src))
	}
	return strings.Join(parts, " · ")
}

func (p ComposePlan) binaryLine() string {
	if len(p.Binaries) == 0 {
		return p.Binary
	}
	names := make([]string, 0, len(p.Binaries))
	for k := range p.Binaries {
		names = append(names, k)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, k := range names {
		parts = append(parts, k+"="+p.Binaries[k])
	}
	return fmt.Sprintf("%s  (per node: %s)", p.Binary, strings.Join(parts, ", "))
}

func (n PlanNodes) line() string {
	s := fmt.Sprintf("bp %d · en %d · pn %d", n.BP, n.EN, n.PN)
	if n.AutoSize {
		s += " (bp fills to the server set)"
	}
	if n.Declared {
		s += " (declared per node)"
	}
	s += ", peering " + n.Peering
	if n.SyncMode != "" {
		s += ", en sync " + n.SyncMode
	}
	return s
}

func (k PlanKeys) line() string {
	s := k.Source + "  " + k.Dir
	if k.Validators > 0 {
		s += fmt.Sprintf(", %d join the validator set", k.Validators)
	}
	if k.Blueprint != "" {
		s += ", blueprint " + k.Blueprint
	}
	return s
}

func (g PlanGenesis) line() string {
	s := "built from the chain template"
	if g.Existing != "" {
		s = "existing: " + g.Existing
	}
	if g.ChainID != 0 {
		s += fmt.Sprintf(", chain id %d", g.ChainID)
	}
	if len(g.Hardforks) > 0 {
		s += ", forks " + strings.Join(g.Hardforks, " ")
	}
	if g.Overlay != "" {
		s += ", overlay " + g.Overlay
	}
	return s
}

// writeKnobScopes prints the launch table, each knob with who asked for it.
func writeKnobScopes(row func(string, string), label string, m map[string][]PlanKnob) {
	if len(m) == 0 {
		return
	}
	plain := make(map[string][]string, len(m))
	for scope, knobs := range m {
		for _, k := range knobs {
			plain[scope] = append(plain[scope], k.String())
		}
	}
	writeScopes(row, label, plain)
}

// writeScopes prints one line per scope, most general first, so the reader sees
// the overrides in the order they will be applied.
func writeScopes(row func(string, string), label string, m map[string][]string) {
	if len(m) == 0 {
		return
	}
	scopes := make([]string, 0, len(m))
	for k := range m {
		scopes = append(scopes, k)
	}
	sort.Slice(scopes, func(i, j int) bool {
		ri, rj := node.ScopeRank(scopes[i]), node.ScopeRank(scopes[j])
		if ri != rj {
			return ri < rj
		}
		return scopes[i] < scopes[j]
	})
	for i, s := range scopes {
		head := ""
		if i == 0 {
			head = label
		}
		row(head, fmt.Sprintf("%-6s %s", s+":", strings.Join(m[s], ", ")))
	}
}

// PlanSuite resolves what RunSuite would compose and returns it without
// composing anything.
//
// It exists for the change that splits a declaration into a shared chain
// document and a case's overrides: the only way to know the split preserved a
// network is to compare the plan before it with the plan after, and comparing
// two runs means composing two networks. It resolves through the same function
// the runner does, so the two cannot disagree.
//
// One file may be written: a declaration carrying a genesis overlay renders it
// into the workspace. The name is the hash of the content, so planning twice
// writes the same bytes to the same path and a later run reuses it.
func PlanSuite(ctx context.Context, in RunSuiteIn) (ComposePlan, error) {
	if len(in.SpecPaths) == 0 && len(in.SpecContent) == 0 {
		return ComposePlan{}, fmt.Errorf("engine: plan suite: no specs given")
	}
	_, parsed, comp, err := resolveComposition(ctx, in)
	if err != nil {
		return ComposePlan{}, err
	}
	return planOf(comp, parsed[0].Chain.Name), nil
}

// nodesLine says how many nodes the profile sizes the handoff to, and where
// that number came from — the question a reader asks first and the one the
// declaration deliberately does not answer.
func (h PlanHandoff) nodesLine() string {
	if h.Producers == 0 && h.Validators == 0 {
		return ""
	}
	return fmt.Sprintf("producers %d · validators %d  (sized by the profile)", h.Producers, h.Validators)
}

// forkLine says where the handoff happens.
func (h PlanHandoff) forkLine() string {
	if h.AtFork == "" {
		return ""
	}
	s := fmt.Sprintf("%s at block %d", h.AtFork, h.ForkBlock)
	if h.NetworkID != 0 {
		s += fmt.Sprintf(", network id %d", h.NetworkID)
	}
	return s
}
