package capability

import (
	"math/big"
	"strconv"
)

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
