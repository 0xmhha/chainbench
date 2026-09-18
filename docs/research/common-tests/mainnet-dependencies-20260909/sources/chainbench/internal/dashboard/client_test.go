package dashboard_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/dashboard"
)

func TestForward(t *testing.T) {
	var mu sync.Mutex
	var got []string
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e collector.Event
		_ = json.NewDecoder(r.Body).Decode(&e)
		mu.Lock()
		got = append(got, e.Message)
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer sink.Close()

	bus := collector.NewBus()
	done := dashboard.Forward(bus, sink.URL, sink.Client())

	bus.Publish(collector.Event{Phase: collector.PhaseSetup, Message: "a"})
	bus.Publish(collector.Event{Phase: collector.PhaseVerify, Message: "b"})
	bus.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("forwarder did not finish")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("forwarded events: %v", got)
	}
}
