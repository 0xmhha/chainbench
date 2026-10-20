package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func exactChainID(args map[string]any) (int64, error) {
	value, present := args["chainId"]
	if !present {
		return 0, nil
	}
	bad := fmt.Errorf("chainId needs an exact nonnegative int64; use a decimal or hex string above the JSON safe range")
	switch value := value.(type) {
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 9007199254740991 || math.Trunc(value) != value {
			return 0, bad
		}
		return int64(value), nil
	case int:
		if value < 0 {
			return 0, bad
		}
		return int64(value), nil
	case int64:
		if value < 0 {
			return 0, bad
		}
		return value, nil
	case json.Number:
		n, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil || n < 0 {
			return 0, bad
		}
		return n, nil
	case string:
		base := 10
		if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
			base = 16
			value = value[2:]
		}
		n, err := strconv.ParseUint(value, base, 63)
		if err != nil {
			return 0, bad
		}
		return int64(n), nil
	default:
		return 0, bad
	}
}
