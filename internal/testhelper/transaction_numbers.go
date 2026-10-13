package testhelper

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
)

// transactionNumber retains exact nonnegative integer quantities. JSON numbers
// beyond JavaScript's safe range must be supplied as decimal or hex strings.
func transactionNumber(value any) (*big.Int, bool) {
	n := new(big.Int)
	switch v := value.(type) {
	case string:
		text := strings.TrimSpace(v)
		base := 10
		if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
			text = text[2:]
			base = 16
		}
		if text == "" {
			return nil, false
		}
		digits := "0123456789"
		if base == 16 {
			digits += "abcdefABCDEF"
		}
		for _, c := range text {
			if !strings.ContainsRune(digits, c) {
				return nil, false
			}
		}
		if _, ok := n.SetString(text, base); !ok {
			return nil, false
		}
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || math.Trunc(v) != v || v > 9007199254740991 {
			return nil, false
		}
		n.SetUint64(uint64(v))
	case int:
		n.SetInt64(int64(v))
	case int64:
		n.SetInt64(v)
	case uint64:
		n.SetUint64(v)
	case json.Number:
		if !json.Valid([]byte(v)) {
			return nil, false
		}
		r, ok := new(big.Rat).SetString(string(v))
		if !ok || !r.IsInt() {
			return nil, false
		}
		n.Set(r.Num())
	case *big.Int:
		if v == nil {
			return nil, false
		}
		n.Set(v)
	default:
		return nil, false
	}
	return n, n.Sign() >= 0
}

func validateQuantity(args map[string]any, field string, bits int) error {
	value, supplied := args[field]
	if !supplied {
		return nil
	}
	n, ok := transactionNumber(value)
	if !ok {
		return fmt.Errorf("dsl: bad %s: expected a nonnegative integral quantity; use decimal/hex strings for large values", field)
	}
	if bits > 0 && n.BitLen() > bits {
		return fmt.Errorf("dsl: bad %s: quantity exceeds %d bits", field, bits)
	}
	return nil
}

func validateTransactionNumbers(args map[string]any) error {
	if err := validateQuantity(args, "gas", 64); err != nil {
		return err
	}
	if err := validateQuantity(args, "nonce", 64); err != nil {
		return err
	}
	if err := validateQuantity(args, "gasPrice", 0); err != nil {
		return err
	}
	if err := validateQuantity(args, "maxFeePerGas", 0); err != nil {
		return err
	}
	return validateQuantity(args, "maxPriorityFeePerGas", 0)
}
