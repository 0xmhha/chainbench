package testhelper

import (
	"fmt"
	"time"
)

// Explicit controls never fall back to defaults or reach a nonpositive ticker.
// An omitted control keeps the consumer's existing default.
func validateDurationControl(name, field string, args map[string]any) error {
	value, present := args[field]
	if !present {
		return nil
	}
	text, ok := value.(string)
	duration, err := time.ParseDuration(text)
	if !ok || err != nil || duration <= 0 {
		return fmt.Errorf("dsl: bad %s: %s expects a positive Go duration", field, name)
	}
	return nil
}
