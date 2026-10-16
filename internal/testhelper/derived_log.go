package testhelper

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// deriveRegexpCount counts matches in one captured stream. An optional exact
// prefix identifies the earlier snapshot; a lost prefix refuses the comparison
// instead of treating rotation or missing history as an error-free interval.
func deriveRegexpCount(spec map[string]any) (any, error) {
	streams, ok := spec["of"].([]any)
	if !ok || len(streams) != 1 {
		return nil, fmt.Errorf("dsl: derive regexpCount requires exactly one captured string in of")
	}
	text, ok := streams[0].(string)
	if !ok {
		return nil, fmt.Errorf("dsl: derive regexpCount requires a string capture")
	}
	pattern, ok := spec["pattern"].(string)
	if !ok || pattern == "" {
		return nil, fmt.Errorf("dsl: derive regexpCount requires a nonempty pattern")
	}
	expression, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("dsl: derive regexpCount pattern: %w", err)
	}
	if raw, present := spec["since"]; present {
		baseline, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("dsl: derive regexpCount since must be a string snapshot")
		}
		if !strings.HasPrefix(text, baseline) {
			return nil, fmt.Errorf("dsl: derive regexpCount capture lost the earlier prefix; log rotation or truncation cannot prove an interval")
		}
		if baseline != "" && len(text) > len(baseline) && !strings.HasSuffix(baseline, "\n") && text[len(baseline)] != '\n' {
			return nil, fmt.Errorf("dsl: derive regexpCount earlier snapshot ends inside a line; interval boundary is ambiguous")
		}
		text = text[len(baseline):]
	}
	return strconv.Itoa(len(expression.FindAllStringIndex(text, -1))), nil
}
