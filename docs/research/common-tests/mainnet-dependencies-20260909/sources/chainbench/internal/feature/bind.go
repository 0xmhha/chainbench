package feature

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// Tag is the struct tag that names an input field on every surface, and Help is
// the one sentence all three show for it.
//
//	DataDir string `cb:"data-dir,required" help:"workspace directory"`
//
// One tag, three bindings: a cobra flag, a JSON schema property, and the name a
// DSL step argument is validated against. They cannot describe the same field
// differently, which is what happened while the schemas were written by hand
// beside the flags.
const (
	Tag     = "cb"
	HelpTag = "help"
	// DefaultTag is the value a surface offers when the caller says nothing.
	//
	// It is separate from the zero value because they are different facts:
	// `--validators` defaults to 4 and zero validators is not a network, while
	// `--endpoints` defaults to 0 and means it. Deriving one from the other
	// would have silently changed what `chain place` does.
	DefaultTag = "default"
)

// field is one input field as the tags describe it.
type field struct {
	// Name is the surface-facing name ("data-dir"). A JSON schema uses the
	// same spelling, so a caller reading MCP's schema and one reading --help
	// are told the same word.
	Name     string
	Help     string
	Required bool
	Kind     reflect.Kind
	// Elem is the element kind for a slice, so a repeated flag and an array
	// schema agree about what they hold.
	Elem reflect.Kind
	// Path is the index chain to the field, so an embedded struct's field is
	// reachable the same way a top-level one is.
	Path []int
	// Default is the tag's literal, empty when the field simply starts at zero.
	Default string
}

// fieldsOf reads the tagged fields of an input struct.
//
// An untagged field is skipped rather than guessed at: a surface can only offer
// what the author named, and inventing a flag from a Go field name would put
// spellings into the CLI that nobody chose.
func fieldsOf(in any) ([]field, error) {
	t := reflect.TypeOf(in)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("feature: an input is a struct, got %T", in)
	}
	return fieldsOfType(t, nil)
}

// fieldsOfType reads one struct's tagged fields, path being how it was reached
// from the input's top level.
func fieldsOfType(t reflect.Type, path []int) ([]field, error) {
	var out []field
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		// An embedded struct contributes its own tagged fields, so a block of
		// options shared by several inputs is declared once and embedded
		// rather than copied into each — which is how two declarations of one
		// field start disagreeing.
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			inner, err := fieldsOfType(sf.Type, append(path, i))
			if err != nil {
				return nil, err
			}
			out = append(out, inner...)
			continue
		}
		tag, ok := sf.Tag.Lookup(Tag)
		if !ok || tag == "-" || !sf.IsExported() {
			continue
		}
		parts := strings.Split(tag, ",")
		f := field{Name: parts[0], Help: sf.Tag.Get(HelpTag), Kind: sf.Type.Kind(), Path: append(append([]int(nil), path...), i)}
		if f.Name == "" {
			return nil, fmt.Errorf("feature: field %s has an empty %s name", sf.Name, Tag)
		}
		for _, opt := range parts[1:] {
			if opt == "required" {
				f.Required = true
			}
		}
		f.Default = sf.Tag.Get(DefaultTag)
		if f.Kind == reflect.Slice {
			f.Elem = sf.Type.Elem().Kind()
		}
		out = append(out, f)
	}
	return out, nil
}

// Flags binds an input's tagged fields onto a flag set, pointing each flag at
// the field itself so a parsed command line fills the struct directly.
//
// in must be a pointer to the input, which is what Registration.Input returns.
func Flags(in any, fs *pflag.FlagSet) error {
	fields, err := fieldsOf(in)
	if err != nil {
		return err
	}
	v := reflect.ValueOf(in)
	if v.Kind() != reflect.Pointer {
		return fmt.Errorf("feature: Flags needs a pointer to the input, got %T", in)
	}
	v = v.Elem()
	for _, f := range fields {
		target := v.FieldByIndex(f.Path).Addr().Interface()
		switch f.Kind {
		case reflect.String:
			fs.StringVar(target.(*string), f.Name, f.Default, f.Help)
		case reflect.Bool:
			fs.BoolVar(target.(*bool), f.Name, f.Default == "true", f.Help)
		case reflect.Int:
			n, err := f.intDefault()
			if err != nil {
				return err
			}
			fs.IntVar(target.(*int), f.Name, int(n), f.Help)
		case reflect.Int64:
			n, err := f.intDefault()
			if err != nil {
				return err
			}
			fs.Int64Var(target.(*int64), f.Name, n, f.Help)
		case reflect.Slice:
			if f.Elem != reflect.String {
				return fmt.Errorf("feature: %s is a slice of %s; only strings repeat as flags", f.Name, f.Elem)
			}
			fs.StringArrayVar(target.(*[]string), f.Name, nil, f.Help)
		default:
			return fmt.Errorf("feature: %s is a %s, which no surface knows how to carry", f.Name, f.Kind)
		}
	}
	return nil
}

// intDefault reads a numeric default, refusing one that is not a number rather
// than starting the flag at zero and letting the difference surface as a
// network of the wrong size.
func (f field) intDefault() (int64, error) {
	if f.Default == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(f.Default, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("feature: %s has default %q, which is not a number", f.Name, f.Default)
	}
	return n, nil
}

// Schema renders an input's tagged fields as a JSON schema, the shape MCP
// publishes in tools/list.
//
// readOnly is carried through as MCP's own readOnlyHint spelling rather than a
// chainbench invention, so a client that knows the field gets the right answer
// and one that does not ignores it.
func Schema(in any, readOnly bool) (map[string]any, error) {
	fields, err := fieldsOf(in)
	if err != nil {
		return nil, err
	}
	props := map[string]any{}
	var required []string
	for _, f := range fields {
		p := map[string]any{"type": jsonType(f.Kind)}
		if f.Kind == reflect.Slice {
			p["items"] = map[string]any{"type": jsonType(f.Elem)}
		}
		if f.Help != "" {
			p["description"] = f.Help
		}
		if f.Default != "" {
			p["default"] = f.Default
		}
		props[f.Name] = p
		if f.Required {
			required = append(required, f.Name)
		}
	}
	sort.Strings(required)
	out := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		out["required"] = required
	}
	if readOnly {
		out["readOnlyHint"] = true
	}
	return out, nil
}

// jsonType maps a Go kind onto the JSON schema type a surface publishes.
func jsonType(k reflect.Kind) string {
	switch k {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int64:
		return "integer"
	case reflect.Slice:
		return "array"
	default:
		return "string"
	}
}
