// Package abiutil isolates JSON-ABI parse / arg coercion / method pack /
// result unpack / event decode helpers used by Sprint 4d's contract / event
// commands. Caller-facing JSON args (decimal/hex strings, json numbers, hex
// addresses, hex []byte, bool, string) are coerced to the Go types
// go-ethereum's abi package expects before Pack. Tuple, nested array, and
// fixed-bytesN (N != 32) are explicitly out of scope — callers that need
// those features must pre-encode and pass raw calldata.
package abiutil

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ParseABI parses a JSON ABI string. Wraps abi.JSON for clearer error context.
func ParseABI(jsonStr string) (abi.ABI, error) {
	parsed, err := abi.JSON(strings.NewReader(jsonStr))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("abiutil.ParseABI: %w", err)
	}
	return parsed, nil
}

// CoerceArgs converts JSON-decoded `[]any` into the Go types each abi.Argument
// expects. Best-effort scalar mapping per spec — tuple / nested array /
// fixed-bytesN (N != 32) are out of scope.
func CoerceArgs(inputs abi.Arguments, jsonArgs []any) ([]any, error) {
	if len(inputs) != len(jsonArgs) {
		return nil, fmt.Errorf("abiutil.CoerceArgs: expected %d args, got %d", len(inputs), len(jsonArgs))
	}
	out := make([]any, len(jsonArgs))
	for i, arg := range inputs {
		v, err := coerceOne(arg.Type, jsonArgs[i])
		if err != nil {
			return nil, fmt.Errorf("abiutil.CoerceArgs[%d] (%s): %w", i, arg.Type.String(), err)
		}
		out[i] = v
	}
	return out, nil
}

func coerceOne(t abi.Type, v any) (any, error) {
	switch t.T {
	case abi.UintTy, abi.IntTy:
		return toBigInt(v)
	case abi.AddressTy:
		return toAddress(v)
	case abi.BytesTy:
		return toBytes(v)
	case abi.FixedBytesTy:
		if t.Size == 32 {
			b, err := toBytes(v)
			if err != nil {
				return nil, err
			}
			if len(b) != 32 {
				return nil, fmt.Errorf("bytes32 requires 32 bytes, got %d", len(b))
			}
			var out [32]byte
			copy(out[:], b)
			return out, nil
		}
		return nil, fmt.Errorf("fixed-bytes%d not supported (only bytes32); use raw calldata", t.Size)
	case abi.BoolTy:
		if b, ok := v.(bool); ok {
			return b, nil
		}
		return nil, fmt.Errorf("expected bool, got %T", v)
	case abi.StringTy:
		if s, ok := v.(string); ok {
			return s, nil
		}
		return nil, fmt.Errorf("expected string, got %T", v)
	default:
		return nil, fmt.Errorf("type %s not supported (use raw calldata)", t.String())
	}
}

func toBigInt(v any) (*big.Int, error) {
	switch x := v.(type) {
	case string:
		if strings.HasPrefix(x, "0x") {
			i, ok := new(big.Int).SetString(strings.TrimPrefix(x, "0x"), 16)
			if !ok {
				return nil, fmt.Errorf("bad hex int: %q", x)
			}
			return i, nil
		}
		i, ok := new(big.Int).SetString(x, 10)
		if !ok {
			return nil, fmt.Errorf("bad decimal int: %q", x)
		}
		return i, nil
	case float64:
		// JSON numbers decode to float64 by default; require integral value
		// to avoid silent truncation when the caller misreads the spec and
		// passes a fractional value for a uint type.
		i := int64(x)
		if float64(i) != x {
			return nil, fmt.Errorf("non-integral number: %v", x)
		}
		return big.NewInt(i), nil
	case json.Number:
		i, ok := new(big.Int).SetString(string(x), 10)
		if !ok {
			return nil, fmt.Errorf("bad json.Number: %q", x)
		}
		return i, nil
	}
	return nil, fmt.Errorf("expected number/string, got %T", v)
}

func toAddress(v any) (common.Address, error) {
	s, ok := v.(string)
	if !ok {
		return common.Address{}, fmt.Errorf("expected hex string, got %T", v)
	}
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("not a valid hex address: %q", s)
	}
	return common.HexToAddress(s), nil
}

func toBytes(v any) ([]byte, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("expected hex string, got %T", v)
	}
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return []byte{}, nil
	}
	return hex.DecodeString(s)
}

// PackMethodCall ABI-encodes a method call: 4-byte selector + packed args.
func PackMethodCall(parsed abi.ABI, methodName string, jsonArgs []any) ([]byte, error) {
	method, ok := parsed.Methods[methodName]
	if !ok {
		return nil, fmt.Errorf("abiutil.PackMethodCall: method %q not in ABI", methodName)
	}
	coerced, err := CoerceArgs(method.Inputs, jsonArgs)
	if err != nil {
		return nil, err
	}
	return parsed.Pack(methodName, coerced...)
}

// UnpackMethodResult decodes the return data of a method call into
// JSON-friendly values.
func UnpackMethodResult(parsed abi.ABI, methodName string, data []byte) ([]any, error) {
	return parsed.Unpack(methodName, data)
}

// PackConstructor encodes constructor args (no selector prefix). Empty args
// returns empty bytes — go-ethereum's abi.Pack("") encodes the constructor
// when the method name is empty, and yields an empty result for a
// constructor with no inputs.
func PackConstructor(parsed abi.ABI, jsonArgs []any) ([]byte, error) {
	if len(jsonArgs) == 0 {
		return parsed.Pack("")
	}
	coerced, err := CoerceArgs(parsed.Constructor.Inputs, jsonArgs)
	if err != nil {
		return nil, err
	}
	return parsed.Pack("", coerced...)
}

// DecodeLog parses log topics+data via an event ABI definition. Returns a
// JSON-friendly map keyed by argument name. Indexed args come from
// log.Topics[1..]; non-indexed args come from log.Data.
func DecodeLog(parsedABI abi.ABI, eventName string, log types.Log) (map[string]any, error) {
	event, ok := parsedABI.Events[eventName]
	if !ok {
		return nil, fmt.Errorf("abiutil.DecodeLog: event %q not in ABI", eventName)
	}
	out := map[string]any{}
	if err := parsedABI.UnpackIntoMap(out, eventName, log.Data); err != nil {
		return nil, fmt.Errorf("abiutil.DecodeLog data: %w", err)
	}
	// Indexed args come from topics[1..]
	var indexed abi.Arguments
	for _, a := range event.Inputs {
		if a.Indexed {
			indexed = append(indexed, a)
		}
	}
	if len(indexed) > 0 && len(log.Topics) >= 1+len(indexed) {
		if err := abi.ParseTopicsIntoMap(out, indexed, log.Topics[1:]); err != nil {
			return nil, fmt.Errorf("abiutil.DecodeLog topics: %w", err)
		}
	}
	return out, nil
}
