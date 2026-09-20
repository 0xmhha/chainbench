package dsl

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"time"
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
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
	ID            string `json:"id"`
	// Description says what this environment is for, in prose. It carries no
	// execution semantics; it exists so a declaration can explain itself where
	// it is read, rather than in a file beside it.
	Description string `json:"description,omitempty"`
	Target      string `json:"target,omitempty"`
	Chain       string `json:"chain"`
	// Manifest is an external, project-supplied chain manifest JSON, run on the
	// built-in family named by Chain; GenesisTemplate is its genesis template.
	// They are the DSL equivalent of the CLI's --manifest/--genesis-template.
	Manifest        string            `json:"manifest,omitempty"`
	GenesisTemplate string            `json:"genesisTemplate,omitempty"`
	Binaries        map[string]string `json:"binaries,omitempty"`
	Keys            *KeysV2           `json:"keys,omitempty"`
	// Blueprint is a network declaration file — the layout AND the node keys in
	// one document. With it, no topology or key set is needed; it is the DSL
	// equivalent of the CLI's --blueprint.
	Blueprint    string                    `json:"blueprint,omitempty"`
	Genesis      *GenesisV2                `json:"genesis,omitempty"`
	Topology     map[string]any            `json:"topology,omitempty"`
	Hardforks    map[string]int            `json:"hardforks,omitempty"`
	Launch       map[string]map[string]any `json:"launch,omitempty"`
	Config       map[string]map[string]any `json:"config,omitempty"`
	Capabilities []string                  `json:"capabilities,omitempty"`
	// Accounts declares test accounts by name, created and funded when the
	// network comes up. They are not in the genesis on purpose: an account
	// funded at run time is one the genesis never has to mention, so preparing
	// a test account stops meaning editing the genesis and re-deriving
	// everything downstream of it.
	Accounts map[string]AccountV2 `json:"accounts,omitempty"`
	// Upgrade declares a mixed-binary handoff: the network starts on the
	// producer's binary and forks to the validators'. With it, Binaries names
	// the two roles ("producer", "validator") rather than a default.
	Upgrade *UpgradeV2 `json:"upgrade,omitempty"`
}

// AccountV2 is one declared test account.
//
// An empty declaration is deliberate and useful: an account with no balance is
// what a test needs to exercise the paths that fail for want of gas.
type AccountV2 struct {
	// Fund is the balance to send it once the chain is up, in wei (decimal or
	// 0x-hex). Empty leaves the account at zero.
	Fund string `json:"fund,omitempty"`
}

// The two binaries an upgrade env names, by the key it names them under.
//
// They are spelled Binary rather than Role because they are not node roles.
// A handoff names the binary that seals up to the fork and the one that takes
// over after it, and calling that a "role" put a third meaning on a word that
// already meant a node's job and an account's function (A7).
const (
	// BinaryBefore seals up to the fork.
	BinaryBefore = "producer"
	// BinaryAfter takes over after it.
	BinaryAfter = "validator"
)

// UpgradeV2 declares a handoff composition: which golden profile shapes it
// and which genesis template the producer's binary generates from. It is a
// declaration only; the composer that runs it lives above the grammar.
type UpgradeV2 struct {
	// Profile is the golden upgrade profile (profiles/*.yaml).
	Profile string `json:"profile"`
	// Template is the producer chain's own genesis template.
	Template string `json:"template"`
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
	// Validators is how many of a generated set join the validator set (0 =
	// all). It has effect only with source "generate": it composes a network
	// whose key set has more identities than producers, the DSL equivalent of
	// the keys step's --validators.
	Validators int `json:"validators,omitempty"`
	// Bootnode named the external BLS-deriving binary. It is accepted so that
	// existing specs keep parsing, and ignored: BLS material is now derived in
	// process (derive.Derive).
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
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
	ID            string `json:"id"`
	// Description says what this case verifies, in prose. Strict parsing means
	// a case cannot carry a note unless the grammar has a place for one, and a
	// test that cannot say what it is for is read by opening its steps.
	Description      string            `json:"description,omitempty"`
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
		return Spec{}, fmt.Errorf("dsl: parse v2: %w", err)
	}
	switch s.Kind {
	case KindCase:
		var c CaseV2
		if err := parseStrict(raw, &c); err != nil {
			return Spec{}, fmt.Errorf("dsl: parse v2 case: %w", err)
		}
		return lowerCase(c)
	case KindEnv:
		return Spec{}, fmt.Errorf("dsl: an env declaration is not runnable — reference it from a case (\"env\": \"<id>\")")
	default:
		return Spec{}, fmt.Errorf("dsl: v2 spec needs \"kind\": %q or %q", KindCase, KindEnv)
	}
}

// IsEnv reports whether raw is a v2 env declaration: a file that is not
// runnable on its own but is what cases reference.
func IsEnv(raw []byte) bool {
	var s sniff
	return json.Unmarshal(raw, &s) == nil && s.SchemaVersion == schemaVersionV2 && s.Kind == KindEnv
}

// ParseEnv parses a v2 env declaration strictly, so a declaration can be
// validated on its own before any case references it.
func ParseEnv(raw []byte) (EnvV2, error) {
	var env EnvV2
	if err := parseStrict(raw, &env); err != nil {
		return EnvV2{}, fmt.Errorf("dsl: env: %w", err)
	}
	if env.Kind != KindEnv {
		return EnvV2{}, fmt.Errorf("dsl: env kind is %q, want %q", env.Kind, KindEnv)
	}
	if env.ID == "" || env.Chain == "" {
		return EnvV2{}, fmt.Errorf("dsl: env needs \"id\" and \"chain\"")
	}
	return env, nil
}

// InlineEnv resolves a case's env reference through lookup and rewrites the
// case with the env object inlined. Three forms are accepted:
//
//	"env": "<id>"                              — the canonical env, verbatim.
//	"env": {"extends": "<id>", "topology": …}  — the canonical env with the
//	                                             named fields overridden.
//	"env": { …a full env object… }             — inline, no lookup.
//
// The override form is a shallow top-level merge: each field the case names
// replaces that field of the canonical env whole (topology, hardforks, keys),
// so a test declares only what differs. A case with an inline env (or a v1
// spec) passes through untouched. lookup receives the env id and returns the
// env file's bytes; the caller owns where env files live.
func InlineEnv(raw []byte, lookup func(id string) ([]byte, error)) ([]byte, error) {
	if !IsV2(raw) {
		return raw, nil
	}
	var probe struct {
		Kind string          `json:"kind"`
		Env  json.RawMessage `json:"env"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("dsl: inline env: %w", err)
	}
	if probe.Kind != KindCase || len(probe.Env) == 0 {
		return raw, nil
	}
	// "env": "<id>" — inline the canonical env verbatim.
	var id string
	if json.Unmarshal(probe.Env, &id) == nil && id != "" {
		envRaw, err := resolveEnv(id, lookup)
		if err != nil {
			return nil, err
		}
		return replaceEnv(raw, envRaw)
	}
	// "env": {"extends": "<id>", …} — inline the canonical env with overrides.
	var envObj map[string]json.RawMessage
	if json.Unmarshal(probe.Env, &envObj) == nil {
		if extRaw, ok := envObj["extends"]; ok {
			var baseID string
			if json.Unmarshal(extRaw, &baseID) != nil || baseID == "" {
				return nil, fmt.Errorf("dsl: env.extends must be an env id string")
			}
			baseRaw, err := resolveEnv(baseID, lookup)
			if err != nil {
				return nil, err
			}
			var base map[string]json.RawMessage
			if err := json.Unmarshal(baseRaw, &base); err != nil {
				return nil, fmt.Errorf("dsl: env %q is not an object: %w", baseID, err)
			}
			// Shallow override: each field the case names replaces the base's.
			delete(envObj, "extends")
			for k, v := range envObj {
				base[k] = v
			}
			merged, err := json.Marshal(base)
			if err != nil {
				return nil, fmt.Errorf("dsl: merge env %q: %w", baseID, err)
			}
			return replaceEnv(raw, merged)
		}
	}
	return raw, nil // inline env object (or malformed — ParseV2 reports it)
}

// resolveEnv looks up a canonical env by id, requiring a resolver.
func resolveEnv(id string, lookup func(id string) ([]byte, error)) ([]byte, error) {
	if lookup == nil {
		return nil, fmt.Errorf("dsl: case references env %q but no env resolver is available", id)
	}
	envRaw, err := lookup(id)
	if err != nil {
		return nil, fmt.Errorf("dsl: resolve env %q: %w", id, err)
	}
	return envRaw, nil
}

// replaceEnv rewrites the case document with env set to the resolved object.
func replaceEnv(raw, envRaw []byte) ([]byte, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("dsl: inline env: %w", err)
	}
	doc["env"] = envRaw
	return json.Marshal(doc)
}

// lowerCase lowers a v2 case (env inline) onto the executable Spec.
func lowerCase(c CaseV2) (Spec, error) {
	if c.ID == "" {
		return Spec{}, fmt.Errorf("dsl: v2 case needs \"id\"")
	}
	if len(c.Env) == 0 {
		return Spec{}, fmt.Errorf("dsl: v2 case %s needs \"env\" (an env id or an inline env object)", c.ID)
	}
	var envID string
	if json.Unmarshal(c.Env, &envID) == nil {
		return Spec{}, fmt.Errorf("dsl: case %s references env %q — resolve it with InlineEnv before parsing", c.ID, envID)
	}
	var env EnvV2
	if err := parseStrict(c.Env, &env); err != nil {
		return Spec{}, fmt.Errorf("dsl: case %s: env: %w", c.ID, err)
	}
	if env.Kind != "" && env.Kind != KindEnv {
		return Spec{}, fmt.Errorf("dsl: case %s: env kind is %q, want %q", c.ID, env.Kind, KindEnv)
	}
	if env.Chain == "" {
		return Spec{}, fmt.Errorf("dsl: case %s: env needs \"chain\"", c.ID)
	}
	// Timeout values are durations; reject an unparsable one here so a typo
	// fails at parse time rather than being silently ignored at run time.
	for name, v := range c.Timeouts {
		if _, err := time.ParseDuration(v); err != nil {
			return Spec{}, fmt.Errorf("dsl: case %s: timeouts.%s %q is not a duration: %w", c.ID, name, v, err)
		}
	}

	spec := Spec{
		SchemaVersion:    supportedSchemaVersion, // lowered form IS the executable v1 shape
		ID:               c.ID,
		ApplicableChains: c.ApplicableChains,
		Requires:         c.Requires,
		Chain:            ChainSpec{Name: env.Chain},
		Topology:         env.Topology,
		Hardforks:        env.Hardforks,
		Placement:        env.Target,
		DefaultOn:        c.On,
		Timeouts:         c.Timeouts,
	}
	// The env's capabilities and the case's requires are both gating inputs, so
	// they union — a case that lists its own requires must not lose the ones the
	// env declares. Duplicates are dropped, case order first.
	if len(env.Capabilities) > 0 {
		seen := make(map[string]bool, len(spec.Requires))
		for _, r := range spec.Requires {
			seen[r] = true
		}
		for _, c := range env.Capabilities {
			if !seen[c] {
				seen[c] = true
				spec.Requires = append(spec.Requires, c)
			}
		}
	}

	// An external manifest (and its genesis template) runs on the family named
	// by chain; both travel on the chain spec for the composer to thread.
	spec.Chain.ManifestPath = env.Manifest
	spec.Chain.TemplatePath = env.GenesisTemplate

	// Binaries: "default" is every node's binary; other keys are per-role.
	if b, ok := env.Binaries["default"]; ok && len(env.Binaries) == 1 {
		spec.Chain.Binary = b
	} else if len(env.Binaries) > 0 {
		spec.Chain.Binaries = env.Binaries
	}

	// An upgrade names its two binaries by role, and nothing else: a default
	// would mean every node runs one binary, which is not a handoff.
	if u := env.Upgrade; u != nil {
		if u.Profile == "" || u.Template == "" {
			return Spec{}, fmt.Errorf("dsl: case %s: upgrade needs \"profile\" and \"template\"", c.ID)
		}
		for _, role := range []string{BinaryBefore, BinaryAfter} {
			if env.Binaries[role] == "" {
				return Spec{}, fmt.Errorf("dsl: case %s: an upgrade env names binaries by role — binaries.%s is missing", c.ID, role)
			}
		}
		if len(env.Binaries) != 2 {
			return Spec{}, fmt.Errorf("dsl: case %s: an upgrade env names exactly the %s and %s binaries", c.ID, BinaryBefore, BinaryAfter)
		}
		spec.EnvUpgrade = u
	}

	// Genesis: template(+overlay/set) is the runtime's proven path; the other
	// declared modes have no support yet and are rejected by name (G2 partial).
	if g := env.Genesis; g != nil {
		if g.Mode != "" && g.Mode != "template" {
			return Spec{}, fmt.Errorf("dsl: case %s: genesis mode %q has no runtime boundary yet (supported: template)", c.ID, g.Mode)
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
	if len(env.Accounts) > 0 {
		spec.EnvAccounts = env.Accounts
	}
	if env.Keys != nil && env.Keys.NodeKeys != nil {
		spec.EnvKeys = env.Keys.NodeKeys
	}
	spec.EnvBlueprint = env.Blueprint
	if len(env.Launch) > 0 {
		spec.EnvLaunch = map[string][]string{}
		for scope, kvs := range env.Launch {
			if !launchScopeRE.MatchString(scope) {
				return Spec{}, fmt.Errorf("dsl: case %s: launch scope %q must be \"all\", a role (bp|validator, en|endpoint, boot), or \"node<N>\"", c.ID, scope)
			}
			for k, v := range kvs {
				spec.EnvLaunch[scope] = append(spec.EnvLaunch[scope], fmt.Sprintf("%s=%v", k, v))
			}
		}
	}
	if len(env.Config) > 0 {
		spec.EnvConfig = map[string][]string{}
		for scope, kvs := range env.Config {
			if scope != "all" && !nodeScopeRE.MatchString(scope) {
				return Spec{}, fmt.Errorf("dsl: case %s: config scope %q must be \"all\" or \"node<N>\"", c.ID, scope)
			}
			for k, v := range kvs {
				spec.EnvConfig[scope] = append(spec.EnvConfig[scope], fmt.Sprintf("%s=%v", k, v))
			}
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
		return Spec{}, fmt.Errorf("dsl: case %s has no steps", c.ID)
	}
	expects := 0
	for i, raw := range c.Steps {
		st, err := lowerStatement(raw)
		if err != nil {
			return Spec{}, fmt.Errorf("dsl: case %s: step %d: %w", c.ID, i+1, err)
		}
		if st.Expect != "" {
			expects++
			spec.Assertions = append(spec.Assertions, StatementAssertion(st))
		}
		spec.Sequence = append(spec.Sequence, st)
	}
	if expects == 0 {
		return Spec{}, fmt.Errorf("dsl: case %s verifies nothing — at least one expect statement is required", c.ID)
	}
	return spec, nil
}

// expectAliases maps proposal-vocabulary source names onto registered
// assertion names.
var expectAliases = map[string]string{"rpc": "rpcCall"}

// expectAdjuncts is the outcome vocabulary a do step's "expect" may name:
// receipt (the default — the tx must be mined), revert (mined with status 0x0),
// reject (the submit itself must fail), and fail (a launched node must not come
// up). A value outside this set is a typo that must be refused, not treated as
// the default success.
var expectAdjuncts = map[string]bool{"receipt": true, "revert": true, "reject": true, "fail": true}

// lowerStatement lowers one v2 statement map onto the runtime vocabulary.
func lowerStatement(m map[string]any) (Statement, error) {
	doName, hasDo := m["do"].(string)
	exName, hasEx := m["expect"].(string)
	// A do statement may carry expect as an ADJUNCT ("expect":"receipt"|"revert"
	// on sendTx). It is a statement head only when "do" is absent.
	if !hasDo && !hasEx {
		return Statement{}, fmt.Errorf("statement needs \"do\" or \"expect\"")
	}
	// A do step's expect names an outcome, not an assertion, so it is checked
	// here rather than by Unresolved (which resolves head expects as assertion
	// names). A value outside the outcome vocabulary would otherwise fall through
	// to the default "must succeed", silently turning a negative case positive.
	if hasDo && hasEx && !expectAdjuncts[strings.ToLower(exName)] {
		return Statement{}, fmt.Errorf("do step's expect adjunct %q is not a known outcome (want receipt, revert, reject, or fail)", exName)
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

// StatementAssertion renders an expect statement in the v1 assertion map shape.
func StatementAssertion(st Statement) map[string]any {
	out := make(map[string]any, len(st.Args)+1)
	maps.Copy(out, st.Args)
	out["assert"] = st.Expect
	return out
}

// StatementStep renders a do statement in the v1 step map shape.
func StatementStep(st Statement) map[string]any {
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
			return nil, fmt.Errorf("dsl: case %s: hooks.%s[%d]: %w", caseID, hook, i, err)
		}
		if st.Do == "" {
			return nil, fmt.Errorf("dsl: case %s: hooks.%s[%d]: hooks take do statements", caseID, hook, i)
		}
		out = append(out, StatementStep(st))
	}
	return out, nil
}

// nodeScopeRE matches a per-node config scope key ("node1", "node12").
var nodeScopeRE = regexp.MustCompile(`^node[1-9][0-9]*$`)

// launchScopeRE matches a launch scope key: "all", a role token, or "node<N>".
// Launch is scoped more widely than config because a launch flag often applies
// to a whole role (every producer mines), not just one node.
var launchScopeRE = regexp.MustCompile(`^(all|bp|validator|en|endpoint|boot|node[1-9][0-9]*)$`)
