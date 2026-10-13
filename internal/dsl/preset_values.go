package dsl

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var presetValueName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validatePresetValues(values map[string]any) error {
	for name := range values {
		if !presetValueName.MatchString(name) {
			return fmt.Errorf("dsl: values: %q is not a binding name", name)
		}
	}
	_, _, err := ExpectedChainID(values)
	return err
}

// ExpectedChainID reads an explicitly declared identity from preset values.
// Presence is separate from the number so a declared zero is still checked.
func ExpectedChainID(values map[string]any) (uint64, bool, error) {
	value, present := values["chainId"]
	if !present {
		return 0, false, nil
	}
	bad := fmt.Errorf("dsl: values.chainId must be an exact nonnegative 64-bit integer; use decimal/hex strings for large values")
	switch v := value.(type) {
	case float64:
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v || v > 9007199254740991 {
			return 0, true, bad
		}
		return uint64(v), true, nil
	case int:
		if v < 0 {
			return 0, true, bad
		}
		return uint64(v), true, nil
	case uint64:
		return v, true, nil
	case string:
		text := strings.TrimSpace(v)
		base := 10
		if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
			base = 16
			text = text[2:]
		}
		if text == "" {
			return 0, true, bad
		}
		digits := "0123456789"
		if base == 16 {
			digits += "abcdefABCDEF"
		}
		for _, c := range text {
			if !strings.ContainsRune(digits, c) {
				return 0, true, bad
			}
		}
		n, err := strconv.ParseUint(text, base, 64)
		if err != nil {
			return 0, true, bad
		}
		return n, true, nil
	default:
		return 0, true, bad
	}
}
