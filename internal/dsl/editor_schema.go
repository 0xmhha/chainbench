package dsl

import (
	"reflect"
	"strings"
)

// LegacyEditorSchema describes the typed v1 declarations without redefining
// them. It lets an editor retain a valid legacy declaration when migration has
// no v2 representation (for example a chain.config file). Statement arguments
// are supplied separately by the registry that owns their implementations.
func LegacyEditorSchema() map[string]any {
	schema := legacyTypeSchema(reflect.TypeOf(Spec{}))
	properties := schema["properties"].(map[string]any)
	properties["schemaVersion"] = map[string]any{"const": supportedSchemaVersion}
	schema["required"] = []string{"schemaVersion", "id", "chain", "assertions"}
	return schema
}
func legacyTypeSchema(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.Pointer:
		return legacyTypeSchema(t.Elem())
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Slice:
		return map[string]any{"type": "array", "items": legacyTypeSchema(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": legacyTypeSchema(t.Elem())}
	case reflect.Struct:
		properties := map[string]any{}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if name == "" || name == "-" {
				continue
			}
			properties[name] = legacyTypeSchema(field.Type)
		}
		return map[string]any{"type": "object", "additionalProperties": false, "properties": properties}
	default:
		return map[string]any{}
	}
}
