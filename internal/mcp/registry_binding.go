package mcp

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/feature"
)

// invokeFeature keeps the legacy decoder and renderer, but selects execution
// from the shared registry. The erased boundary checks both input and output.
func invokeFeature[In, Out any](ctx context.Context, deps app.Deps, name string, in In) (Out, error) {
	var zero Out
	entry, ok := feature.Lookup(name)
	if !ok {
		return zero, fmt.Errorf("mcp: missing registered operation %s", name)
	}
	var input any = &in
	prototype := entry.Input()
	if reflect.TypeOf(prototype) != reflect.TypeOf(input) {
		// Older app helpers take a scalar RPC URL or a Node directly. Their
		// registered inputs wrap that one value; copy it without JSON so a
		// Node's authenticated transport and exact integer types survive.
		target := reflect.ValueOf(prototype).Elem()
		source := reflect.ValueOf(&in).Elem()
		if target.Kind() != reflect.Struct || target.NumField() != 1 || target.Field(0).Type() != source.Type() || !target.Field(0).CanSet() {
			return zero, fmt.Errorf("mcp: operation %s input is %T, want %T", name, input, prototype)
		}
		target.Field(0).Set(source)
		input = prototype
	}
	out, err := entry.Invoke(ctx, deps, input)
	if err != nil {
		return zero, err
	}
	result, ok := out.(Out)
	if !ok {
		return zero, fmt.Errorf("mcp: operation %s output is %T, want %T", name, out, zero)
	}
	return result, nil
}

func invokeFeatureError[In any](ctx context.Context, deps app.Deps, name string, in In) error {
	_, err := invokeFeature[In, struct{}](ctx, deps, name, in)
	return err
}

// invokeFeatureFields adapts trusted legacy decoder fields to a registered
// struct without serializing authentication descriptors or exact numbers.
func invokeFeatureFields[Out any](ctx context.Context, deps app.Deps, name string, values map[string]any) (Out, error) {
	var zero Out
	entry, ok := feature.Lookup(name)
	if !ok {
		return zero, fmt.Errorf("mcp: missing registered operation %s", name)
	}
	input := entry.Input()
	target := reflect.ValueOf(input).Elem()
	for fieldName, value := range values {
		field := target.FieldByName(fieldName)
		if !field.IsValid() || !field.CanSet() {
			return zero, fmt.Errorf("mcp: operation %s has no input field %s", name, fieldName)
		}
		source := reflect.ValueOf(value)
		if !source.IsValid() {
			switch field.Kind() {
			case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
				field.SetZero()
				continue
			}
		}
		if !source.IsValid() || !source.Type().AssignableTo(field.Type()) {
			return zero, fmt.Errorf("mcp: operation %s field %s has incompatible type", name, fieldName)
		}
		field.Set(source)
	}
	out, err := entry.Invoke(ctx, deps, input)
	if err != nil {
		return zero, err
	}
	result, ok := out.(Out)
	if !ok {
		return zero, fmt.Errorf("mcp: operation %s output is %T, want %T", name, out, zero)
	}
	return result, nil
}

func invokeFeatureFieldsError(ctx context.Context, deps app.Deps, name string, values map[string]any) error {
	_, err := invokeFeatureFields[struct{}](ctx, deps, name, values)
	return err
}

var commonSchemaSources = map[string][2]string{
	"rpc":          {"node.status", "rpc"},
	"workspaceDir": {"chain.stop", "workspace-dir"},
	"chain":        {"chain.describe", "chain"},
}

func commonProperty(name string, extras map[string]any) map[string]any {
	source, ok := commonSchemaSources[name]
	if !ok {
		panic("unknown shared schema property " + name)
	}
	entry, ok := feature.Lookup(source[0])
	if !ok {
		panic("missing shared schema source " + source[0])
	}
	schema, err := feature.Schema(entry.Input(), entry.ReadOnly)
	if err != nil {
		panic(err)
	}
	shared := schema["properties"].(map[string]any)[source[1]].(map[string]any)
	out := map[string]any{}
	for key, value := range extras {
		out[key] = value
	}
	out["type"] = shared["type"]
	description, _ := shared["description"].(string)
	if details, _ := extras["description"].(string); details != "" && details != description && !strings.HasPrefix(details, description) {
		description += ". " + details
	} else if details != "" {
		description = details
	}
	out["description"] = description
	return out
}

// registrySchemas replaces only the common scalar definition. Tool-specific
// required alternatives, enum/range constraints and compound inputs stay intact.
func registrySchemas(tool Tool) Tool {
	properties, _ := tool.InputSchema["properties"].(map[string]any)
	for name := range commonSchemaSources {
		property, ok := properties[name].(map[string]any)
		if !ok || property["type"] != "string" {
			continue
		}
		properties[name] = commonProperty(name, property)
	}
	return tool
}
