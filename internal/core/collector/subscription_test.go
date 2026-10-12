package collector

import (
	"sync"
	"testing"
)

func TestCancelledSubscriberDoesNotAccumulateDeliveryDrops(t *testing.T) {
	bus := NewBus()
	sub, cancel := bus.SubscribeWithCancel()
	cancel()
	cancel()
	if _, ok := <-sub; ok {
		t.Fatal("subscription remains open")
	}
	for i := 0; i < 1024; i++ {
		bus.Publish(Event{Message: "after disconnect"})
	}
	if bus.Dropped() != 0 {
		t.Fatal("disconnected subscriber counted as a delivery drop")
	}
	bus.Close()
	cancel()
}

func TestConcurrentUnsubscribeAndPublication(t *testing.T) {
	bus := NewBus()
	defer bus.Close()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		_, cancel := bus.SubscribeWithCancel()
		wg.Go(func() { cancel() })
		wg.Go(func() {
			for n := 0; n < 100; n++ {
				bus.Publish(Event{})
			}
		})
	}
	wg.Wait()
}
