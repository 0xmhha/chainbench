package collector

import (
	"sync"
	"time"
)

// subBuffer bounds each subscriber's backlog. A subscriber that falls behind
// past this many events has further events dropped (counted in Dropped) rather
// than blocking the publisher — observation must never stall the pipeline.
const subBuffer = 256

// Bus is an in-process publish/subscribe event bus. It is safe for concurrent
// use. Publishers call Publish; consumers (dashboard streamer, log sink, test
// reporter) call Subscribe and drain the returned channel.
type Bus struct {
	mu      sync.Mutex
	subs    []chan Event
	closed  bool
	dropped uint64
	now     func() time.Time // injectable clock for tests
}

// NewBus returns an empty bus.
func NewBus() *Bus {
	return &Bus{now: time.Now}
}

// Subscribe returns a new channel receiving every event published after the
// call. The channel is closed when the bus is closed. Drain it promptly;
// events beyond the buffer are dropped, not blocked.
func (b *Bus) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, subBuffer)
	if b.closed {
		close(ch)
		return ch
	}
	b.subs = append(b.subs, ch)
	return ch
}

// Publish delivers e to all current subscribers. A zero e.Time is stamped with
// the bus clock. Delivery to a full subscriber is skipped and counted.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	if e.Time.IsZero() {
		e.Time = b.now()
	}
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default:
			b.dropped++
		}
	}
}

// Dropped returns the number of events dropped due to full subscriber buffers.
func (b *Bus) Dropped() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.dropped
}

// Close closes all subscriber channels and rejects further Publish/Subscribe.
// It is idempotent.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, ch := range b.subs {
		close(ch)
	}
	b.subs = nil
}
