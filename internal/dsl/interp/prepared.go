package interp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// ClonePreparedValue copies public data while preserving numbers and binding
// types. Private byte buffers, live handles and sensitive fields are refused.
func ClonePreparedValue(value any) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("interp: prepared result is not serializable public data: %w", err)
	}
	if !bytes.Equal(raw, session.Scrub(raw)) {
		return nil, fmt.Errorf("interp: prepared result contains sensitive fields")
	}
	return clonePrepared(value)
}

func clonePrepared(value any) (any, error) {
	switch v := value.(type) {
	case nil, string, bool, json.Number, float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return v, nil
	case *big.Int:
		if v == nil {
			return (*big.Int)(nil), nil
		}
		return new(big.Int).Set(v), nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for name, child := range v {
			copied, err := clonePrepared(child)
			if err != nil {
				return nil, err
			}
			out[name] = copied
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			copied, err := clonePrepared(child)
			if err != nil {
				return nil, err
			}
			out[i] = copied
		}
		return out, nil
	default:
		return nil, fmt.Errorf("interp: prepared result type %T is not public binding data", value)
	}
}

func importPrepared(s dsl.Spec, binds Bindings) error {
	clear(s.PreparedOutputs)
	if s.Prepared == nil {
		return nil
	}
	for name := range s.Prepared.Imports {
		value, found := s.PreparedInputs[name]
		if !found {
			return fmt.Errorf("interp: prepared input %q is unavailable", name)
		}
		if _, exists := binds[name]; exists {
			return fmt.Errorf("interp: prepared input %q conflicts with a preset value", name)
		}
		copied, err := ClonePreparedValue(value)
		if err != nil {
			return err
		}
		binds[name] = copied
	}
	return nil
}

func exportPrepared(s dsl.Spec, binds Bindings) error {
	if s.Prepared == nil || len(s.Prepared.Exports) == 0 {
		return nil
	}
	if s.PreparedOutputs == nil {
		return fmt.Errorf("interp: prepared result collector is unavailable")
	}
	outputs := make(map[string]any, len(s.Prepared.Exports))
	for name, ref := range s.Prepared.Exports {
		binding := dsl.PreparedBinding(ref)
		value, found := binds[binding]
		if binding == "" || !found {
			return fmt.Errorf("interp: prepared export %q refers to an unbound result", name)
		}
		copied, err := ClonePreparedValue(value)
		if err != nil {
			return fmt.Errorf("interp: prepared export %q: %w", name, err)
		}
		outputs[name] = copied
	}
	for name, value := range outputs {
		s.PreparedOutputs[name] = value
	}
	return nil
}
