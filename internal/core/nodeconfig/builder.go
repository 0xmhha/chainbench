package nodeconfig

import (
	"errors"
	"fmt"
	"strings"
)

// Override is one high-precedence knob from the env.launch or case layer of a
// test declaration. Value is ignored for boolean knobs.
type Override struct {
	Key   OptionKey
	Value string
	Layer Layer
}

// Builder assembles modules into a command line in a fixed order and then runs
// the cross-module checks no single module can see. Module order is fixed —
// deterministic output is part of the contract. It followed §3.3 of a design
// since retired (formerly docs/dev/archive/chain-binary-flag-graph.md;
// `git show cde3a08f:docs/dev/archive/chain-binary-flag-graph.md`).
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
			layer = LayerCommand
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

// String renders an argv for logs and errors.
func String(argv []string) string { return strings.Join(argv, " ") }
