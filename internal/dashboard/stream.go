package dashboard

import "github.com/0xmhha/chainbench/internal/core/collector"

// Stream opens the bus that carries orchestration events to a running
// chainbench-dashboard, and the function that drains it.
//
// It lives here because every surface that streams needs the same two halves
// and the same teardown: the bus has to be closed and the forwarder drained
// before the caller returns, or the last events of a run are lost. A surface
// assembling this itself is one that can get the teardown subtly wrong.
//
// With no URL there is nothing to carry, so there is no bus. Emission is not
// free, and a run that streams nowhere should not pay for it.
func Stream(url string) (*collector.Bus, func()) {
	if url == "" {
		return nil, func() {}
	}
	bus := collector.NewBus()
	done := Forward(bus, url, nil)
	return bus, func() {
		bus.Close()
		<-done
	}
}
