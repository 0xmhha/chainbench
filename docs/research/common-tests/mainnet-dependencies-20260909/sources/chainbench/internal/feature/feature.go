// Package feature is the one place a capability is registered, so the three
// surfaces render it instead of each describing it again.
//
// The project's rule is not to abstract until there is a second use. Here there
// are three — CLI, MCP and the DSL — and they have already diverged: a feature
// exists on one surface and not another, and the MCP schemas are written by
// hand beside the cobra flags that say the same thing. This is de-duplication
// rather than anticipation (surface-unification-design §3.1).
//
// What it does NOT do is generate commands. Cobra commands stay hand-written:
// the name, the place in the tree and the wording of the help are better
// decided by a person. Only the flag binding is derived, from the same tags the
// JSON schema comes from, so those two cannot disagree (§3.4).
package feature

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/0xmhha/chainbench/internal/app"
)

// Stage is which of the three phases a feature belongs to. It is what a surface
// groups by: CLI command groups, DSL sections, MCP namespaces.
type Stage string

const (
	// StageCompose builds and runs a network.
	StageCompose Stage = "compose"
	// StageTest drives a network with specs.
	StageTest Stage = "test"
	// StageReport reads what happened.
	StageReport Stage = "report"
)

// Registration is one feature, described so a surface can render it without
// knowing what it does. Authors do not build one by hand; Register does.
//
// The design called this Descriptor. That name already means two other things —
// a chain plugin descriptor in core/registry and its re-export in app — and the
// name-collision ratchet said so before this landed, which is what it is for.
type Registration struct {
	// Name is the feature's registry spelling ("chain.genesis", "tx.send").
	Name  string
	Stage Stage
	// Summary is one line, shared by every surface's help.
	Summary string
	// ReadOnly declares that invoking this changes nothing — no file, no
	// process, no chain state — and that its output carries no secret.
	//
	// A declaration, not an inference. `keyring export` changes nothing and
	// still must not carry it, because it prints a private key; and nothing
	// about `node rpc` says whether it writes, since the method is an argument.
	// This is the attribute the query projection and the MCP read-only tool
	// list are both derived from (§4.4).
	ReadOnly bool

	// Input returns a fresh zero input. A surface fills it — cobra from flags,
	// MCP from JSON, the DSL from step arguments — and hands it back.
	Input func() any
	// Invoke runs the feature with a filled input.
	Invoke func(ctx context.Context, d app.Deps, in any) (any, error)
}

// registry holds every registered feature. Registration happens in init, so it
// is guarded rather than assumed single-threaded.
var registry = struct {
	sync.RWMutex
	byName map[string]Registration
}{byName: map[string]Registration{}}

// Register wraps a typed use case into the registry.
//
// Authoring stays typed — the function keeps its real input and output — and
// only the registry is erased. That is the same trade the DSL's action registry
// already makes, and it is what lets a surface hold a feature it has never
// heard of.
func Register[In, Out any](d Registration, fn func(context.Context, app.Deps, In) (Out, error)) {
	if d.Name == "" {
		panic("feature: a descriptor needs a name")
	}
	d.Input = func() any { return new(In) }
	d.Invoke = func(ctx context.Context, deps app.Deps, in any) (any, error) {
		typed, ok := in.(*In)
		if !ok {
			return nil, fmt.Errorf("feature %s: input is %T, want %T", d.Name, in, new(In))
		}
		return fn(ctx, deps, *typed)
	}
	registry.Lock()
	defer registry.Unlock()
	if _, dup := registry.byName[d.Name]; dup {
		panic("feature: " + d.Name + " is registered twice")
	}
	registry.byName[d.Name] = d
}

// Lookup returns the feature registered under name.
func Lookup(name string) (Registration, bool) {
	registry.RLock()
	defer registry.RUnlock()
	d, ok := registry.byName[name]
	return d, ok
}

// All returns every registered feature, by name.
func Registered() []Registration {
	registry.RLock()
	defer registry.RUnlock()
	out := make([]Registration, 0, len(registry.byName))
	for _, d := range registry.byName {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Queries returns the features that declared themselves read-only, by name. It
// is the one answer both the CLI's query projection and MCP's read-only tool
// list are meant to read, so the two cannot disagree.
func Queries() []string {
	var out []string
	for _, d := range Registered() {
		if d.ReadOnly {
			out = append(out, d.Name)
		}
	}
	return out
}
