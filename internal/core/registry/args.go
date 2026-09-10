package registry

import (
	"math/big"
	"strconv"
)

// This file is the argument decoding for a capability or tool call. Both
// surfaces that invoke a capability hand it the same shape — a JSON object
// decoded into map[string]any — so the rules for reading one value out of it (a
// JSON number arrives as float64, a numeric string is accepted, an absent or
// wrong-typed key falls back to a default) belong to one package. They lived
// here and in a private copy inside internal/mcp; the surface now delegates.

// ArgString returns a string argument, or def if absent/not a string.
func ArgString(args map[string]any, key, def string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

// ArgInt returns an integer argument (JSON numbers decode as float64; a numeric
// string is also accepted), or def.
func ArgInt(args map[string]any, key string, def int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// ArgBigInt returns a decimal-string argument parsed as a big.Int, or nil if
// absent/unparseable.
func ArgBigInt(args map[string]any, key string) *big.Int {
	s := ArgString(args, key, "")
	if s == "" {
		return nil
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil
	}
	return n
}

// ArgStrings returns a []string argument (a JSON array of strings), or nil.
// A non-string element is skipped rather than failing the call: the schema is
// what rejects a malformed argument, and a decoder that panics on one bad
// element would turn a validation problem into a crash.
func ArgStrings(args map[string]any, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// ArgBool returns a boolean argument, or def if absent/not a boolean.
func ArgBool(args map[string]any, key string, def bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return def
}
