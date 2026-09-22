package chainsetup

import (
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
)

// Recording who asked for each value a launch uses.
//
// A config knob or a launch flag can come from the declaration, from the
// command, or from the harness's own default, and the one that surprises a
// reader is the third — nothing they wrote chose it. Keeping the source beside
// the value is what lets a plan say so.

// errBuildBadOption is the one way the command stage fails: a launch knob the
// vocabulary does not take, or a --set that is not one.
var errBuildBadOption = errors.New("a launch option is not one this accepts")

// errBuildSplitNetwork is the other: the commands were assembled and do not all
// name the same devp2p network, so the nodes would come up unable to peer.
var errBuildSplitNetwork = errors.New("the assembled commands name more than one network")

// applyConfigOverrides applies the workspace's config-knob overrides to one
// node's spec, most-general-first, so the narrowest scope wins. Each entry is a
// dot-path "key=value"; an unknown key or a malformed entry is an error, never
// a silent no-op.
func (w *Workspace) applyConfigOverrides(spec *nodeconfig.Spec, role node.Role, index int) error {
	for _, kv := range w.configOverridesFor(role, index) {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || key == "" {
			return lifecycle.Mark(errConfigBadOverride, fmt.Errorf("config override %q must be key=value", kv))
		}
		if err := nodeconfig.ApplyConfigOverride(spec, key, value); err != nil {
			return lifecycle.Mark(errConfigBadOverride, err)
		}
	}
	return nil
}

// configOverridesFor returns the config overrides that apply to one node,
// most-general-first: "all", then the node's role, then the node itself, so the
// narrowest scope wins.
//
// It is the single source of which overrides shape a node's config —
// applyConfigOverrides applies them and the config step records them as
// provenance, so the two never diverge. A node reads only its own "node<N>"
// scope, so one node's override never leaks into another's config.
func (w *Workspace) configOverridesFor(role node.Role, index int) []string {
	var out []string
	for _, scope := range node.ScopeFor(role, index) {
		out = append(out, w.state.ConfigSet[scope]...)
	}
	return out
}

// ConfigProvenance is one REVISION of one node's config: the overrides applied to
// it, the checksum ("sha256:<hex>") of the config that resulted, and when. Fixture
// names a mid-test config swap (config-<test-purpose>), empty for the initial
// compose.
//
// Revisions accumulate. The word was already in this file's comments — "a fresh
// config is a new revision" — while the writer replaced the node's entry, so a
// swapNode erased the config the node had been composed with. What a run is asked
// afterwards is "which config did node N have, and since when", and one entry with
// no time answers neither half. A fresh compose clears the list; a swap appends.
type ConfigProvenance struct {
	Node      int      `json:"node"`
	Fixture   string   `json:"fixture,omitempty"`
	Overrides []string `json:"overrides,omitempty"`
	Checksum  string   `json:"checksum"`
	// At is when this revision was applied (RFC3339, UTC). The requirement asks
	// for the node AND the time, and without it two revisions of one node's
	// config cannot be ordered — which is the only question a swap makes anyone
	// ask.
	At string `json:"at,omitempty"`
}

// RecordLaunchSet stores launch-argv overrides under a scope ("all", a role,
// or "node<N>"). Each entry is validated as a launch override up front, so a
// bad knob is refused where it is set rather than at argv assembly.
//
// A scope holds one entry per key. Setting a key it already has replaces that
// entry where it stands rather than appending a second one: argv assembly is
// last-write-wins, so two entries for one key mean the same node either way,
// and the record is what says what was ASKED for. It used to append
// unconditionally, so composing the same declaration twice over one workspace
// wrote the knob twice and a workspace reused all week grew a line per run.
// Replacing in place keeps the order a reader sees stable across runs.
func (w *Workspace) RecordLaunchSet(scope string, sets []string) error {
	if len(sets) == 0 {
		return nil
	}
	if !node.ValidScope(scope) {
		return lifecycle.Mark(errBuildBadOption,
			fmt.Errorf("launch scope %q must be %s", scope, node.ScopeWords()))
	}
	overrides, err := ParseOverrides(sets)
	if err != nil {
		return lifecycle.Mark(errBuildBadOption, err)
	}
	if err := refuseSharedPerNodeKnob(scope, overrides); err != nil {
		return err
	}
	if w.state.LaunchSet == nil {
		w.state.LaunchSet = map[string][]string{}
	}
	w.state.LaunchSet[scope] = holdOnePerKey(w.state.LaunchSet[scope], sets)
	return nil
}

// refuseSharedPerNodeKnob refuses a knob only one node can be told, named on a
// scope that covers more than one.
//
// The allocator gives each node its own port slot, data root and key files, and
// one value handed to a scope replaces all of them with the same one. Measured
// before this existed: a two-node network with launch.all.port=39999 assembled
// both nodes with --port 39999, so the second could not bind and the network
// that came up was not the one declared.
//
// Refusing rather than ignoring is the point. The line this repository fixed
// puts the allocator above a declaration, which would mean dropping the value —
// and a value silently dropped is the shape of every defect this track has
// found. A scope naming one node is allowed: overriding node1's port is a thing
// somebody may mean, and it collapses nothing.
func refuseSharedPerNodeKnob(scope string, overrides []nodeconfig.Override) error {
	if node.ScopeIndex(scope) > 0 {
		return nil
	}
	for _, o := range overrides {
		if !nodeconfig.IsPerNode(o.Key) {
			continue
		}
		return lifecycle.Mark(errBuildBadOption, fmt.Errorf(
			"chainsetup: launch %q on scope %q: the allocator gives each node its own, so one value for several nodes would collide — name a single node (node1) or change it in the server set. Per-node knobs: %s",
			o.Key, scope, strings.Join(nodeconfig.PerNodeKeys(), ", ")))
	}
	return nil
}

// holdOnePerKey folds new overrides into what a scope already holds, keeping
// one entry per key. A key already there is replaced where it stands; a new one
// is appended.
//
// An override is "key=value" or a bare key, and the key is the text before the
// first "=" — the same split ParseOverrides makes, so the two agree on what
// counts as one key. Replacing in place rather than appending keeps the order a
// reader sees stable no matter how many times a workspace is recomposed.
func holdOnePerKey(held, sets []string) []string {
	at := make(map[string]int, len(held))
	for i, kv := range held {
		key, _, _ := strings.Cut(kv, "=")
		at[key] = i
	}
	for _, kv := range sets {
		key, _, _ := strings.Cut(kv, "=")
		if j, ok := at[key]; ok {
			held[j] = kv
			continue
		}
		at[key] = len(held)
		held = append(held, kv)
	}
	return held
}

// launchOverridesFor returns the launch-argv overrides for one node, folding the
// scopes that apply to it most-general-first: "all", then the node's role, then
// the node itself. Later entries win at assembly (nodeconfig.Argv override
// layer is last-write-wins), so a node override beats a role override beats all.
func (w *Workspace) launchOverridesFor(role string, index int) []string {
	var out []string
	for _, scope := range node.ScopeFor(node.Role(role), index) {
		out = append(out, w.state.LaunchSet[scope]...)
	}
	// Last, so nothing a document says can be more specific than the line the
	// operator typed.
	return append(out, w.state.LaunchCommand...)
}

// RecordLaunchCommand stores what the invocation overrode. It is held to the
// same rules a scope is — the knobs are the same knobs, and an invocation
// speaks for every node, so a per-node knob collides here exactly as it does on
// scope "all".
func (w *Workspace) RecordLaunchCommand(sets []string) error {
	if len(sets) == 0 {
		return nil
	}
	overrides, err := ParseOverrides(sets)
	if err != nil {
		return lifecycle.Mark(errBuildBadOption, err)
	}
	if err := refuseSharedPerNodeKnob(node.ScopeAll, overrides); err != nil {
		return err
	}
	w.state.LaunchCommand = holdOnePerKey(w.state.LaunchCommand, sets)
	return nil
}

// RecordConfigSet stores config overrides under a scope, one entry per key, the
// same way RecordLaunchSet does and for the same reason: render is
// last-write-wins, so a second entry for one key means the same node either way
// and only makes the record grow every time the workspace is recomposed.
//
// Both halves are validated up front, so a bad override is refused where it is
// set rather than at render. The scope was not checked at all before: a typo
// stored values under a key nothing reads, and the node it was meant for came
// up with a config that silently lacked them.
func (w *Workspace) RecordConfigSet(scope string, sets []string) error {
	if len(sets) == 0 {
		return nil
	}
	if !node.ValidScope(scope) {
		return lifecycle.Mark(errConfigBadOverride,
			fmt.Errorf("config scope %q must be %s", scope, node.ScopeWords()))
	}
	var probe nodeconfig.Spec
	for _, kv := range sets {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || key == "" {
			return lifecycle.Mark(errConfigBadOverride, fmt.Errorf("config override %q must be key=value", kv))
		}
		if err := nodeconfig.ApplyConfigOverride(&probe, key, value); err != nil {
			return lifecycle.Mark(errConfigBadOverride, err)
		}
	}
	if w.state.ConfigSet == nil {
		w.state.ConfigSet = map[string][]string{}
	}
	w.state.ConfigSet[scope] = holdOnePerKey(w.state.ConfigSet[scope], sets)
	return nil
}

// markStep records that step ran with detail, stamping the completion time.

// BuildFailure is what the command stage's failures are.
//
// The default is assembling itself: a peer list that cannot be built, a node
// whose plugin cannot be resolved. A bad option is a line somebody wrote, a
// split network is the commands disagreeing with each other, and the rest are
// the composition disagreeing with itself.
func BuildFailure(err error) lifecycle.Status {
	switch {
	case errors.Is(err, errBuildBadOption):
		return lifecycle.ChainBuildNodeCommandFailBadOption
	case errors.Is(err, errBuildSplitNetwork):
		return lifecycle.ChainBuildNodeCommandFailSplitNetwork
	}
	return lifecycle.FailStageUnclassified
}
