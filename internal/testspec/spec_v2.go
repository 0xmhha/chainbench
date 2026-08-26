package testspec

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

// SchemaV2 is the canonical v2 grammar (schema/v2.schema.json). The strict
// parser is the enforcement; this document is the field-level source of truth
// the parser, docs, and external tooling share.
//
//go:embed schema/v2.schema.json
var SchemaV2 []byte

// DSL v2 (docs/dev/dsl-v2-proposal.md). v2 separates the
// declaration (env — the fingerprint/reuse unit) from the scenario (case), and
// projects every predicate through one statement form:
//
//	{ "do": "<action>", ...complement, ...adjunct }
//	{ "expect": "<source>", "is": <expected>, ...adjunct }
//
// v2 is not a second runtime: a case LOWERS onto the v1 Spec the engine
// already executes, and v1 files desugar into the same unified statement
// sequence — one execution path, two grammars. Unknown fields are errors
// (strict): v1 let typos flow through map[string]any to the runtime.

// KindEnv and KindCase are the v2 file kinds.
const (
	KindEnv  = "env"
	KindCase = "case"
)

// schemaVersionV2 is the v2 grammar version.
const schemaVersionV2 = "2"

// Statement is one unified v2 statement: exactly one of Do or Expect names the
// head; Args carry complement and adjuncts (on/save/timeout/compare/...).
type Statement struct {
	// Do is the action name ("" when this is an expect statement).
	Do string
	// Expect is the assertion/source name ("" when this is a do statement).
	Expect string
	// Args are the statement's remaining fields, in the runtime's (v1) arg
	// vocabulary — lowering already renamed "is" to "expected".
	Args map[string]any
}

// EnvV2 is the v2 environment declaration — the reuse unit.
type EnvV2 struct {
	SchemaVersion string                    `json:"schemaVersion"`
	Kind          string                    `json:"kind"`
	ID            string                    `json:"id"`
	Target        string                    `json:"target,omitempty"`
	Chain         string                    `json:"chain"`
	Binaries      map[string]string         `json:"binaries,omitempty"`
	Keys          *KeysV2                   `json:"keys,omitempty"`
	Genesis       *GenesisV2                `json:"genesis,omitempty"`
	Topology      map[string]any            `json:"topology,omitempty"`
	Hardforks     map[string]int            `json:"hardforks,omitempty"`
	Launch        map[string]map[string]any `json:"launch,omitempty"`
	Config        string                    `json:"config,omitempty"`
	Capabilities  []string                  `json:"capabilities,omitempty"`
}

// KeysV2 declares where node identities come from (background 1.4/1.5,
// algorithm steps 2-3 — gap G1's grammar side).
type KeysV2 struct {
	NodeKeys *KeySourceV2 `json:"nodekeys,omitempty"`
}

// KeySourceV2 is one key-material source declaration.
type KeySourceV2 struct {
	// Source is preset (default) | generate.
	Source string `json:"source"`
	// Ref is the key-set directory.
	Ref string `json:"ref,omitempty"`
	// Bootnode named the external BLS-deriving binary. It is accepted so that
	// existing specs keep parsing, and ignored: BLS material is now derived in
	// process (keyring.Derive).
	//
	// Deprecated: has no effect.
	Bootnode string `json:"bootnode,omitempty"`
}

// GenesisV2 declares the genesis build (gap G2). The runtime's proven path is
// template + overlay; Set is dot-path sugar over the same overlay.
type GenesisV2 struct {
	// Mode is "template" (default). The other design modes (existing | build |
	// inherit) are declared in the design but have no runtime boundary yet; they
	// are rejected by name rather than silently treated as template.
	Mode string `json:"mode,omitempty"`
	// Set applies dot-path single values (e.g. "config.chainId": 8284).
	Set map[string]any `json:"set,omitempty"`
	// Overlay deep-merges into the built genesis.
	Overlay map[string]any `json:"overlay,omitempty"`
}

// HooksV2 are the case hooks. Override hooks (gap G5) are deliberately not
// accepted yet: parsing a declaration the runtime cannot execute would repeat
// the declared-but-never-emitted defect.
type HooksV2 struct {
	Pre    []map[string]any `json:"pre,omitempty"`
	Post   []map[string]any `json:"post,omitempty"`
	OnFail []map[string]any `json:"onFail,omitempty"`
}

// CaseV2 is the v2 scenario file.
type CaseV2 struct {
	SchemaVersion    string            `json:"schemaVersion"`
	Kind             string            `json:"kind"`
	ID               string            `json:"id"`
	Env              json.RawMessage   `json:"env"`
	ApplicableChains string            `json:"applicableChains,omitempty"`
	Requires         []string          `json:"requires,omitempty"`
	On               string            `json:"on,omitempty"`
	Timeouts         map[string]string `json:"timeouts,omitempty"`
	Hooks            *HooksV2          `json:"hooks,omitempty"`
	Steps            []map[string]any  `json:"steps"`
}

// sniff reads just enough to route a raw spec to its grammar.
type sniff struct {
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
}

// IsV2 reports whether raw declares the v2 grammar.
func IsV2(raw []byte) bool {
	var s sniff
	return json.Unmarshal(raw, &s) == nil && s.SchemaVersion == schemaVersionV2
}

// parseStrict decodes into out rejecting unknown fields.
func parseStrict(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

// ParseV2 parses a v2 case (env inline) and lowers it onto the executable
// Spec. An env file alone is not runnable and is rejected with guidance.
func ParseV2(raw []byte) (Spec, error) {
	var s sniff
	if err := json.Unmarshal(raw, &s); err != nil {
		return Spec{}, fmt.Errorf("testspec: parse v2: %w", err)
	}
	switch s.Kind {
	case KindCase:
		var c CaseV2
		if err := parseStrict(raw, &c); err != nil {
			return Spec{}, fmt.Errorf("testspec: parse v2 case: %w", err)
		}
		return lowerCase(c)
	case KindEnv:
		return Spec{}, fmt.Errorf("testspec: an env declaration is not runnable — reference it from a case (\"env\": \"<id>\")")
	default:
		return Spec{}, fmt.Errorf("testspec: v2 spec needs \"kind\": %q or %q", KindCase, KindEnv)
	}
}

// InlineEnv resolves a case's "env": "<id>" reference through lookup and
// rewrites the case with the env object inlined. A case with an inline env
// (or a v1 spec) passes through untouched. lookup receives the env id and
// returns the env file's bytes; the caller owns where env files live.
func InlineEnv(raw []byte, lookup func(id string) ([]byte, error)) ([]byte, error) {
	if !IsV2(raw) {
		return raw, nil
	}
	var probe struct {
		Kind string          `json:"kind"`
		Env  json.RawMessage `json:"env"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("testspec: inline env: %w", err)
	}
	if probe.Kind != KindCase || len(probe.Env) == 0 {
		return raw, nil
	}
	var id string
	if json.Unmarshal(probe.Env, &id) != nil || id == "" {
		return raw, nil // inline env object (or malformed — ParseV2 reports it)
	}
	if lookup == nil {
		return nil, fmt.Errorf("testspec: case references env %q but no env resolver is available", id)
	}
	envRaw, err := lookup(id)
	if err != nil {
		return nil, fmt.Errorf("testspec: resolve env %q: %w", id, err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("testspec: inline env: %w", err)
	}
	doc["env"] = envRaw
	return json.Marshal(doc)
}

// lowerCase lowers a v2 case (env inline) onto the executable Spec.
func lowerCase(c CaseV2) (Spec, error) {
	if c.ID == "" {
		return Spec{}, fmt.Errorf("testspec: v2 case needs \"id\"")
	}
	if len(c.Env) == 0 {
		return Spec{}, fmt.Errorf("testspec: v2 case %s needs \"env\" (an env id or an inline env object)", c.ID)
	}
	var envID string
	if json.Unmarshal(c.Env, &envID) == nil {
		return Spec{}, fmt.Errorf("testspec: case %s references env %q — resolve it with InlineEnv before parsing", c.ID, envID)
	}
	var env EnvV2
	if err := parseStrict(c.Env, &env); err != nil {
		return Spec{}, fmt.Errorf("testspec: case %s: env: %w", c.ID, err)
	}
	if env.Kind != "" && env.Kind != KindEnv {
		return Spec{}, fmt.Errorf("testspec: case %s: env kind is %q, want %q", c.ID, env.Kind, KindEnv)
	}
	if env.Chain == "" {
		return Spec{}, fmt.Errorf("testspec: case %s: env needs \"chain\"", c.ID)
	}

	spec := Spec{
		SchemaVersion:    supportedSchemaVersion, // lowered form IS the executable v1 shape
		ID:               c.ID,
		ApplicableChains: c.ApplicableChains,
		Requires:         c.Requires,
		Chain:            ChainSpec{Name: env.Chain, Config: env.Config},
		Topology:         env.Topology,
		Hardforks:        env.Hardforks,
		Placement:        env.Target,
		DefaultOn:        c.On,
		Timeouts:         c.Timeouts,
	}
	if len(env.Capabilities) > 0 && len(spec.Requires) == 0 {
		spec.Requires = env.Capabilities
	}

	// Binaries: "default" is every node's binary; other keys are per-role.
	if b, ok := env.Binaries["default"]; ok && len(env.Binaries) == 1 {
		spec.Chain.Binary = b
	} else if len(env.Binaries) > 0 {
		spec.Chain.Binaries = env.Binaries
	}

	// Genesis: template(+overlay/set) is the runtime's proven path; the other
	// declared modes have no support yet and are rejected by name (G2 partial).
	if g := env.Genesis; g != nil {
		if g.Mode != "" && g.Mode != "template" {
			return Spec{}, fmt.Errorf("testspec: case %s: genesis mode %q has no runtime boundary yet (supported: template)", c.ID, g.Mode)
		}
		overlay := map[string]any{}
		maps.Copy(overlay, g.Overlay)
		for path, v := range g.Set {
			mergeDotPath(overlay, path, v)
		}
		if len(overlay) > 0 {
			spec.Chain.GenesisOverlay = overlay
		}
	}

	// Keys/launch declarations carry through for the surface (cmd run) to fold
	// into the engine's construction boundaries.
	if env.Keys != nil && env.Keys.NodeKeys != nil {
		spec.EnvKeys = env.Keys.NodeKeys
	}
	if len(env.Launch) > 0 {
		all, ok := env.Launch["all"]
		if !ok || len(env.Launch) > 1 {
			return Spec{}, fmt.Errorf("testspec: case %s: launch supports the \"all\" scope today; role-scoped launch lands with per-role wiring", c.ID)
		}
		for k, v := range all {
			spec.EnvLaunch = append(spec.EnvLaunch, LaunchKV{Key: k, Value: fmt.Sprintf("%v", v)})
		}
	}

	// Hooks.
	if h := c.Hooks; h != nil {
		var err error
		if spec.PreActions, err = lowerHookActions(c.ID, "pre", h.Pre); err != nil {
			return Spec{}, err
		}
		if spec.PostActions, err = lowerHookActions(c.ID, "post", h.Post); err != nil {
			return Spec{}, err
		}
		if spec.OnFailActions, err = lowerHookActions(c.ID, "onFail", h.OnFail); err != nil {
			return Spec{}, err
		}
	}

	// Statements.
	if len(c.Steps) == 0 {
		return Spec{}, fmt.Errorf("testspec: case %s has no steps", c.ID)
	}
	expects := 0
	for i, raw := range c.Steps {
		st, err := lowerStatement(raw)
		if err != nil {
			return Spec{}, fmt.Errorf("testspec: case %s: step %d: %w", c.ID, i+1, err)
		}
		if st.Expect != "" {
			expects++
			spec.Assertions = append(spec.Assertions, statementAssertion(st))
		}
		spec.Sequence = append(spec.Sequence, st)
	}
	if expects == 0 {
		return Spec{}, fmt.Errorf("testspec: case %s verifies nothing — at least one expect statement is required", c.ID)
	}
	return spec, nil
}

// expectAliases maps proposal-vocabulary source names onto registered
// assertion names.
var expectAliases = map[string]string{"rpc": "rpcCall"}

// lowerStatement lowers one v2 statement map onto the runtime vocabulary.
func lowerStatement(m map[string]any) (Statement, error) {
	doName, hasDo := m["do"].(string)
	exName, hasEx := m["expect"].(string)
	// A do statement may carry expect as an ADJUNCT ("expect":"receipt"|"revert"
	// on sendTx). It is a statement head only when "do" is absent.
	if !hasDo && !hasEx {
		return Statement{}, fmt.Errorf("statement needs \"do\" or \"expect\"")
	}
	if _, isOverride := m["override"]; isOverride {
		return Statement{}, fmt.Errorf("override hooks (G5) have no execution semantics yet and are not accepted")
	}
	args := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "do":
			// head, not an arg
		case "is":
			args["expected"] = v
		default:
			args[k] = v
		}
	}
	if hasDo {
		return Statement{Do: doName, Args: args}, nil
	}
	delete(args, "expect")
	if alias, ok := expectAliases[exName]; ok {
		exName = alias
	}
	return Statement{Expect: exName, Args: args}, nil
}

// statementAssertion renders an expect statement in the v1 assertion map shape.
func statementAssertion(st Statement) map[string]any {
	out := make(map[string]any, len(st.Args)+1)
	maps.Copy(out, st.Args)
	out["assert"] = st.Expect
	return out
}

// statementStep renders a do statement in the v1 step map shape.
func statementStep(st Statement) map[string]any {
	return map[string]any{st.Do: st.Args}
}

// mergeDotPath sets a dot-path value inside a nested map, creating levels.
func mergeDotPath(m map[string]any, path string, v any) {
	parts := strings.Split(path, ".")
	cur := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
	cur[parts[len(parts)-1]] = v
}

// lowerHookActions lowers hook statements (do form) onto v1 action maps.
func lowerHookActions(caseID, hook string, stmts []map[string]any) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(stmts))
	for i, raw := range stmts {
		st, err := lowerStatement(raw)
		if err != nil {
			return nil, fmt.Errorf("testspec: case %s: hooks.%s[%d]: %w", caseID, hook, i, err)
		}
		if st.Do == "" {
			return nil, fmt.Errorf("testspec: case %s: hooks.%s[%d]: hooks take do statements", caseID, hook, i)
		}
		out = append(out, statementStep(st))
	}
	return out, nil
}
