package wire

import (
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// Message is a sealed interface implemented by EventMessage, ProgressMessage,
// and ResultMessage. Consumers use a type switch to handle each case.
type Message interface {
	isMessage()
}

// EventMessage wraps a decoded non-terminator event line.
type EventMessage struct{ types.Event }

// ProgressMessage wraps a decoded progress line.
type ProgressMessage struct{ types.Progress }

// ResultMessage wraps a decoded result (terminator) line.
//
// Note: types.Result is defined as interface{} in the generated types package,
// so ResultMessage carries explicit fields rather than embedding that type.
type ResultMessage struct {
	Ok    bool               `json:"ok"`
	Data  types.ResultData   `json:"data,omitempty"`
	Error *types.ResultError `json:"error,omitempty"`
}

func (EventMessage) isMessage()    {}
func (ProgressMessage) isMessage() {}
func (ResultMessage) isMessage()   {}

// DecodeMessage parses one NDJSON line into a Message. Dispatches on the
// "type" discriminator field. Returns an error for unknown type or malformed JSON.
func DecodeMessage(line []byte) (Message, error) {
	var hdr struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &hdr); err != nil {
		return nil, fmt.Errorf("wire: decode header: %w", err)
	}
	switch hdr.Type {
	case "event":
		var ev types.Event
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, fmt.Errorf("wire: decode event: %w", err)
		}
		return EventMessage{ev}, nil
	case "progress":
		var p types.Progress
		if err := json.Unmarshal(line, &p); err != nil {
			return nil, fmt.Errorf("wire: decode progress: %w", err)
		}
		return ProgressMessage{p}, nil
	case "result":
		var r ResultMessage
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("wire: decode result: %w", err)
		}
		return r, nil
	default:
		return nil, fmt.Errorf("wire: unknown message type %q", hdr.Type)
	}
}
