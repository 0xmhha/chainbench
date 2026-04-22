package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// allowedCommandFields enumerates the top-level keys permitted in a command
// envelope. Any key outside this set causes DecodeCommand to fail.
var allowedCommandFields = map[string]struct{}{
	"command": {},
	"args":    {},
	"env":     {},
}

// DecodeCommand reads one JSON command envelope from r and returns
// a validated Command. Unknown top-level fields are rejected.
// Schema-level validation (enum membership, required fields) is
// already enforced by the generated types.Command.UnmarshalJSON.
func DecodeCommand(r io.Reader) (*types.Command, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("wire: command envelope: %w", err)
	}
	// First pass: reject unknown top-level fields. The generated
	// types.Command.UnmarshalJSON does not honor DisallowUnknownFields
	// because it delegates to json.Unmarshal internally, so we enforce
	// the constraint explicitly here.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("wire: command envelope: %w", err)
	}
	if probe == nil {
		// JSON "null" unmarshals to a nil map without error; the generated
		// Command.UnmarshalJSON also skips required-field validation in that
		// case. Reject explicitly so callers cannot receive a zero-value Command.
		return nil, fmt.Errorf("wire: command envelope: expected JSON object, got null")
	}
	for key := range probe {
		if _, ok := allowedCommandFields[key]; !ok {
			return nil, fmt.Errorf("wire: command envelope: unknown field %q", key)
		}
	}
	// Second pass: full decode with strict unknown-field checks. This
	// engages the generated UnmarshalJSON for enum/required validation.
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var cmd types.Command
	if err := dec.Decode(&cmd); err != nil {
		return nil, fmt.Errorf("wire: command envelope: %w", err)
	}
	return &cmd, nil
}
