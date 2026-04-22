package events

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

func newTestBus() (*Bus, *bytes.Buffer) {
	var buf bytes.Buffer
	emitter := wire.NewEmitter(&buf)
	bus := NewBus(emitter)
	return bus, &buf
}

func TestBus_PublishDeliversToSubscribers(t *testing.T) {
	bus, _ := newTestBus()
	defer bus.Close()
	sub1 := bus.Subscribe()
	sub2 := bus.Subscribe()
	ev := Event{Name: types.EventName("chain.block"), Data: map[string]any{"height": 42}}
	if err := bus.Publish(ev); err != nil {
		t.Fatalf("publish: %v", err)
	}
	for i, ch := range []<-chan Event{sub1, sub2} {
		select {
		case got := <-ch:
			if got.Name != ev.Name {
				t.Errorf("sub %d name: got %q, want %q", i, got.Name, ev.Name)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("sub %d timeout", i)
		}
	}
}

func TestBus_PublishAlsoEmitsToWire(t *testing.T) {
	bus, buf := newTestBus()
	defer bus.Close()
	ev := Event{Name: types.EventName("chain.block"), Data: map[string]any{"h": 1}}
	if err := bus.Publish(ev); err != nil {
		t.Fatalf("publish: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"type":"event"`) {
		t.Errorf("wire output missing type:event: %q", out)
	}
	if !strings.Contains(out, `"name":"chain.block"`) {
		t.Errorf("wire output missing name: %q", out)
	}
}

func TestBus_NonBlockingDropWhenSubscriberFull(t *testing.T) {
	bus, _ := newTestBus()
	defer bus.Close()
	sub := bus.Subscribe() // buffer = DefaultSubscriberBuffer (16)
	// Publish 100 events without draining; should not block.
	start := time.Now()
	for i := 0; i < 100; i++ {
		if err := bus.Publish(Event{Name: types.EventName("chain.block")}); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Errorf("publish blocked unexpectedly (%.2fms)", float64(elapsed.Microseconds())/1000.0)
	}
	// Consume what fits in the buffer to confirm drops occurred (received < published).
	received := 0
drain:
	for {
		select {
		case <-sub:
			received++
		case <-time.After(50 * time.Millisecond):
			break drain
		}
	}
	if received >= 100 {
		t.Errorf("expected drops, got all %d events", received)
	}
}

func TestBus_CloseRejectsSubsequentPublish(t *testing.T) {
	bus, _ := newTestBus()
	if err := bus.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	err := bus.Publish(Event{Name: types.EventName("chain.block")})
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("post-close publish: got %v, want ErrBusClosed", err)
	}
}

func TestBus_CloseClosesSubscriberChannels(t *testing.T) {
	bus, _ := newTestBus()
	sub := bus.Subscribe()
	if err := bus.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// Channel should close (recv returns zero value, ok=false).
	select {
	case _, ok := <-sub:
		if ok {
			t.Error("expected channel closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("subscriber channel not closed after bus.Close")
	}
}

func TestBus_ConcurrentPublish_RaceSafe(t *testing.T) {
	bus, _ := newTestBus()
	defer bus.Close()
	bus.Subscribe() // one passive sub

	const goroutines = 50
	const perGoroutine = 20
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				_ = bus.Publish(Event{Name: types.EventName("chain.block")})
			}
		}()
	}
	wg.Wait()
}
