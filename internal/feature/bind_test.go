package feature_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/spf13/pflag"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/feature"
)

// probeIn is an input with one field of each shape a surface has to carry.
type probeIn struct {
	DataDir string   `cb:"data-dir,required" help:"workspace directory"`
	ChainID int64    `cb:"chain-id"          help:"override the manifest chain id"`
	Set     []string `cb:"set"               help:"genesis config override key=value"`
	Verbose bool     `cb:"verbose"           help:"say more"`
	Count   int      `cb:"count"             help:"how many"`
	hidden  string   //nolint:unused // untagged and unexported: must not reach a surface
	Skipped string   `cb:"-"`
}

// TestTags_MakeBothBindings is S0's point: one tag, and the flag a person types
// and the schema an agent reads cannot describe the field differently.
//
// They were written by hand beside each other before, which is how the same
// argument came to be spelled two ways on two surfaces.
func TestTags_MakeBothBindings(t *testing.T) {
	in := &probeIn{}
	fs := pflag.NewFlagSet("probe", pflag.ContinueOnError)
	if err := feature.Flags(in, fs); err != nil {
		t.Fatalf("flags: %v", err)
	}
	schema, err := feature.Schema(in, false)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	props, _ := schema["properties"].(map[string]any)

	// Every tagged field is on both surfaces, under the same name.
	for _, name := range []string{"data-dir", "chain-id", "set", "verbose", "count"} {
		if fs.Lookup(name) == nil {
			t.Errorf("%q has no flag", name)
		}
		if _, ok := props[name]; !ok {
			t.Errorf("%q is not in the schema", name)
		}
	}
	// And nothing else is. An untagged field is not a surface's business, and
	// deriving a flag from a Go field name would put spellings into the CLI
	// that nobody chose.
	if len(props) != 5 {
		t.Errorf("the schema holds %d properties, want the 5 tagged ones: %v", len(props), props)
	}
	if fs.Lookup("hidden") != nil || fs.Lookup("Skipped") != nil {
		t.Error("an untagged or skipped field reached the flag set")
	}

	// Required travels too.
	req, _ := schema["required"].([]string)
	if !reflect.DeepEqual(req, []string{"data-dir"}) {
		t.Errorf("required = %v, want [data-dir]", req)
	}
	if got := props["data-dir"].(map[string]any)["description"]; got != "workspace directory" {
		t.Errorf("the schema lost the help text: %v", got)
	}
	if got := fs.Lookup("data-dir").Usage; got != "workspace directory" {
		t.Errorf("the flag lost the help text: %q", got)
	}
}

// TestFlags_FillTheInput: the flags point at the input's own fields, so parsing
// a command line fills the struct the feature is invoked with. A binding that
// merely declared the flags would leave every surface to copy values across by
// hand, which is the step that drops one.
func TestFlags_FillTheInput(t *testing.T) {
	in := &probeIn{}
	fs := pflag.NewFlagSet("probe", pflag.ContinueOnError)
	if err := feature.Flags(in, fs); err != nil {
		t.Fatalf("flags: %v", err)
	}
	err := fs.Parse([]string{"--data-dir", "/tmp/ws", "--chain-id", "8285",
		"--set", "a=1", "--set", "b=2", "--verbose", "--count", "3"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := probeIn{DataDir: "/tmp/ws", ChainID: 8285, Set: []string{"a=1", "b=2"}, Verbose: true, Count: 3}
	if !reflect.DeepEqual(*in, want) {
		t.Errorf("parsed input = %+v, want %+v", *in, want)
	}
}

// TestSchema_CarriesReadOnly is the third of S0's gates: the declaration has to
// reach the schema, because that is where an agent reads it.
func TestSchema_CarriesReadOnly(t *testing.T) {
	plain, _ := feature.Schema(&probeIn{}, false)
	if _, has := plain["readOnlyHint"]; has {
		t.Error("a feature that writes advertised readOnlyHint")
	}
	query, _ := feature.Schema(&probeIn{}, true)
	if query["readOnlyHint"] != true {
		t.Errorf("a read-only feature's schema = %v, want readOnlyHint true", query)
	}
}

// TestFlags_RefusesAShapeNoSurfaceCanCarry: a field type that has no flag and
// no schema form is an error at registration rather than a field that silently
// never appears.
func TestFlags_RefusesAShapeNoSurfaceCanCarry(t *testing.T) {
	type bad struct {
		M map[string]string `cb:"m"`
	}
	if err := feature.Flags(&bad{}, pflag.NewFlagSet("x", pflag.ContinueOnError)); err == nil {
		t.Error("a map field was accepted as a flag")
	}
	type badSlice struct {
		N []int `cb:"n"`
	}
	if err := feature.Flags(&badSlice{}, pflag.NewFlagSet("x", pflag.ContinueOnError)); err == nil {
		t.Error("a slice of ints was accepted as a repeated flag")
	}
}

// TestRegister_KeepsTheTypes: authoring stays typed, and only the registry is
// erased. A caller hands back the input the descriptor gave it and gets the
// feature's own output.
func TestRegister_KeepsTheTypes(t *testing.T) {
	feature.Register(feature.Registration{
		Name: "probe.echo", Stage: feature.StageReport, Summary: "echo", ReadOnly: true,
	}, func(_ context.Context, _ app.Deps, in probeIn) (string, error) {
		return in.DataDir, nil
	})

	d, ok := feature.Lookup("probe.echo")
	if !ok {
		t.Fatal("the feature did not register")
	}
	in := d.Input().(*probeIn)
	in.DataDir = "/tmp/ws"
	out, err := d.Invoke(context.Background(), app.Deps{}, in)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if out != "/tmp/ws" {
		t.Errorf("out = %v, want the input's data dir", out)
	}

	// Queries holds every read-only feature, this probe among the real ones.
	var found bool
	for _, q := range feature.Queries() {
		if q == "probe.echo" {
			found = true
		}
	}
	if !found {
		t.Errorf("Queries() = %v, want it to hold the read-only probe", feature.Queries())
	}
}

// TestInvoke_RefusesTheWrongInput: an erased registry can be handed the wrong
// type, and it has to say so rather than panic inside the feature.
func TestInvoke_RefusesTheWrongInput(t *testing.T) {
	feature.Register(feature.Registration{Name: "probe.typed", Stage: feature.StageTest},
		func(_ context.Context, _ app.Deps, in probeIn) (int, error) { return 1, nil })
	d, _ := feature.Lookup("probe.typed")
	if _, err := d.Invoke(context.Background(), app.Deps{}, "not the input"); err == nil {
		t.Error("the wrong input type was accepted")
	}
}
