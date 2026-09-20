package dsl

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
)

// Parsing a declaration, and resolving the env it names.
//
// A case says `"env": "<id>"` or extends one inline, so reading a spec means
// finding that declaration, merging what the case overrode onto it, and doing
// it the same way on every surface — a case that parses differently under the
// CLI than under MCP is a case nobody can reason about.

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
// The override form is a deep merge: an object meets an object by key, a null
// removes the key, and anything else replaces — so a case declares what differs
// and keeps the rest of the shared env. See mergeEnv for why each rule is what
// it is. A case with an inline env (or a v1 spec) passes through untouched.
// lookup receives the env id and returns the env file's bytes; the caller owns
// where env files live.
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
			base, err := objectOf(baseRaw, baseID)
			if err != nil {
				return nil, err
			}
			if _, chained := base["extends"]; chained {
				return nil, fmt.Errorf("dsl: env %q extends another env; a shared env is the base, not a step in a chain", baseID)
			}
			over, err := objectOf(probe.Env, "the case's env")
			if err != nil {
				return nil, err
			}
			delete(over, "extends")
			merged, err := json.Marshal(mergeEnv(base, over))
			if err != nil {
				return nil, fmt.Errorf("dsl: merge env %q: %w", baseID, err)
			}
			return replaceEnv(raw, merged)
		}
	}
	return raw, nil // inline env object (or malformed — ParseV2 reports it)
}

// UseEnv rewrites a case so it runs on the env named envID, keeping whatever
// the case itself overrode.
//
// It is how one case runs on more than one chain without being edited. A case
// names its network, and that name is the last mainnet-specific thing left in
// the common cases; replacing it at read time is what lets the same steps meet
// a different chain.
//
// The two reference forms are rewritten; an inline env object is refused. An
// inline object IS the case's declaration, and swapping it would discard what
// the case asked for with no way to tell which parts mattered. Such a case is
// converted to the extends form first, which says out loud what it keeps.
//
// A non-case document, or a case with no env, passes through untouched.
func UseEnv(raw []byte, envID string) ([]byte, error) {
	if envID == "" || !IsV2(raw) {
		return raw, nil
	}
	var probe struct {
		Kind string          `json:"kind"`
		ID   string          `json:"id"`
		Env  json.RawMessage `json:"env"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("dsl: use env: %w", err)
	}
	if probe.Kind != KindCase || len(probe.Env) == 0 {
		return raw, nil
	}
	var id string
	if json.Unmarshal(probe.Env, &id) == nil && id != "" {
		return replaceEnv(raw, mustQuote(envID))
	}
	over, err := objectOf(probe.Env, "the case's env")
	if err != nil {
		return nil, err
	}
	if _, ok := over["extends"]; !ok {
		return nil, fmt.Errorf("dsl: case %s declares its env inline, so it cannot be moved onto env %q — give it \"extends\" and keep only what differs", probe.ID, envID)
	}
	over["extends"] = envID
	merged, err := json.Marshal(over)
	if err != nil {
		return nil, fmt.Errorf("dsl: use env %q: %w", envID, err)
	}
	return replaceEnv(raw, merged)
}

// mustQuote renders a string as a JSON scalar. The input is an env id that has
// already round-tripped through the resolver, so encoding cannot fail.
func mustQuote(s string) []byte {
	b, _ := json.Marshal(s)
	return b
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
