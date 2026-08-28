package testspec

import (
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Bindings are the values earlier steps saved, keyed by the name given in a
// step's "save" field. A binding scope is one test run (one Spec), so bindings
// never leak between tests and the interpreter stays free of shared state.
//
// The DSL grammar (design §3.2b):
//
//		{"read": {"source":"call", "to":"0x..", "data":"0x..", "save":"supply"}}
//		{"assert":"call", ..., "compare":"GreaterOrEqual", "expected":"$supply"}
//
//	  - "$name" as a WHOLE string is replaced by the bound value with its type
//	    intact, so a number stays a number and a comparator still sees a number.
//	  - "${name}" inside a longer string interpolates the value's textual form,
//	    which is how calldata is assembled from a saved address.
//	  - "$$" is a literal "$".
//
// An unbound reference is an error, never a silently empty string: a typo has to
// fail the step rather than compare against "".
type Bindings map[string]any

// refPattern matches the two reference forms and the "$$" escape. Order matters:
// the escape is matched first so "$$name" never reads as a reference.
var refPattern = regexp.MustCompile(`\$\$|\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// wholeRefPattern matches a string that is exactly one bare reference, which is
// the case where the bound value's type is preserved rather than stringified.
var wholeRefPattern = regexp.MustCompile(`^\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?$`)

// resolveRefs returns a copy of v with every binding reference substituted. Maps
// and slices are rebuilt rather than edited, so the parsed spec stays pristine
// and a spec can be run more than once.
func resolveRefs(v any, b Bindings) (any, error) {
	switch x := v.(type) {
	case string:
		return resolveString(x, b)
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			rv, err := resolveRefs(val, b)
			if err != nil {
				return nil, err
			}
			out[k] = rv
		}
		return out, nil
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			rv, err := resolveRefs(val, b)
			if err != nil {
				return nil, err
			}
			out[i] = rv
		}
		return out, nil
	default:
		return v, nil
	}
}

// resolveString substitutes references in one string. A string that is exactly
// one reference yields the bound value itself; anything else yields a string.
func resolveString(s string, b Bindings) (any, error) {
	if s == "$$" {
		return "$", nil
	}
	if m := wholeRefPattern.FindStringSubmatch(s); m != nil {
		val, ok := b[m[1]]
		if !ok {
			return nil, fmt.Errorf("testspec: unbound reference %q (no earlier step saved it)", s)
		}
		return val, nil
	}

	var firstErr error
	out := refPattern.ReplaceAllStringFunc(s, func(match string) string {
		if match == "$$" {
			return "$"
		}
		name := strings.Trim(match, "${}")
		val, ok := b[name]
		if !ok {
			if firstErr == nil {
				firstErr = fmt.Errorf("testspec: unbound reference %q in %q (no earlier step saved it)", name, s)
			}
			return match
		}
		return refText(val)
	})
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

// refText renders a bound value for interpolation into a larger string. Numbers
// use their decimal form (JSON decodes every number as float64, and "12" is
// wanted where Go's default would print "12" or "1.2e+01" by width).
func refText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case uint64:
		return strconv.FormatUint(x, 10)
	case bool:
		return strconv.FormatBool(x)
	case *big.Int:
		return BigString(x)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

// saveName returns the binding name an entry's args declare via "save", or "".
func saveName(args map[string]any) string {
	s, _ := args["save"].(string)
	return s
}

// refNames returns every binding name v references, sorted and de-duplicated.
// It powers the offline check that each reference has an earlier "save".
func refNames(v any) []string {
	seen := map[string]bool{}
	collectRefs(v, seen)
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// collectRefs walks v and records every referenced binding name in seen.
func collectRefs(v any, seen map[string]bool) {
	switch x := v.(type) {
	case string:
		for _, m := range refPattern.FindAllStringSubmatch(x, -1) {
			switch {
			case m[0] == "$$":
				// escape, not a reference
			case m[1] != "":
				seen[m[1]] = true
			case m[2] != "":
				seen[m[2]] = true
			}
		}
	case map[string]any:
		for _, val := range x {
			collectRefs(val, seen)
		}
	case []any:
		for _, val := range x {
			collectRefs(val, seen)
		}
	}
}

// BigString renders a big.Int as a decimal string ("0" for nil). It is the one
// spelling a bound value and a read value share, so a saved balance compares
// equal to an asserted one.
func BigString(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}
