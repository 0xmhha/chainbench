package assert

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strings"
)

// Func is a two-argument assertion primitive, suitable for name dispatch.
type Func func(actual, expected any) (pass bool, detail string)

// Equal reports value equality. Two 0x-hex strings (addresses, hashes) compare
// exactly (use EqualCI for case-insensitive); otherwise integer-valued inputs
// (int, float, decimal string, or hex-vs-decimal, incl. large wei) compare
// numerically, booleans compare as booleans, and everything else falls back to
// deep equality.
func Equal(actual, expected any) (bool, string) {
	if !bothHexStrings(actual, expected) {
		if ai, ok := toBigInt(actual); ok {
			if ei, ok := toBigInt(expected); ok {
				return report(ai.Cmp(ei) == 0, actual, expected)
			}
		}
	}
	if ab, ok := actual.(bool); ok {
		if eb, ok := toBool(expected); ok {
			return report(ab == eb, actual, expected)
		}
	}
	return report(reflect.DeepEqual(actual, expected), actual, expected)
}

// bothHexStrings reports whether both values are 0x-prefixed hex strings, in
// which case Equal compares them exactly rather than as numbers (so addresses
// and hashes are not case-folded away).
func bothHexStrings(a, b any) bool { return isHexString(a) && isHexString(b) }

// isHexString reports whether v is a 0x-prefixed string.
func isHexString(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X")
}

// NotEqual is the negation of Equal.
func NotEqual(actual, expected any) (bool, string) {
	pass, _ := Equal(actual, expected)
	return report(!pass, actual, expected)
}

// EqualCI compares two values as case-insensitive strings (e.g. addresses).
func EqualCI(actual, expected any) (bool, string) {
	return report(strings.EqualFold(fmt.Sprint(actual), fmt.Sprint(expected)), actual, expected)
}

// Len reports whether actual's length equals expected.
func Len(actual, expected any) (bool, string) {
	n, ok := lengthOf(actual)
	if !ok {
		return false, fmt.Sprintf("value of type %T has no length", actual)
	}
	want, ok := toBigInt(expected)
	if !ok {
		return false, "Len expected must be an integer"
	}
	return report(big.NewInt(int64(n)).Cmp(want) == 0, n, expected)
}

// Greater reports actual > expected numerically.
func Greater(actual, expected any) (bool, string) {
	return compare(actual, expected, func(c int) bool { return c > 0 })
}

// GreaterOrEqual reports actual >= expected numerically.
func GreaterOrEqual(actual, expected any) (bool, string) {
	return compare(actual, expected, func(c int) bool { return c >= 0 })
}

// Less reports actual < expected numerically.
func Less(actual, expected any) (bool, string) {
	return compare(actual, expected, func(c int) bool { return c < 0 })
}

// LessOrEqual reports actual <= expected numerically.
func LessOrEqual(actual, expected any) (bool, string) {
	return compare(actual, expected, func(c int) bool { return c <= 0 })
}

// InDelta reports |actual-expected| <= tol as floats (e.g. reward +/- gas).
func InDelta(actual, expected, tol any) (bool, string) {
	a, ok1 := toFloat(actual)
	e, ok2 := toFloat(expected)
	d, ok3 := toFloat(tol)
	if !ok1 || !ok2 || !ok3 {
		return false, "InDelta needs numeric actual, expected, tol"
	}
	return report(math.Abs(a-e) <= d, actual, expected)
}

// Contains reports whether a string actual holds the expected substring, or a
// slice actual holds an element equal to expected.
func Contains(actual, expected any) (bool, string) {
	if s, ok := actual.(string); ok {
		return report(strings.Contains(s, fmt.Sprint(expected)), actual, expected)
	}
	rv := reflect.ValueOf(actual)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			if pass, _ := Equal(rv.Index(i).Interface(), expected); pass {
				return report(true, actual, expected)
			}
		}
		return report(false, actual, expected)
	}
	return false, fmt.Sprintf("Contains needs a string or slice, got %T", actual)
}

// NotContains is the negation of Contains.
func NotContains(actual, expected any) (bool, string) {
	pass, _ := Contains(actual, expected)
	return report(!pass, actual, expected)
}

// Regexp reports whether the expected pattern matches actual (stringified).
func Regexp(actual, expected any) (bool, string) {
	re, err := regexp.Compile(fmt.Sprint(expected))
	if err != nil {
		return false, "invalid regexp: " + err.Error()
	}
	return report(re.MatchString(fmt.Sprint(actual)), actual, expected)
}

// True reports whether actual is (or parses as) the boolean true.
func True(actual, _ any) (bool, string) {
	b, ok := toBool(actual)
	return report(ok && b, actual, true)
}

// False reports whether actual is (or parses as) the boolean false.
func False(actual, _ any) (bool, string) {
	b, ok := toBool(actual)
	return report(ok && !b, actual, false)
}

// Nil reports whether actual is nil (including a typed-nil pointer/map/slice).
func Nil(actual, _ any) (bool, string) { return report(isNil(actual), actual, nil) }

// NotNil is the negation of Nil.
func NotNil(actual, _ any) (bool, string) { return report(!isNil(actual), actual, nil) }

// ElementsMatch reports whether two slices hold the same multiset of elements
// regardless of order.
func ElementsMatch(actual, expected any) (bool, string) {
	as, ok1 := toSlice(actual)
	es, ok2 := toSlice(expected)
	if !ok1 || !ok2 {
		return false, "ElementsMatch needs two slices"
	}
	if len(as) != len(es) {
		return report(false, actual, expected)
	}
	used := make([]bool, len(es))
	for _, a := range as {
		found := false
		for j, e := range es {
			if used[j] {
				continue
			}
			if pass, _ := Equal(a, e); pass {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return report(false, actual, expected)
		}
	}
	return report(true, actual, expected)
}

// HashesEqual reports whether every hash is identical (cross-node no-fork check,
// backing EqualHashAt). Fewer than two hashes trivially pass.
func HashesEqual(hashes []string) (bool, string) {
	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			return false, fmt.Sprintf("hash divergence: %v", hashes)
		}
	}
	return true, ""
}

// funcs is the name -> primitive dispatch for two-argument assertions. InDelta
// (three args) and HashesEqual (a slice) are called directly by the interpreter.
var funcs = map[string]Func{
	"Equal":          Equal,
	"NotEqual":       NotEqual,
	"EqualCI":        EqualCI,
	"Len":            Len,
	"Greater":        Greater,
	"GreaterOrEqual": GreaterOrEqual,
	"Less":           Less,
	"LessOrEqual":    LessOrEqual,
	"Contains":       Contains,
	"NotContains":    NotContains,
	"Regexp":         Regexp,
	"True":           True,
	"False":          False,
	"Nil":            Nil,
	"NotNil":         NotNil,
	"ElementsMatch":  ElementsMatch,
}

// Lookup returns the assertion primitive registered under name.
func Lookup(name string) (Func, bool) {
	fn, ok := funcs[name]
	return fn, ok
}

// --- helpers ---

// report formats a mismatch detail, empty when pass.
func report(pass bool, actual, expected any) (bool, string) {
	if pass {
		return true, ""
	}
	return false, fmt.Sprintf("actual=%v expected=%v", actual, expected)
}

// compare applies pred to the numeric comparison of a and b.
func compare(a, b any, pred func(int) bool) (bool, string) {
	if ai, ok := toBigInt(a); ok {
		if bi, ok := toBigInt(b); ok {
			return report(pred(ai.Cmp(bi)), a, b)
		}
	}
	if af, ok := toFloat(a); ok {
		if bf, ok := toFloat(b); ok {
			c := 0
			switch {
			case af < bf:
				c = -1
			case af > bf:
				c = 1
			}
			return report(pred(c), a, b)
		}
	}
	return false, fmt.Sprintf("non-numeric comparison: %v, %v", a, b)
}

// toBigInt parses integer-valued inputs (int, int64, integral float64,
// json.Number, decimal or 0x-hex string) into a big.Int.
func toBigInt(v any) (*big.Int, bool) {
	switch x := v.(type) {
	case int:
		return big.NewInt(int64(x)), true
	case int64:
		return big.NewInt(x), true
	case float64:
		if x == math.Trunc(x) && !math.IsInf(x, 0) {
			bi, _ := big.NewFloat(x).Int(nil)
			return bi, true
		}
		return nil, false
	case json.Number:
		bi, ok := new(big.Int).SetString(x.String(), 10)
		return bi, ok
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil, false
		}
		base := 10
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			base, s = 16, s[2:]
		}
		bi, ok := new(big.Int).SetString(s, base)
		return bi, ok
	}
	return nil, false
}

// toFloat parses numeric inputs into a float64.
func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case string:
		if bi, ok := toBigInt(x); ok {
			f, _ := new(big.Float).SetInt(bi).Float64()
			return f, true
		}
	}
	return 0, false
}

// toBool parses a boolean from a bool or "true"/"false" string.
func toBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

// lengthOf returns len(v) for strings, slices, arrays, and maps.
func lengthOf(v any) (int, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return rv.Len(), true
	}
	return 0, false
}

// isNil reports whether v is nil, including typed-nil reference kinds.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}

// toSlice converts a slice/array value to []any.
func toSlice(v any) ([]any, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}
