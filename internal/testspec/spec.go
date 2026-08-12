package testspec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/session"
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
// surface to fold into the engine's launch-override seam.
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
	// fold into the engine's KeySource seam. Runtime-only.
	EnvKeys *KeySourceV2 `json:"-"`
	// EnvLaunch are the v2 env.launch knobs (the "all" scope), for the
	// surface to fold into the engine's launch-override seam. Runtime-only.
	EnvLaunch []LaunchKV `json:"-"`
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
		return Spec{}, fmt.Errorf("testspec: parse: %w", err)
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
		return fmt.Errorf("testspec: missing required field(s): %s", strings.Join(missing, ", "))
	}
	if s.SchemaVersion != supportedSchemaVersion {
		return fmt.Errorf("testspec: unsupported schemaVersion %q (supported: %s)", s.SchemaVersion, supportedSchemaVersion)
	}
	return nil
}

// fpInput is the canonical, order-independent hash input for a fingerprint.
// json.Marshal sorts map keys, so equal declared values hash equally regardless
// of insertion order.
type fpInput struct {
	Binary         string            `json:"binary"`
	Binaries       map[string]string `json:"binaries"`
	Config         string            `json:"config"`
	GenesisOverlay map[string]any    `json:"genesisOverlay"`
	Topology       map[string]any    `json:"topology"`
	Hardforks      map[string]int    `json:"hardforks"`
	Placement      string            `json:"placement"`
	Resolved       map[string]string `json:"resolved"`
}

// Fingerprint hashes the resolved declared values
// (binaries+genesis+config+topology+hardforks+placement) to a reuse key. config
// comes from resolved; the rest come from the receiver. It never touches a chain.
func (s Spec) Fingerprint(resolved config.Values) session.Fingerprint {
	in := fpInput{
		Binary:         s.Chain.Binary,
		Binaries:       s.Chain.Binaries,
		Config:         s.Chain.Config,
		GenesisOverlay: s.Chain.GenesisOverlay,
		Topology:       s.Topology,
		Hardforks:      s.Hardforks,
		Placement:      s.Placement,
		Resolved:       resolved,
	}
	b, err := json.Marshal(in)
	if err != nil {
		// Inputs originate from JSON, so this should not happen; fall back to a
		// stable representation rather than silently ignoring the error.
		b = []byte(fmt.Sprintf("fingerprint-error:%v", err))
	}
	sum := sha256.Sum256(b)
	return session.Fingerprint(hex.EncodeToString(sum[:]))
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
