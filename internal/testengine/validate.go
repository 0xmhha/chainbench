package testengine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"github.com/0xmhha/chainbench/internal/testhelper"
)

// ValidateResult is one spec's offline validation outcome. OK is true when the
// spec parses and everything it names resolves; Result carries the human detail
// (and, with a chain, applicability/capability).
type ValidateResult struct {
	Spec   string `json:"spec"`
	ID     string `json:"id,omitempty"`
	OK     bool   `json:"ok"`
	Result string `json:"result"`
}

// ValidateSpecs parses each spec file and reports a per-file result: parse
// errors, unresolved action/assertion/reader/reference names, malformed node
// selectors, and — when chain is set — applicability and required capabilities.
// It writes nothing and composes nothing, so it is the pre-flight both the CLI
// `validate` and the MCP validate surface run, and it reaches the same verdict
// for both. It never errors on an unreadable/invalid file — that is reported in
// the result; the caller decides the exit status from OK.
func ValidateSpecs(paths []string, chain string) ([]ValidateResult, error) {
	var caps []string
	if chain != "" {
		plugin, err := registry.Get(chain)
		if err != nil {
			return nil, fmt.Errorf("validate: --chain: %w", err)
		}
		caps = plugin.Manifest().Capabilities
	}
	reg := testhelper.Registry()

	results := make([]ValidateResult, 0, len(paths))
	for _, p := range paths {
		raws, err := dsl.ReadFiles([]string{p})
		if err != nil {
			results = append(results, ValidateResult{Spec: p, Result: "ERROR: " + err.Error()})
			continue
		}
		r := validateRaw(raws[0], chain, caps, reg)
		r.Spec = p
		results = append(results, r)
	}
	return results, nil
}

// ValidateContent runs the same offline validation on spec bytes rather than
// file paths — the form the MCP surface passes (a spec is a JSON string, not a
// file on the host). label names each spec in the result (the MCP surface has no
// path). It reaches the same verdict as ValidateSpecs, so the two surfaces agree.
func ValidateContent(raws [][]byte, labels []string, chain string) ([]ValidateResult, error) {
	var caps []string
	if chain != "" {
		plugin, err := registry.Get(chain)
		if err != nil {
			return nil, fmt.Errorf("validate: --chain: %w", err)
		}
		caps = plugin.Manifest().Capabilities
	}
	reg := testhelper.Registry()
	results := make([]ValidateResult, 0, len(raws))
	for i, raw := range raws {
		r := validateRaw(raw, chain, caps, reg)
		if i < len(labels) {
			r.Spec = labels[i]
		}
		results = append(results, r)
	}
	return results, nil
}

// validateRaw validates one spec's bytes: parse, name resolution, selectors, and
// (with a chain) applicability/capability.
func validateRaw(raw []byte, chain string, caps []string, reg interp.Registry) ValidateResult {
	var r ValidateResult
	switch {
	case dsl.IsEnv(raw):
		// An env is a declaration, not a run: it validates on its own terms and
		// is exercised through the cases that name it.
		env, perr := dsl.ParseEnv(raw)
		if perr != nil {
			r.Result = "INVALID: " + perr.Error()
			return r
		}
		r.ID, r.OK, r.Result = env.ID, true, "env declaration for chain "+env.Chain
		return r
	}
	s, perr := dsl.Parse(raw)
	if perr != nil {
		r.Result = "INVALID: " + perr.Error()
		return r
	}
	r.ID = s.ID
	if unresolved := interp.Unresolved(s, reg); len(unresolved) > 0 {
		r.Result = "UNRESOLVED: " + strings.Join(unresolved, ", ")
		return r
	}
	if bad := malformedSelectors(s); len(bad) > 0 {
		r.Result = "INVALID SELECTOR: " + strings.Join(bad, ", ")
		return r
	}
	r.OK, r.Result = true, specResult(s, chain, caps)
	return r
}

// Precheck validates already-parsed specs before any network is composed, so an
// unsupported or mistyped spec fails with nothing allocated and nothing written.
// It runs the same name and selector checks as ValidateSpecs (capability
// applicability is a per-spec SKIP the engine applies, not a hard failure).
func Precheck(specs []dsl.Spec) error {
	reg := testhelper.Registry()
	for _, s := range specs {
		if unresolved := interp.Unresolved(s, reg); len(unresolved) > 0 {
			return fmt.Errorf("spec %s has unresolved references (nothing composed): %s", s.ID, strings.Join(unresolved, ", "))
		}
		if bad := malformedSelectors(s); len(bad) > 0 {
			return fmt.Errorf("spec %s has malformed node selectors: %s", s.ID, strings.Join(bad, ", "))
		}
	}
	return nil
}

// malformedSelectors returns the on/onEach/defaultOn selectors in a spec that
// are not well-formed. Range and existence (does node7 exist in this topology?)
// are resolved at run time against the real node set; this catches the offline
// mistakes: an empty selector, node0, or an unknown role.
func malformedSelectors(s dsl.Spec) []string {
	var bad []string
	seen := map[string]bool{}
	add := func(sel string) {
		if sel == "" || seen[sel] {
			return
		}
		seen[sel] = true
		if !selectorWellFormed(sel) {
			bad = append(bad, sel)
		}
	}
	add(s.DefaultOn)
	for _, st := range dsl.SequenceOf(s) {
		if on, ok := st.Args["on"].(string); ok {
			add(on)
		}
		if oe, ok := st.Args["onEach"].([]any); ok {
			for _, v := range oe {
				if sv, ok := v.(string); ok {
					add(sv)
				}
			}
		}
	}
	return bad
}

// selectorWellFormed reports whether sel is a syntactically valid node selector:
// "node<N>" (N>=1), a role ("bp"), a role ordinal ("bp2", N>=1), or a role with
// a resolver suffix ("bp:any", "en:0"). It mirrors the forms env.Resolve accepts
// without resolving against a concrete node set.
func selectorWellFormed(sel string) bool {
	if sel == "" {
		return false
	}
	if n, ok := strings.CutPrefix(sel, "node"); ok {
		v, err := strconv.Atoi(n)
		return err == nil && v >= 1
	}
	var base string
	suffix, hasSuffix := "", false
	if i := strings.IndexByte(sel, ':'); i >= 0 {
		base, suffix, hasSuffix = sel[:i], sel[i+1:], true
	} else {
		base = strings.TrimRightFunc(sel, func(r rune) bool { return r >= '0' && r <= '9' })
	}
	if _, err := node.NormalizeRole(base); err != nil {
		return false
	}
	if hasSuffix && suffix != "any" {
		if v, err := strconv.Atoi(suffix); err != nil || v < 0 {
			return false
		}
	}
	return true
}

// specResult describes a parsed spec's status against an optional target chain:
// plain OK without a chain, else OK / SKIP (not applicable) / SKIP (needs caps).
func specResult(s dsl.Spec, chain string, caps []string) string {
	if chain == "" {
		return "OK"
	}
	if !chainApplies(s.ApplicableChains, chain) {
		return "SKIP (chain not applicable)"
	}
	if missing := missingCaps(s.Requires, caps); len(missing) > 0 {
		return "SKIP (needs caps: " + strings.Join(missing, ",") + ")"
	}
	return "OK"
}

// chainApplies reports whether a spec's applicableChains (comma/space separated,
// empty = all) includes chain.
func chainApplies(applicableChains, chain string) bool {
	list := strings.FieldsFunc(applicableChains, func(r rune) bool { return r == ',' || r == ' ' })
	if len(list) == 0 {
		return true
	}
	for _, c := range list {
		if c == chain {
			return true
		}
	}
	return false
}

// missingCaps returns the required capabilities not present in have.
func missingCaps(required, have []string) []string {
	set := make(map[string]bool, len(have))
	for _, c := range have {
		set[c] = true
	}
	var missing []string
	for _, r := range required {
		if !set[r] {
			missing = append(missing, r)
		}
	}
	return missing
}
