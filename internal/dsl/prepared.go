package dsl

import (
	"fmt"
	"strings"
)

// PreparedV2 explicitly imports and exports public results between ordered cases.
// Values belong to one execution and one environment; declarations never carry
// the results themselves. Export references use the existing whole-binding form.
type PreparedV2 struct {
	Imports map[string]PreparedRef `json:"imports,omitempty"`
	Exports map[string]string      `json:"exports,omitempty"`
}

// PreparedRef names an exported result of an earlier successful case.
type PreparedRef struct {
	Case string `json:"case"`
	Name string `json:"name"`
}

// PreparedBinding returns the binding a whole-value export reference names.
func PreparedBinding(ref string) string {
	if !strings.HasPrefix(ref, "$") {
		return ""
	}
	name := strings.TrimPrefix(ref, "$")
	if strings.HasPrefix(name, "{") && strings.HasSuffix(name, "}") {
		name = strings.TrimSuffix(strings.TrimPrefix(name, "{"), "}")
	}
	if !presetValueName.MatchString(name) {
		return ""
	}
	return name
}

func validatePrepared(id string, declaration *PreparedV2, values map[string]any) error {
	if declaration == nil {
		return nil
	}
	for alias, ref := range declaration.Imports {
		if !presetValueName.MatchString(alias) || ref.Case == "" || !presetValueName.MatchString(ref.Name) {
			return fmt.Errorf("dsl: prepared import %q needs a binding alias, case and exported name", alias)
		}
		if ref.Case == id {
			return fmt.Errorf("dsl: prepared import %q references its own case", alias)
		}
		if _, exists := values[alias]; exists {
			return fmt.Errorf("dsl: prepared import %q conflicts with a preset value", alias)
		}
	}
	for name, ref := range declaration.Exports {
		if !presetValueName.MatchString(name) || PreparedBinding(ref) == "" {
			return fmt.Errorf("dsl: prepared export %q needs a name and whole binding reference", name)
		}
	}
	return nil
}

// CheckPreparedSequence validates dependencies in execution order before any
// network is changed. A skipped or failed producer is handled again at runtime.
func CheckPreparedSequence(specs []Spec) error {
	producers := map[string]map[string]string{}
	for _, spec := range specs {
		if spec.Prepared == nil {
			continue
		}
		for alias, ref := range spec.Prepared.Imports {
			exports, found := producers[ref.Case]
			if !found {
				return fmt.Errorf("dsl: prepared import %q in case %q needs earlier producer %q", alias, spec.ID, ref.Case)
			}
			if _, found := exports[ref.Name]; !found {
				return fmt.Errorf("dsl: prepared import %q in case %q names an undeclared export %q", alias, spec.ID, ref.Name)
			}
		}
		if len(spec.Prepared.Exports) > 0 {
			if _, found := producers[spec.ID]; found {
				return fmt.Errorf("dsl: prepared producer case %q is duplicated", spec.ID)
			}
			producers[spec.ID] = spec.Prepared.Exports
		}
	}
	return nil
}
