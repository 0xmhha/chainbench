package chainsetup

import (
	"fmt"
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

func (w *Workspace) applyConfigOverrides(spec *nodeconfig.Spec, role node.Role, index int) error {
	for _, kv := range w.configOverridesFor(role, index) {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || key == "" {
			return ofKind(errConfigBadOverride, fmt.Errorf("config override %q must be key=value", kv))
		}
		if err := nodeconfig.ApplyConfigOverride(spec, key, value); err != nil {
			return ofKind(errConfigBadOverride, err)
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

// recordLaunchSet stores launch-argv overrides under a scope ("all", a role,
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
func (w *Workspace) recordLaunchSet(scope string, sets []string) error {
	if len(sets) == 0 {
		return nil
	}
	if !node.ValidScope(scope) {
		return fmt.Errorf("launch scope %q must be %s", scope, node.ScopeWords())
	}
	if _, err := ParseOverrides(sets); err != nil {
		return err
	}
	if w.state.LaunchSet == nil {
		w.state.LaunchSet = map[string][]string{}
	}
	w.state.LaunchSet[scope] = holdOnePerKey(w.state.LaunchSet[scope], sets)
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
	return out
}

// recordConfigSet stores config overrides under a scope, one entry per key, the
// same way recordLaunchSet does and for the same reason: render is
// last-write-wins, so a second entry for one key means the same node either way
// and only makes the record grow every time the workspace is recomposed.
//
// Both halves are validated up front, so a bad override is refused where it is
// set rather than at render. The scope was not checked at all before: a typo
// stored values under a key nothing reads, and the node it was meant for came
// up with a config that silently lacked them.
func (w *Workspace) recordConfigSet(scope string, sets []string) error {
	if len(sets) == 0 {
		return nil
	}
	if !node.ValidScope(scope) {
		return ofKind(errConfigBadOverride,
			fmt.Errorf("config scope %q must be %s", scope, node.ScopeWords()))
	}
	var probe nodeconfig.Spec
	for _, kv := range sets {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || key == "" {
			return ofKind(errConfigBadOverride, fmt.Errorf("config override %q must be key=value", kv))
		}
		if err := nodeconfig.ApplyConfigOverride(&probe, key, value); err != nil {
			return ofKind(errConfigBadOverride, err)
		}
	}
	if w.state.ConfigSet == nil {
		w.state.ConfigSet = map[string][]string{}
	}
	w.state.ConfigSet[scope] = holdOnePerKey(w.state.ConfigSet[scope], sets)
	return nil
}

// markStep records that step ran with detail, stamping the completion time.
