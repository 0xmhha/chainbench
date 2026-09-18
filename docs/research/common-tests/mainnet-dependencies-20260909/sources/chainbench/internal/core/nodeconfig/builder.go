package nodeconfig

import (
	"errors"
	"fmt"
	"strings"
)

// Override is one high-precedence knob from the env.launch or case layer of a
// test declaration. Value is ignored for boolean knobs.
type Override struct {
	Key   Key
	Value string
	Layer Layer
}

// Builder assembles modules into a command line in a fixed order and then runs
// the cross-module checks no single module can see. Module order is the
// declaration order in docs/chain-binary-flag-graph.md §3.3 — deterministic
// output is part of the contract.
type Builder struct {
	dialect   Dialect
	modules   []Module
	overrides []Override
}

// New creates a Builder for one dialect and module set. Modules emit in the
// given order; overrides apply after every module, last write wins.
func New(d Dialect, modules ...Module) *Builder {
	return &Builder{dialect: d, modules: modules}
}

// WithOverrides appends high-precedence knobs (env.launch / case layers).
func (b *Builder) WithOverrides(ov ...Override) *Builder {
	b.overrides = append(b.overrides, ov...)
	return b
}

// Build assembles the argv. Every classified problem — module invariant,
// unsupported knob, cross-module conflict — is collected and joined, so one
// failed assembly reports all its defects at once instead of one per run.
func (b *Builder) Build() ([]string, error) {
	a := NewArgs(b.dialect)
	var errs []error
	for _, m := range b.modules {
		if err := m.Apply(a); err != nil {
			errs = append(errs, fmt.Errorf("launchopt: %s: %w", m.Name(), err))
		}
	}
	for _, ov := range b.overrides {
		layer := ov.Layer
		if layer == "" {
			layer = LayerCase
		}
		if b.dialect.IsBool(ov.Key) {
			a.Enable(ov.Key, layer)
		} else {
			a.Set(ov.Key, ov.Value, layer)
		}
	}
	errs = append(errs, a.Problems()...)
	errs = append(errs, b.crossChecks(a)...)
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return a.Argv(), nil
}

// crossChecks are the combination rules that span modules.
func (b *Builder) crossChecks(a *Args) []error {
	var errs []error
	// Unlocking over HTTP without the insecure-unlock acknowledgment is
	// rejected by the binary at startup; catch it at assembly.
	if a.Has(KeyUnlock) && !a.Has(KeyAllowInsecureUnlock) {
		errs = append(errs, fmt.Errorf(
			"launchopt: --unlock with an HTTP endpoint needs --allow-insecure-unlock"))
	}
	// An API list on a disabled endpoint silently serves nothing.
	if a.Has(KeyHTTPAPI) && !a.Has(KeyHTTP) {
		errs = append(errs, fmt.Errorf("launchopt: http.api set but http endpoint not enabled"))
	}
	if a.Has(KeyWSAPI) && !a.Has(KeyWS) {
		errs = append(errs, fmt.Errorf("launchopt: ws.api set but ws endpoint not enabled"))
	}
	if a.Has(KeyMetricsPort) && !a.Has(KeyMetrics) {
		errs = append(errs, fmt.Errorf("launchopt: metrics.port set but metrics not enabled"))
	}
	return errs
}

// FamilyPolicy is the consensus family's contribution translated to typed
// knobs. It exists so families can keep exposing their launch posture as data
// while argv assembly stays single-sited here.
type FamilyPolicy struct {
	AllowInsecureUnlock bool
	DeprecatedPersonal  bool
	UnprotectedTxs      bool
	Mine                bool
}

// ParseFamilyFlags maps a family's legacy StartFlags vocabulary onto a
// FamilyPolicy. The vocabulary is closed (families emit only these four); an
// unknown flag is an error so a family extending its surface is forced to
// extend the typed model instead of leaking a raw string through.
//
// Transitional: this shim exists so registry.ChainPlugin keeps its StartFlags
// signature during the golden conversion. It goes away when families declare
// a FamilyPolicy directly.
func ParseFamilyFlags(flags []string) (FamilyPolicy, error) {
	var p FamilyPolicy
	for _, f := range flags {
		switch f {
		case "--allow-insecure-unlock":
			p.AllowInsecureUnlock = true
		case "--rpc.enabledeprecatedpersonal":
			p.DeprecatedPersonal = true
		case "--rpc.allow-unprotected-txs":
			p.UnprotectedTxs = true
		case "--mine":
			p.Mine = true
		default:
			return FamilyPolicy{}, fmt.Errorf(
				"launchopt: family flag %q is outside the typed vocabulary %s",
				f, "[--allow-insecure-unlock --rpc.enabledeprecatedpersonal --rpc.allow-unprotected-txs --mine]")
		}
	}
	return p, nil
}

// String renders an argv for logs and errors.
func String(argv []string) string { return strings.Join(argv, " ") }
