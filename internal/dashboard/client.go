package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/obs"
)

// Forward subscribes to bus and POSTs every event to a running chainbenchd's
// /api/events, so a CLI (or any producer) can feed the dashboard live. It
// returns a channel closed when forwarding finishes (after the bus is closed
// and its buffered events are drained), so callers can flush before exiting.
func Forward(bus *obs.Bus, dashboardURL string, client *http.Client) <-chan struct{} {
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := strings.TrimRight(dashboardURL, "/") + "/api/events"
	sub := bus.Subscribe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for e := range sub {
			b, err := json.Marshal(e)
			if err != nil {
				continue
			}
			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			if resp, err := client.Do(req); err == nil {
				_ = resp.Body.Close()
			}
		}
	}()
	return done
}
