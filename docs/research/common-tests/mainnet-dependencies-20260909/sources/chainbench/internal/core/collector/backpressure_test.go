package collector

import (
	"testing"
	"time"
)

// TestBus_DropsWhenFull asserts the bus applies backpressure by dropping — never
// blocking — when a subscriber falls behind, and counts the drops (O7).
func TestBus_DropsWhenFull(t *testing.T) {
	b := NewBus()
	_ = b.Subscribe() // never drained

	const overflow = 50
	total := subBuffer + overflow

	done := make(chan struct{})
	go func() {
		for i := 0; i < total; i++ {
			b.Publish(Event{Message: "e"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a full subscriber (no backpressure)")
	}

	if got := b.Dropped(); got != overflow {
		t.Fatalf("Dropped() = %d, want %d (buffer %d, published %d)", got, overflow, subBuffer, total)
	}
}
