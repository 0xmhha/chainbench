package dsl

import (
	"fmt"
	"regexp"
)

var presetValueName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validatePresetValues(values map[string]any) error {
	for name := range values {
		if !presetValueName.MatchString(name) {
			return fmt.Errorf("dsl: values: %q is not a binding name", name)
		}
	}
	return nil
}
