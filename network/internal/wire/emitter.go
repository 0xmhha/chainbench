package wire

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// ErrStreamClosed is returned when Emit* is called after the stream has been
// terminated by EmitResult or EmitResultError.
var ErrStreamClosed = errors.New("wire: stream already closed by result")

// Emitter writes NDJSON stream messages to an underlying io.Writer.
// Concurrent use by multiple goroutines is safe; each emitted line is
// serialized under an internal mutex.
//
// After EmitResult/EmitResultError, further Emit* calls return ErrStreamClosed.
type Emitter struct {
	mu     sync.Mutex
	enc    *json.Encoder
	clock  func() time.Time
	closed bool
}

// NewEmitter creates an Emitter that writes to w and uses time.Now for timestamps.
func NewEmitter(w io.Writer) *Emitter {
	return NewEmitterWithClock(w, time.Now)
}

// NewEmitterWithClock creates an Emitter that writes to w and obtains event
// timestamps by calling clock(). Useful for deterministic tests.
func NewEmitterWithClock(w io.Writer, clock func() time.Time) *Emitter {
	return &Emitter{
		enc:   json.NewEncoder(w),
		clock: clock,
	}
}

func (e *Emitter) EmitEvent(name types.EventName, data map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ErrStreamClosed
	}
	msg := map[string]any{
		"type": "event",
		"name": string(name),
		"ts":   e.clock().UTC().Format(time.RFC3339Nano),
	}
	if data != nil {
		msg["data"] = data
	}
	return e.enc.Encode(msg)
}

// EmitProgress emits a progress message. total must be >= 1 and done must be
// in [0, total], matching the event.json schema constraints.
func (e *Emitter) EmitProgress(step string, done, total int) error {
	if total < 1 {
		return fmt.Errorf("wire: EmitProgress: total must be >= 1, got %d", total)
	}
	if done < 0 || done > total {
		return fmt.Errorf("wire: EmitProgress: done must be in [0, %d], got %d", total, done)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ErrStreamClosed
	}
	msg := map[string]any{
		"type":  "progress",
		"step":  step,
		"done":  done,
		"total": total,
	}
	return e.enc.Encode(msg)
}

func (e *Emitter) EmitResult(ok bool, data map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ErrStreamClosed
	}
	if !ok {
		// Guard: callers must use EmitResultError for ok=false.
		return errors.New("wire: EmitResult requires ok=true; use EmitResultError for failures")
	}
	msg := map[string]any{
		"type": "result",
		"ok":   true,
	}
	if data != nil {
		msg["data"] = data
	}
	e.closed = true
	return e.enc.Encode(msg)
}

func (e *Emitter) EmitResultError(code types.ResultErrorCode, message string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ErrStreamClosed
	}
	msg := map[string]any{
		"type": "result",
		"ok":   false,
		"error": map[string]any{
			"code":    string(code),
			"message": message,
		},
	}
	e.closed = true
	return e.enc.Encode(msg)
}
