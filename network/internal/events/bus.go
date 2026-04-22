package events

import (
	"errors"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

// ErrBusClosed is returned by Publish after Close has been called.
var ErrBusClosed = errors.New("events: bus closed")

// DefaultSubscriberBuffer is the channel capacity for each Subscribe() result.
const DefaultSubscriberBuffer = 16

// Event is the in-process representation of a published event.
// The corresponding wire-level message is emitted by Publish via the
// bound wire.Emitter.
type Event struct {
	Name types.EventName
	Data map[string]any
	TS   time.Time
}

// Bus is a non-blocking pub/sub that also mirrors events to a wire.Emitter.
type Bus struct {
	mu      sync.RWMutex
	subs    []chan Event
	emitter *wire.Emitter
	clock   func() time.Time
	closed  bool
}

// NewBus creates a Bus that delegates wire emission to emitter.
func NewBus(emitter *wire.Emitter) *Bus {
	return &Bus{
		emitter: emitter,
		clock:   time.Now,
	}
}

// Publish fans out ev to in-process subscribers (non-blocking; dropped for any
// subscriber whose buffer is full) and writes it to the bound wire.Emitter.
// Returns ErrBusClosed after Close().
func (b *Bus) Publish(ev Event) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}
	// Fan-out under read lock (subs slice only mutated under write lock).
	for _, ch := range b.subs {
		select {
		case ch <- ev:
		default:
			// drop: subscriber is slow or not draining
		}
	}
	b.mu.RUnlock()

	// Emit outside the lock — wire.Emitter has its own mutex.
	return b.emitter.EmitEvent(ev.Name, ev.Data)
}

// Subscribe returns a new buffered channel that will receive published events
// until Close is called.
func (b *Bus) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, DefaultSubscriberBuffer)
	b.subs = append(b.subs, ch)
	return ch
}

// Close closes all subscriber channels and rejects subsequent Publish calls.
// Safe to call multiple times.
func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	for _, ch := range b.subs {
		close(ch)
	}
	b.subs = nil
	return nil
}
