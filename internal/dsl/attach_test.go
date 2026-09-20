package dsl_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// attachCase builds a v2 case whose env is the given object.
func attachCase(t *testing.T, env map[string]any) []byte {
	t.Helper()
	doc := map[string]any{
		"schemaVersion": "2",
		"kind":          "case",
		"id":            "attach-case",
		"env":           env,
		"steps": []map[string]any{
			{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "0"},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestAttachEnvIsExclusiveWithComposing is the rule that makes the two forms
// tell each other apart. A declaration that both builds a network and attaches
// to another has not said which network its assertions are about: it would
// build one, run nothing against it, and report on the other. That reads as a
// pass, which is the worst way to be wrong.
func TestAttachEnvIsExclusiveWithComposing(t *testing.T) {
	for name, extra := range map[string]map[string]any{
		"topology":  {"topology": map[string]any{"bp": 4}},
		"genesis":   {"genesis": map[string]any{"overlay": map[string]any{}}},
		"binaries":  {"binaries": map[string]any{"default": "gstable"}},
		"upgrade":   {"upgrade": map[string]any{"preset": "p", "fork": "boho", "at": 10}},
		"hardforks": {"hardforks": map[string]any{"boho": 0}},
		"accounts":  {"accounts": map[string]any{"dev1": map[string]any{}}},
	} {
		t.Run(name, func(t *testing.T) {
			env := map[string]any{
				"schemaVersion": "2", "kind": "env", "id": "e", "chain": "stablenet",
				"attach": map[string]any{"rpc": []string{"http://127.0.0.1:8600"}},
			}
			for k, v := range extra {
				env[k] = v
			}
			_, err := dsl.Parse(attachCase(t, env))
			if err == nil {
				t.Fatalf("an env that attaches AND declares %s was accepted", name)
			}
			if !strings.Contains(err.Error(), name) {
				t.Errorf("the refusal does not name %s, so it does not say what to drop: %v", name, err)
			}
		})
	}
}

// TestAttachEnvNeedsAnEndpoint keeps an attach declaration that names no
// network from parsing. Without one there is nothing to attach to, and the
// failure would otherwise surface as a dial error with no idea why the list was
// empty.
func TestAttachEnvNeedsAnEndpoint(t *testing.T) {
	for name, attach := range map[string]map[string]any{
		"no rpc key":  {"provides": []string{"rpc"}},
		"empty list":  {"rpc": []string{}},
		"empty entry": {"rpc": []string{"  "}},
	} {
		t.Run(name, func(t *testing.T) {
			env := map[string]any{
				"schemaVersion": "2", "kind": "env", "id": "e", "chain": "stablenet",
				"attach": attach,
			}
			if _, err := dsl.Parse(attachCase(t, env)); err == nil {
				t.Fatal("an attach declaration with no endpoint was accepted")
			}
		})
	}
}

// TestAttachEnvReachesTheSpec checks the declaration survives lowering. A rule
// that is validated and then dropped is the shape that let a resolved value be
// thrown away elsewhere in this package.
func TestAttachEnvReachesTheSpec(t *testing.T) {
	env := map[string]any{
		"schemaVersion": "2", "kind": "env", "id": "e", "chain": "stablenet",
		"attach": map[string]any{
			"rpc":      []string{"http://127.0.0.1:8600", "http://127.0.0.1:8610"},
			"keysDir":  "keys/preset",
			"provides": []string{"rpc", "consensus"},
		},
	}
	spec, err := dsl.Parse(attachCase(t, env))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if spec.EnvAttach == nil {
		t.Fatal("the attach declaration did not reach the spec")
	}
	if len(spec.EnvAttach.RPC) != 2 {
		t.Errorf("RPC = %v, want both endpoints", spec.EnvAttach.RPC)
	}
	if spec.EnvAttach.KeysDir != "keys/preset" {
		t.Errorf("KeysDir = %q", spec.EnvAttach.KeysDir)
	}
	if len(spec.EnvAttach.Provides) != 2 {
		t.Errorf("Provides = %v", spec.EnvAttach.Provides)
	}
	// An attaching env composes nothing, so it must not leave a topology behind
	// for the composer to act on.
	if len(spec.Topology) != 0 {
		t.Errorf("Topology = %v, want none", spec.Topology)
	}
}

// TestComposingEnvHasNoAttach is the other direction: the ordinary form must
// not acquire one, or every composed run would look like a candidate to attach.
func TestComposingEnvHasNoAttach(t *testing.T) {
	env := map[string]any{
		"schemaVersion": "2", "kind": "env", "id": "e", "chain": "stablenet",
		"topology": map[string]any{"bp": 4},
	}
	spec, err := dsl.Parse(attachCase(t, env))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if spec.EnvAttach != nil {
		t.Errorf("EnvAttach = %+v, want nil for an env that composes", spec.EnvAttach)
	}
}
