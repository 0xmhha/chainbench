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
	"github.com/0xmhha/chainbench/internal/core/node"
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
	Launch map[string][]string `json:"launch,omitempty"`
	Config map[string][]string `json:"config,omitempty"`

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

// PlanHandoff is the upgrade form: two binaries and a profile, not a layout.
type PlanHandoff struct {
	Profile    string `json:"profile"`
	Template   string `json:"template,omitempty"`
	FromBinary string `json:"fromBinary"`
	ToBinary   string `json:"toBinary"`
}

// planOf renders the plan from the composer's input. It reads only what is
// already decided: nothing here computes a value the composition does not
// already hold.
func planOf(c composition, chain string) ComposePlan {
	if c.handoff != nil {
		h := c.handoff
		return ComposePlan{
			Chain:     chain,
			Workspace: h.DataDir,
			Target:    "this machine",
			Keys:      PlanKeys{Dir: h.KeysDir},
			Genesis:   PlanGenesis{Overlay: h.GenesisOverlay},
			Handoff: &PlanHandoff{
				Profile: h.ProfilePath, Template: h.Template,
				FromBinary: h.FromBinary, ToBinary: h.ToBinary,
			},
		}
	}
	up := c.up
	p := ComposePlan{
		Chain:     up.Chain,
		Workspace: up.DataDir,
		Target:    describeTarget(up),
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
	}
	if p.Keys.Source == "" {
		p.Keys.Source = keySourceKeyPreset
	}
	p.Launch = mergeScopes(up.LaunchScoped, up.LaunchSet)
	return p
}

// mergeScopes folds the flat "all"-scope list into the scoped map so a reader
// sees one table. The flat list is what the command passed and the scoped map
// is what the declaration asked for, and the command wins, so it goes last.
func mergeScopes(scoped map[string][]string, all []string) map[string][]string {
	if len(scoped) == 0 && len(all) == 0 {
		return nil
	}
	out := make(map[string][]string, len(scoped)+1)
	for k, v := range scoped {
		out[k] = append([]string(nil), v...)
	}
	if len(all) > 0 {
		out[node.ScopeAll] = append(out[node.ScopeAll], all...)
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

// describeTarget says where the nodes run in one phrase.
func describeTarget(up *chainsetup.NetUpIn) string {
	var where string
	switch {
	case up.Server.All:
		where = "every server in the set"
	case up.Server.Name != "":
		where = "server " + up.Server.Name
	case up.Server.SetPath != "":
		where = "the server set's default entry"
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
	writeScopes(row, "launch", p.Launch)
	writeScopes(row, "config", p.Config)
	return b.String()
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
