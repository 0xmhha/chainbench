package dsl

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ChainSpec selects the chain, its binary/binaries, and genesis overlay for a
// test. A single Binary applies to all nodes; Binaries maps roles to binaries
// for mixed (handoff) environments.
type ChainSpec struct {
	Name           string            `json:"name"`
	Binary         string            `json:"binary,omitempty"`
	Binaries       map[string]string `json:"binaries,omitempty"`
	Config         string            `json:"config,omitempty"`
	GenesisOverlay map[string]any    `json:"genesisOverlay,omitempty"`
}

// LaunchKV is one env.launch knob carried from a v2 declaration for the
// surface to fold into the engine's launch-override boundary.
type LaunchKV struct {
	Key   string
	Value string
}

// Spec is a parsed, validated test definition (schema in design §4.3).
type Spec struct {
	SchemaVersion    string            `json:"schemaVersion"`
	ID               string            `json:"id"`
	ApplicableChains string            `json:"applicableChains,omitempty"`
	Requires         []string          `json:"requires,omitempty"`
	Chain            ChainSpec         `json:"chain"`
	Topology         map[string]any    `json:"topology,omitempty"`
	Hardforks        map[string]int    `json:"hardforks,omitempty"`
	Placement        string            `json:"placement,omitempty"`
	DefaultOn        string            `json:"defaultOn,omitempty"`
	PreActions       []map[string]any  `json:"preActions,omitempty"`
	Steps            []map[string]any  `json:"steps,omitempty"`
	Assertions       []map[string]any  `json:"assertions"`
	PostActions      []map[string]any  `json:"postActions,omitempty"`
	Timeouts         map[string]string `json:"timeouts,omitempty"`

	// Sequence is the unified statement list (v2 steps; v1 desugars its steps
	// then assertions into it). Runtime-only — never serialized.
	Sequence []Statement `json:"-"`
	// OnFailActions run when the case fails (v2 hooks.onFail). Runtime-only.
	OnFailActions []map[string]any `json:"-"`
	// EnvKeys is the v2 env's node-key source declaration, for the surface to
	// fold into the engine's KeySource boundary. Runtime-only.
	EnvKeys *KeySourceV2 `json:"-"`
	// EnvLaunch are the v2 env.launch knobs (the "all" scope), for the
	// surface to fold into the engine's launch-override boundary. Runtime-only.
	EnvLaunch []LaunchKV `json:"-"`
	// EnvUpgrade is the v2 env's handoff declaration, for the composer to run
	// the network as a mixed-binary handoff. Nil is a single-binary network.
	// Runtime-only.
	EnvUpgrade *UpgradeV2 `json:"-"`
}

// Parse routes raw JSON to its grammar (v1, or v2 by schemaVersion sniff),
// validates it, and returns the executable Spec. A v2 case referencing an env
// by id must be resolved with InlineEnv first — the caller owns file lookup.
func Parse(raw []byte) (Spec, error) {
	if IsV2(raw) {
		return ParseV2(raw)
	}
	var s Spec
	if err := json.Unmarshal(raw, &s); err != nil {
		return Spec{}, fmt.Errorf("dsl: parse: %w", err)
	}
	if err := s.validate(); err != nil {
		return Spec{}, err
	}
	return s, nil
}

// supportedSchemaVersion is the only spec schemaVersion the interpreter accepts.
// Specs are long-lived assets, so an unknown version is rejected explicitly
// (forward-compat guard, F16-O2) rather than parsed on a best-effort basis.
const supportedSchemaVersion = "1"

// validate reports the first set of missing required fields, naming each, then
// rejects an unsupported schemaVersion.
func (s Spec) validate() error {
	var missing []string
	if s.SchemaVersion == "" {
		missing = append(missing, "schemaVersion")
	}
	if s.ID == "" {
		missing = append(missing, "id")
	}
	if s.Chain.Name == "" {
		missing = append(missing, "chain.name")
	}
	if s.Chain.Binary == "" && len(s.Chain.Binaries) == 0 {
		missing = append(missing, "chain.binary|binaries")
	}
	if len(s.Assertions) == 0 {
		missing = append(missing, "assertions")
	}
	if len(missing) > 0 {
		return fmt.Errorf("dsl: missing required field(s): %s", strings.Join(missing, ", "))
	}
	if s.SchemaVersion != supportedSchemaVersion {
		return fmt.Errorf("dsl: unsupported schemaVersion %q (supported: %s)", s.SchemaVersion, supportedSchemaVersion)
	}
	return nil
}

// Get resolves a dot-path (a.b.c) within the spec. ok is false when the path is
// absent. It navigates the spec's JSON form, so paths use the JSON field names.
func (s Spec) Get(dotPath string) (any, bool) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, false
	}
	var cur any
	if err := json.Unmarshal(b, &cur); err != nil {
		return nil, false
	}
	for _, part := range strings.Split(dotPath, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}
