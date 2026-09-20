package dsl

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Lowering v1 to v2.
//
// A v1 spec is read by translating it into the v2 shape and then running the v2
// path, so there is one grammar to reason about and one place a statement can
// be wrong. The vocabulary tables here — aliases, timeout keys, adjuncts — are
// what the older spelling maps onto.

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
	if err := checkAttach(c.ID, env); err != nil {
		return Spec{}, err
	}
	// Timeout values are durations; reject an unparsable one here so a typo
	// fails at parse time rather than being silently ignored at run time.
	//
	// The KEY is checked too, and for the same reason. Only case (alias test) is
	// consumed — the interpreter bounds the whole case with it — so any other key
	// was validated as a duration and then ignored, which is the worst of both:
	// the spec looks like it set a budget and no budget exists. A name outside the
	// vocabulary is a typo, not a feature request.
	for name, v := range c.Timeouts {
		if !timeoutKeys[name] {
			return Spec{}, fmt.Errorf("dsl: case %s: timeouts.%s is not a known timeout (want %s)", c.ID, name, timeoutKeyList())
		}
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
		EnvAttach:        env.Attach,
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

	// A definition names a binary; it does not place one. A path here is a
	// fact about one machine, and a case that carries it runs nowhere else.
	names := map[string]string{}
	for key, ref := range env.Binaries {
		if err := binaryRefIsAName(ref.Binary); err != nil {
			return Spec{}, fmt.Errorf("dsl: case %s: binaries.%s %q %w", c.ID, key, ref.Binary, err)
		}
		names[key] = ref.Binary
		if ref.Chain == "" {
			continue
		}
		if spec.Chain.BinaryChains == nil {
			spec.Chain.BinaryChains = map[string]string{}
		}
		spec.Chain.BinaryChains[key] = ref.Chain
	}

	// Binaries: "default" is every node's binary; other keys are per-role.
	if b, ok := names[BinaryDefault]; ok && len(names) == 1 {
		spec.Chain.Binary = b
	} else if len(names) > 0 {
		spec.Chain.Binaries = names
	}

	// An upgrade names its two binaries by which side of the fork each seals,
	// and nothing else: a default would mean every node runs one binary, which
	// is not a handoff.
	if u := env.Upgrade; u != nil {
		if err := checkUpgrade(c.ID, u, env); err != nil {
			return Spec{}, err
		}
		spec.EnvUpgrade = u
	}

	// Genesis: template(+overlay/set) is the runtime's proven path; the other
	// declared modes have no support yet and are rejected by name (G2 partial).
	if g := env.Genesis; g != nil {
		switch g.Mode {
		case "", "template":
			overlay := map[string]any{}
			maps.Copy(overlay, g.Overlay)
			for path, v := range g.Set {
				mergeDotPath(overlay, path, v)
			}
			if len(overlay) > 0 {
				spec.Chain.GenesisOverlay = overlay
			}
			spec.Chain.GenesisProvides = g.Provides
			spec.Chain.GenesisHaltsAt = g.HaltsAt
			for name, side := range g.PerBinary {
				if _, ok := env.Binaries[name]; !ok {
					return Spec{}, fmt.Errorf("dsl: case %s: genesis.perBinary names %q, which binaries does not declare", c.ID, name)
				}
				one := map[string]any{}
				maps.Copy(one, side.Overlay)
				for path, v := range side.Set {
					mergeDotPath(one, path, v)
				}
				if len(one) == 0 {
					return Spec{}, fmt.Errorf("dsl: case %s: genesis.perBinary.%s says nothing — give it a set or an overlay, or drop it", c.ID, name)
				}
				if spec.Chain.GenesisPerBinary == nil {
					spec.Chain.GenesisPerBinary = map[string]map[string]any{}
				}
				spec.Chain.GenesisPerBinary[name] = one
			}
		case "existing":
			// A finished genesis is used verbatim, so set/overlay — which edit a
			// built one — have nothing to act on and are refused rather than
			// silently ignored.
			if g.Ref == "" {
				return Spec{}, fmt.Errorf("dsl: case %s: genesis mode existing needs a ref", c.ID)
			}
			if len(g.Set) > 0 || len(g.Overlay) > 0 {
				return Spec{}, fmt.Errorf("dsl: case %s: genesis mode existing uses the file verbatim — set/overlay do not apply", c.ID)
			}
			spec.Chain.GenesisExisting = g.Ref
		default:
			return Spec{}, fmt.Errorf("dsl: case %s: genesis mode %q has no runtime boundary yet (supported: template, existing)", c.ID, g.Mode)
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
			if !node.ValidScope(scope) {
				return Spec{}, fmt.Errorf("dsl: case %s: launch scope %q must be %s", c.ID, scope, node.ScopeWords())
			}
			for k, v := range kvs {
				spec.EnvLaunch[scope] = append(spec.EnvLaunch[scope], fmt.Sprintf("%s=%v", k, v))
			}
		}
	}
	if len(env.Config) > 0 {
		spec.EnvConfig = map[string][]string{}
		for scope, kvs := range env.Config {
			if !node.ValidScope(scope) {
				return Spec{}, fmt.Errorf("dsl: case %s: config scope %q must be %s", c.ID, scope, node.ScopeWords())
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
	if err := checkCrossFork(c.ID, spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

// ActionCrossFork is the step that crosses the hardfork a network is composed
// to cross.
//
// The name lives in the grammar because the grammar checks it and because the
// composer reads a case's steps for it: a case that names the step crosses the
// fork itself, and one that does not gets a network already past it. Three
// places have to mean the same word, so there is one.
const ActionCrossFork = "crossFork"

// checkCrossFork holds a case to what it said about crossing the fork.
//
// A case that names the step on a network with no hardfork is refused. The step
// would fail at run time with the same reason, but by then the network is up
// and the case has usually done something first — and the likeliest cause is an
// env that was meant to declare an upgrade and does not.
//
// Naming it twice is refused for the same kind of reason. The step is
// idempotent, so the second one reports "already crossed" and the case passes
// while reading as though it crossed twice.
func checkCrossFork(caseID string, spec Spec) error {
	crossings := 0
	for _, st := range spec.Sequence {
		if st.Do == ActionCrossFork {
			crossings++
		}
	}
	if crossings == 0 {
		return nil
	}
	if spec.EnvUpgrade == nil {
		return fmt.Errorf("dsl: case %s names the %s step, but its env declares no hardfork to cross — declare one under env.upgrade", caseID, ActionCrossFork)
	}
	if crossings > 1 {
		return fmt.Errorf("dsl: case %s names the %s step %d times, and a network crosses its fork once", caseID, ActionCrossFork, crossings)
	}
	return nil
}

// expectAliases maps proposal-vocabulary source names onto registered
// assertion names.
var expectAliases = map[string]string{"rpc": "rpcCall"}

// timeoutKeys is the timeout vocabulary a case's "timeouts" may name. Only these
// are read — interp.caseTimeout bounds the whole case with case (or its alias
// test) — so the set is the parser's promise that a declared budget takes effect.
// Widening it means giving the new key a consumer in the same change.
var timeoutKeys = map[string]bool{"case": true, "test": true}

// timeoutKeyList renders the vocabulary for an error message, sorted so the text
// does not depend on map order.
func timeoutKeyList() string {
	keys := make([]string, 0, len(timeoutKeys))
	for k := range timeoutKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

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

// envDefaultRE matches the ${VAR:-default} form and captures the default, which
// is the only part of an expansion this file can judge.
var envDefaultRE = regexp.MustCompile(`^\$\{[A-Za-z_][A-Za-z0-9_]*:-(.*)\}$`)

// binaryRefIsAName reports why a declared binary reference is not one.
//
// A definition says WHICH binary; a workspace-config says WHERE binaries live
// on the target (dataRoot plus paths.binaries) and binaryAliases says which
// file this environment calls that name. Writing the path in the definition
// says both at once, in the document that is supposed to travel: five specs
// carried /data/chainbench/bin/... and ran on one docker environment and
// nowhere else, while the environment file beside them already produced the
// same path from the name.
//
// An expansion is judged by its default, because that is the part the
// definition wrote. ${GWBFT_BIN:-gwbft} is a name with a machine-local escape
// hatch; ${GWBFT_BIN:-/opt/gwbft} is the path problem wearing a variable. A
// bare $VAR names nothing this file can see, so it passes and placeBinary
// judges what it expands to.
func binaryRefIsAName(ref string) error {
	if strings.TrimSpace(ref) == "" {
		return errors.New("is empty — name the binary, or leave it out and let the chain name it")
	}
	lit := ref
	if m := envDefaultRE.FindStringSubmatch(ref); m != nil {
		lit = m[1]
	} else if strings.Contains(ref, "$") {
		return nil // an expansion with no default: only the machine knows
	}
	switch {
	case strings.HasPrefix(lit, "~"):
		return errors.New("starts at a home directory, which is a fact about one machine")
	case strings.ContainsRune(lit, '/'):
		return errors.New("is a path — name the binary, and let a workspace-config say where binaries live on the target")
	}
	return nil
}

// objectOf decodes a JSON object, naming what failed. A null decodes into a nil
// map without error, and a caller that then assigns into it panics — so it is
// refused here with the same words a non-object gets.
func objectOf(raw []byte, what string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("dsl: %s is not an object: %w", what, err)
	}
	if m == nil {
		return nil, fmt.Errorf("dsl: %s is not an object: it is null", what)
	}
	return m, nil
}

// mergeEnv lays a case's overrides over a shared env and returns the result.
//
// Three rules, and each exists because the alternative loses something the
// author wrote:
//
//   - An object meets an object by KEY, recursively. Replacing the whole object
//     was the old rule, and under it a case that wanted node2 to differ wrote
//     config.node2 and silently dropped the shared env's "all" scope and every
//     other node's — the case ran, and only the result was different.
//   - null REMOVES the key. Deep merge alone can only add and overwrite, so
//     there would be no way to switch off something the shared env turned on.
//     That is not hypothetical: the commonest difference between the 205 inline
//     envs and the shape they share is a capability list that is absent.
//   - An array REPLACES. Unioning reads as generous and takes away the only way
//     to drop an entry, which is the same hole as having no delete.
//
// The merged object is parsed strictly afterwards, so a key neither side should
// have is refused there rather than checked twice here.
func mergeEnv(base, over map[string]any) map[string]any {
	for k, v := range over {
		if v == nil {
			delete(base, k)
			continue
		}
		bo, baseIsObject := base[k].(map[string]any)
		oo, overIsObject := v.(map[string]any)
		if baseIsObject && overIsObject {
			base[k] = mergeEnv(bo, oo)
			continue
		}
		base[k] = v
	}
	return base
}

// checkAttach holds an attaching env to saying only what attaching needs.
//
// The two forms are exclusive on purpose. Every composition field describes a
// network to build, and a declaration that both builds one and attaches to
// another has not said which network its assertions are about — it would build
// a network, run nothing against it, and report on a different one. That is the
// kind of wrong answer that reads as a pass.
//
// "chain" stays required. The direction recorded for this work was to read the
// chain's identity over RPC after attaching, and that is still right, but it
// cannot be done first: a spec that names a contract ("govMinter") needs the
// chain's table before the first call, and the table comes from the manifest.
func checkAttach(caseID string, env EnvV2) error {
	if env.Attach == nil {
		return nil
	}
	if len(env.Attach.RPC) == 0 {
		return fmt.Errorf("dsl: case %s: env.attach needs \"rpc\" (the endpoints of the network to run against)", caseID)
	}
	for i, u := range env.Attach.RPC {
		if strings.TrimSpace(u) == "" {
			return fmt.Errorf("dsl: case %s: env.attach.rpc[%d] is empty", caseID, i)
		}
	}
	var composing []string
	for name, set := range map[string]bool{
		"binaries":        len(env.Binaries) > 0,
		"keys":            env.Keys != nil,
		"blueprint":       env.Blueprint != "",
		"genesis":         env.Genesis != nil,
		"topology":        len(env.Topology) > 0,
		"hardforks":       len(env.Hardforks) > 0,
		"launch":          len(env.Launch) > 0,
		"config":          len(env.Config) > 0,
		"accounts":        len(env.Accounts) > 0,
		"upgrade":         env.Upgrade != nil,
		"target":          env.Target != "",
		"manifest":        env.Manifest != "",
		"genesisTemplate": env.GenesisTemplate != "",
	} {
		if set {
			composing = append(composing, name)
		}
	}
	if len(composing) > 0 {
		sort.Strings(composing)
		return fmt.Errorf("dsl: case %s: env.attach runs against a network that is already up, so it cannot also compose one — drop %s",
			caseID, strings.Join(composing, ", "))
	}
	return nil
}
